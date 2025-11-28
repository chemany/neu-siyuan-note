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

package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/88250/gulu"
	"github.com/gin-gonic/gin"
	"github.com/siyuan-note/logging"
	"github.com/siyuan-note/siyuan/kernel/model"
)

// Web认证API处理器

func webAuthLogin(c *gin.Context) {
	ret := gulu.Ret.NewResult()
	defer c.JSON(http.StatusOK, ret)

	var req model.LoginRequest
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		logging.LogErrorf("Failed to decode login request: %s", err)
		ret.Code = -1
		ret.Msg = "请求格式错误"
		return
	}

	// 验证请求格式
	if strings.TrimSpace(req.Email) == "" || strings.TrimSpace(req.Password) == "" {
		ret.Code = -1
		ret.Msg = "邮箱和密码不能为空"
		return
	}

	// 获取Web认证服务
	authService := model.GetWebAuthService()
	if authService == nil {
		ret.Code = -1
		ret.Msg = "认证服务未初始化"
		return
	}

	// 验证用户凭据
	authResp, err := authService.Login(&req)
	if err != nil {
		ret.Code = -1
		ret.Msg = "登录失败: " + err.Error()
		return
	}

	ret.Code = 0
	ret.Msg = "登录成功"
	ret.Data = authResp
}

func webAuthRegister(c *gin.Context) {
	ret := gulu.Ret.NewResult()
	defer c.JSON(http.StatusOK, ret)

	var req model.RegisterRequest
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		logging.LogErrorf("Failed to decode register request: %s", err)
		ret.Code = -1
		ret.Msg = "请求格式错误"
		return
	}

	// 验证请求格式
	if strings.TrimSpace(req.Username) == "" || strings.TrimSpace(req.Email) == "" || strings.TrimSpace(req.Password) == "" {
		ret.Code = -1
		ret.Msg = "用户名、邮箱和密码不能为空"
		return
	}

	// 获取Web认证服务
	authService := model.GetWebAuthService()
	if authService == nil {
		ret.Code = -1
		ret.Msg = "认证服务未初始化"
		return
	}

	// 注册用户
	authResp, err := authService.Register(&req)
	if err != nil {
		ret.Code = -1
		ret.Msg = "注册失败: " + err.Error()
		return
	}

	ret.Code = 0
	ret.Msg = "注册成功"
	ret.Data = authResp
}

func webAuthUnifiedLogin(c *gin.Context) {
	ret := gulu.Ret.NewResult()
	defer c.JSON(http.StatusOK, ret)

	var req model.UnifiedLoginRequest
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		logging.LogErrorf("Failed to decode unified login request: %s", err)
		ret.Code = -1
		ret.Msg = "请求格式错误"
		return
	}

	// 验证请求格式
	if strings.TrimSpace(req.UnifiedToken) == "" {
		ret.Code = -1
		ret.Msg = "统一令牌不能为空"
		return
	}

	// 获取统一认证服务
	unifiedService := model.GetUnifiedAuthService()
	if unifiedService == nil {
		ret.Code = -1
		ret.Msg = "统一认证服务未初始化"
		return
	}

	// 通过统一服务登录
	response, err := unifiedService.LoginWithUnifiedToken(req.UnifiedToken)
	if err != nil {
		ret.Code = -1
		ret.Msg = "统一登录失败: " + err.Error()
		return
	}

	ret.Code = 0
	ret.Msg = "统一登录成功"
	ret.Data = response
}

func webAuthUnifiedStatus(c *gin.Context) {
	ret := gulu.Ret.NewResult()
	defer c.JSON(http.StatusOK, ret)

	// 检查统一认证服务状态
	unifiedService := model.GetUnifiedAuthService()
	if unifiedService == nil {
		ret.Code = -1
		ret.Msg = "统一认证服务未初始化"
		return
	}

	// 检查统一服务连接状态
	ok, msg := unifiedService.CheckServiceStatus()
	if !ok {
		ret.Code = -1
		ret.Msg = "检查服务状态失败: " + msg
		return
	}

	ret.Code = 0
	ret.Msg = "服务状态正常"
	ret.Data = map[string]interface{}{
		"status":  "connected",
		"message": msg,
	}
}

func webAuthProfile(c *gin.Context) {
	ret := gulu.Ret.NewResult()
	defer c.JSON(http.StatusOK, ret)

	// 从Authorization header获取令牌
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		ret.Code = -1
		ret.Msg = "缺少认证令牌"
		return
	}

	// 移除"Bearer "前缀
	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == authHeader {
		ret.Code = -1
		ret.Msg = "认证令牌格式错误"
		return
	}

	// 获取Web认证服务
	authService := model.GetWebAuthService()
	if authService == nil {
		ret.Code = -1
		ret.Msg = "认证服务未初始化"
		return
	}

	// 验证令牌
	user, err := authService.ValidateToken(token)
	if err != nil {
		ret.Code = -1
		ret.Msg = "令牌无效: " + err.Error()
		return
	}

	ret.Code = 0
	ret.Msg = "获取用户信息成功"
	ret.Data = *user
}

func webAuthUpdateProfile(c *gin.Context) {
	ret := gulu.Ret.NewResult()
	defer c.JSON(http.StatusOK, ret)

	// 从Authorization header获取令牌
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		ret.Code = -1
		ret.Msg = "缺少认证令牌"
		return
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == authHeader {
		ret.Code = -1
		ret.Msg = "认证令牌格式错误"
		return
	}

	// 获取Web认证服务
	authService := model.GetWebAuthService()
	if authService == nil {
		ret.Code = -1
		ret.Msg = "认证服务未初始化"
		return
	}

	// 验证令牌
	user, err := authService.ValidateToken(token)
	if err != nil {
		ret.Code = -1
		ret.Msg = "令牌无效: " + err.Error()
		return
	}

	var req model.UpdateProfileRequest
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		logging.LogErrorf("Failed to decode update profile request: %s", err)
		ret.Code = -1
		ret.Msg = "请求格式错误"
		return
	}

	// 更新用户信息
	updatedUser, err := authService.UpdateUserProfile(user.ID, req.Username, req.Email)
	if err != nil {
		ret.Code = -1
		ret.Msg = "更新用户信息失败: " + err.Error()
		return
	}

	ret.Code = 0
	ret.Msg = "更新用户信息成功"
	ret.Data = *updatedUser
}

func webAuthChangePassword(c *gin.Context) {
	ret := gulu.Ret.NewResult()
	defer c.JSON(http.StatusOK, ret)

	// 从Authorization header获取令牌
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		ret.Code = -1
		ret.Msg = "缺少认证令牌"
		return
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == authHeader {
		ret.Code = -1
		ret.Msg = "认证令牌格式错误"
		return
	}

	// 获取Web认证服务
	authService := model.GetWebAuthService()
	if authService == nil {
		ret.Code = -1
		ret.Msg = "认证服务未初始化"
		return
	}

	// 验证令牌
	user, err := authService.ValidateToken(token)
	if err != nil {
		ret.Code = -1
		ret.Msg = "令牌无效: " + err.Error()
		return
	}

	var req model.ChangePasswordRequest
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		logging.LogErrorf("Failed to decode change password request: %s", err)
		ret.Code = -1
		ret.Msg = "请求格式错误"
		return
	}

	// 验证请求格式
	if strings.TrimSpace(req.OldPassword) == "" || strings.TrimSpace(req.NewPassword) == "" {
		ret.Code = -1
		ret.Msg = "旧密码和新密码不能为空"
		return
	}

	// 修改密码
	err = authService.ChangePassword(user.ID, req.OldPassword, req.NewPassword)
	if err != nil {
		ret.Code = -1
		ret.Msg = "修改密码失败: " + err.Error()
		return
	}

	ret.Code = 0
	ret.Msg = "修改密码成功"
}

func webAuthLogout(c *gin.Context) {
	ret := gulu.Ret.NewResult()
	defer c.JSON(http.StatusOK, ret)

	// 从Authorization header获取令牌
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		ret.Code = -1
		ret.Msg = "缺少认证令牌"
		return
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == authHeader {
		ret.Code = -1
		ret.Msg = "认证令牌格式错误"
		return
	}

	// 获取Web认证服务
	authService := model.GetWebAuthService()
	if authService == nil {
		ret.Code = -1
		ret.Msg = "认证服务未初始化"
		return
	}

	// 验证令牌并注销
	user, err := authService.ValidateToken(token)
	if err != nil {
		ret.Code = -1
		ret.Msg = "令牌无效: " + err.Error()
		return
	}

	// 注销令牌（将令牌加入黑名单）
	err = authService.Logout(user.ID, token)
	if err != nil {
		ret.Code = -1
		ret.Msg = "注销失败: " + err.Error()
		return
	}

	ret.Code = 0
	ret.Msg = "注销成功"
}

func webAuthRefreshToken(c *gin.Context) {
	ret := gulu.Ret.NewResult()
	defer c.JSON(http.StatusOK, ret)

	// 从Authorization header获取令牌
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		ret.Code = -1
		ret.Msg = "缺少认证令牌"
		return
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == authHeader {
		ret.Code = -1
		ret.Msg = "认证令牌格式错误"
		return
	}

	// 获取Web认证服务
	authService := model.GetWebAuthService()
	if authService == nil {
		ret.Code = -1
		ret.Msg = "认证服务未初始化"
		return
	}

	// 验证令牌并生成新令牌
	user, err := authService.ValidateToken(token)
	if err != nil {
		ret.Code = -1
		ret.Msg = "令牌无效: " + err.Error()
		return
	}

	// 生成新的JWT令牌
	newToken, err := authService.GenerateToken(user)
	if err != nil {
		ret.Code = -1
		ret.Msg = "生成新令牌失败: " + err.Error()
		return
	}

	// 将旧令牌加入黑名单
	_ = authService.Logout(user.ID, token)

	// 构建响应数据
	response := model.RefreshTokenResponse{
		Token: newToken,
	}

	ret.Code = 0
	ret.Msg = "刷新令牌成功"
	ret.Data = response
}

// webAuthHealth 检查认证服务健康状态
func webAuthHealth(c *gin.Context) {
	ret := gulu.Ret.NewResult()
	defer c.JSON(http.StatusOK, ret)

	// 获取Web认证服务
	authService := model.GetWebAuthService()
	if authService == nil {
		ret.Code = -1
		ret.Msg = "认证服务未初始化"
		return
	}

	// 检查服务状态
	status := map[string]interface{}{
		"status":    "healthy",
		"service":   "web-auth",
		"timestamp": time.Now().Unix(),
	}

	ret.Code = 0
	ret.Msg = "认证服务运行正常"
	ret.Data = status
}

// webAuthVerifyToken 验证令牌有效性
func webAuthVerifyToken(c *gin.Context) {
	ret := gulu.Ret.NewResult()
	defer c.JSON(http.StatusOK, ret)

	var req struct {
		Token string `json:"token"`
	}

	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		logging.LogErrorf("Failed to decode verify token request: %s", err)
		ret.Code = -1
		ret.Msg = "请求格式错误"
		return
	}

	if strings.TrimSpace(req.Token) == "" {
		ret.Code = -1
		ret.Msg = "令牌不能为空"
		return
	}

	// 获取Web认证服务
	authService := model.GetWebAuthService()
	if authService == nil {
		ret.Code = -1
		ret.Msg = "认证服务未初始化"
		return
	}

	// 验证令牌
	user, err := authService.ValidateToken(req.Token)
	if err != nil {
		ret.Code = -1
		ret.Msg = "令牌无效: " + err.Error()
		return
	}

	// 构建验证结果
	result := map[string]interface{}{
		"valid":    true,
		"user_id":  user.ID,
		"username": user.Username,
		"email":    user.Email,
		"expires":  time.Now().Add(24 * time.Hour).Unix(), // 近似值，因为ValidateToken不返回过期时间
	}

	ret.Code = 0
	ret.Msg = "令牌有效"
	ret.Data = result
}

// webAuthMiddleware Web认证中间件
func webAuthMiddleware(c *gin.Context) {
	// 从Authorization header获取令牌
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		ret := gulu.Ret.NewResult()
		ret.Code = -1
		ret.Msg = "缺少认证令牌"
		c.JSON(http.StatusUnauthorized, ret)
		c.Abort()
		return
	}

	// 移除"Bearer "前缀
	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == authHeader {
		ret := gulu.Ret.NewResult()
		ret.Code = -1
		ret.Msg = "认证令牌格式错误"
		c.JSON(http.StatusUnauthorized, ret)
		c.Abort()
		return
	}

	// 获取Web认证服务
	authService := model.GetWebAuthService()
	if authService == nil {
		ret := gulu.Ret.NewResult()
		ret.Code = -1
		ret.Msg = "认证服务未初始化"
		c.JSON(http.StatusInternalServerError, ret)
		c.Abort()
		return
	}

	// 验证令牌
	user, err := authService.ValidateToken(token)
	if err != nil {
		ret := gulu.Ret.NewResult()
		ret.Code = -1
		ret.Msg = "令牌无效: " + err.Error()
		c.JSON(http.StatusUnauthorized, ret)
		c.Abort()
		return
	}

	// 将用户信息存储到context中
	c.Set("user_id", user.ID)
	c.Set("username", user.Username)
	c.Set("email", user.Email)
	c.Set("workspace", user.Workspace)

	c.Next()
}
