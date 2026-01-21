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

package cache

import (
	"sync"
	"time"

	"github.com/dgraph-io/ristretto"
	gcache "github.com/patrickmn/go-cache"
	"github.com/siyuan-note/logging"
)

// WorkspaceContext 接口，避免循环依赖
type WorkspaceContext interface {
	GetWorkspaceDir() string
	GetConfDir() string
	GetDataDir() string
}

// UserCache 用户级缓存结构
type UserCache struct {
	// Block 缓存 (使用 ristretto)
	blockCache *ristretto.Cache
	
	// Ref 缓存 (使用 go-cache)
	refCache *gcache.Cache
	
	// 最后访问时间
	lastAccess time.Time
	
	// 缓存是否启用
	enabled bool
}

// UserCacheManager 用户缓存管理器
type UserCacheManager struct {
	caches     map[string]*UserCache // workspace -> UserCache
	mutex      sync.RWMutex
	maxIdle    time.Duration         // 最大空闲时间
	maxCaches  int                   // 最大缓存数
}

// globalUserCacheManager 全局用户缓存管理器实例
var globalUserCacheManager *UserCacheManager

// initUserCacheManager 初始化用户缓存管理器
func initUserCacheManager() {
	globalUserCacheManager = &UserCacheManager{
		caches:    make(map[string]*UserCache),
		maxIdle:   30 * time.Minute, // 30分钟未使用则清理
		maxCaches: 100,               // 最多100个用户缓存
	}
	
	// 启动定期清理协程
	go globalUserCacheManager.cleanupRoutine()
}

// GetUserCache 获取指定用户的缓存
func (m *UserCacheManager) GetUserCache(ctx WorkspaceContext) *UserCache {
	workspaceKey := ctx.GetWorkspaceDir()
	
	// 1. 尝试获取已有缓存
	m.mutex.RLock()
	cache, exists := m.caches[workspaceKey]
	m.mutex.RUnlock()
	
	if exists {
		// 更新最后访问时间
		m.mutex.Lock()
		cache.lastAccess = time.Now()
		m.mutex.Unlock()
		return cache
	}
	
	// 2. 创建新缓存
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	// 双重检查
	if cache, exists := m.caches[workspaceKey]; exists {
		cache.lastAccess = time.Now()
		return cache
	}
	
	// 检查缓存数限制
	if len(m.caches) >= m.maxCaches {
		logging.LogWarnf("用户缓存数已达上限 %d，尝试清理空闲缓存", m.maxCaches)
		m.cleanupIdleCachesLocked()
	}
	
	// 创建新的用户缓存
	cache = newUserCache()
	m.caches[workspaceKey] = cache
	
	logging.LogInfof("为 workspace [%s] 创建新的用户缓存，当前缓存数: %d", workspaceKey, len(m.caches))
	
	return cache
}

// ClearUserCache 清除指定用户的缓存
func (m *UserCacheManager) ClearUserCache(workspaceKey string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	if cache, exists := m.caches[workspaceKey]; exists {
		cache.Clear()
		delete(m.caches, workspaceKey)
		logging.LogInfof("清除 workspace [%s] 的缓存", workspaceKey)
	}
}

// cleanupRoutine 定期清理空闲缓存
func (m *UserCacheManager) cleanupRoutine() {
	ticker := time.NewTicker(5 * time.Minute) // 每5分钟检查一次
	defer ticker.Stop()
	
	for range ticker.C {
		m.mutex.Lock()
		m.cleanupIdleCachesLocked()
		m.mutex.Unlock()
	}
}

// cleanupIdleCachesLocked 清理空闲缓存（需要持有锁）
func (m *UserCacheManager) cleanupIdleCachesLocked() {
	now := time.Now()
	var toRemove []string
	
	for workspace, cache := range m.caches {
		if now.Sub(cache.lastAccess) > m.maxIdle {
			toRemove = append(toRemove, workspace)
		}
	}
	
	for _, workspace := range toRemove {
		if cache, exists := m.caches[workspace]; exists {
			cache.Clear()
			delete(m.caches, workspace)
			logging.LogInfof("清理空闲缓存 [%s]", workspace)
		}
	}
	
	if len(toRemove) > 0 {
		logging.LogInfof("清理了 %d 个空闲缓存，剩余缓存数: %d", len(toRemove), len(m.caches))
	}
}

// GetStats 获取缓存管理器统计信息
func (m *UserCacheManager) GetStats() map[string]interface{} {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	return map[string]interface{}{
		"total_caches": len(m.caches),
		"max_caches":   m.maxCaches,
		"max_idle":     m.maxIdle.String(),
	}
}

// newUserCache 创建新的用户缓存
func newUserCache() *UserCache {
	blockCache, _ := ristretto.NewCache(&ristretto.Config{
		NumCounters: 102400,
		MaxCost:     10240,
		BufferItems: 64,
	})
	
	refCache := gcache.New(30*time.Minute, 5*time.Minute)
	
	return &UserCache{
		blockCache: blockCache,
		refCache:   refCache,
		lastAccess: time.Now(),
		enabled:    true,
	}
}

// Clear 清除缓存内容
func (c *UserCache) Clear() {
	c.blockCache.Clear()
	c.refCache.Flush()
}

// Enable 启用缓存
func (c *UserCache) Enable() {
	c.enabled = true
}

// Disable 禁用缓存
func (c *UserCache) Disable() {
	c.enabled = false
}

// IsEnabled 检查缓存是否启用
func (c *UserCache) IsEnabled() bool {
	return c.enabled
}

// GetBlockCache 获取 Block 缓存
func (c *UserCache) GetBlockCache() *ristretto.Cache {
	return c.blockCache
}

// GetRefCache 获取 Ref 缓存
func (c *UserCache) GetRefCache() *gcache.Cache {
	return c.refCache
}

// GetUserCacheWithContext 获取指定 WorkspaceContext 的用户缓存（全局函数）
func GetUserCacheWithContext(ctx WorkspaceContext) *UserCache {
	if globalUserCacheManager == nil {
		initUserCacheManager()
	}
	return globalUserCacheManager.GetUserCache(ctx)
}

// ClearUserCacheByWorkspace 清除指定 workspace 的缓存
func ClearUserCacheByWorkspace(workspaceKey string) {
	if globalUserCacheManager != nil {
		globalUserCacheManager.ClearUserCache(workspaceKey)
	}
}

// GetUserCacheStats 获取用户缓存统计信息
func GetUserCacheStats() map[string]interface{} {
	if globalUserCacheManager == nil {
		return map[string]interface{}{
			"total_caches": 0,
			"max_caches":   0,
		}
	}
	return globalUserCacheManager.GetStats()
}
