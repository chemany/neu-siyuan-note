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

package filesys

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"

	"github.com/88250/lute"
	"github.com/88250/lute/parse"
	"github.com/88250/lute/render"
	jsoniter "github.com/json-iterator/go"
	"github.com/panjf2000/ants/v2"
	"github.com/siyuan-note/dataparser"
	"github.com/siyuan-note/filelock"
	"github.com/siyuan-note/logging"
	"github.com/siyuan-note/siyuan/kernel/cache"
	"github.com/siyuan-note/siyuan/kernel/treenode"
	"github.com/siyuan-note/siyuan/kernel/util"
)

// GetDataDirFunc 是一个函数变量，用于获取当前的 DataDir
// 在 Web 模式下，model 包会注入一个返回用户特定 DataDir 的函数
var GetDataDirFunc func() string

// init 初始化 GetDataDirFunc 为默认实现
func init() {
	GetDataDirFunc = func() string {
		return util.DataDir
	}
}

func LoadTrees(ids []string) (ret map[string]*parse.Tree) {
	ret = map[string]*parse.Tree{}
	if 1 > len(ids) {
		return ret
	}

	bts := treenode.GetBlockTrees(ids)
	luteEngine := util.NewLute()
	var boxIDs []string
	var paths []string
	blockIDs := map[string][]string{}
	for _, bt := range bts {
		boxIDs = append(boxIDs, bt.BoxID)
		paths = append(paths, bt.Path)
		if _, ok := blockIDs[bt.RootID]; !ok {
			blockIDs[bt.RootID] = []string{}
		}
		blockIDs[bt.RootID] = append(blockIDs[bt.RootID], bt.ID)
	}

	trees, errs := batchLoadTrees(boxIDs, paths, luteEngine)
	for i := range trees {
		tree := trees[i]
		err := errs[i]
		if err != nil || tree == nil {
			logging.LogErrorf("load tree failed: %s", err)
			continue
		}

		bIDs := blockIDs[tree.Root.ID]
		for _, bID := range bIDs {
			ret[bID] = tree
		}
	}
	return
}

// LoadTreesWithDataDir 使用指定的 dataDir 加载多个树
func LoadTreesWithDataDir(dataDir string, ids []string) (ret map[string]*parse.Tree) {
	ret = map[string]*parse.Tree{}
	if 1 > len(ids) {
		return ret
	}

	bts := treenode.GetBlockTrees(ids)
	luteEngine := util.NewLute()
	var boxIDs []string
	var paths []string
	blockIDs := map[string][]string{}
	for _, bt := range bts {
		boxIDs = append(boxIDs, bt.BoxID)
		paths = append(paths, bt.Path)
		if _, ok := blockIDs[bt.RootID]; !ok {
			blockIDs[bt.RootID] = []string{}
		}
		blockIDs[bt.RootID] = append(blockIDs[bt.RootID], bt.ID)
	}

	trees, errs := batchLoadTreesWithDataDir(dataDir, boxIDs, paths, luteEngine)
	for i := range trees {
		tree := trees[i]
		err := errs[i]
		if err != nil || tree == nil {
			logging.LogErrorf("load tree failed: %s", err)
			continue
		}

		bIDs := blockIDs[tree.Root.ID]
		for _, bID := range bIDs {
			ret[bID] = tree
		}
	}
	return
}

func batchLoadTrees(boxIDs, paths []string, luteEngine *lute.Lute) (ret []*parse.Tree, errs []error) {
	waitGroup := sync.WaitGroup{}
	lock := sync.Mutex{}
	poolSize := runtime.NumCPU()
	if 8 < poolSize {
		poolSize = 8
	}
	p, _ := ants.NewPoolWithFunc(poolSize, func(arg interface{}) {
		defer waitGroup.Done()

		i := arg.(int)
		boxID := boxIDs[i]
		path := paths[i]
		tree, err := LoadTree(boxID, path, luteEngine)
		lock.Lock()
		ret = append(ret, tree)
		errs = append(errs, err)
		lock.Unlock()
	})
	loaded := map[string]bool{}
	for i := range paths {
		if loaded[boxIDs[i]+paths[i]] {
			continue
		}

		loaded[boxIDs[i]+paths[i]] = true

		waitGroup.Add(1)
		p.Invoke(i)
	}
	waitGroup.Wait()
	p.Release()
	return
}

// batchLoadTreesWithDataDir 使用指定的 dataDir 批量加载树
func batchLoadTreesWithDataDir(dataDir string, boxIDs, paths []string, luteEngine *lute.Lute) (ret []*parse.Tree, errs []error) {
	waitGroup := sync.WaitGroup{}
	lock := sync.Mutex{}
	poolSize := runtime.NumCPU()
	if 8 < poolSize {
		poolSize = 8
	}
	p, _ := ants.NewPoolWithFunc(poolSize, func(arg interface{}) {
		defer waitGroup.Done()

		i := arg.(int)
		boxID := boxIDs[i]
		path := paths[i]
		tree, err := LoadTreeWithDataDir(dataDir, boxID, path, luteEngine)
		lock.Lock()
		ret = append(ret, tree)
		errs = append(errs, err)
		lock.Unlock()
	})
	loaded := map[string]bool{}
	for i := range paths {
		if loaded[boxIDs[i]+paths[i]] {
			continue
		}

		loaded[boxIDs[i]+paths[i]] = true

		waitGroup.Add(1)
		p.Invoke(i)
	}
	waitGroup.Wait()
	p.Release()
	return
}

func LoadTree(boxID, p string, luteEngine *lute.Lute) (ret *parse.Tree, err error) {
	// 使用 GetDataDirFunc 获取 DataDir
	// 在 Web 模式下，这会返回当前用户的 DataDir
	dataDir := GetDataDirFunc()
	filePath := filepath.Join(dataDir, boxID, p)
	data, err := filelock.ReadFile(filePath)
	if err != nil {
		logging.LogErrorf("load tree [%s] failed: %s", p, err)
		return
	}

	ret, err = LoadTreeByData(data, boxID, p, luteEngine)
	return
}

// LoadTreeWithDataDir 使用指定的 dataDir 加载树
func LoadTreeWithDataDir(dataDir, boxID, p string, luteEngine *lute.Lute) (ret *parse.Tree, err error) {
	filePath := filepath.Join(dataDir, boxID, p)
	data, err := filelock.ReadFile(filePath)
	if err != nil {
		logging.LogErrorf("load tree [%s] failed: %s", p, err)
		return
	}

	ret, err = LoadTreeByDataWithDataDir(data, boxID, p, dataDir, luteEngine)
	return
}

func LoadTreeByData(data []byte, boxID, p string, luteEngine *lute.Lute) (ret *parse.Tree, err error) {
	// 使用 GetDataDirFunc 获取 DataDir
	dataDir := GetDataDirFunc()
	return LoadTreeByDataWithDataDir(data, boxID, p, dataDir, luteEngine)
}

// LoadTreeByDataWithDataDir 使用指定的 dataDir 从数据加载树
func LoadTreeByDataWithDataDir(data []byte, boxID, p string, dataDir string, luteEngine *lute.Lute) (ret *parse.Tree, err error) {
	ret = parseJSON2TreeWithDataDir(dataDir, boxID, p, data, luteEngine)
	if nil == ret {
		logging.LogErrorf("parse tree [%s] failed", p)
		err = errors.New("parse tree failed")
		return
	}
	ret.Path = p
	ret.Root.Path = p
	return
}

func parseJSON2TreeWithDataDir(dataDir, boxID, p string, jsonData []byte, luteEngine *lute.Lute) (ret *parse.Tree) {
	ret = parseJSON2Tree(boxID, p, jsonData, luteEngine)
	if nil == ret {
		return
	}

	parts := strings.Split(p, "/")
	parts = parts[1 : len(parts)-1] // 去掉开头的斜杆和结尾的自己
	if 1 > len(parts) {
		ret.HPath = "/" + ret.Root.IALAttr("title")
		ret.Hash = treenode.NodeHash(ret.Root, ret, luteEngine)
		return
	}

	// 构造 HPath
	hPathBuilder := bytes.Buffer{}
	hPathBuilder.WriteString("/")
	for i, _ := range parts {
		var parentAbsPath string
		if 0 < i {
			parentAbsPath = strings.Join(parts[:i+1], "/")
		} else {
			parentAbsPath = parts[0]
		}
		parentAbsPath += ".sy"
		parentPath := parentAbsPath
		// 使用传入的 dataDir 而不是 util.DataDir
		parentAbsPath = filepath.Join(dataDir, boxID, parentAbsPath)

		parentDocIAL := DocIAL(parentAbsPath)
		if 1 > len(parentDocIAL) {
			// 子文档缺失父文档时自动补全 https://github.com/siyuan-note/siyuan/issues/7376
			parentTree := treenode.NewTree(boxID, parentPath, hPathBuilder.String()+"Untitled", "Untitled")
			// 使用 WriteTreeWithDataDir 而不是 WriteTree，确保使用正确的 dataDir
			if _, writeErr := WriteTreeWithDataDir(parentTree, dataDir); nil != writeErr {
				logging.LogErrorf("rebuild parent tree [%s] failed: %s", parentAbsPath, writeErr)
			} else {
				logging.LogInfof("rebuilt parent tree [%s]", parentAbsPath)
				treenode.UpsertBlockTree(parentTree)
			}
			hPathBuilder.WriteString("Untitled/")
			continue
		}

		title := parentDocIAL["title"]
		if "" == title {
			title = "Untitled"
		}
		hPathBuilder.WriteString(util.UnescapeHTML(title))
		hPathBuilder.WriteString("/")
	}
	hPathBuilder.WriteString(ret.Root.IALAttr("title"))
	ret.HPath = hPathBuilder.String()
	ret.Hash = treenode.NodeHash(ret.Root, ret, luteEngine)
	return
}

func DocIAL(absPath string) (ret map[string]string) {
	filelock.Lock(absPath)
	file, err := os.Open(absPath)
	if err != nil {
		logging.LogErrorf("open file [%s] failed: %s", absPath, err)
		filelock.Unlock(absPath)
		return nil
	}

	iter := jsoniter.Parse(jsoniter.ConfigCompatibleWithStandardLibrary, file, 512)
	for field := iter.ReadObject(); field != ""; field = iter.ReadObject() {
		if field == "Properties" {
			iter.ReadVal(&ret)
			break
		} else {
			iter.Skip()
		}
	}
	file.Close()
	filelock.Unlock(absPath)
	return
}

func TreeSize(tree *parse.Tree) (size uint64) {
	luteEngine := util.NewLute() // 不关注用户的自定义解析渲染选项
	renderer := render.NewJSONRenderer(tree, luteEngine.RenderOptions)
	return uint64(len(renderer.Render()))
}

func WriteTree(tree *parse.Tree) (size uint64, err error) {
	// 添加日志以追踪使用全局 DataDir 的调用，包括堆栈信息
	logging.LogWarnf("WriteTree: writing tree [%s] using GLOBAL util.DataDir [%s] - this should be avoided in multi-user mode", tree.ID, util.DataDir)
	logging.LogWarnf("WriteTree: Stack trace: %s", string(debug.Stack()))
	return WriteTreeWithDataDir(tree, util.DataDir)
}

// WriteTreeWithDataDir 使用指定的 dataDir 写入树
func WriteTreeWithDataDir(tree *parse.Tree, dataDir string) (size uint64, err error) {
	data, filePath, err := prepareWriteTreeWithDataDir(tree, dataDir)
	if err != nil {
		return
	}

	// 添加详细日志以追踪文件写入
	logging.LogInfof("WriteTreeWithDataDir: writing tree [%s] to [%s] with dataDir [%s]", tree.ID, filePath, dataDir)

	size = uint64(len(data))
	if err = filelock.WriteFile(filePath, data); err != nil {
		msg := fmt.Sprintf("write data [%s] failed: %s", filePath, err)
		logging.LogErrorf(msg)
		err = errors.New(msg)
		return
	}

	if util.ExceedLargeFileWarningSize(len(data)) {
		msg := fmt.Sprintf(util.Langs[util.Lang][268], tree.Root.IALAttr("title")+" "+filepath.Base(filePath), util.LargeFileWarningSize)
		util.PushErrMsg(msg, 7000)
	}

	afterWriteTree(tree)
	return
}

func prepareWriteTree(tree *parse.Tree) (data []byte, filePath string, err error) {
	return prepareWriteTreeWithDataDir(tree, util.DataDir)
}

// prepareWriteTreeWithDataDir 使用指定的 dataDir 准备写入树
func prepareWriteTreeWithDataDir(tree *parse.Tree, dataDir string) (data []byte, filePath string, err error) {
	luteEngine := util.NewLute() // 不关注用户的自定义解析渲染选项

	if nil == tree.Root.FirstChild {
		newP := treenode.NewParagraph("")
		tree.Root.AppendChild(newP)
		tree.Root.SetIALAttr("updated", util.TimeFromID(newP.ID))
		treenode.UpsertBlockTree(tree)
	}

	filePath = filepath.Join(dataDir, tree.Box, tree.Path)
	if oldSpec := tree.Root.Spec; "" == oldSpec {
		parse.NestedInlines2FlattedSpans(tree, false)
		tree.Root.Spec = "1"
		logging.LogInfof("migrated tree [%s] from spec [%s] to [%s]", filePath, oldSpec, tree.Root.Spec)
	}
	tree.Root.SetIALAttr("type", "doc")
	renderer := render.NewJSONRenderer(tree, luteEngine.RenderOptions)
	data = renderer.Render()
	data = bytes.ReplaceAll(data, []byte("\u0000"), []byte(""))

	if !util.UseSingleLineSave {
		buf := bytes.Buffer{}
		buf.Grow(1024 * 1024 * 2)
		if err = json.Indent(&buf, data, "", "\t"); err != nil {
			logging.LogErrorf("json indent failed: %s", err)
			return
		}
		data = buf.Bytes()
	}

	if err = os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return
	}
	return
}

func afterWriteTree(tree *parse.Tree) {
	docIAL := parse.IAL2MapUnEsc(tree.Root.KramdownIAL)
	cache.PutDocIAL(tree.Path, docIAL)
}

func parseJSON2Tree(boxID, p string, jsonData []byte, luteEngine *lute.Lute) (ret *parse.Tree) {
	var err error
	var needFix bool
	ret, needFix, err = dataparser.ParseJSON(jsonData, luteEngine.ParseOptions)
	if err != nil {
		logging.LogErrorf("parse json [%s] to tree failed: %s", boxID+p, err)
		return
	}

	ret.Box = boxID
	ret.Path = p

	filePath := filepath.Join(util.DataDir, ret.Box, ret.Path)
	if oldSpec := ret.Root.Spec; "" == oldSpec {
		parse.NestedInlines2FlattedSpans(ret, false)
		ret.Root.Spec = "1"
		needFix = true
		logging.LogInfof("migrated tree [%s] from spec [%s] to [%s]", filePath, oldSpec, ret.Root.Spec)
	}

	if pathID := util.GetTreeID(p); pathID != ret.Root.ID {
		needFix = true
		logging.LogInfof("reset tree id from [%s] to [%s]", ret.Root.ID, pathID)
		ret.Root.ID = pathID
		ret.ID = pathID
		ret.Root.SetIALAttr("id", ret.ID)
	}

	if needFix {
		renderer := render.NewJSONRenderer(ret, luteEngine.RenderOptions)
		data := renderer.Render()

		if !util.UseSingleLineSave {
			buf := bytes.Buffer{}
			buf.Grow(1024 * 1024 * 2)
			if err = json.Indent(&buf, data, "", "\t"); err != nil {
				return
			}
			data = buf.Bytes()
		}

		if err = os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			return
		}
		if err = filelock.WriteFile(filePath, data); err != nil {
			msg := fmt.Sprintf("write data [%s] failed: %s", filePath, err)
			logging.LogErrorf(msg)
		}
	}
	return
}
