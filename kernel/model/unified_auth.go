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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/siyuan-note/logging"
)

// UnifiedUser 统一注册服务用户信息
type UnifiedUser struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Nickname string `json:"nickname,omitempty"`
	Avatar   string `json:"avatar,omitempty"`
	Phone    string `json:"phone,omitempty"`
	IsActive bool   `json:"is_active,omitempty"`
	Created  string `json:"created_at"` // 统一服务使用created_at
	Updated  string `json:"updated_at"` // 统一服务使用updated_at
}

// UnifiedLoginRequest 统一登录请求
type UnifiedLoginRequest struct {
	UnifiedToken string `json:"unified_token"`
}

// UnifiedAuthRequest 统一认证请求（发送给统一服务）
type UnifiedAuthRequest struct {
	Token string `json:"token"`
}

// UnifiedAuthResponse 统一认证响应
type UnifiedAuthResponse struct {
	Valid   bool         `json:"valid"`
	User    *UnifiedUser `json:"user,omitempty"`
	Message string       `json:"message,omitempty"`
}

// UnifiedAuthService 统一注册服务客户端
type UnifiedAuthService struct {
	baseURL    string
	httpClient *http.Client
	tokenCache map[string]*CachedToken
}

// CachedToken 缓存的令牌信息
type CachedToken struct {
	User   *UnifiedUser
	Expiry time.Time
}

// NewUnifiedAuthService 创建统一认证服务客户端
func NewUnifiedAuthService() *UnifiedAuthService {
	baseURL := "http://localhost:3002"
	if envURL := os.Getenv("UNIFIED_AUTH_SERVICE_URL"); envURL != "" {
		baseURL = envURL
	}

	return &UnifiedAuthService{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		tokenCache: make(map[string]*CachedToken),
	}
}

// SetBaseURL 设置统一认证服务地址
func (s *UnifiedAuthService) SetBaseURL(url string) {
	s.baseURL = url
}

// VerifyToken 验证统一注册服务令牌
func (s *UnifiedAuthService) VerifyToken(token string) (*UnifiedUser, error) {
	// 首先检查缓存
	if cached, exists := s.tokenCache[token]; exists {
		if time.Now().Before(cached.Expiry) {
			return cached.User, nil
		}
		// 缓存过期，删除
		delete(s.tokenCache, token)
	}

	// 调用统一注册服务验证令牌 (GET请求,通过Authorization header传递token)
	url := fmt.Sprintf("%s/api/auth/verify", s.baseURL)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 设置Authorization header
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		logging.LogErrorf("Failed to call unified auth service: %s", err)
		return nil, fmt.Errorf("failed to connect to unified auth service: %w", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		logging.LogWarnf("Unified auth service returned status %d: %s", resp.StatusCode, string(body))
		return nil, fmt.Errorf("unified auth service error: %s", string(body))
	}

	// 解析统一服务的响应格式
	var authResp struct {
		Valid bool         `json:"valid"`
		User  *UnifiedUser `json:"user"`
	}

	if err := json.Unmarshal(body, &authResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if !authResp.Valid || authResp.User == nil {
		return nil, fmt.Errorf("invalid token")
	}

	// 缓存成功的验证结果（30分钟）
	s.tokenCache[token] = &CachedToken{
		User:   authResp.User,
		Expiry: time.Now().Add(30 * time.Minute),
	}

	return authResp.User, nil
}

// EnsureLocalUser 确保本地用户存在
func (s *UnifiedAuthService) EnsureLocalUser(unifiedUser *UnifiedUser) (*User, error) {
	userStore := GetUserStore()
	if userStore == nil {
		return nil, fmt.Errorf("user store not initialized")
	}

	// 通过邮箱查找本地用户
	localUser, err := userStore.GetByEmail(unifiedUser.Email)
	if err == nil {
		// 用户已存在，更新信息
		localUser.Username = unifiedUser.Username
		// 强制保持激活状态，防止被统一认证服务覆盖为false
		localUser.IsActive = true
		if err := userStore.Update(localUser); err != nil {
			logging.LogErrorf("Failed to update local user: %s", err)
		}
		return localUser, nil
	}

	// 用户不存在，创建新用户
	newUser := &User{
		Username: unifiedUser.Username,
		Email:    unifiedUser.Email,
		Password: "unified_auth_placeholder", // 统一认证用户使用占位密码
		IsActive: unifiedUser.IsActive,
	}

	// 解析创建时间
	if unifiedUser.Created != "" {
		if createdTime, err := time.Parse(time.RFC3339, unifiedUser.Created); err == nil {
			newUser.CreatedAt = createdTime
		}
	}

	if err := userStore.Create(newUser); err != nil {
		return nil, fmt.Errorf("failed to create local user: %w", err)
	}

	logging.LogInfof("Created new local user for unified auth: %s", unifiedUser.Email)
	return newUser, nil
}

// SyncUserFromUnified 从统一服务同步用户信息
func (s *UnifiedAuthService) SyncUserFromUnified(token string) (*User, error) {
	// 验证统一服务令牌
	unifiedUser, err := s.VerifyToken(token)
	if err != nil {
		return nil, fmt.Errorf("failed to verify unified token: %w", err)
	}

	// 确保本地用户存在
	localUser, err := s.EnsureLocalUser(unifiedUser)
	if err != nil {
		return nil, fmt.Errorf("failed to ensure local user: %w", err)
	}

	return localUser, nil
}

// LoginWithUnifiedToken 使用统一服务令牌登录
func (s *UnifiedAuthService) LoginWithUnifiedToken(token string) (*AuthResponse, error) {
	// 从统一服务同步用户
	user, err := s.SyncUserFromUnified(token)
	if err != nil {
		return nil, fmt.Errorf("failed to sync user from unified service: %w", err)
	}

	// 生成本地JWT令牌
	authService := GetWebAuthService()
	if authService == nil {
		authService = NewWebAuthService()
		InitWebAuthService()
	}

	localToken, err := authService.GenerateToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate local token: %w", err)
	}

	// 转换为WebUser
	webUser := &WebUser{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		Created:   user.CreatedAt,
		Workspace: user.Workspace,
		IsActive:  user.IsActive,
	}

	return &AuthResponse{
		Token:    localToken,
		User:     webUser,
		Expires:  time.Now().Add(24 * time.Hour).Unix(),
		Messages: []string{"通过统一注册服务登录成功"},
	}, nil
}

// CheckServiceStatus 检查统一服务状态
func (s *UnifiedAuthService) CheckServiceStatus() (bool, string) {
	url := fmt.Sprintf("%s/api/health", s.baseURL)

	resp, err := s.httpClient.Get(url)
	if err != nil {
		return false, fmt.Sprintf("无法连接到统一注册服务: %s", err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return true, "统一注册服务运行正常"
	}

	return false, fmt.Sprintf("统一注册服务状态异常: HTTP %d", resp.StatusCode)
}

// ClearTokenCache 清理过期的令牌缓存
func (s *UnifiedAuthService) ClearTokenCache() {
	now := time.Now()
	for token, cached := range s.tokenCache {
		if now.After(cached.Expiry) {
			delete(s.tokenCache, token)
		}
	}
}

// 全局统一认证服务实例
var globalUnifiedAuthService *UnifiedAuthService

// InitUnifiedAuthService 初始化统一认证服务
func InitUnifiedAuthService() {
	globalUnifiedAuthService = NewUnifiedAuthService()

	// 启动定期清理缓存的goroutine
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			if globalUnifiedAuthService != nil {
				globalUnifiedAuthService.ClearTokenCache()
			}
		}
	}()
}

// GetUnifiedAuthService 获取统一认证服务
func GetUnifiedAuthService() *UnifiedAuthService {
	return globalUnifiedAuthService
}
