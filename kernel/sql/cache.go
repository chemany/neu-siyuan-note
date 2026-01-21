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
	"strings"
	"time"

	"github.com/88250/lute/ast"
	"github.com/88250/lute/parse"
	"github.com/dgraph-io/ristretto"
	"github.com/jinzhu/copier"
	gcache "github.com/patrickmn/go-cache"
	"github.com/siyuan-note/logging"
	"github.com/siyuan-note/siyuan/kernel/cache"
	"github.com/siyuan-note/siyuan/kernel/search"
)

var cacheDisabled = true

func enableCache() {
	cacheDisabled = false
}

func disableCache() {
	cacheDisabled = true
}

var blockCache, _ = ristretto.NewCache(&ristretto.Config{
	NumCounters: 102400,
	MaxCost:     10240,
	BufferItems: 64,
})

func ClearCache() {
	blockCache.Clear()
}

func putBlockCache(block *Block) {
	if cacheDisabled {
		return
	}

	cloned := &Block{}
	if err := copier.Copy(cloned, block); err != nil {
		logging.LogErrorf("clone block failed: %v", err)
		return
	}
	cloned.Content = strings.ReplaceAll(cloned.Content, search.SearchMarkLeft, "")
	cloned.Content = strings.ReplaceAll(cloned.Content, search.SearchMarkRight, "")
	blockCache.Set(cloned.ID, cloned, 1)
}

func getBlockCache(id string) (ret *Block) {
	if cacheDisabled {
		return
	}

	b, _ := blockCache.Get(id)
	if nil != b {
		ret = b.(*Block)
	}
	return
}

func removeBlockCache(id string) {
	blockCache.Del(id)
	removeRefCacheByDefID(id)
}

var defIDRefsCache = gcache.New(30*time.Minute, 5*time.Minute) // [defBlockID]map[refBlockID]*Ref

func GetRefsCacheByDefID(defID string) (ret []*Ref) {
	for defBlockID, refs := range defIDRefsCache.Items() {
		if defBlockID == defID {
			for _, ref := range refs.Object.(map[string]*Ref) {
				ret = append(ret, ref)
			}
		}
	}
	if 1 > len(ret) {
		ret = QueryRefsByDefID(defID, false)
		for _, ref := range ret {
			putRefCache(ref)
		}
	}
	return
}

func CacheRef(tree *parse.Tree, refNode *ast.Node) {
	ref := buildRef(tree, refNode)
	putRefCache(ref)
}

func putRefCache(ref *Ref) {
	defBlockRefs, ok := defIDRefsCache.Get(ref.DefBlockID)
	if !ok {
		defBlockRefs = map[string]*Ref{}
	}
	defBlockRefs.(map[string]*Ref)[ref.BlockID] = ref
	defIDRefsCache.SetDefault(ref.DefBlockID, defBlockRefs)
}

func removeRefCacheByDefID(defID string) {
	defIDRefsCache.Delete(defID)
}

// ==================== 带 Context 的缓存函数 ====================

// putBlockCacheWithContext 使用用户缓存存储 Block
func putBlockCacheWithContext(ctx WorkspaceContext, block *Block) {
	userCache := cache.GetUserCacheWithContext(ctx)
	if !userCache.IsEnabled() {
		return
	}

	cloned := &Block{}
	if err := copier.Copy(cloned, block); err != nil {
		logging.LogErrorf("clone block failed: %v", err)
		return
	}
	cloned.Content = strings.ReplaceAll(cloned.Content, search.SearchMarkLeft, "")
	cloned.Content = strings.ReplaceAll(cloned.Content, search.SearchMarkRight, "")
	userCache.GetBlockCache().Set(cloned.ID, cloned, 1)
}

// getBlockCacheWithContext 从用户缓存获取 Block
func getBlockCacheWithContext(ctx WorkspaceContext, id string) (ret *Block) {
	userCache := cache.GetUserCacheWithContext(ctx)
	if !userCache.IsEnabled() {
		return
	}

	b, _ := userCache.GetBlockCache().Get(id)
	if nil != b {
		ret = b.(*Block)
	}
	return
}

// removeBlockCacheWithContext 从用户缓存删除 Block
func removeBlockCacheWithContext(ctx WorkspaceContext, id string) {
	userCache := cache.GetUserCacheWithContext(ctx)
	userCache.GetBlockCache().Del(id)
	removeRefCacheByDefIDWithContext(ctx, id)
}

// GetRefsCacheByDefIDWithContext 从用户缓存获取引用
func GetRefsCacheByDefIDWithContext(ctx WorkspaceContext, defID string) (ret []*Ref) {
	userCache := cache.GetUserCacheWithContext(ctx)
	refCache := userCache.GetRefCache()
	
	for defBlockID, refs := range refCache.Items() {
		if defBlockID == defID {
			for _, ref := range refs.Object.(map[string]*Ref) {
				ret = append(ret, ref)
			}
		}
	}
	if 1 > len(ret) {
		ret = QueryRefsByDefIDWithContext(ctx, defID, false)
		for _, ref := range ret {
			putRefCacheWithContext(ctx, ref)
		}
	}
	return
}

// CacheRefWithContext 缓存引用（带 Context）
func CacheRefWithContext(ctx WorkspaceContext, tree *parse.Tree, refNode *ast.Node) {
	ref := buildRef(tree, refNode)
	putRefCacheWithContext(ctx, ref)
}

// putRefCacheWithContext 存储引用到用户缓存
func putRefCacheWithContext(ctx WorkspaceContext, ref *Ref) {
	userCache := cache.GetUserCacheWithContext(ctx)
	refCache := userCache.GetRefCache()
	
	defBlockRefs, ok := refCache.Get(ref.DefBlockID)
	if !ok {
		defBlockRefs = map[string]*Ref{}
	}
	defBlockRefs.(map[string]*Ref)[ref.BlockID] = ref
	refCache.SetDefault(ref.DefBlockID, defBlockRefs)
}

// removeRefCacheByDefIDWithContext 从用户缓存删除引用
func removeRefCacheByDefIDWithContext(ctx WorkspaceContext, defID string) {
	userCache := cache.GetUserCacheWithContext(ctx)
	userCache.GetRefCache().Delete(defID)
}

// ClearCacheWithContext 清除用户缓存
func ClearCacheWithContext(ctx WorkspaceContext) {
	userCache := cache.GetUserCacheWithContext(ctx)
	userCache.Clear()
}

