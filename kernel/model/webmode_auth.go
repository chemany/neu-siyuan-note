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

	// 放行公开路径
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
		"/appearance/",
	}

	requestPath := c.Request.URL.Path
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

		// 切换到用户workspace
		util.WorkspaceDir = user.Workspace
		util.DataDir = user.Workspace
		util.ConfDir = filepath.Join(user.Workspace, "conf")

		logging.LogInfof("[Web Mode] Switched workspace to: %s", user.Workspace)

		// 确保用户workspace目录结构存在
		dirs := []string{
			user.Workspace,
			filepath.Join(user.Workspace, "conf"),
			filepath.Join(user.Workspace, "data"),
			filepath.Join(user.Workspace, "temp"),
		}
		for _, dir := range dirs {
			if err := os.MkdirAll(dir, 0755); err != nil {
				logging.LogErrorf("[Web Mode] Failed to create workspace dir %s: %s", dir, err)
			}
		}
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
