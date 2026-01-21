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

package sql

import (
	"database/sql"
	"errors"
	"path/filepath"
	"sync"
	"time"

	"github.com/siyuan-note/logging"
)

// 错误定义
var (
	ErrTooManyConnections = errors.New("数据库连接数已达上限")
)

// WorkspaceContext 接口，避免循环依赖
type WorkspaceContext interface {
	GetWorkspaceDir() string
	GetConfDir() string
	GetDataDir() string
}

// DBPool 数据库连接池，按 workspace 管理数据库连接
type DBPool struct {
	connections map[string]*sql.DB      // workspace -> db connection
	lastAccess  map[string]time.Time    // workspace -> last access time
	mutex       sync.RWMutex
	maxIdle     time.Duration           // 最大空闲时间
	maxConns    int                     // 最大连接数
}

// globalDBPool 全局数据库连接池实例
var globalDBPool *DBPool

// initDBPool 初始化数据库连接池
func initDBPool() {
	globalDBPool = &DBPool{
		connections: make(map[string]*sql.DB),
		lastAccess:  make(map[string]time.Time),
		maxIdle:     30 * time.Minute, // 30分钟未使用则关闭连接
		maxConns:    100,               // 最多100个并发连接
	}
	
	// 启动定期清理协程
	go globalDBPool.cleanupRoutine()
}

// GetDB 获取指定 workspace 的数据库连接
func (pool *DBPool) GetDB(ctx WorkspaceContext) (*sql.DB, error) {
	workspaceKey := ctx.GetWorkspaceDir()
	
	// 1. 尝试获取已有连接
	pool.mutex.RLock()
	db, exists := pool.connections[workspaceKey]
	pool.mutex.RUnlock()
	
	if exists {
		// 更新最后访问时间
		pool.mutex.Lock()
		pool.lastAccess[workspaceKey] = time.Now()
		pool.mutex.Unlock()
		return db, nil
	}
	
	// 2. 创建新连接
	pool.mutex.Lock()
	defer pool.mutex.Unlock()
	
	// 双重检查
	if db, exists := pool.connections[workspaceKey]; exists {
		pool.lastAccess[workspaceKey] = time.Now()
		return db, nil
	}
	
	// 检查连接数限制
	if len(pool.connections) >= pool.maxConns {
		logging.LogWarnf("数据库连接数已达上限 %d，尝试清理空闲连接", pool.maxConns)
		pool.cleanupIdleConnectionsLocked()
		
		// 如果清理后仍然超限，返回错误
		if len(pool.connections) >= pool.maxConns {
			logging.LogErrorf("数据库连接数超限，无法创建新连接")
			return nil, ErrTooManyConnections
		}
	}
	
	// 打开数据库
	dbPath := filepath.Join(ctx.GetConfDir(), "siyuan.db")
	db, err := openDatabase(dbPath)
	if err != nil {
		logging.LogErrorf("打开数据库失败 [%s]: %s", dbPath, err)
		return nil, err
	}
	
	pool.connections[workspaceKey] = db
	pool.lastAccess[workspaceKey] = time.Now()
	
	logging.LogInfof("为 workspace [%s] 创建新的数据库连接，当前连接数: %d", workspaceKey, len(pool.connections))
	
	return db, nil
}

// CloseDB 关闭指定 workspace 的数据库连接
func (pool *DBPool) CloseDB(workspaceKey string) error {
	pool.mutex.Lock()
	defer pool.mutex.Unlock()
	
	db, exists := pool.connections[workspaceKey]
	if !exists {
		return nil
	}
	
	err := db.Close()
	if err != nil {
		logging.LogErrorf("关闭数据库连接失败 [%s]: %s", workspaceKey, err)
		return err
	}
	
	delete(pool.connections, workspaceKey)
	delete(pool.lastAccess, workspaceKey)
	
	logging.LogInfof("关闭 workspace [%s] 的数据库连接，剩余连接数: %d", workspaceKey, len(pool.connections))
	
	return nil
}

// cleanupRoutine 定期清理空闲连接
func (pool *DBPool) cleanupRoutine() {
	ticker := time.NewTicker(5 * time.Minute) // 每5分钟检查一次
	defer ticker.Stop()
	
	for range ticker.C {
		pool.mutex.Lock()
		pool.cleanupIdleConnectionsLocked()
		pool.mutex.Unlock()
	}
}

// cleanupIdleConnectionsLocked 清理空闲连接（需要持有锁）
func (pool *DBPool) cleanupIdleConnectionsLocked() {
	now := time.Now()
	var toClose []string
	
	for workspace, lastTime := range pool.lastAccess {
		if now.Sub(lastTime) > pool.maxIdle {
			toClose = append(toClose, workspace)
		}
	}
	
	for _, workspace := range toClose {
		if db, exists := pool.connections[workspace]; exists {
			err := db.Close()
			if err != nil {
				logging.LogErrorf("清理空闲连接失败 [%s]: %s", workspace, err)
			} else {
				logging.LogInfof("清理空闲连接 [%s]", workspace)
			}
			delete(pool.connections, workspace)
			delete(pool.lastAccess, workspace)
		}
	}
	
	if len(toClose) > 0 {
		logging.LogInfof("清理了 %d 个空闲连接，剩余连接数: %d", len(toClose), len(pool.connections))
	}
}

// CloseAll 关闭所有数据库连接
func (pool *DBPool) CloseAll() {
	pool.mutex.Lock()
	defer pool.mutex.Unlock()
	
	for workspace, db := range pool.connections {
		err := db.Close()
		if err != nil {
			logging.LogErrorf("关闭数据库连接失败 [%s]: %s", workspace, err)
		}
	}
	
	pool.connections = make(map[string]*sql.DB)
	pool.lastAccess = make(map[string]time.Time)
	
	logging.LogInfof("已关闭所有数据库连接")
}

// GetStats 获取连接池统计信息
func (pool *DBPool) GetStats() map[string]interface{} {
	pool.mutex.RLock()
	defer pool.mutex.RUnlock()
	
	return map[string]interface{}{
		"total_connections": len(pool.connections),
		"max_connections":   pool.maxConns,
		"max_idle_time":     pool.maxIdle.String(),
	}
}

// openDatabase 打开数据库连接
func openDatabase(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}
	
	// 设置连接池参数
	db.SetMaxOpenConns(1)  // SQLite 建议单连接
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)
	
	// 测试连接
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	
	return db, nil
}

// GetDBWithContext 获取指定 WorkspaceContext 的数据库连接（全局函数）
func GetDBWithContext(ctx WorkspaceContext) (*sql.DB, error) {
	if globalDBPool == nil {
		initDBPool()
	}
	return globalDBPool.GetDB(ctx)
}

// CloseDBPool 关闭数据库连接池
func CloseDBPool() {
	if globalDBPool != nil {
		globalDBPool.CloseAll()
	}
}

// GetDBPoolStats 获取连接池统计信息
func GetDBPoolStats() map[string]interface{} {
	if globalDBPool == nil {
		return map[string]interface{}{
			"total_connections": 0,
			"max_connections":   0,
		}
	}
	return globalDBPool.GetStats()
}
