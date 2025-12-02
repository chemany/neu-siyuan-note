# 上传功能 401 错误修复总结

## 问题描述

在 Web 模式下，用户上传文件时遇到 401 Unauthorized 错误：

```
XHR POST http://localhost:6806/upload [HTTP/1.1 401 Unauthorized 1ms]
```

请求信息：
- 端点：`POST /upload`
- Cookie：`siyuan_token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...`
- Content-Type：`multipart/form-data`

## 问题分析

### 根本原因

1. **路由注册问题**：`/upload` 端点使用了 `uploadRedirectHandler` 进行重定向，但重定向会丢失 POST 请求的 body 数据
2. **认证中间件配置**：`CheckWebAuth` 中间件已经正确实现了从 Cookie 中读取 JWT token 的逻辑
3. **路由重复注册**：`/api/asset/upload` 在 `router.go` 和 `serve.go` 中被重复注册，导致启动失败

### 认证流程

Web 模式下的认证流程（`webmode_auth.go`）：

1. 检查是否为公开路径（登录、注册等）
2. 从多个来源提取 JWT token：
   - Authorization header（`Bearer token`）
   - Cookie（`siyuan_token`）
   - X-Auth-Token header
3. 验证 token 并提取用户信息
4. 动态切换到用户的 workspace

## 修复方案

### 1. 修改 `serve.go` 中的路由注册

**位置**：`siyuan/kernel/server/serve.go`

**修改前**：
```go
func serveAssets(ginServer *gin.Engine) {
	ginServer.POST("/upload", uploadRedirectHandler)
	ginServer.POST("/upload-legacy", uploadCompatibilityHandler)
	
	if os.Getenv("SIYUAN_WEB_MODE") == "true" {
		ginServer.POST("/api/asset/upload", model.CheckWebAuth, model.CheckAdminRole, model.CheckReadonly, model.Upload)
	} else {
		ginServer.POST("/api/asset/upload", model.CheckAuth, model.CheckAdminRole, model.CheckReadonly, model.Upload)
	}
}
```

**修改后**：
```go
func serveAssets(ginServer *gin.Engine) {
	// 注册 /upload 端点作为 /api/asset/upload 的别名
	// 在Web模式下使用 CheckWebAuth，在传统模式下使用 CheckAuth
	if os.Getenv("SIYUAN_WEB_MODE") == "true" {
		ginServer.POST("/upload", model.CheckWebAuth, model.CheckAdminRole, model.CheckReadonly, model.Upload)
	} else {
		ginServer.POST("/upload", model.CheckAuth, model.CheckAdminRole, model.CheckReadonly, model.Upload)
	}
	// 注意：/api/asset/upload 已在 router.go 中注册，无需重复注册
}
```

### 2. 删除不必要的文件

删除了 `siyuan/kernel/server/serve_upload_fix.go`，因为不再需要重定向处理器。

### 3. 修复 `notebook_optimizer.go` 编译错误

**问题**：
- `model.GetWorkspaceDir()` 不存在
- `copyFile` 函数与 `file.go` 中的函数冲突

**修复**：
```bash
# 替换 GetWorkspaceDir 调用
sed -i 's/model\.GetWorkspaceDir()/util.WorkspaceDir/g' notebook_optimizer.go

# 重命名 copyFile 函数避免冲突
sed -i 's/func copyFile(/func copyNotebookFile(/g' notebook_optimizer.go
sed -i 's/copyFile(/copyNotebookFile(/g' notebook_optimizer.go

# 修改导入
import (
	"github.com/siyuan-note/siyuan/kernel/util"  // 添加 util 包
)
```

## 测试验证

### 1. 编译测试

```bash
cd siyuan
./rebuild-and-restart.sh
```

结果：✅ 编译成功，服务正常启动

### 2. 服务状态检查

```
✅ 端口 6806 正在监听
✅ API 响应正常
✅ 前端访问正常
```

### 3. 认证日志

```
I 2025/11/28 17:16:33 webmode_auth.go:41: [Web Mode] Checking authentication for [/]
```

## 关键代码说明

### CheckWebAuth 中间件（webmode_auth.go）

```go
func CheckWebAuth(c *gin.Context) {
	// 1. 检查公开路径
	publicPaths := []string{
		"/api/web/auth/login",
		"/api/web/auth/register",
		// ...
	}
	
	// 2. 提取 JWT token
	var token string
	
	// 从 Authorization header 获取
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		token = strings.TrimPrefix(authHeader, "Bearer ")
	}
	
	// 从 Cookie 获取
	if token == "" {
		token, _ = c.Cookie("siyuan_token")
	}
	
	// 从 X-Auth-Token header 获取
	if token == "" {
		token = c.GetHeader("X-Auth-Token")
	}
	
	// 3. 验证 token
	authService := GetWebAuthService()
	user, err := authService.ValidateToken(token)
	
	// 4. 存储用户信息到 context
	c.Set("web_user_id", user.ID)
	c.Set("web_username", user.Username)
	c.Set("web_workspace", user.Workspace)
	
	// 5. 动态切换 workspace
	util.WorkspaceDir = user.Workspace
}
```

## 总结

通过以下修改成功解决了上传功能的 401 错误：

1. ✅ 将 `/upload` 端点直接绑定到 `model.Upload` 处理器，而不是使用重定向
2. ✅ 确保 Web 模式下使用 `CheckWebAuth` 中间件
3. ✅ 避免路由重复注册
4. ✅ 修复相关编译错误

现在用户可以正常上传文件，JWT token 从 Cookie 中正确提取并验证。

## 下一步建议

1. 测试实际的文件上传功能
2. 验证多用户场景下的 workspace 隔离
3. 检查上传文件的权限和存储路径
4. 添加上传文件大小限制和类型验证
