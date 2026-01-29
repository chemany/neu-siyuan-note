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
	"fmt"
	"os"
	"sync"
	"unsafe"

	"github.com/gin-gonic/gin"
	"github.com/siyuan-note/logging"
	"github.com/siyuan-note/siyuan/kernel/filesys"
	"github.com/siyuan-note/siyuan/kernel/treenode"
	"github.com/siyuan-note/siyuan/kernel/util"
)

// currentExecutionContext 存储当前正在执行的 goroutine 的上下文
// 使用 sync.Map 存储，key 使用 goroutine 特定的标识
var currentExecutionContext = &sync.Map{}

// SetCurrentExecutionContext 设置当前 goroutine 的执行上下文
// 在 WebSocket 消息处理或 HTTP 请求处理前调用
func SetCurrentExecutionContext(ctx *WorkspaceContext) {
	if ctx == nil {
		return
	}
	// 使用栈指针的哈希作为 key（区分不同 goroutine）
	key := getGoroutineKey()
	currentExecutionContext.Store(key, ctx)
}

// ClearCurrentExecutionContext 清除当前 goroutine 的执行上下文
// 在 WebSocket 消息处理或 HTTP 请求处理完成后调用
func ClearCurrentExecutionContext() {
	key := getGoroutineKey()
	currentExecutionContext.Delete(key)
}

// GetCurrentExecutionContext 获取当前 goroutine 的执行上下文
func GetCurrentExecutionContext() *WorkspaceContext {
	key := getGoroutineKey()
	if actual, ok := currentExecutionContext.Load(key); ok {
		return actual.(*WorkspaceContext)
	}
	return nil
}

// getGoroutineKey 获取当前 goroutine 的 key
// 使用指针地址区分不同的 goroutine
func getGoroutineKey() string {
	// 使用栈上变量的地址来区分不同的 goroutine
	// 每个 goroutine 有自己的栈，地址范围不同
	var x int
	ptr := uintptr(unsafe.Pointer(&x))
	return fmt.Sprintf("g-%d", ptr>>20) // 取高位，避免栈扩容的影响
}

// init 初始化 filesys 和 treenode 包的 GetDataDirFunc
func init() {
	// 注入 GetDataDirFunc，让 filesys.LoadTree 和 treenode.RootChildIDs 能够使用正确的 DataDir
	getDataDir := func() string {
		ctx := GetDefaultWorkspaceContext()
		dataDir := ctx.GetDataDir()
		// 添加调试日志
		if os.Getenv("SIYUAN_WEB_MODE") == "true" {
			logging.LogInfof("GetDataDirFunc called: returning [%s], userID=[%s]", dataDir, ctx.UserID)
		}
		return dataDir
	}
	
	filesys.GetDataDirFunc = getDataDir
	treenode.GetDataDirFunc = getDataDir
}

// WorkspaceContext 包含用户的 workspace 信息
// 用于在多用户环境下隔离不同用户的数据
type WorkspaceContext struct {
	// 基础路径
	WorkspaceDir string // workspace 根目录
	DataDir      string // 数据目录（笔记本存储位置）
	ConfDir      string // 配置目录
	RepoDir      string // 仓库目录（同步相关）
	HistoryDir   string // 历史记录目录
	TempDir      string // 临时文件目录
	
	// 数据库路径
	BlockTreeDBPath    string // BlockTree 数据库路径
	AssetContentDBPath string // 附件内容数据库路径
	
	// 用户信息
	UserID   string // 用户 ID
	Username string // 用户名
	
	// 元数据
	WorkspaceName string // workspace 名称
}

// NewWorkspaceContext 创建一个新的 WorkspaceContext
// workspace: workspace 根目录路径
func NewWorkspaceContext(workspace string) *WorkspaceContext {
	tempDir := workspace + "/temp"
	return &WorkspaceContext{
		WorkspaceDir:       workspace,
		DataDir:            workspace,           // 注意：用户的笔记本直接在 workspace 根目录
		ConfDir:            workspace + "/conf",  // 配置目录在 workspace/conf（appearance、conf.json 等）
		RepoDir:            workspace + "/repo",
		HistoryDir:         workspace + "/history",
		TempDir:            tempDir,
		BlockTreeDBPath:    tempDir + "/blocktree.db", // 用户特定的 BlockTree 数据库
		AssetContentDBPath: tempDir + "/asset_content.db", // 用户特定的附件内容数据库
		WorkspaceName:      "",
	}
}

// NewWorkspaceContextWithUser 创建一个包含用户信息的 WorkspaceContext
func NewWorkspaceContextWithUser(workspace, userID, username string) *WorkspaceContext {
	ctx := NewWorkspaceContext(workspace)
	ctx.UserID = userID
	ctx.Username = username
	ctx.WorkspaceName = username
	return ctx
}

// GetWorkspaceContext 从 Gin Context 中获取 WorkspaceContext
// 如果不存在，返回默认的全局 workspace context
func GetWorkspaceContext(c *gin.Context) *WorkspaceContext {
	// 尝试从 context 获取
	if ctx, exists := c.Get("workspace_context"); exists {
		return ctx.(*WorkspaceContext)
	}
	
	// 如果不存在，返回默认的全局 workspace
	return GetDefaultWorkspaceContext()
}

// GetDefaultWorkspaceContext 获取默认的全局 workspace context
// 用于非 Web 模式或未认证的请求
// 在 Web 模式下,如果有当前用户 Context,则返回当前用户的 Context
func GetDefaultWorkspaceContext() *WorkspaceContext {
	// 首先检查当前是否有执行上下文（WebSocket/HTTP 请求处理中）
	if os.Getenv("SIYUAN_WEB_MODE") == "true" {
		if ctx := GetCurrentExecutionContext(); ctx != nil {
			return ctx
		}
	}

	// 在 Web 模式下,尝试获取当前用户的 Context
	if os.Getenv("SIYUAN_WEB_MODE") == "true" {
		currentUserMutex.RLock()
		userID := currentUserID
		currentUserMutex.RUnlock()

		if userID != "" {
			userContextsMutex.RLock()
			ctx, exists := userContexts[userID]
			userContextsMutex.RUnlock()

			if exists {
				return ctx
			}
		}
	}

	// 非 Web 模式或没有当前用户,返回全局 workspace
	return &WorkspaceContext{
		WorkspaceDir:       util.WorkspaceDir,
		DataDir:            util.DataDir,
		ConfDir:            util.ConfDir,
		RepoDir:            util.RepoDir,
		HistoryDir:         util.HistoryDir,
		TempDir:            util.TempDir,
		BlockTreeDBPath:    util.BlockTreeDBPath,
		AssetContentDBPath: util.AssetContentDBPath,
		WorkspaceName:      util.WorkspaceName,
		UserID:             "",
		Username:           "",
	}
}

// 全局用户 Context 管理器
var (
	userContexts      = make(map[string]*WorkspaceContext)
	userContextsMutex sync.RWMutex
	currentUserID     string
	currentUserMutex  sync.RWMutex
)

// SetCurrentUserContext 设置当前用户的 Context
func SetCurrentUserContext(userID string, ctx *WorkspaceContext) {
	userContextsMutex.Lock()
	defer userContextsMutex.Unlock()
	userContexts[userID] = ctx
	
	currentUserMutex.Lock()
	defer currentUserMutex.Unlock()
	currentUserID = userID
}

// GetCurrentUserContext 获取当前用户的 Context
func GetCurrentUserContext() *WorkspaceContext {
	currentUserMutex.RLock()
	userID := currentUserID
	currentUserMutex.RUnlock()
	
	if userID == "" {
		return GetDefaultWorkspaceContext()
	}
	
	userContextsMutex.RLock()
	defer userContextsMutex.RUnlock()
	
	if ctx, exists := userContexts[userID]; exists {
		return ctx
	}
	
	return GetDefaultWorkspaceContext()
}

// GetUserContext 根据用户 ID 获取 Context
func GetUserContext(userID string) *WorkspaceContext {
	if userID == "" {
		return GetDefaultWorkspaceContext()
	}
	
	userContextsMutex.RLock()
	defer userContextsMutex.RUnlock()
	
	if ctx, exists := userContexts[userID]; exists {
		return ctx
	}
	
	return GetDefaultWorkspaceContext()
}

// SetWorkspaceContext 将 WorkspaceContext 存储到 Gin Context
func SetWorkspaceContext(c *gin.Context, ctx *WorkspaceContext) {
	c.Set("workspace_context", ctx)
}

// GetDataDir 获取数据目录
func (ctx *WorkspaceContext) GetDataDir() string {
	return ctx.DataDir
}

// GetConfDir 获取配置目录
func (ctx *WorkspaceContext) GetConfDir() string {
	return ctx.ConfDir
}

// GetRepoDir 获取仓库目录
func (ctx *WorkspaceContext) GetRepoDir() string {
	return ctx.RepoDir
}

// GetHistoryDir 获取历史记录目录
func (ctx *WorkspaceContext) GetHistoryDir() string {
	return ctx.HistoryDir
}

// GetTempDir 获取临时文件目录
func (ctx *WorkspaceContext) GetTempDir() string {
	return ctx.TempDir
}

// GetBlockTreeDBPath 获取 BlockTree 数据库路径
func (ctx *WorkspaceContext) GetBlockTreeDBPath() string {
	return ctx.BlockTreeDBPath
}

// GetAssetContentDBPath 获取附件内容数据库路径
func (ctx *WorkspaceContext) GetAssetContentDBPath() string {
	return ctx.AssetContentDBPath
}

// GetWorkspaceDir 获取 workspace 根目录
func (ctx *WorkspaceContext) GetWorkspaceDir() string {
	return ctx.WorkspaceDir
}

// IsDefaultWorkspace 判断是否为默认 workspace
func (ctx *WorkspaceContext) IsDefaultWorkspace() bool {
	return ctx.WorkspaceDir == util.WorkspaceDir
}

// IsWebMode 判断是否为 Web 模式（多用户模式）
func (ctx *WorkspaceContext) IsWebMode() bool {
	return ctx.UserID != ""
}
