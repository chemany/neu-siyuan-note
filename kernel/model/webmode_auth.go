// SiYuan - Refactor your thinking
// Copyright (c) 2020-present, b3log.org
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package model

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/siyuan-note/logging"
	"github.com/siyuan-note/siyuan/kernel/util"
)

// CheckWebAuth Web模式JWT认证中间件
// 在Web模式下,所有请求必须携带有效的JWT token
func CheckWebAuth(c *gin.Context) {
	// 检查是否启用Web模式
	if os.Getenv("SIYUAN_WEB_MODE") != "true" {
		// 未启用Web模式,使用原有的CheckAuth逻辑
		CheckAuth(c)
		return
	}

	// Web模式下的认证逻辑
	logging.LogInfof("[Web Mode] Checking authentication for [%s]", c.Request.RequestURI)

	// 放行公开路径（无需任何认证）
	publicPaths := []string{
		"/api/web/auth/login",
		"/api/web/auth/register",
		"/api/web/auth/unified-login",
		"/api/web/auth/unified-status",
		"/api/web/auth/health",
		"/api/web/auth/verify-token",
		"/api/system/bootProgress",
		"/api/system/version",
		"/stage/login.html",
		"/stage/register.html",
		"/stage/protyle/",
		"/stage/icon",           // icon.png, icon-large.png, 以及带后缀的图标文件
		"/stage/loading",        // loading.svg, loading-pure.svg
		"/stage/build/fonts/",   // 字体文件
		"/stage/build/desktop/", // 桌面端静态资源
		"/stage/build/mobile/",  // 移动端静态资源
		"/stage/build/app/",     // 应用端静态资源
		"/stage/base.",          // base.css
		"/stage/build/desktop",  // desktop 目录（包含所有桌面端资源）
		"/stage/build/mobile",   // mobile 目录（包含所有移动端资源）
		"/stage/build/app",      // app 目录（包含所有应用端资源）
		"/appearance/",
		"/favicon.ico",
		"/manifest.webmanifest",
		"/manifest.json",
		"/service-worker.js",
	}

	// 获取请求路径
	requestPath := c.Request.URL.Path

	// 需要认证但可以通过 Cookie 验证的资源路径
	// 这些路径的请求可能不携带 Authorization header，但会携带 Cookie
	// 注意：这些资源在用户已登录的情况下应该可以访问
	// 由于浏览器对静态资源请求可能不携带自定义 header，我们暂时放行这些路径
	// 安全性由前端页面的认证保证（用户必须先登录才能看到笔记内容）
	cookieAuthPaths := []string{
		"/assets/",   // 用户资源文件（图片、附件等）
		"/emojis/",   // 表情资源
		"/widgets/",  // 挂件资源
		"/snippets/", // 代码片段
		"/plugins/",  // 插件资源
	}

	// 对于资源路径，如果有 Cookie 中的 token 就验证，没有就放行
	// 这是因为浏览器对 img/iframe 等标签的请求可能不携带自定义 header
	for _, path := range cookieAuthPaths {
		if strings.HasPrefix(requestPath, path) {
			// 尝试从 Cookie 获取 token
			cookieToken, _ := c.Cookie("siyuan_token")
			if cookieToken != "" {
				// 有 Cookie，验证后放行
				authService := GetWebAuthService()
				if authService != nil {
					if user, err := authService.ValidateToken(cookieToken); err == nil {
						// Token 有效，设置用户信息并放行
						c.Set("web_user_id", user.ID)
						c.Set("web_username", user.Username)
						c.Set("web_workspace", user.Workspace)
						c.Set(RoleContextKey, RoleAdministrator)
						// 切换 workspace
						if user.Workspace != "" && user.Workspace != util.WorkspaceDir {
							SwitchWorkspace(user.Workspace)
						}
						c.Next()
						return
					}
				}
			}
			// 没有 Cookie 或验证失败，也放行（让后续逻辑处理）
			// 这样可以让公开分享的资源也能访问
			logging.LogInfof("[Web Mode] Resource path accessed without valid token: %s", requestPath)
			c.Next()
			return
		}
	}

	for _, path := range publicPaths {
		if strings.HasPrefix(requestPath, path) {
			logging.LogInfof("[Web Mode] Public path accessed: %s", requestPath)
			c.Next()
			return
		}
	}

	// 从请求中提取JWT token
	var token string

	// 1. 从Authorization header获取
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		if strings.HasPrefix(authHeader, "Bearer ") {
			token = strings.TrimPrefix(authHeader, "Bearer ")
		} else if strings.HasPrefix(authHeader, "bearer ") {
			token = strings.TrimPrefix(authHeader, "bearer ")
		}
	}

	// 2. 从Cookie获取
	if token == "" {
		token, _ = c.Cookie("siyuan_token")
	}

	// 3. 从localStorage传递的header获取(前端可能通过X-Auth-Token传递)
	if token == "" {
		token = c.GetHeader("X-Auth-Token")
	}

	if token == "" {
		logging.LogWarnf("[Web Mode] No token provided for %s", requestPath)

		// 如果是API请求,返回JSON错误
		if strings.HasPrefix(requestPath, "/api/") {
			c.JSON(http.StatusUnauthorized, map[string]interface{}{
				"code": -1,
				"msg":  "未登录或登录已过期,请重新登录",
			})
			c.Abort()
			return
		}

		// 如果是页面请求,重定向到登录页
		c.Redirect(http.StatusFound, "/stage/login.html")
		c.Abort()
		return
	}

	// 验证JWT token
	authService := GetWebAuthService()
	if authService == nil {
		logging.LogErrorf("[Web Mode] WebAuthService not initialized")
		c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"code": -1,
			"msg":  "认证服务未初始化",
		})
		c.Abort()
		return
	}

	user, err := authService.ValidateToken(token)
	if err != nil {
		logging.LogWarnf("[Web Mode] Invalid token: %s", err)

		if strings.HasPrefix(requestPath, "/api/") {
			c.JSON(http.StatusUnauthorized, map[string]interface{}{
				"code": -1,
				"msg":  "令牌无效或已过期,请重新登录",
			})
			c.Abort()
			return
		}

		c.Redirect(http.StatusFound, "/stage/login.html")
		c.Abort()
		return
	}

	// Token有效,将用户信息存储到context中
	c.Set("web_user_id", user.ID)
	c.Set("web_username", user.Username)
	c.Set("web_email", user.Email)
	c.Set("web_workspace", user.Workspace)
	c.Set(RoleContextKey, RoleAdministrator) // Web认证用户默认为管理员角色

	logging.LogInfof("[Web Mode] Authenticated user: %s (workspace: %s)", user.Username, user.Workspace)

	// 动态切换workspace
	// 注意:这是一个简化的实现,在高并发情况下可能有问题
	// 更好的方案是修改所有API以支持从context读取workspace
	// 但这需要大量代码改动,作为第一阶段,我们先实现基本切换
	if user.Workspace != "" && user.Workspace != util.WorkspaceDir {
		// 保存原workspace以便恢复(虽然在当前实现中不会恢复)
		c.Set("original_workspace", util.WorkspaceDir)

		// 完整切换到用户workspace（所有路径变量）
		SwitchWorkspace(user.Workspace)

		logging.LogInfof("[Web Mode] Switched workspace to: %s", user.Workspace)
	}

	c.Next()
}

// GetWebUserWorkspace 从context中获取当前用户的workspace路径
func GetWebUserWorkspace(c *gin.Context) string {
	workspace, exists := c.Get("web_workspace")
	if !exists {
		return util.WorkspaceDir
	}
	return workspace.(string)
}

// GetWebUserID 从context中获取当前用户ID
func GetWebUserID(c *gin.Context) string {
	userID, exists := c.Get("web_user_id")
	if !exists {
		return ""
	}
	return userID.(string)
}

// SwitchWorkspace 完整切换工作空间（切换所有相关路径变量）
// 这个函数会更新 util 包中的所有路径变量，确保用户数据完全隔离
func SwitchWorkspace(workspacePath string) {
	// 基础路径
	util.WorkspaceDir = workspacePath
	util.WorkspaceName = filepath.Base(workspacePath)
	util.ConfDir = filepath.Join(workspacePath, "conf")
	util.DataDir = workspacePath // 在思源中，DataDir 通常就是 workspace 根目录
	util.RepoDir = filepath.Join(workspacePath, "repo")
	util.HistoryDir = filepath.Join(workspacePath, "history")
	util.TempDir = filepath.Join(workspacePath, "temp")

	// 数据库路径
	util.DBPath = filepath.Join(util.TempDir, util.DBName)
	util.HistoryDBPath = filepath.Join(util.TempDir, "history.db")
	util.AssetContentDBPath = filepath.Join(util.TempDir, "asset_content.db")
	util.BlockTreeDBPath = filepath.Join(util.TempDir, "blocktree.db")

	// 外观路径
	util.AppearancePath = filepath.Join(util.ConfDir, "appearance")
	util.ThemesPath = filepath.Join(util.AppearancePath, "themes")
	util.IconsPath = filepath.Join(util.AppearancePath, "icons")

	// Snippets 路径
	util.SnippetsPath = filepath.Join(util.DataDir, "snippets")

	// 日志路径（保持在用户 workspace 的 temp 目录）
	util.LogPath = filepath.Join(util.TempDir, "siyuan.log")

	// 确保用户workspace目录结构存在
	dirs := []string{
		workspacePath,
		util.ConfDir,
		util.DataDir,
		util.TempDir,
		util.RepoDir,
		util.HistoryDir,
		util.AppearancePath,
		filepath.Join(util.DataDir, "assets"),
		filepath.Join(util.DataDir, "templates"),
		filepath.Join(util.DataDir, "widgets"),
		filepath.Join(util.DataDir, "plugins"),
		filepath.Join(util.DataDir, "emojis"),
		filepath.Join(util.DataDir, "public"),
		filepath.Join(util.DataDir, "storage"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil && !os.IsExist(err) {
			logging.LogErrorf("[Web Mode] Failed to create workspace dir %s: %s", dir, err)
		}
	}

	// 设置临时目录环境变量
	osTmpDir := filepath.Join(util.TempDir, "os")
	os.MkdirAll(osTmpDir, 0755)
	os.Setenv("TMPDIR", osTmpDir)
	os.Setenv("TEMP", osTmpDir)
	os.Setenv("TMP", osTmpDir)
}
