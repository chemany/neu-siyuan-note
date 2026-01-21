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
	"github.com/gin-gonic/gin"
	"github.com/siyuan-note/siyuan/kernel/util"
)

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
	
	// 用户信息
	UserID   string // 用户 ID
	Username string // 用户名
	
	// 元数据
	WorkspaceName string // workspace 名称
}

// NewWorkspaceContext 创建一个新的 WorkspaceContext
// workspace: workspace 根目录路径
func NewWorkspaceContext(workspace string) *WorkspaceContext {
	return &WorkspaceContext{
		WorkspaceDir:  workspace,
		DataDir:       workspace,        // 注意：用户的笔记本直接在 workspace 根目录（旧版本结构）
		ConfDir:       workspace + "/conf",
		RepoDir:       workspace + "/repo",
		HistoryDir:    workspace + "/history",
		TempDir:       workspace + "/temp",
		WorkspaceName: "",
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
func GetDefaultWorkspaceContext() *WorkspaceContext {
	return &WorkspaceContext{
		WorkspaceDir:  util.WorkspaceDir,
		DataDir:       util.DataDir,
		ConfDir:       util.ConfDir,
		RepoDir:       util.RepoDir,
		HistoryDir:    util.HistoryDir,
		TempDir:       util.TempDir,
		WorkspaceName: util.WorkspaceName,
		UserID:        "",
		Username:      "",
	}
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
