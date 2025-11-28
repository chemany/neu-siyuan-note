# 思源笔记多用户Web应用改造技术总结

## 项目概述

本文档总结了将思源笔记从桌面应用改造为多用户Web应用的完整技术实现方案。该改造实现了与灵枢笔记统一注册服务的集成，支持多用户独立数据隔离和完整的Web认证体系。

## 核心架构改造

### 1. Web模式配置
- **文件**: `kernel/model/process.go`
- **核心改动**: 修改`HookDesktopUIProcJob()`函数，增加`SIYUAN_WEB_MODE`环境变量检查
- **功能**: 在Web模式下禁用UI进程检测，确保后端服务持久运行
- **环境变量**: `SIYUAN_WEB_MODE=true`

### 2. 用户管理系统
- **文件**: `kernel/model/user.go`
- **核心功能**:
  - 基于文件存储的用户数据管理
  - 完整的CRUD操作接口
  - bcrypt密码加密
  - 用户数据结构定义（ID、Username、Email、Password、Workspace、CreatedAt、UpdatedAt、IsActive）
- **核心结构体**: `UserStore`接口、`FileUserStore`实现、`User`结构体

### 3. Web认证服务
- **文件**: `kernel/model/webauth.go`
- **核心功能**:
  - JWT令牌生成和验证
  - 用户登录、注册、密码修改
  - 令牌刷新和黑名单机制
  - 用户信息更新
- **JWT配置**: 24小时有效期，HS256签名算法
- **核心结构体**: `WebAuthService`、`CustomClaims`、各种请求/响应结构体

### 4. 统一注册服务集成
- **文件**: `kernel/model/unified_auth.go`
- **核心功能**:
  - 与灵枢笔记统一注册服务（localhost:3002）集成
  - 令牌缓存机制（1小时有效期）
  - 用户数据同步和本地存储
  - 统一登录接口
- **核心结构体**: `UnifiedAuthService`、`UnifiedUser`、`UnifiedAuthResponse`

### 5. Web API接口
- **文件**: `kernel/api/web_auth.go`
- **API端点**:
  - `POST /api/web/auth/login` - 用户登录
  - `POST /api/web/auth/register` - 用户注册
  - `POST /api/web/auth/unified-login` - 统一登录
  - `GET /api/web/auth/unified-status` - 统一服务状态
  - `GET /api/web/auth/health` - 健康检查
  - `POST /api/web/auth/verify-token` - 令牌验证
  - `POST /api/web/auth/refresh-token` - 令牌刷新
  - `POST /api/web/auth/profile` - 获取用户信息
  - `POST /api/web/auth/update-profile` - 更新用户信息
  - `POST /api/web/auth/change-password` - 修改密码
  - `POST /api/web/auth/logout` - 用户注销
- **中间件**: `webAuthMiddleware` - JWT令牌验证中间件

### 6. 路由配置
- **文件**: `kernel/api/router.go`
- **改动**: 在`ServeAPI()`函数中添加Web认证相关路由
- **路由分组**:
  - 公开接口（无需认证）
  - 受保护接口（需要JWT认证）

### 7. 服务初始化
- **文件**: `kernel/main.go`
- **改动**: 在应用启动时初始化所有认证服务
- **初始化顺序**:
  1. `InitUserStore()` - 用户存储服务
  2. `InitWebAuthService()` - Web认证服务
  3. `InitUnifiedAuthService()` - 统一认证服务

## 数据隔离机制

### 1. 用户工作空间
- 每个用户拥有独立的工作空间目录
- 工作空间路径: `workspace/{user_id}/`
- 数据完全隔离，确保用户间数据安全

### 2. JWT令牌机制
- 令牌包含用户ID、用户名、邮箱、工作空间路径等信息
- 服务端验证确保用户只能访问自己的数据
- 令牌黑名单机制支持主动注销

### 3. 中间件保护
- 受保护接口通过中间件验证JWT令牌
- 将用户信息注入请求上下文
- 统一的错误处理和响应格式

## 统一注册服务集成

### 1. 服务发现
- 默认连接地址: `http://localhost:3002`
- 支持服务健康检查
- 连接失败时提供友好错误信息

### 2. 令牌验证流程
1. 客户端提供统一服务令牌
2. 调用统一服务验证令牌并获取用户信息
3. 在本地创建或更新用户数据
4. 生成本地JWT令牌返回给客户端
5. 缓存统一令牌避免重复验证

### 3. 数据同步
- 自动同步用户基本信息
- 保持本地数据与统一服务数据一致性
- 支持增量更新机制

## 技术栈依赖

### 1. 核心依赖
- `github.com/gin-gonic/gin` - Web框架
- `github.com/golang-jwt/jwt/v4` - JWT令牌处理
- `golang.org/x/crypto/bcrypt` - 密码加密
- `github.com/88250/gulu` - 工具库

### 2. Go版本要求
- Go 1.21.0+
- 支持Go modules依赖管理

## 部署配置

### 1. 环境变量
```bash
# 启用Web模式
export SIYUAN_WEB_MODE=true

# 可选配置
export SIYUAN_AUTH_SECRET=your-secret-key
export UNIFIED_AUTH_SERVICE_URL=http://localhost:3002
```

### 2. 构建命令
```bash
cd kernel
export SIYUAN_WEB_MODE=true
go build -o siyuan-kernel main.go
```

### 3. 运行配置
- 后端服务: 默认端口6806
- 支持Docker容器化部署
- 静态文件服务集成

## API接口规范

### 1. 统一响应格式
```json
{
  "code": 0,        // 0表示成功，-1表示失败
  "msg": "消息",     // 响应消息
  "data": {}        // 响应数据
}
```

### 2. 认证方式
- JWT Bearer Token
- Header: `Authorization: Bearer {token}`
- 令牌有效期: 24小时

### 3. 错误处理
- 统一错误码和错误消息
- 详细的错误日志记录
- 友好的用户提示信息

## 安全特性

### 1. 密码安全
- bcrypt哈希算法
- 自适应加盐
- 防止彩虹表攻击

### 2. 令牌安全
- HMAC-SHA256签名
- 令牌有效期控制
- 黑名单机制

### 3. 输入验证
- 邮箱格式验证
- 密码强度检查
- 用户名长度限制

## 性能优化

### 1. 数据存储
- 内存缓存用户数据
- 定期持久化到文件
- 读写锁并发控制

### 2. 令牌缓存
- 统一服务令牌缓存
- 减少网络请求
- 缓存过期控制

### 3. 并发控制
- 读写分离锁机制
- 原子操作保证
- 协程安全设计

## 扩展性设计

### 1. 插件化架构
- 服务接口抽象
- 可插拔认证方式
- 模块化设计

### 2. 配置化
- 环境变量配置
- 服务发现机制
- 动态配置更新

### 3. 监控集成
- 健康检查接口
- 性能指标暴露
- 日志聚合支持

## 后续优化建议

### 1. 数据库集成
- 支持MySQL/PostgreSQL等关系型数据库
- 数据迁移工具
- 分库分表支持

### 2. 分布式部署
- 微服务架构
- 负载均衡
- 服务注册发现

### 3. 高可用性
- 集群部署
- 故障转移
- 数据备份恢复

### 4. 监控运维
- Prometheus指标
- Grafana仪表盘
- 告警机制

## 文件清单

### 核心文件
- `kernel/model/process.go` - Web模式配置
- `kernel/model/user.go` - 用户数据管理
- `kernel/model/webauth.go` - Web认证服务
- `kernel/model/unified_auth.go` - 统一服务集成
- `kernel/api/web_auth.go` - Web API接口
- `kernel/api/router.go` - 路由配置
- `kernel/main.go` - 服务初始化

### 配置文件
- `go.mod` - Go模块依赖
- `.env` - 环境变量配置
- `Dockerfile` - 容器化配置

### 部署脚本
- `scripts/linux-build.sh` - Linux构建脚本
- `start-web.sh` - Web模式启动脚本

## 总结

本次改造成功将思源笔记从单用户桌面应用转换为支持多用户的Web应用，实现了与灵枢笔记统一注册服务的无缝集成。改造后的系统具备以下特点：

1. **完整的多用户支持**: 用户注册、登录、数据隔离
2. **统一认证集成**: 与灵枢笔记服务无缝对接
3. **安全性保障**: JWT认证、密码加密、令牌管理
4. **可扩展架构**: 模块化设计、插件化支持
5. **部署友好**: 容器化支持、环境配置

该改造为思源笔记的Web化和多用户化奠定了坚实的技术基础，支持后续的功能扩展和性能优化。