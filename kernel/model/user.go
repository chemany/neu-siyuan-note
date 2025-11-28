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
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/siyuan-note/logging"
	"github.com/siyuan-note/siyuan/kernel/util"
	"golang.org/x/crypto/bcrypt"
)

// User 用户结构
type User struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Password  string    `json:"-"` // 不序列化密码
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Workspace string    `json:"workspace"` // 用户工作空间路径
	IsActive  bool      `json:"is_active"`
}

// UserStore 用户存储接口
type UserStore interface {
	Create(user *User) error
	GetByID(id string) (*User, error)
	GetByEmail(email string) (*User, error)
	GetByUsername(username string) (*User, error)
	Update(user *User) error
	Delete(id string) error
	List() ([]*User, error)
	VerifyPassword(email, password string) (*User, error)
}

// FileUserStore 基于文件的用户存储
type FileUserStore struct {
	filePath    string
	users       map[string]*User
	emailMap    map[string]*User
	usernameMap map[string]*User
	mutex       sync.RWMutex
}

// NewFileUserStore 创建文件用户存储
func NewFileUserStore(dataDir string) (*FileUserStore, error) {
	store := &FileUserStore{
		filePath:    filepath.Join(dataDir, "users.json"),
		users:       make(map[string]*User),
		emailMap:    make(map[string]*User),
		usernameMap: make(map[string]*User),
	}

	if err := store.load(); err != nil {
		logging.LogErrorf("Failed to load user store: %s", err)
		return nil, err
	}

	return store, nil
}

// load 加载用户数据
func (s *FileUserStore) load() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if _, err := os.Stat(s.filePath); os.IsNotExist(err) {
		// 文件不存在，创建空的存储
		return s.save()
	}

	data, err := ioutil.ReadFile(s.filePath)
	if err != nil {
		return fmt.Errorf("failed to read users file: %w", err)
	}

	var users []*User
	if err := json.Unmarshal(data, &users); err != nil {
		return fmt.Errorf("failed to unmarshal users: %w", err)
	}

	// 重建映射
	s.users = make(map[string]*User)
	s.emailMap = make(map[string]*User)
	s.usernameMap = make(map[string]*User)

	for _, user := range users {
		s.users[user.ID] = user
		s.emailMap[user.Email] = user
		s.usernameMap[user.Username] = user
	}

	return nil
}

// save 保存用户数据
func (s *FileUserStore) save() error {
	var users []*User
	for _, user := range s.users {
		users = append(users, user)
	}

	data, err := json.MarshalIndent(users, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal users: %w", err)
	}

	// 确保目录存在
	if err := os.MkdirAll(filepath.Dir(s.filePath), 0755); err != nil {
		return fmt.Errorf("failed to create users directory: %w", err)
	}

	if err := ioutil.WriteFile(s.filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write users file: %w", err)
	}

	return nil
}

// Create 创建用户
func (s *FileUserStore) Create(user *User) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// 检查邮箱是否已存在
	if _, exists := s.emailMap[user.Email]; exists {
		return fmt.Errorf("email already exists")
	}

	// 检查用户名是否已存在
	if _, exists := s.usernameMap[user.Username]; exists {
		return fmt.Errorf("username already exists")
	}

	// 生成ID
	user.ID = generateUUID()
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	user.IsActive = true

	// 加密密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}
	user.Password = string(hashedPassword)

	// 创建用户工作空间
	if err := s.createUserWorkspace(user); err != nil {
		return fmt.Errorf("failed to create user workspace: %w", err)
	}

	// 存储用户
	s.users[user.ID] = user
	s.emailMap[user.Email] = user
	s.usernameMap[user.Username] = user

	return s.save()
}

// createUserWorkspace 创建用户工作空间
func (s *FileUserStore) createUserWorkspace(user *User) error {
	// 从环境变量获取用户数据根路径,如果未设置则使用默认路径
	userDataRoot := os.Getenv("SIYUAN_USER_DATA_ROOT")
	if userDataRoot == "" {
		userDataRoot = "/mnt/nas-sata12/MindOcean/user-data/notes"
	}

	workspaceDir := filepath.Join(userDataRoot, user.Username)

	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		return fmt.Errorf("failed to create workspace directory: %w", err)
	}

	user.Workspace = workspaceDir
	logging.LogInfof("Created workspace for user %s at %s", user.Username, workspaceDir)
	return nil
}

// GetByID 根据ID获取用户
func (s *FileUserStore) GetByID(id string) (*User, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	user, exists := s.users[id]
	if !exists {
		return nil, fmt.Errorf("user not found")
	}
	return user, nil
}

// GetByEmail 根据邮箱获取用户
func (s *FileUserStore) GetByEmail(email string) (*User, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	user, exists := s.emailMap[email]
	if !exists {
		return nil, fmt.Errorf("user not found")
	}
	return user, nil
}

// GetByUsername 根据用户名获取用户
func (s *FileUserStore) GetByUsername(username string) (*User, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	user, exists := s.usernameMap[username]
	if !exists {
		return nil, fmt.Errorf("user not found")
	}
	return user, nil
}

// Update 更新用户
func (s *FileUserStore) Update(user *User) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	existing, exists := s.users[user.ID]
	if !exists {
		return fmt.Errorf("user not found")
	}

	// 如果邮箱或用户名发生变化，需要检查冲突
	if user.Email != existing.Email {
		if _, exists := s.emailMap[user.Email]; exists {
			return fmt.Errorf("email already exists")
		}
	}

	if user.Username != existing.Username {
		if _, exists := s.usernameMap[user.Username]; exists {
			return fmt.Errorf("username already exists")
		}
	}

	user.UpdatedAt = time.Now()
	s.users[user.ID] = user
	s.emailMap[user.Email] = user
	s.usernameMap[user.Username] = user

	return s.save()
}

// Delete 删除用户
func (s *FileUserStore) Delete(id string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	user, exists := s.users[id]
	if !exists {
		return fmt.Errorf("user not found")
	}

	delete(s.users, id)
	delete(s.emailMap, user.Email)
	delete(s.usernameMap, user.Username)

	return s.save()
}

// List 列出所有用户
func (s *FileUserStore) List() ([]*User, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	var users []*User
	for _, user := range s.users {
		users = append(users, user)
	}
	return users, nil
}

// VerifyPassword 验证密码
func (s *FileUserStore) VerifyPassword(email, password string) (*User, error) {
	user, err := s.GetByEmail(email)
	if err != nil {
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, fmt.Errorf("invalid password")
	}

	return user, nil
}

// generateUUID 生成UUID
func generateUUID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// 全局用户存储实例
var globalUserStore UserStore

// InitUserStore 初始化用户存储
func InitUserStore() error {
	dataDir := filepath.Join(util.WorkingDir, "data", "users")
	store, err := NewFileUserStore(dataDir)
	if err != nil {
		return err
	}
	globalUserStore = store
	return nil
}

// GetUserStore 获取用户存储
func GetUserStore() UserStore {
	return globalUserStore
}
