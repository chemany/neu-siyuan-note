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
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/siyuan-note/logging"
	"golang.org/x/crypto/bcrypt"
)

// WebUser Web用户结构（扩展基础用户）
type WebUser struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Created   time.Time `json:"created"`
	Workspace string    `json:"workspace"`
	Avatar    string    `json:"avatar,omitempty"`
	IsActive  bool      `json:"is_active"`
}

// CustomClaims 自定义JWT声明
type CustomClaims struct {
	UserID    string `json:"user_id"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	Workspace string `json:"workspace"`
	jwt.RegisteredClaims
}

// WebAuthService Web认证服务
type WebAuthService struct {
	userStore UserStore
	secretKey []byte
}

// NewWebAuthService 创建Web认证服务
func NewWebAuthService() *WebAuthService {
	// 从环境变量获取JWT密钥，如果没有则使用默认密钥
	secret := []byte("siyuan-web-auth-secret-change-in-production")
	if envSecret := os.Getenv("SIYUAN_JWT_SECRET"); envSecret != "" {
		secret = []byte(envSecret)
	}

	return &WebAuthService{
		userStore: GetUserStore(),
		secretKey: secret,
	}
}

// GenerateToken 生成JWT令牌
func (a *WebAuthService) GenerateToken(user *User) (string, error) {
	claims := CustomClaims{
		UserID:    user.ID,
		Username:  user.Username,
		Email:     user.Email,
		Workspace: user.Workspace,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)), // 24小时过期
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "siyuan-web",
			Subject:   user.ID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(a.secretKey)
}

// ValidateToken 验证JWT令牌
func (a *WebAuthService) ValidateToken(tokenString string) (*User, error) {
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return a.secretKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if claims, ok := token.Claims.(*CustomClaims); ok && token.Valid {
		user, err := a.userStore.GetByID(claims.UserID)
		if err != nil {
			return nil, fmt.Errorf("user not found: %w", err)
		}
		return user, nil
	}

	return nil, fmt.Errorf("invalid token")
}

// RefreshToken 刷新令牌
func (a *WebAuthService) RefreshToken(tokenString string) (string, error) {
	user, err := a.ValidateToken(tokenString)
	if err != nil {
		return "", err
	}

	return a.GenerateToken(user)
}

// LoginRequest 登录请求
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// RegisterRequest 注册请求
type RegisterRequest struct {
	Username string `json:"username" validate:"required,min=3,max=50"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
}

// UpdateProfileRequest 更新资料请求
type UpdateProfileRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
}

// ChangePasswordRequest 修改密码请求
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

// RefreshTokenResponse 刷新令牌响应
type RefreshTokenResponse struct {
	Token string `json:"token"`
}

// AuthResponse 认证响应
type AuthResponse struct {
	Token    string   `json:"token"`
	User     *WebUser `json:"user"`
	Expires  int64    `json:"expires"`
	Messages []string `json:"messages,omitempty"`
}

// GetUserStore 获取用户存储
func (a *WebAuthService) GetUserStore() UserStore {
	return a.userStore
}

// UpdateUserProfile 更新用户资料
func (a *WebAuthService) UpdateUserProfile(userID, username, email string) (*WebUser, error) {
	updates := make(map[string]interface{})
	if username != "" {
		updates["username"] = username
	}
	if email != "" {
		updates["email"] = email
	}
	return a.UpdateProfile(userID, updates)
}

// Logout 用户注销
func (a *WebAuthService) Logout(userID, token string) error {
	// TODO: 实现令牌黑名单
	return nil
}

// Login 用户登录
func (a *WebAuthService) Login(req *LoginRequest) (*AuthResponse, error) {
	user, err := a.userStore.VerifyPassword(req.Email, req.Password)
	if err != nil {
		return nil, fmt.Errorf("邮箱或密码错误")
	}

	if !user.IsActive {
		return nil, fmt.Errorf("账户已被禁用")
	}

	token, err := a.GenerateToken(user)
	if err != nil {
		return nil, fmt.Errorf("生成令牌失败: %w", err)
	}

	// 转换为WebUser（不包含密码）
	webUser := &WebUser{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		Created:   user.CreatedAt,
		Workspace: user.Workspace,
		IsActive:  user.IsActive,
	}

	return &AuthResponse{
		Token:    token,
		User:     webUser,
		Expires:  time.Now().Add(24 * time.Hour).Unix(),
		Messages: []string{"登录成功"},
	}, nil
}

// Register 用户注册
func (a *WebAuthService) Register(req *RegisterRequest) (*AuthResponse, error) {
	// 检查邮箱是否已存在
	if _, err := a.userStore.GetByEmail(req.Email); err == nil {
		return nil, fmt.Errorf("邮箱已被注册")
	}

	// 检查用户名是否已存在
	if _, err := a.userStore.GetByUsername(req.Username); err == nil {
		return nil, fmt.Errorf("用户名已被使用")
	}

	// 创建新用户
	user := &User{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
		IsActive: true,
	}

	if err := a.userStore.Create(user); err != nil {
		return nil, fmt.Errorf("创建用户失败: %w", err)
	}

	// 自动登录
	return a.Login(&LoginRequest{
		Email:    req.Email,
		Password: req.Password,
	})
}

// GetCurrentUser 获取当前用户
func (a *WebAuthService) GetCurrentUser(tokenString string) (*WebUser, error) {
	user, err := a.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}

	return &WebUser{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		Created:   user.CreatedAt,
		Workspace: user.Workspace,
		IsActive:  user.IsActive,
	}, nil
}

// UpdateProfile 更新用户资料
func (a *WebAuthService) UpdateProfile(userID string, updates map[string]interface{}) (*WebUser, error) {
	user, err := a.userStore.GetByID(userID)
	if err != nil {
		return nil, fmt.Errorf("用户不存在")
	}

	// 更新允许的字段
	if username, ok := updates["username"].(string); ok && username != "" {
		// 检查新用户名是否已被使用
		if existingUser, err := a.userStore.GetByUsername(username); err == nil && existingUser.ID != userID {
			return nil, fmt.Errorf("用户名已被使用")
		}
		user.Username = username
	}

	if email, ok := updates["email"].(string); ok && email != "" {
		// 检查新邮箱是否已被使用
		if existingUser, err := a.userStore.GetByEmail(email); err == nil && existingUser.ID != userID {
			return nil, fmt.Errorf("邮箱已被使用")
		}
		user.Email = email
	}

	user.UpdatedAt = time.Now()

	if err := a.userStore.Update(user); err != nil {
		return nil, fmt.Errorf("更新用户资料失败: %w", err)
	}

	return &WebUser{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		Created:   user.CreatedAt,
		Workspace: user.Workspace,
		IsActive:  user.IsActive,
	}, nil
}

// ChangePassword 修改密码
func (a *WebAuthService) ChangePassword(userID, oldPassword, newPassword string) error {
	user, err := a.userStore.GetByID(userID)
	if err != nil {
		return fmt.Errorf("用户不存在")
	}

	// 验证旧密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(oldPassword)); err != nil {
		return fmt.Errorf("原密码错误")
	}

	// 加密新密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("密码加密失败: %w", err)
	}

	user.Password = string(hashedPassword)
	user.UpdatedAt = time.Now()

	return a.userStore.Update(user)
}

// ExtractTokenFromRequest 从请求中提取JWT令牌
func ExtractTokenFromRequest(r *http.Request) (string, error) {
	// 尝试从Authorization头获取
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		const bearerPrefix = "Bearer "
		if strings.HasPrefix(authHeader, bearerPrefix) {
			return authHeader[len(bearerPrefix):], nil
		}
	}

	// 尝试从Cookie获取
	cookie, err := r.Cookie("siyuan_auth_token")
	if err == nil && cookie.Value != "" {
		return cookie.Value, nil
	}

	// 尝试从查询参数获取（适用于某些特殊情况）
	token := r.URL.Query().Get("token")
	if token != "" {
		return token, nil
	}

	return "", fmt.Errorf("未找到认证令牌")
}

// WebAuthMiddleware Web认证中间件
func WebAuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 设置CORS头
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// 处理预检请求
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// 对于公开的API路径，跳过认证检查
		publicPaths := []string{
			"/api/web/auth/login",
			"/api/web/auth/register",
			"/api/web/auth/health",
		}

		for _, path := range publicPaths {
			if strings.HasPrefix(r.URL.Path, path) {
				next(w, r)
				return
			}
		}

		// 提取并验证令牌
		tokenString, err := ExtractTokenFromRequest(r)
		if err != nil {
			logging.LogWarnf("Auth middleware: %s", err)
			http.Error(w, `{"error":"Unauthorized","message":"认证令牌缺失"}`, http.StatusUnauthorized)
			return
		}

		authService := NewWebAuthService()
		user, err := authService.ValidateToken(tokenString)
		if err != nil {
			logging.LogWarnf("Invalid token: %s", err)
			http.Error(w, `{"error":"Unauthorized","message":"认证令牌无效"}`, http.StatusUnauthorized)
			return
		}

		if !user.IsActive {
			http.Error(w, `{"error":"Forbidden","message":"账户已被禁用"}`, http.StatusForbidden)
			return
		}

		// 将用户信息添加到请求上下文
		ctx := context.WithValue(r.Context(), "user", user)
		next(w, r.WithContext(ctx))
	}
}

// GetUserFromContext 从上下文获取用户
func GetUserFromContext(r *http.Request) (*User, bool) {
	user, ok := r.Context().Value("user").(*User)
	return user, ok
}

// 全局Web认证服务实例
var globalWebAuthService *WebAuthService

// InitWebAuthService 初始化Web认证服务
func InitWebAuthService() {
	globalWebAuthService = NewWebAuthService()
}

// GetWebAuthService 获取Web认证服务
func GetWebAuthService() *WebAuthService {
	return globalWebAuthService
}
