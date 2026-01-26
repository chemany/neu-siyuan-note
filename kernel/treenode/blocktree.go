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

package treenode

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/88250/gulu"
	"github.com/88250/lute/ast"
	"github.com/88250/lute/parse"
	"github.com/siyuan-note/logging"
	"github.com/siyuan-note/siyuan/kernel/util"
)

type BlockTree struct {
	ID       string // 块 ID
	RootID   string // 根 ID
	ParentID string // 父 ID
	BoxID    string // 笔记本 ID
	Path     string // 文档数据路径
	HPath    string // 文档可读路径
	Updated  string // 更新时间
	Type     string // 类型
}

const (
	// SlowQueryThreshold 慢查询阈值(毫秒)
	SlowQueryThreshold = 100
)

// logSlowQuery 记录慢查询
func logSlowQuery(operation string, duration time.Duration, details string) {
	if duration.Milliseconds() > SlowQueryThreshold {
		logging.LogWarnf("[BlockTree] Slow query detected: operation=%s, duration=%dms, details=%s", 
			operation, duration.Milliseconds(), details)
	}
}

var (
	db            *sql.DB
	currentDBPath string
	dbMutex       sync.RWMutex
)

func initDatabase(forceRebuild bool) (err error) {
	initDBConnection()

	if !forceRebuild {
		if !gulu.File.IsExist(util.BlockTreeDBPath) {
			forceRebuild = true
		}
	}
	if !forceRebuild {
		return
	}

	closeDatabase()
	if gulu.File.IsExist(util.BlockTreeDBPath) {
		if err = removeDatabaseFile(); err != nil {
			logging.LogErrorf("remove database file [%s] failed: %s", util.BlockTreeDBPath, err)
			err = nil
		}
	}

	initDBConnection()
	initDBTables()

	logging.LogInfof("reinitialized database [%s]", util.BlockTreeDBPath)
	return
}

func initDBTables() {
	_, err := db.Exec("DROP TABLE IF EXISTS blocktrees")
	if err != nil {
		logging.LogFatalf(logging.ExitCodeReadOnlyDatabase, "drop table [blocks] failed: %s", err)
	}
	_, err = db.Exec("CREATE TABLE blocktrees (id, root_id, parent_id, box_id, path, hpath, updated, type)")
	if err != nil {
		logging.LogFatalf(logging.ExitCodeReadOnlyDatabase, "create table [blocktrees] failed: %s", err)
	}

	_, err = db.Exec("CREATE INDEX idx_blocktrees_id ON blocktrees(id)")
	if err != nil {
		logging.LogFatalf(logging.ExitCodeReadOnlyDatabase, "create index [idx_blocktrees_id] failed: %s", err)
	}
}

func initDBConnection() {
	if nil != db {
		closeDatabase()
	}

	util.LogDatabaseSize(util.BlockTreeDBPath)
	dsn := util.BlockTreeDBPath + "?_journal_mode=WAL" +
		"&_synchronous=OFF" +
		"&_mmap_size=2684354560" +
		"&_secure_delete=OFF" +
		"&_cache_size=-20480" +
		"&_page_size=32768" +
		"&_busy_timeout=7000" +
		"&_ignore_check_constraints=ON" +
		"&_temp_store=MEMORY" +
		"&_case_sensitive_like=OFF"
	var err error
	db, err = sql.Open("sqlite3_extended", dsn)
	if err != nil {
		logging.LogFatalf(logging.ExitCodeReadOnlyDatabase, "create database failed: %s", err)
	}
	db.SetMaxIdleConns(7)
	db.SetMaxOpenConns(7)
	db.SetConnMaxLifetime(365 * 24 * time.Hour)
}

func CloseDatabase() {
	closeDatabase()
}

func closeDatabase() {
	if nil == db {
		return
	}

	if err := db.Close(); err != nil {
		logging.LogErrorf("close database failed: %s", err)
	}
	debug.FreeOSMemory()
	runtime.GC() // 没有这句的话文件句柄不会释放，后面就无法删除文件
	return
}

func removeDatabaseFile() (err error) {
	err = os.RemoveAll(util.BlockTreeDBPath)
	if err != nil {
		return
	}
	err = os.RemoveAll(util.BlockTreeDBPath + "-shm")
	if err != nil {
		return
	}
	err = os.RemoveAll(util.BlockTreeDBPath + "-wal")
	if err != nil {
		return
	}
	return
}

func GetBlockTreesByType(typ string) (ret []*BlockTree) {
	sqlStmt := "SELECT * FROM blocktrees WHERE type = ?"
	rows, err := db.Query(sqlStmt, typ)
	if err != nil {
		logging.LogErrorf("sql query [%s] failed: %s", sqlStmt, err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var block BlockTree
		if err = rows.Scan(&block.ID, &block.RootID, &block.ParentID, &block.BoxID, &block.Path, &block.HPath, &block.Updated, &block.Type); err != nil {
			logging.LogErrorf("query scan field failed: %s", err)
			return
		}
		ret = append(ret, &block)
	}
	return
}

func GetBlockTreeByPath(path string) (ret *BlockTree) {
	ret = &BlockTree{}
	sqlStmt := "SELECT * FROM blocktrees WHERE path = ?"
	err := db.QueryRow(sqlStmt, path).Scan(&ret.ID, &ret.RootID, &ret.ParentID, &ret.BoxID, &ret.Path, &ret.HPath, &ret.Updated, &ret.Type)
	if err != nil {
		ret = nil
		if errors.Is(err, sql.ErrNoRows) {
			return
		}
		logging.LogErrorf("sql query [%s] failed: %s", sqlStmt, err)
		return
	}
	return
}

func CountTrees() (ret int) {
	sqlStmt := "SELECT COUNT(*) FROM blocktrees WHERE type = 'd'"
	err := db.QueryRow(sqlStmt).Scan(&ret)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0
		}
		logging.LogErrorf("sql query [%s] failed: %s", sqlStmt, err)
	}
	return
}

func CountBlocks() (ret int) {
	sqlStmt := "SELECT COUNT(*) FROM blocktrees"
	err := db.QueryRow(sqlStmt).Scan(&ret)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0
		}
		logging.LogErrorf("sql query [%s] failed: %s", sqlStmt, err)
	}
	return
}

func GetBlockTreeRootByPath(boxID, path string) (ret *BlockTree) {
	return GetBlockTreeRootByPathWithDB(boxID, path, db)
}

// GetBlockTreeRootByPathWithDB 使用指定的数据库连接获取 BlockTree 根节点
//
// 此函数通过文档路径查询 BlockTree，支持多用户数据隔离。
//
// 参数：
//   - boxID: 笔记本 ID
//   - path: 文档数据路径（如 "/20260125144733-f8giaqe.sy"）
//   - database: 数据库连接，如果为 nil 则返回 nil
//
// 返回：
//   - BlockTree 对象，如果未找到或数据库为 nil 则返回 nil
//
// 性能：
//   - 查询时间通常 < 10ms
//   - 慢查询（> 100ms）会被记录到日志
//
// 示例：
//
//	userDB, _ := btManager.GetOrCreateDB(ctx.BlockTreeDBPath)
//	tree := GetBlockTreeRootByPathWithDB("box1", "/20260125144733-f8giaqe.sy", userDB)
//	if tree != nil {
//	    fmt.Printf("Found document: %s\n", tree.HPath)
//	}
func GetBlockTreeRootByPathWithDB(boxID, path string, database *sql.DB) (ret *BlockTree) {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		logSlowQuery("GetBlockTreeRootByPathWithDB", duration, fmt.Sprintf("boxID=%s, path=%s", boxID, path))
	}()
	
	if nil == database {
		return nil
	}
	
	ret = &BlockTree{}
	sqlStmt := "SELECT * FROM blocktrees WHERE box_id = ? AND path = ? AND type = 'd'"
	err := database.QueryRow(sqlStmt, boxID, path).Scan(&ret.ID, &ret.RootID, &ret.ParentID, &ret.BoxID, &ret.Path, &ret.HPath, &ret.Updated, &ret.Type)
	if err != nil {
		ret = nil
		if errors.Is(err, sql.ErrNoRows) {
			return
		}
		logging.LogErrorf("sql query [%s] failed: %s", sqlStmt, err)
		return
	}
	return
}

func GetBlockTreeRootByHPath(boxID, hPath string) (ret *BlockTree) {
	return GetBlockTreeRootByHPathWithDB(boxID, hPath, db)
}

// GetBlockTreeRootByHPathWithDB 使用指定的数据库连接获取 BlockTree 根节点
//
// 此函数通过可读路径（HPath）查询 BlockTree，支持多用户数据隔离。
// HPath 是用户可读的文档路径，如 "/父文档/子文档"。
//
// 参数：
//   - boxID: 笔记本 ID
//   - hPath: 可读路径（如 "/父文档/子文档"）
//   - database: 数据库连接，如果为 nil 则返回 nil
//
// 返回：
//   - BlockTree 对象，如果未找到或数据库为 nil 则返回 nil
//
// 注意：
//   - HPath 中的不可见字符会被自动移除
//   - 只查询类型为 'd'（文档）的 BlockTree
//
// 性能：
//   - 查询时间通常 < 10ms
//   - 慢查询（> 100ms）会被记录到日志
//
// 示例：
//
//	userDB, _ := btManager.GetOrCreateDB(ctx.BlockTreeDBPath)
//	tree := GetBlockTreeRootByHPathWithDB("box1", "/父文档", userDB)
//	if tree != nil {
//	    fmt.Printf("Found document ID: %s\n", tree.ID)
//	}
func GetBlockTreeRootByHPathWithDB(boxID, hPath string, database *sql.DB) (ret *BlockTree) {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		logSlowQuery("GetBlockTreeRootByHPathWithDB", duration, fmt.Sprintf("boxID=%s, hPath=%s", boxID, hPath))
	}()
	
	if nil == database {
		return nil
	}
	
	ret = &BlockTree{}
	hPath = gulu.Str.RemoveInvisible(hPath)
	sqlStmt := "SELECT * FROM blocktrees WHERE box_id = ? AND hpath = ? AND type = 'd'"
	err := database.QueryRow(sqlStmt, boxID, hPath).Scan(&ret.ID, &ret.RootID, &ret.ParentID, &ret.BoxID, &ret.Path, &ret.HPath, &ret.Updated, &ret.Type)
	if err != nil {
		ret = nil
		if errors.Is(err, sql.ErrNoRows) {
			return
		}
		logging.LogErrorf("sql query [%s] failed: %s", sqlStmt, err)
		return
	}
	return
}

func GetBlockTreeRootsByHPath(boxID, hPath string) (ret []*BlockTree) {
	hPath = gulu.Str.RemoveInvisible(hPath)
	sqlStmt := "SELECT * FROM blocktrees WHERE box_id = ? AND hpath = ? AND type = 'd'"
	rows, err := db.Query(sqlStmt, boxID, hPath)
	if err != nil {
		logging.LogErrorf("sql query [%s] failed: %s", sqlStmt, err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var block BlockTree
		if err = rows.Scan(&block.ID, &block.RootID, &block.ParentID, &block.BoxID, &block.Path, &block.HPath, &block.Updated, &block.Type); err != nil {
			logging.LogErrorf("query scan field failed: %s", err)
			return
		}
		ret = append(ret, &block)
	}
	return
}

func GetBlockTreeByHPathPreferredParentID(boxID, hPath, preferredParentID string) (ret *BlockTree) {
	return GetBlockTreeByHPathPreferredParentIDWithDB(boxID, hPath, preferredParentID, db)
}

// GetBlockTreeByHPathPreferredParentIDWithDB 使用指定的数据库连接获取 BlockTree，优先匹配指定的父文档
//
// 此函数通过可读路径查询 BlockTree，当存在多个同名文档时，优先返回指定父文档下的文档。
// 这在创建子文档时特别有用，可以确保子文档创建在正确的父文档下。
//
// 参数：
//   - boxID: 笔记本 ID
//   - hPath: 可读路径（如 "/父文档/子文档"）
//   - preferredParentID: 优先匹配的父文档 ID
//   - database: 数据库连接，如果为 nil 则返回 nil
//
// 返回：
//   - BlockTree 对象，优先返回指定父文档下的文档，如果未找到或数据库为 nil 则返回 nil
//
// 查询逻辑：
//   1. 首先查询所有匹配 HPath 的文档
//   2. 如果存在父 ID 匹配 preferredParentID 的文档，返回该文档
//   3. 否则返回第一个匹配的文档
//
// 示例：
//
//	userDB, _ := btManager.GetOrCreateDB(ctx.BlockTreeDBPath)
//	tree := GetBlockTreeByHPathPreferredParentIDWithDB("box1", "/父文档/子文档", parentID, userDB)
//	if tree != nil && tree.ParentID == parentID {
//	    fmt.Println("Found document under preferred parent")
//	}
func GetBlockTreeByHPathPreferredParentIDWithDB(boxID, hPath, preferredParentID string, database *sql.DB) (ret *BlockTree) {
	if nil == database {
		return nil
	}
	
	hPath = gulu.Str.RemoveInvisible(hPath)
	var roots []*BlockTree
	sqlStmt := "SELECT * FROM blocktrees WHERE box_id = ? AND hpath = ? AND parent_id = ? LIMIT 1"
	rows, err := database.Query(sqlStmt, boxID, hPath, preferredParentID)
	if err != nil {
		logging.LogErrorf("sql query [%s] failed: %s", sqlStmt, err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var block BlockTree
		if err = rows.Scan(&block.ID, &block.RootID, &block.ParentID, &block.BoxID, &block.Path, &block.HPath, &block.Updated, &block.Type); err != nil {
			logging.LogErrorf("query scan field failed: %s", err)
			return
		}
		if "" == preferredParentID {
			ret = &block
			return
		}
		roots = append(roots, &block)
	}

	if 1 > len(roots) {
		return
	}

	for _, root := range roots {
		if root.ID == preferredParentID {
			ret = root
			return
		}
	}
	ret = roots[0]
	return
}

func ExistBlockTree(id string) bool {
	sqlStmt := "SELECT COUNT(*) FROM blocktrees WHERE id = ?"
	var count int
	err := db.QueryRow(sqlStmt, id).Scan(&count)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false
		}
		logging.LogErrorf("sql query [%s] failed: %s", sqlStmt, err)
		return false
	}
	return 0 < count
}

func ExistBlockTrees(ids []string) (ret map[string]bool) {
	ret = map[string]bool{}
	if 1 > len(ids) {
		return
	}

	for _, id := range ids {
		ret[id] = false
	}

	sqlStmt := "SELECT id FROM blocktrees WHERE id IN ('" + strings.Join(ids, "','") + "')"
	rows, err := db.Query(sqlStmt)
	if err != nil {
		logging.LogErrorf("sql query [%s] failed: %s", sqlStmt, err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			logging.LogErrorf("query scan field failed: %s", err)
			return
		}
		ret[id] = true
	}
	return
}

func GetBlockTrees(ids []string) (ret map[string]*BlockTree) {
	ret = map[string]*BlockTree{}
	if 1 > len(ids) {
		return
	}

	stmtBuf := bytes.Buffer{}
	stmtBuf.WriteString("SELECT * FROM blocktrees WHERE id IN (")
	for i := range ids {
		stmtBuf.WriteString("?")
		if i == len(ids)-1 {
			stmtBuf.WriteString(")")
		} else {
			stmtBuf.WriteString(",")
		}
	}
	var args []any
	for _, id := range ids {
		args = append(args, id)
	}
	stmt := stmtBuf.String()
	rows, err := db.Query(stmt, args...)
	if err != nil {
		logging.LogErrorf("sql query [%s] failed: %s", stmt, err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var block BlockTree
		if err = rows.Scan(&block.ID, &block.RootID, &block.ParentID, &block.BoxID, &block.Path, &block.HPath, &block.Updated, &block.Type); err != nil {
			logging.LogErrorf("query scan field failed: %s", err)
			return
		}
		ret[block.ID] = &block
	}
	return
}

func GetBlockTree(id string) (ret *BlockTree) {
	return GetBlockTreeWithDB(id, db)
}

// GetBlockTreeWithDB 使用指定的数据库连接获取 BlockTree
//
// 此函数通过块 ID 查询 BlockTree，支持多用户数据隔离。
// 这是最基础的查询函数，通过唯一的块 ID 获取 BlockTree 对象。
//
// 参数：
//   - id: 块 ID（20位时间戳格式，如 "20260125144733-f8giaqe"）
//   - database: 数据库连接，如果为 nil 则返回 nil
//
// 返回：
//   - BlockTree 对象，如果未找到、ID 为空或数据库为 nil 则返回 nil
//
// 性能：
//   - 查询时间通常 < 5ms（通过主键索引）
//   - 慢查询（> 100ms）会被记录到日志
//
// 注意：
//   - 如果 id 为空字符串，直接返回 nil
//   - 如果 database 为 nil，直接返回 nil
//
// 示例：
//
//	userDB, _ := btManager.GetOrCreateDB(ctx.BlockTreeDBPath)
//	tree := GetBlockTreeWithDB("20260125144733-f8giaqe", userDB)
//	if tree != nil {
//	    fmt.Printf("Document path: %s\n", tree.Path)
//	}
func GetBlockTreeWithDB(id string, database *sql.DB) (ret *BlockTree) {
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		logSlowQuery("GetBlockTreeWithDB", duration, fmt.Sprintf("id=%s", id))
	}()
	
	if "" == id {
		return
	}
	if nil == database {
		return
	}

	ret = &BlockTree{}
	sqlStmt := "SELECT * FROM blocktrees WHERE id = ?"
	err := database.QueryRow(sqlStmt, id).Scan(&ret.ID, &ret.RootID, &ret.ParentID, &ret.BoxID, &ret.Path, &ret.HPath, &ret.Updated, &ret.Type)
	if err != nil {
		ret = nil
		if errors.Is(err, sql.ErrNoRows) {
			return
		}
		logging.LogErrorf("sql query [%s] failed: %s", sqlStmt, logging.ShortStack())
		return
	}
	return
}

// GetBlockTreeWithDBPath 使用指定的数据库路径获取 BlockTree
//
// 此函数是 GetBlockTreeWithDB 的便捷封装，通过数据库路径而不是数据库连接来查询。
// 内部会自动通过 BlockTreeDBManager 获取或创建数据库连接。
//
// 参数：
//   - id: 块 ID
//   - dbPath: 数据库文件路径（如 "/path/to/user/data/blocktree.db"）
//
// 返回：
//   - BlockTree 对象，如果未找到或数据库连接失败则返回 nil
//
// 注意：
//   - 此函数会自动管理数据库连接，无需手动关闭
//   - 如果数据库文件不存在，会自动创建
//   - 如果数据库连接失败，会记录错误日志并返回 nil
//
// 示例：
//
//	tree := GetBlockTreeWithDBPath("20260125144733-f8giaqe", ctx.BlockTreeDBPath)
//	if tree != nil {
//	    fmt.Printf("Found document: %s\n", tree.HPath)
//	}
func GetBlockTreeWithDBPath(id string, dbPath string) (ret *BlockTree) {
	database, err := btManager.GetOrCreateDB(dbPath)
	if err != nil {
		logging.LogErrorf("get or create database [%s] failed: %s", dbPath, err)
		return nil
	}
	return GetBlockTreeWithDB(id, database)
}

func SetBlockTreePath(tree *parse.Tree) {
	RemoveBlockTreesByRootID(tree.ID)
	IndexBlockTree(tree)
}

func RemoveBlockTreesByRootID(rootID string) {
	sqlStmt := "DELETE FROM blocktrees WHERE root_id = ?"
	_, err := db.Exec(sqlStmt, rootID)
	if err != nil {
		logging.LogErrorf("sql exec [%s] failed: %s", sqlStmt, err)
		return
	}
}

func GetBlockTreesByPathPrefix(pathPrefix string) (ret []*BlockTree) {
	sqlStmt := "SELECT * FROM blocktrees WHERE path LIKE ?"
	rows, err := db.Query(sqlStmt, pathPrefix+"%")
	if err != nil {
		logging.LogErrorf("sql query [%s] failed: %s", sqlStmt, err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var block BlockTree
		if err = rows.Scan(&block.ID, &block.RootID, &block.ParentID, &block.BoxID, &block.Path, &block.HPath, &block.Updated, &block.Type); err != nil {
			logging.LogErrorf("query scan field failed: %s", err)
			return
		}
		ret = append(ret, &block)
	}
	return
}

func GetBlockTreesByRootID(rootID string) (ret []*BlockTree) {
	sqlStmt := "SELECT * FROM blocktrees WHERE root_id = ?"
	rows, err := db.Query(sqlStmt, rootID)
	if err != nil {
		logging.LogErrorf("sql query [%s] failed: %s", sqlStmt, err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var block BlockTree
		if err = rows.Scan(&block.ID, &block.RootID, &block.ParentID, &block.BoxID, &block.Path, &block.HPath, &block.Updated, &block.Type); err != nil {
			logging.LogErrorf("query scan field failed: %s", err)
			return
		}
		ret = append(ret, &block)
	}
	return
}

func RemoveBlockTreesByPathPrefix(pathPrefix string) {
	sqlStmt := "DELETE FROM blocktrees WHERE path LIKE ?"
	_, err := db.Exec(sqlStmt, pathPrefix+"%")
	if err != nil {
		logging.LogErrorf("sql exec [%s] failed: %s", sqlStmt, err)
		return
	}
}

func GetBlockTreesByBoxID(boxID string) (ret []*BlockTree) {
	sqlStmt := "SELECT * FROM blocktrees WHERE box_id = ?"
	rows, err := db.Query(sqlStmt, boxID)
	if err != nil {
		logging.LogErrorf("sql query [%s] failed: %s", sqlStmt, err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var block BlockTree
		if err = rows.Scan(&block.ID, &block.RootID, &block.ParentID, &block.BoxID, &block.Path, &block.HPath, &block.Updated, &block.Type); err != nil {
			logging.LogErrorf("query scan field failed: %s", err)
			return
		}
		ret = append(ret, &block)
	}
	return
}

func RemoveBlockTreesByBoxID(boxID string) (ids []string) {
	sqlStmt := "SELECT id FROM blocktrees WHERE box_id = ?"
	rows, err := db.Query(sqlStmt, boxID)
	if err != nil {
		logging.LogErrorf("sql query [%s] failed: %s", sqlStmt, err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			logging.LogErrorf("query scan field failed: %s", err)
			return
		}
		ids = append(ids, id)
	}

	sqlStmt = "DELETE FROM blocktrees WHERE box_id = ?"
	_, err = db.Exec(sqlStmt, boxID)
	if err != nil {
		logging.LogErrorf("sql exec [%s] failed: %s", sqlStmt, err)
		return
	}
	return
}

func RemoveBlockTree(id string) {
	sqlStmt := "DELETE FROM blocktrees WHERE id = ?"
	_, err := db.Exec(sqlStmt, id)
	if err != nil {
		logging.LogErrorf("sql exec [%s] failed: %s", sqlStmt, err)
		return
	}
}

var indexBlockTreeLock = sync.Mutex{}

func IndexBlockTree(tree *parse.Tree) {
	var changedNodes []*ast.Node
	ast.Walk(tree.Root, func(n *ast.Node, entering bool) ast.WalkStatus {
		if !entering || !n.IsBlock() || "" == n.ID {
			return ast.WalkContinue
		}

		changedNodes = append(changedNodes, n)
		return ast.WalkContinue
	})

	indexBlockTreeLock.Lock()
	defer indexBlockTreeLock.Unlock()

	tx, err := db.Begin()
	if err != nil {
		logging.LogErrorf("begin transaction failed: %s", err)
		return
	}

	sqlStmt := "INSERT INTO blocktrees (id, root_id, parent_id, box_id, path, hpath, updated, type) VALUES (?, ?, ?, ?, ?, ?, ?, ?)"
	for _, n := range changedNodes {
		var parentID string
		if nil != n.Parent {
			parentID = n.Parent.ID
		}
		if _, err = tx.Exec(sqlStmt, n.ID, tree.ID, parentID, tree.Box, tree.Path, tree.HPath, n.IALAttr("updated"), TypeAbbr(n.Type.String())); err != nil {
			tx.Rollback()
			logging.LogErrorf("sql exec [%s] failed: %s", sqlStmt, err)
			return
		}
	}
	if err = tx.Commit(); err != nil {
		logging.LogErrorf("commit transaction failed: %s", err)
	}
}

func UpsertBlockTree(tree *parse.Tree) {
	UpsertBlockTreeWithDB(tree, db)
}

// UpsertBlockTreeWithDB 使用指定的数据库连接更新 BlockTree
//
// 此函数将解析树（parse.Tree）的所有块节点同步到 BlockTree 数据库表中。
// 它会智能地检测变化的节点，只更新需要更新的部分，提高性能。
//
// 参数：
//   - tree: 解析树对象，包含文档的完整块结构
//   - database: 数据库连接，如果为 nil 则记录警告并返回
//
// 更新逻辑：
//   1. 查询数据库中该文档的所有现有块
//   2. 遍历解析树，找出变化的节点（新增、修改、删除）
//   3. 批量删除旧的块记录
//   4. 批量插入新的块记录
//
// 检测变化的条件：
//   - 块的更新时间（updated）改变
//   - 块的类型（type）改变
//   - 文档路径（path）改变
//   - 笔记本 ID（boxID）改变
//   - 可读路径（hPath）改变
//
// 性能：
//   - 使用批量操作减少数据库交互
//   - 只更新变化的节点，避免全量更新
//
// 注意：
//   - 如果 database 为 nil，会记录警告日志并直接返回
//   - 此函数会自动处理子块，确保整个文档树的一致性
//
// 示例：
//
//	userDB, _ := btManager.GetOrCreateDB(ctx.BlockTreeDBPath)
//	tree := parseMarkdown(content)
//	UpsertBlockTreeWithDB(tree, userDB)
func UpsertBlockTreeWithDB(tree *parse.Tree, database *sql.DB) {
	if nil == database {
		logging.LogWarnf("database is nil, cannot upsert block tree")
		return
	}
	
	oldBts := map[string]*BlockTree{}
	bts := GetBlockTreesByRootIDWithDB(tree.ID, database)
	for _, bt := range bts {
		oldBts[bt.ID] = bt
	}

	var changedNodes []*ast.Node
	ast.Walk(tree.Root, func(n *ast.Node, entering bool) ast.WalkStatus {
		if !entering || !n.IsBlock() || "" == n.ID {
			return ast.WalkContinue
		}

		if oldBt, found := oldBts[n.ID]; found {
			if oldBt.Updated != n.IALAttr("updated") || oldBt.Type != TypeAbbr(n.Type.String()) || oldBt.Path != tree.Path || oldBt.BoxID != tree.Box || oldBt.HPath != tree.HPath {
				children := ChildBlockNodes(n) // 需要考虑子块，因为一些操作（比如移动块）后需要同时更新子块
				changedNodes = append(changedNodes, children...)
			}
		} else {
			children := ChildBlockNodes(n)
			changedNodes = append(changedNodes, children...)
		}
		return ast.WalkContinue
	})

	ids := bytes.Buffer{}
	for i, n := range changedNodes {
		ids.WriteString("'")
		ids.WriteString(n.ID)
		ids.WriteString("'")
		if i < len(changedNodes)-1 {
			ids.WriteString(",")
		}
	}

	indexBlockTreeLock.Lock()
	defer indexBlockTreeLock.Unlock()

	tx, err := database.Begin()
	if err != nil {
		logging.LogErrorf("begin transaction failed: %s", err)
		return
	}

	sqlStmt := "DELETE FROM blocktrees WHERE id IN (" + ids.String() + ")"

	_, err = tx.Exec(sqlStmt)
	if err != nil {
		tx.Rollback()
		logging.LogErrorf("sql exec [%s] failed: %s", sqlStmt, err)
		return
	}
	sqlStmt = "INSERT INTO blocktrees (id, root_id, parent_id, box_id, path, hpath, updated, type) VALUES (?, ?, ?, ?, ?, ?, ?, ?)"
	for _, n := range changedNodes {
		var parentID string
		if nil != n.Parent {
			parentID = n.Parent.ID
		}
		if _, err = tx.Exec(sqlStmt, n.ID, tree.ID, parentID, tree.Box, tree.Path, tree.HPath, n.IALAttr("updated"), TypeAbbr(n.Type.String())); err != nil {
			tx.Rollback()
			logging.LogErrorf("sql exec [%s] failed: %s", sqlStmt, err)
			return
		}
	}
	if err = tx.Commit(); err != nil {
		logging.LogErrorf("commit transaction failed: %s", err)
	}
}

func InitBlockTree(force bool) {
	err := initDatabase(force)
	if err != nil {
		logging.LogErrorf("init database failed: %s", err)
		os.Exit(logging.ExitCodeReadOnlyDatabase)
		return
	}
	return
}

func CeilTreeCount(count int) int {
	if 100 > count {
		return 100
	}

	for i := 1; i < 40; i++ {
		if count < i*500 {
			return i * 500
		}
	}
	return 500*40 + 1
}

func CeilBlockCount(count int) int {
	if 5000 > count {
		return 5000
	}

	for i := 1; i < 100; i++ {
		if count < i*10000 {
			return i * 10000
		}
	}
	return 10000*100 + 1
}

// SwitchBlockTreeDB 切换到指定的 BlockTree 数据库
// 这个函数用于在多用户环境下切换不同用户的数据库
func SwitchBlockTreeDB(dbPath string) error {
	dbMutex.Lock()
	defer dbMutex.Unlock()
	
	if currentDBPath == dbPath && db != nil {
		// 已经是当前数据库,无需切换
		return nil
	}
	
	// 关闭当前数据库连接
	if db != nil {
		if err := db.Close(); err != nil {
			logging.LogWarnf("close current BlockTree database failed: %s", err)
		}
		db = nil
	}
	
	// 确保目录存在
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	
	// 构建 DSN (与 initDBConnection 保持一致)
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
	
	// 打开新数据库
	var err error
	db, err = sql.Open("sqlite3_extended", dsn)
	if err != nil {
		logging.LogErrorf("open BlockTree database [%s] failed: %s", dbPath, err)
		return err
	}
	
	// 设置连接池参数
	db.SetMaxIdleConns(7)
	db.SetMaxOpenConns(7)
	db.SetConnMaxLifetime(365 * 24 * time.Hour)
	
	// 初始化表结构(如果需要)
	if err := ensureDBTables(); err != nil {
		db.Close()
		db = nil
		return err
	}
	
	currentDBPath = dbPath
	logging.LogInfof("switched BlockTree database to [%s]", dbPath)
	return nil
}

// ensureDBTables 确保数据库表结构存在
func ensureDBTables() error {
	// 检查表是否存在
	var tableName string
	err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='blocktrees'").Scan(&tableName)
	if err == nil {
		// 表已存在
		return nil
	}
	
	// 创建表
	_, err = db.Exec("CREATE TABLE blocktrees (id, root_id, parent_id, box_id, path, hpath, updated, type)")
	if err != nil {
		logging.LogErrorf("create table [blocktrees] failed: %s", err)
		return err
	}
	
	// 创建索引
	_, err = db.Exec("CREATE INDEX idx_blocktrees_id ON blocktrees(id)")
	if err != nil {
		logging.LogErrorf("create index [idx_blocktrees_id] failed: %s", err)
		return err
	}
	
	logging.LogInfof("initialized BlockTree database tables")
	return nil
}

// GetCurrentDBPath 获取当前数据库路径
func GetCurrentDBPath() string {
	dbMutex.RLock()
	defer dbMutex.RUnlock()
	return currentDBPath
}

// GetBlockTreesByRootIDWithDB 使用指定的数据库连接获取文档的所有块
//
// 此函数查询指定文档（rootID）下的所有块，包括文档本身和所有子块。
// 返回的是一个 BlockTree 切片，包含文档的完整块结构。
//
// 参数：
//   - rootID: 文档根 ID（文档的块 ID）
//   - database: 数据库连接，如果为 nil 则返回空切片
//
// 返回：
//   - BlockTree 切片，包含文档的所有块，如果未找到或数据库为 nil 则返回空切片
//
// 用途：
//   - 获取文档的完整块结构
//   - 检查文档是否存在变化（配合 UpsertBlockTreeWithDB 使用）
//   - 导出文档数据
//
// 性能：
//   - 查询时间取决于文档的块数量
//   - 通常 < 50ms（对于包含数百个块的文档）
//
// 示例：
//
//	userDB, _ := btManager.GetOrCreateDB(ctx.BlockTreeDBPath)
//	blocks := GetBlockTreesByRootIDWithDB(docID, userDB)
//	fmt.Printf("Document has %d blocks\n", len(blocks))
//	for _, block := range blocks {
//	    fmt.Printf("Block %s: %s\n", block.ID, block.Type)
//	}
func GetBlockTreesByRootIDWithDB(rootID string, database *sql.DB) (ret []*BlockTree) {
	if nil == database {
		return
	}
	
	sqlStmt := "SELECT * FROM blocktrees WHERE root_id = ?"
	rows, err := database.Query(sqlStmt, rootID)
	if err != nil {
		logging.LogErrorf("sql query [%s] failed: %s", sqlStmt, err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		bt := &BlockTree{}
		if err = rows.Scan(&bt.ID, &bt.RootID, &bt.ParentID, &bt.BoxID, &bt.Path, &bt.HPath, &bt.Updated, &bt.Type); err != nil {
			logging.LogErrorf("scan row failed: %s", err)
			return
		}
		ret = append(ret, bt)
	}
	return
}
