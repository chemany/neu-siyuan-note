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
	"fmt"
	"os"
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
	GetTempDir() string
	GetAssetContentDBPath() string
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
func (pool *DBPool) GetDB(ctx WorkspaceContextInterface) (*sql.DB, error) {
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
	logging.LogInfof("正在打开数据库: %s", dbPath)
	
	// 确保父目录存在
	dbDir := filepath.Dir(dbPath)
	logging.LogInfof("确保数据库目录存在: %s", dbDir)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		logging.LogErrorf("创建数据库目录失败 [%s]: %s", dbDir, err)
		return nil, err
	}
	logging.LogInfof("数据库目录已确保存在: %s", dbDir)
	
	// 检查数据库文件是否存在
	dbExists := fileExists(dbPath)
	logging.LogInfof("数据库文件是否存在: %v [%s]", dbExists, dbPath)
	
	dsn := dbPath + "?_journal_mode=WAL" +
		"&_synchronous=OFF" +
		"&_mmap_size=2684354560" +
		"&_secure_delete=OFF" +
		"&_cache_size=-20480" +
		"&_page_size=32768" +
		"&_busy_timeout=7000" +
		"&_ignore_check_constraints=ON" +
		"&_temp_store=MEMORY" +
		"&_case_sensitive_like=OFF"
	
	logging.LogInfof("正在打开 SQLite 数据库...")
	db, err := sql.Open("sqlite3_extended", dsn)
	if err != nil {
		logging.LogErrorf("打开数据库失败: %s", err)
		return nil, err
	}
	logging.LogInfof("SQLite 数据库已打开")
	
	// 设置连接池参数
	db.SetMaxOpenConns(1)  // SQLite 建议单连接
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)
	
	// 测试连接
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	
	// 如果数据库是新创建的，初始化表结构
	if !dbExists {
		logging.LogInfof("数据库不存在，正在初始化: %s", dbPath)
		if err := initDatabaseTables(db); err != nil {
			logging.LogErrorf("初始化数据库表失败 [%s]: %s", dbPath, err)
			db.Close()
			return nil, err
		}
		logging.LogInfof("数据库初始化完成: %s", dbPath)
	}
	
	// 确保 FTS 表存在
	if err := ensureFTSTables(db); err != nil {
		logging.LogErrorf("确保 FTS 表存在失败 [%s]: %s", dbPath, err)
		// 不返回错误，继续使用数据库（FTS 表不是必需的）
	}
	
	return db, nil
}

// fileExists 检查文件是否存在
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// initDatabaseTables 初始化数据库表结构
func initDatabaseTables(db *sql.DB) error {
	// 创建 stat 表
	_, err := db.Exec("CREATE TABLE IF NOT EXISTS stat (key TEXT, value TEXT)")
	if err != nil {
		return fmt.Errorf("创建 stat 表失败: %w", err)
	}
	
	// 设置数据库版本
	_, err = db.Exec("INSERT INTO stat (key, value) VALUES ('dbver', '20211230184000')")
	if err != nil {
		return fmt.Errorf("设置数据库版本失败: %w", err)
	}
	
	// 创建 blocks 表
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS blocks (
		id TEXT, 
		parent_id TEXT, 
		root_id TEXT, 
		hash TEXT, 
		box TEXT, 
		path TEXT, 
		hpath TEXT, 
		name TEXT, 
		alias TEXT, 
		memo TEXT, 
		tag TEXT, 
		content TEXT, 
		fcontent TEXT, 
		markdown TEXT, 
		length INTEGER, 
		type TEXT, 
		subtype TEXT, 
		ial TEXT, 
		sort INTEGER, 
		created TEXT, 
		updated TEXT
	)`)
	if err != nil {
		return fmt.Errorf("创建 blocks 表失败: %w", err)
	}
	
	// 创建索引
	_, err = db.Exec("CREATE INDEX IF NOT EXISTS idx_blocks_id ON blocks(id)")
	if err != nil {
		return fmt.Errorf("创建 idx_blocks_id 索引失败: %w", err)
	}
	
	_, err = db.Exec("CREATE INDEX IF NOT EXISTS idx_blocks_parent_id ON blocks(parent_id)")
	if err != nil {
		return fmt.Errorf("创建 idx_blocks_parent_id 索引失败: %w", err)
	}
	
	_, err = db.Exec("CREATE INDEX IF NOT EXISTS idx_blocks_root_id ON blocks(root_id)")
	if err != nil {
		return fmt.Errorf("创建 idx_blocks_root_id 索引失败: %w", err)
	}
	
	// 创建其他必要的表
	tables := []string{
		`CREATE TABLE IF NOT EXISTS spans (
			id TEXT, 
			block_id TEXT, 
			root_id TEXT, 
			box TEXT, 
			path TEXT, 
			content TEXT, 
			markdown TEXT, 
			type TEXT, 
			ial TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS assets (
			id TEXT, 
			block_id TEXT, 
			root_id TEXT, 
			box TEXT, 
			docpath TEXT, 
			path TEXT, 
			name TEXT, 
			title TEXT, 
			hash TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS attributes (
			id TEXT, 
			name TEXT, 
			value TEXT, 
			type TEXT, 
			block_id TEXT, 
			root_id TEXT, 
			box TEXT, 
			path TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS refs (
			id TEXT, 
			def_block_id TEXT, 
			def_block_parent_id TEXT, 
			def_block_root_id TEXT, 
			def_block_path TEXT, 
			block_id TEXT, 
			root_id TEXT, 
			box TEXT, 
			path TEXT, 
			content TEXT, 
			markdown TEXT, 
			type TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS file_annotation_refs (
			id TEXT, 
			file_path TEXT, 
			annotation_id TEXT, 
			block_id TEXT, 
			root_id TEXT, 
			box TEXT, 
			path TEXT, 
			content TEXT, 
			type TEXT
		)`,
	}
	
	for _, createSQL := range tables {
		if _, err := db.Exec(createSQL); err != nil {
			return fmt.Errorf("创建表失败: %w", err)
		}
	}
	
	// 创建必要的索引
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_spans_root_id ON spans(root_id)",
		"CREATE INDEX IF NOT EXISTS idx_assets_root_id ON assets(root_id)",
		"CREATE INDEX IF NOT EXISTS idx_attributes_block_id ON attributes(block_id)",
		"CREATE INDEX IF NOT EXISTS idx_attributes_root_id ON attributes(root_id)",
	}
	
	for _, indexSQL := range indexes {
		if _, err := db.Exec(indexSQL); err != nil {
			return fmt.Errorf("创建索引失败: %w", err)
		}
	}
	
	logging.LogInfof("数据库表结构初始化完成")
	return nil
}

// ensureFTSTables 确保 FTS 表存在
func ensureFTSTables(db *sql.DB) error {
	// 检查 blocks_fts 表是否存在
	var tableName string
	err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='blocks_fts'").Scan(&tableName)
	if err == sql.ErrNoRows {
		// 表不存在，创建 FTS 表
		logging.LogInfof("FTS 表不存在，正在创建...")
		
		// 创建 blocks_fts 表
		_, err = db.Exec("CREATE VIRTUAL TABLE blocks_fts USING fts5(id UNINDEXED, parent_id UNINDEXED, root_id UNINDEXED, hash UNINDEXED, box UNINDEXED, path UNINDEXED, hpath, name, alias, memo, tag, content, fcontent, markdown UNINDEXED, length UNINDEXED, type UNINDEXED, subtype UNINDEXED, ial, sort UNINDEXED, created UNINDEXED, updated UNINDEXED, tokenize=\"siyuan\")")
		if err != nil {
			logging.LogErrorf("创建 blocks_fts 表失败: %s", err)
			return err
		}
		
		// 创建 blocks_fts_case_insensitive 表
		_, err = db.Exec("CREATE VIRTUAL TABLE blocks_fts_case_insensitive USING fts5(id UNINDEXED, parent_id UNINDEXED, root_id UNINDEXED, hash UNINDEXED, box UNINDEXED, path UNINDEXED, hpath, name, alias, memo, tag, content, fcontent, markdown UNINDEXED, length UNINDEXED, type UNINDEXED, subtype UNINDEXED, ial, sort UNINDEXED, created UNINDEXED, updated UNINDEXED, tokenize=\"siyuan case_insensitive\")")
		if err != nil {
			logging.LogErrorf("创建 blocks_fts_case_insensitive 表失败: %s", err)
			return err
		}
		
		logging.LogInfof("FTS 表创建成功")
		
		// 从 blocks 表复制数据到 FTS 表
		_, err = db.Exec("INSERT INTO blocks_fts SELECT * FROM blocks")
		if err != nil {
			logging.LogWarnf("复制数据到 blocks_fts 失败: %s", err)
			// 不返回错误，因为 blocks 表可能为空
		}
		
		_, err = db.Exec("INSERT INTO blocks_fts_case_insensitive SELECT * FROM blocks")
		if err != nil {
			logging.LogWarnf("复制数据到 blocks_fts_case_insensitive 失败: %s", err)
			// 不返回错误，因为 blocks 表可能为空
		}
		
		logging.LogInfof("FTS 表数据同步完成")
	} else if err != nil {
		logging.LogErrorf("检查 FTS 表是否存在失败: %s", err)
		return err
	} else {
		logging.LogInfof("FTS 表已存在: %s", tableName)
	}
	
	return nil
}

// GetDBWithContext 获取指定 WorkspaceContext 的数据库连接（全局函数）
func GetDBWithContext(ctx WorkspaceContextInterface) (*sql.DB, error) {
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
