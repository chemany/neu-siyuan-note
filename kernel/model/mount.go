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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/88250/gulu"
	"github.com/88250/lute/ast"
	"github.com/siyuan-note/filelock"
	"github.com/siyuan-note/logging"
	"github.com/siyuan-note/siyuan/kernel/task"
	"github.com/siyuan-note/siyuan/kernel/util"
)

// CreateBoxWithContext 使用 WorkspaceContext 创建笔记本
func CreateBoxWithContext(ctx *WorkspaceContext, name string) (id string, err error) {
	name = util.RemoveInvalid(name)
	if 512 < utf8.RuneCountInString(name) {
		// 限制笔记本名和文档名最大长度为 `512` https://github.com/siyuan-note/siyuan/issues/6299
		err = errors.New(Conf.Language(106))
		return
	}
	if "" == name {
		name = Conf.language(105)
	}

	FlushTxQueue()

	createDocLock.Lock()
	defer createDocLock.Unlock()

	boxes, _ := ListNotebooks(ctx)
	for i, b := range boxes {
		c := b.GetConfWithDataDir(ctx.GetDataDir())
		c.Sort = i + 1
		b.SaveConfWithDataDir(ctx.GetDataDir(), c)
	}

	id = ast.NewNodeID()
	// 使用 ctx.GetDataDir() 替代 util.DataDir
	boxLocalPath := filepath.Join(ctx.GetDataDir(), id)
	err = os.MkdirAll(boxLocalPath, 0755)
	if err != nil {
		return
	}

	box := &Box{ID: id, Name: name}
	boxConf := box.GetConfWithDataDir(ctx.GetDataDir())
	boxConf.Name = name
	boxConf.Closed = false  // 新创建的笔记本应该是打开状态
	
	// 使用带 dataDir 的保存方法
	box.SaveConfWithDataDir(ctx.GetDataDir(), boxConf)
	
	IncSync()
	logging.LogInfof("created box [%s] with name [%s]", id, name)
	return
}

// CreateBox 创建笔记本（向后兼容）
func CreateBox(name string) (id string, err error) {
	return CreateBoxWithContext(GetDefaultWorkspaceContext(), name)
}

func RenameBox(boxID, name string) (err error) {
	box := Conf.Box(boxID)
	if nil == box {
		return errors.New(Conf.Language(0))
	}

	if 512 < utf8.RuneCountInString(name) {
		// 限制笔记本名和文档名最大长度为 `512` https://github.com/siyuan-note/siyuan/issues/6299
		err = errors.New(Conf.Language(106))
		return
	}

	if "" == name {
		name = Conf.language(105)
	}

	boxConf := box.GetConf()
	boxConf.Name = name
	box.Name = name
	box.SaveConf(boxConf)
	IncSync()
	logging.LogInfof("renamed box [%s] to [%s]", boxID, name)
	return
}

var boxLock = sync.Map{}

// RemoveBoxWithContext 使用 WorkspaceContext 删除笔记本
func RemoveBoxWithContext(ctx *WorkspaceContext, boxID string) (err error) {
	if _, ok := boxLock.Load(boxID); ok {
		err = fmt.Errorf(Conf.language(239))
		return
	}

	boxLock.Store(boxID, true)
	defer boxLock.Delete(boxID)

	if util.IsReservedFilename(boxID) {
		return errors.New(fmt.Sprintf("can not remove [%s] caused by it is a reserved file", boxID))
	}

	FlushTxQueue()
	isUserGuide := IsUserGuide(boxID)
	createDocLock.Lock()
	defer createDocLock.Unlock()

	// 使用 ctx.GetDataDir() 替代 util.DataDir
	localPath := filepath.Join(ctx.GetDataDir(), boxID)
	if !filelock.IsExist(localPath) {
		return
	}
	if !gulu.File.IsDir(localPath) {
		return errors.New(fmt.Sprintf("can not remove [%s] caused by it is not a dir", boxID))
	}

	if !isUserGuide {
		var historyDir string
		historyDir, err = GetHistoryDir(HistoryOpDelete)
		if err != nil {
			logging.LogErrorf("get history dir failed: %s", err)
			return
		}
		// 使用 ctx.GetDataDir() 替代 util.DataDir
		p := strings.TrimPrefix(localPath, ctx.GetDataDir())
		historyPath := filepath.Join(historyDir, p)
		if err = filelock.Copy(localPath, historyPath); err != nil {
			logging.LogErrorf("gen sync history failed: %s", err)
			return
		}

		copyBoxAssetsToDataAssets(boxID)
	}

	unmount0(boxID)
	if err = filelock.Remove(localPath); err != nil {
		return
	}
	IncSync()

	logging.LogInfof("removed box [%s]", boxID)
	return
}

// RemoveBox 删除笔记本（向后兼容）
func RemoveBox(boxID string) (err error) {
	return RemoveBoxWithContext(GetDefaultWorkspaceContext(), boxID)
}

func Unmount(boxID string) {
	FlushTxQueue()

	unmount0(boxID)
	evt := util.NewCmdResult("unmount", 0, util.PushModeBroadcast)
	evt.Data = map[string]interface{}{
		"box": boxID,
	}
	util.PushEvent(evt)
}

// UnmountWithContext 使用 WorkspaceContext 关闭笔记本
func UnmountWithContext(ctx *WorkspaceContext, boxID string) {
	FlushTxQueue()

	unmount0WithContext(ctx, boxID)
	evt := util.NewCmdResult("unmount", 0, util.PushModeBroadcast)
	evt.Data = map[string]interface{}{
		"box": boxID,
	}
	util.PushEvent(evt)
}

func unmount0(boxID string) {
	box := Conf.Box(boxID)
	if nil == box {
		return
	}

	boxConf := box.GetConf()
	boxConf.Closed = true
	box.SaveConf(boxConf)
	box.Unindex()
}

// unmount0WithContext 使用 WorkspaceContext 关闭笔记本的内部实现
func unmount0WithContext(ctx *WorkspaceContext, boxID string) {
	box := Conf.BoxWithContext(ctx, boxID)
	if nil == box {
		return
	}

	boxConf := box.GetConf()
	boxConf.Closed = true
	box.SaveConf(boxConf)
	box.Unindex()
}

// MountWithContext 使用 WorkspaceContext 打开笔记本
func MountWithContext(ctx *WorkspaceContext, boxID string) (alreadyMount bool, err error) {
	if _, ok := boxLock.Load(boxID); ok {
		err = fmt.Errorf(Conf.language(239))
		return
	}

	boxLock.Store(boxID, true)
	defer boxLock.Delete(boxID)

	FlushTxQueue()
	isUserGuide := IsUserGuide(boxID)

	// 使用 ctx.GetDataDir() 替代 util.DataDir
	localPath := filepath.Join(ctx.GetDataDir(), boxID)
	var reMountGuide bool
	if isUserGuide {
		// 重新挂载帮助文档

		guideBox := Conf.Box(boxID)
		if nil != guideBox {
			unmount0(guideBox.ID)
			reMountGuide = true
		}

		if err = filelock.Remove(localPath); err != nil {
			return
		}

		boxes, _ := ListNotebooks(ctx)
		var sort int
		if len(boxes) > 0 {
			sort = boxes[0].Sort - 1
		}

		p := filepath.Join(util.WorkingDir, "guide", boxID)
		if err = filelock.Copy(p, localPath); err != nil {
			return
		}

		avDirPath := filepath.Join(util.WorkingDir, "guide", boxID, "storage", "av")
		if filelock.IsExist(avDirPath) {
			// 使用 ctx.GetDataDir() 替代 util.DataDir
			if err = filelock.Copy(avDirPath, filepath.Join(ctx.GetDataDir(), "storage", "av")); err != nil {
				return
			}
		}

		if box := Conf.Box(boxID); nil != box {
			boxConf := box.GetConf()
			boxConf.Closed = true
			boxConf.Sort = sort
			box.SaveConf(boxConf)
		}

		if Conf.OpenHelp {
			Conf.OpenHelp = false
			Conf.Save()
		}

		task.AppendAsyncTaskWithDelay(task.PushMsg, 3*time.Second, util.PushErrMsg, Conf.Language(52), 7000)
		go func() {
			// 每次打开帮助文档时自动检查版本更新并提醒 https://github.com/siyuan-note/siyuan/issues/5057
			time.Sleep(time.Second * 10)
			CheckUpdate(true)
		}()
	}

	if !gulu.File.IsDir(localPath) {
		return false, errors.New("can not open file, just support open folder only")
	}

	// 使用 GetOpenedBoxesWithContext 检查笔记本是否已打开
	for _, box := range Conf.GetOpenedBoxesWithContext(ctx) {
		if box.ID == boxID {
			return true, nil
		}
	}

	box := &Box{ID: boxID}
	boxConf := box.GetConfWithDataDir(ctx.GetDataDir())
	boxConf.Closed = false
	
	// 使用带 dataDir 的保存方法
	box.SaveConfWithDataDir(ctx.GetDataDir(), boxConf)

	// 使用带 Context 的索引方法
	// 使用带 Context 的索引方法(通过后台任务系统)
	box.IndexWithContext(ctx)
	// 缓存根一级的文档树展开
	ListDocTree(ctx, box.ID, "/", util.SortModeUnassigned, false, false, Conf.FileTree.MaxListCount)
	util.ClearPushProgress(100)

	if reMountGuide {
		return true, nil
	}
	return false, nil
}

// Mount 打开笔记本（向后兼容）
func Mount(boxID string) (alreadyMount bool, err error) {
	return MountWithContext(GetDefaultWorkspaceContext(), boxID)
}

func IsUserGuide(boxID string) bool {
	return "20210808180117-czj9bvb" == boxID || "20210808180117-6v0mkxr" == boxID || "20211226090932-5lcq56f" == boxID || "20240530133126-axarxgx" == boxID
}
