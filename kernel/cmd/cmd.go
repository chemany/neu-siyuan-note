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

package cmd

import (
	"github.com/olahol/melody"
	"github.com/siyuan-note/logging"
	"github.com/siyuan-note/siyuan/kernel/model"
	"github.com/siyuan-note/siyuan/kernel/util"
)

type Cmd interface {
	Name() string
	IsRead() bool // 非读即写
	Id() float64
	Exec()
	Context() *model.WorkspaceContext // ✅ 获取执行上下文（用于多用户隔离）
}

type BaseCmd struct {
	id          float64
	param       map[string]interface{}
	session     *melody.Session
	ctx         *model.WorkspaceContext // ✅ 添加 WorkspaceContext
	PushPayload *util.Result
}

func (cmd *BaseCmd) Id() float64 {
	return cmd.id
}

func (cmd *BaseCmd) Push() {
	cmd.PushPayload.Callback = cmd.param["callback"]
	appId, _ := cmd.session.Get("app")
	cmd.PushPayload.AppId = appId.(string)
	sid, _ := cmd.session.Get("id")
	cmd.PushPayload.SessionId = sid.(string)
	util.PushEvent(cmd.PushPayload)
}

// Context 返回当前命令的 WorkspaceContext
// 用于多用户环境下的数据隔离
func (cmd *BaseCmd) Context() *model.WorkspaceContext {
	return cmd.ctx
}

func NewCommand(cmdStr string, cmdId float64, param map[string]interface{}, session *melody.Session) (ret Cmd) {
	// ✅ 从 session 中获取 WorkspaceContext
	var ctx *model.WorkspaceContext
	if ctxVal, exists := session.Get("workspaceContext"); exists {
		ctx = ctxVal.(*model.WorkspaceContext)
		logging.LogInfof("Command [%s] using WorkspaceContext: %s", cmdStr, ctx.DataDir)
	} else {
		logging.LogWarnf("Command [%s] has no WorkspaceContext", cmdStr)
	}
	
	baseCmd := &BaseCmd{
		id:      cmdId,
		param:   param,
		session: session,
		ctx:     ctx, // ✅ 传递 WorkspaceContext
	}
	switch cmdStr {
	case "closews":
		ret = &closews{baseCmd}
	case "ping":
		ret = &ping{baseCmd}
	}

	if nil == ret {
		return
	}

	pushMode := util.PushModeSingleSelf
	if pushModeParam := param["pushMode"]; nil != pushModeParam {
		pushMode = util.PushMode(pushModeParam.(float64))
	}
	baseCmd.PushPayload = util.NewCmdResult(ret.Name(), cmdId, pushMode)
	appId, _ := baseCmd.session.Get("app")
	baseCmd.PushPayload.AppId = appId.(string)
	sid, _ := baseCmd.session.Get("id")
	baseCmd.PushPayload.SessionId = sid.(string)
	return
}

func Exec(cmd Cmd) {
	go func() {
		defer logging.Recover()

		// ✅ 设置当前 goroutine 的执行上下文（用于多用户隔离）
		if ctx := cmd.Context(); ctx != nil {
			model.SetCurrentExecutionContext(ctx)
			defer model.ClearCurrentExecutionContext()
		}

		cmd.Exec()
	}()
}
