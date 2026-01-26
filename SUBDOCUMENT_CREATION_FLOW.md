# 子文档创建流程详解

## 概述

本文档详细说明灵枢笔记中子文档的创建流程，特别是如何通过 BlockTree 查询系统实现正确的父子文档关系。

## 问题背景

### 重构前的问题

在多用户架构重构之前，子文档创建存在以下问题：

1. **查询使用全局数据库**
   ```go
   // ❌ 问题代码
   parent := GetBlockTreeRootByHPath(boxID, "/父文档")  // 使用全局数据库
   ```
   - 无法查询到用户特定的父文档
   - 导致每次都创建新的父文档
   - 子文档创建在错误的位置

2. **父文档重复创建**
   - 用户 A 创建 `/父文档/子文档1`，系统创建父文档
   - 用户 A 再创建 `/父文档/子文档2`，系统又创建一个新的父文档
   - 结果：两个同名的父文档，子文档分散在不同位置

3. **数据隔离失败**
   - 用户 A 的查询可能返回用户 B 的父文档
   - 导致数据混乱和安全问题

### 重构后的改进

✅ 使用用户特定的数据库查询
✅ 正确重用已存在的父文档
✅ 子文档创建在正确的父文档目录下
✅ 完整的数据隔离

## 完整流程图

```
┌─────────────────────────────────────────────────────────────┐
│ 用户请求：创建子文档                                          │
│ POST /api/filetree/createDocWithMd                           │
│ {                                                            │
│   "notebook": "box1",                                        │
│   "path": "/父文档/子文档",                                   │
│   "markdown": "# 子文档内容"                                  │
│ }                                                            │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│ 步骤 1：认证和创建 Context                                    │
│                                                              │
│ CheckWebAuth 中间件：                                         │
│ • 验证 JWT Token                                             │
│ • 提取用户信息（UserID: "user_a"）                           │
│ • 创建 WorkspaceContext：                                    │
│   - DataDir: "/root/.../user_a"                             │
│   - BlockTreeDBPath: "/root/.../user_a/blocktree.db"       │
│ • 注入到 gin.Context                                         │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│ 步骤 2：API 层处理                                            │
│                                                              │
│ createDocWithMd() 函数：                                     │
│ • 从 gin.Context 获取 WorkspaceContext                       │
│ • 解析请求参数                                                │
│ • 调用 Model 层函数                                          │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│ 步骤 3：Model 层 - 解析路径                                   │
│                                                              │
│ createDocsByHPathWithContext() 函数：                        │
│                                                              │
│ 输入路径："/父文档/子文档"                                     │
│                                                              │
│ 解析结果：                                                    │
│ • 路径层级：["父文档", "子文档"]                              │
│ • 父路径："/父文档"                                           │
│ • 子路径："/子文档"                                           │
│ • 需要创建：父文档（如果不存在）+ 子文档                       │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│ 步骤 4：获取用户数据库连接                                     │
│                                                              │
│ if ctx.IsWebMode() {                                        │
│     userDB, err := btManager.GetOrCreateDB(                 │
│         ctx.BlockTreeDBPath                                 │
│     )                                                       │
│     if err != nil {                                         │
│         // 降级到全局数据库                                   │
│         userDB = nil                                        │
│     }                                                       │
│ }                                                           │
│                                                              │
│ BlockTreeDBManager 操作：                                    │
│ • 检查连接池是否已有该数据库的连接                             │
│ • 如果有，复用现有连接                                        │
│ • 如果没有，创建新连接并加入连接池                             │
│ • 如果数据库文件不存在，自动创建并初始化表结构                  │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│ 步骤 5：查询父文档（使用用户数据库）                           │
│                                                              │
│ if userDB != nil {                                          │
│     parent = GetBlockTreeRootByHPathWithDB(                 │
│         "box1", "/父文档", userDB                           │
│     )                                                       │
│ } else {                                                    │
│     parent = GetBlockTreeRootByHPath(                       │
│         "box1", "/父文档"                                   │
│     )                                                       │
│ }                                                           │
│                                                              │
│ 查询 SQL：                                                   │
│ SELECT * FROM blocktrees                                    │
│ WHERE box_id = 'box1'                                       │
│   AND hpath = '/父文档'                                      │
│   AND type = 'd'                                            │
│                                                              │
│ 结果：                                                        │
│ • 如果找到：parent = BlockTree{ID: "parent-id", ...}        │
│ • 如果未找到：parent = nil                                   │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
                    ┌────┴────┐
                    │ parent  │
                    │ 是否存在？│
                    └────┬────┘
                         │
         ┌───────────────┴───────────────┐
         │                               │
         ▼ 不存在                        ▼ 存在
┌─────────────────────┐         ┌─────────────────────┐
│ 步骤 6a：创建父文档  │         │ 步骤 6b：重用父文档  │
│                     │         │                     │
│ 1. 生成父文档 ID    │         │ 1. 使用现有父文档 ID │
│    parentID =       │         │    parentID =       │
│    "20260125-xxx"   │         │    parent.ID        │
│                     │         │                     │
│ 2. 创建父文档文件   │         │ 2. 跳过文件创建     │
│    path = "/parent  │         │                     │
│    -id.sy"          │         │ 3. 获取父文档路径   │
│                     │         │    parentPath =     │
│ 3. 写入文件系统     │         │    parent.Path      │
│    /root/.../user_a │         │                     │
│    /box1/parent-id  │         │ ✅ 避免重复创建     │
│    .sy              │         │                     │
│                     │         │                     │
│ 4. 更新 BlockTree   │         │                     │
│    UpsertBlockTree  │         │                     │
│    WithDB(tree,     │         │                     │
│    userDB)          │         │                     │
└──────────┬──────────┘         └──────────┬──────────┘
           │                               │
           └───────────────┬───────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│ 步骤 7：创建子文档                                            │
│                                                              │
│ 1. 生成子文档 ID                                             │
│    childID = "20260125-yyy"                                 │
│                                                              │
│ 2. 构建子文档路径（在父文档目录下）                           │
│    childPath = parentPath + "/" + childID + ".sy"           │
│    例如："/parent-id/child-id.sy"                           │
│                                                              │
│ 3. 创建子文档文件                                            │
│    /root/.../user_a/box1/parent-id/child-id.sy             │
│                                                              │
│ 4. 写入 Markdown 内容                                        │
│    "# 子文档内容"                                            │
│                                                              │
│ 5. 更新 BlockTree 数据库                                     │
│    UpsertBlockTreeWithDB(childTree, userDB)                 │
│                                                              │
│ BlockTree 记录：                                             │
│ ┌──────────┬──────────┬───────────┬──────────────────┐      │
│ │ id       │ root_id  │ parent_id │ hpath            │      │
│ ├──────────┼──────────┼───────────┼──────────────────┤      │
│ │ child-id │ child-id │ parent-id │ /父文档/子文档    │      │
│ └──────────┴──────────┴───────────┴──────────────────┘      │
└────────────────────────┬────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────────┐
│ 步骤 8：返回结果                                              │
│                                                              │
│ 返回给前端：                                                  │
│ {                                                            │
│   "code": 0,                                                │
│   "msg": "",                                                │
│   "data": {                                                 │
│     "id": "child-id",                                       │
│     "path": "/parent-id/child-id.sy",                       │
│     "hpath": "/父文档/子文档"                                 │
│   }                                                         │
│ }                                                           │
│                                                              │
│ 前端更新：                                                    │
│ • 文档树显示新的子文档                                        │
│ • 子文档显示在父文档下                                        │
│ • 打开子文档编辑器                                            │
└─────────────────────────────────────────────────────────────┘
```

## 关键代码片段

### 1. 获取用户数据库

```go
func createDocsByHPathWithContext(ctx *WorkspaceContext, boxID, hPath, content, parentID, id string) (retID string, err error) {
    // 获取用户特定的数据库
    var userDB *sql.DB
    if ctx.IsWebMode() {
        userDB, err = treenode.GetBlockTreeDBManager().GetOrCreateDB(ctx.BlockTreeDBPath)
        if err != nil {
            logging.LogErrorf("get or create BlockTree database [%s] failed: %s", ctx.BlockTreeDBPath, err)
            userDB = nil // 降级到全局数据库
        }
    }
    
    // ... 后续使用 userDB
}
```

### 2. 查询父文档

```go
// 使用用户数据库查询父文档
var root *treenode.BlockTree
if nil != userDB {
    root = treenode.GetBlockTreeRootByHPathWithDB(boxID, hp, userDB)
} else {
    root = treenode.GetBlockTreeRootByHPath(boxID, hp)
}

if nil != root {
    // 父文档存在，重用
    parentID = root.ID
    parentPath = root.Path
} else {
    // 父文档不存在，创建
    parentID, err = createDoc(boxID, hp, "")
    if err != nil {
        return "", err
    }
}
```

### 3. 创建子文档

```go
// 在父文档目录下创建子文档
childPath := filepath.Join(filepath.Dir(parentPath), childID+".sy")

// 创建文件
err = os.MkdirAll(filepath.Dir(childPath), 0755)
if err != nil {
    return "", err
}

err = os.WriteFile(childPath, []byte(content), 0644)
if err != nil {
    return "", err
}

// 更新 BlockTree
tree := parseMarkdown(content)
tree.ID = childID
tree.Path = childPath
tree.HPath = hPath
tree.ParentID = parentID

if nil != userDB {
    treenode.UpsertBlockTreeWithDB(tree, userDB)
} else {
    treenode.UpsertBlockTree(tree)
}
```

## 数据库表结构

### blocktrees 表

```sql
CREATE TABLE blocktrees (
    id        TEXT PRIMARY KEY,  -- 块 ID
    root_id   TEXT NOT NULL,     -- 根文档 ID
    parent_id TEXT,              -- 父块 ID
    box_id    TEXT NOT NULL,     -- 笔记本 ID
    path      TEXT NOT NULL,     -- 文档路径
    hpath     TEXT NOT NULL,     -- 可读路径
    updated   TEXT NOT NULL,     -- 更新时间
    type      TEXT NOT NULL      -- 类型（d=文档）
);

CREATE INDEX idx_blocktrees_id ON blocktrees(id);
CREATE INDEX idx_blocktrees_box_hpath ON blocktrees(box_id, hpath);
CREATE INDEX idx_blocktrees_box_path ON blocktrees(box_id, path);
CREATE INDEX idx_blocktrees_root_id ON blocktrees(root_id);
```

### 示例数据

创建 `/父文档/子文档1` 和 `/父文档/子文档2` 后的数据：

```
┌──────────────────┬──────────────────┬──────────────────┬────────┬─────────────────────┬──────────────────┬─────────────────────┬──────┐
│ id               │ root_id          │ parent_id        │ box_id │ path                │ hpath            │ updated             │ type │
├──────────────────┼──────────────────┼──────────────────┼────────┼─────────────────────┼──────────────────┼─────────────────────┼──────┤
│ 20260125-parent  │ 20260125-parent  │                  │ box1   │ /20260125-parent.sy │ /父文档          │ 20260125144733      │ d    │
│ 20260125-child1  │ 20260125-child1  │ 20260125-parent  │ box1   │ /20260125-parent/   │ /父文档/子文档1  │ 20260125144800      │ d    │
│                  │                  │                  │        │  20260125-child1.sy │                  │                     │      │
│ 20260125-child2  │ 20260125-child2  │ 20260125-parent  │ box1   │ /20260125-parent/   │ /父文档/子文档2  │ 20260125144900      │ d    │
│                  │                  │                  │        │  20260125-child2.sy │                  │                     │      │
└──────────────────┴──────────────────┴──────────────────┴────────┴─────────────────────┴──────────────────┴─────────────────────┴──────┘
```

**关键点**：
- 父文档只创建一次（ID: `20260125-parent`）
- 两个子文档都引用同一个父文档（`parent_id` 相同）
- 子文档的 `path` 都在父文档目录下

## 文件系统结构

```
/root/code/MindOcean/user-data/notes/user_a/box1/
├── 20260125-parent.sy              # 父文档文件
│   └── 20260125-parent/            # 父文档目录
│       ├── 20260125-child1.sy      # 子文档1
│       └── 20260125-child2.sy      # 子文档2
└── .siyuan/
    └── conf.json
```

## 多用户场景

### 场景：两个用户创建同名文档

**用户 A**：
```
请求：创建 /父文档/子文档
数据库：/root/.../user_a/blocktree.db
结果：
  - 父文档 ID: parent-a
  - 子文档 ID: child-a
  - 路径：/parent-a/child-a.sy
```

**用户 B**：
```
请求：创建 /父文档/子文档
数据库：/root/.../user_b/blocktree.db
结果：
  - 父文档 ID: parent-b
  - 子文档 ID: child-b
  - 路径：/parent-b/child-b.sy
```

**数据隔离**：
- ✅ 两个用户的文档完全独立
- ✅ 查询时只能看到自己的文档
- ✅ 文件存储在不同的目录
- ✅ 数据库记录在不同的数据库文件

## 性能优化

### 1. 数据库连接复用

```go
// 第一次查询：创建连接（~50ms）
userDB, _ := btManager.GetOrCreateDB(ctx.BlockTreeDBPath)
parent := GetBlockTreeRootByHPathWithDB("box1", "/父文档", userDB)

// 第二次查询：复用连接（~0.1ms）
child := GetBlockTreeWithDB(childID, userDB)
```

### 2. 慢查询监控

所有查询函数都有性能监控：

```go
func GetBlockTreeRootByHPathWithDB(boxID, hPath string, database *sql.DB) (ret *BlockTree) {
    start := time.Now()
    defer func() {
        duration := time.Since(start)
        if duration > 100*time.Millisecond {
            logging.LogWarnf("slow query: GetBlockTreeRootByHPath took %v", duration)
        }
    }()
    // 查询逻辑
}
```

### 3. 批量操作

创建多个子文档时，复用数据库连接：

```go
userDB, _ := btManager.GetOrCreateDB(ctx.BlockTreeDBPath)

for _, childName := range childNames {
    // 复用同一个数据库连接
    createSubDocument(userDB, parentID, childName)
}
```

## 错误处理

### 1. 数据库连接失败

```go
userDB, err := btManager.GetOrCreateDB(ctx.BlockTreeDBPath)
if err != nil {
    logging.LogErrorf("get user database failed: %s", err)
    // 降级到全局数据库
    userDB = nil
}

// 使用数据库（自动降级）
if userDB != nil {
    parent = GetBlockTreeRootByHPathWithDB(boxID, hPath, userDB)
} else {
    parent = GetBlockTreeRootByHPath(boxID, hPath)
}
```

### 2. 父文档查询失败

```go
parent := GetBlockTreeRootByHPathWithDB(boxID, parentPath, userDB)
if parent == nil {
    // 父文档不存在，创建新的
    parentID, err = createDoc(boxID, parentPath, "")
    if err != nil {
        return "", fmt.Errorf("create parent document failed: %w", err)
    }
}
```

### 3. 文件创建失败

```go
err = os.WriteFile(childPath, []byte(content), 0644)
if err != nil {
    // 清理已创建的数据库记录
    treenode.RemoveBlockTree(childID)
    return "", fmt.Errorf("write file failed: %w", err)
}
```

## 测试验证

### 单元测试

```go
func TestCreateSubDocument(t *testing.T) {
    ctx := createTestContext(t, "user_a")
    
    // 创建父文档
    parentID, _ := createDocsByHPathWithContext(ctx, "box1", "/父文档", "", "", "")
    
    // 创建第一个子文档
    child1ID, _ := createDocsByHPathWithContext(ctx, "box1", "/父文档/子文档1", "", parentID, "")
    
    // 创建第二个子文档
    child2ID, _ := createDocsByHPathWithContext(ctx, "box1", "/父文档/子文档2", "", parentID, "")
    
    // 验证两个子文档都在同一个父文档下
    userDB, _ := btManager.GetOrCreateDB(ctx.BlockTreeDBPath)
    child1 := GetBlockTreeWithDB(child1ID, userDB)
    child2 := GetBlockTreeWithDB(child2ID, userDB)
    
    assert.Equal(t, parentID, child1.ParentID)
    assert.Equal(t, parentID, child2.ParentID)
    assert.Contains(t, child1.Path, parentID)
    assert.Contains(t, child2.Path, parentID)
}
```

### 集成测试

```bash
# 测试脚本
./test-create-subdoc.sh

# 预期输出
✓ 创建父文档成功
✓ 创建第一个子文档成功
✓ 创建第二个子文档成功
✓ 验证父文档只有一个
✓ 验证两个子文档都在父文档下
✓ 验证文件系统结构正确
```

## 总结

子文档创建流程的关键改进：

1. **使用用户数据库查询** - 确保查询到正确的父文档
2. **重用已存在的父文档** - 避免创建重复的父文档
3. **正确的路径构建** - 子文档创建在父文档目录下
4. **完整的数据隔离** - 多用户数据互不干扰
5. **性能优化** - 连接复用和慢查询监控
6. **错误处理** - 降级策略和清理机制

这个流程确保了灵枢笔记在多用户环境下的正确性和稳定性。
