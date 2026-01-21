# 多用户架构重构 - 任务列表

## 阶段 1：基础设施准备

- [x] 1.1 创建 WorkspaceContext 结构体
  - [x] 1.1.1 创建 `kernel/model/workspace_context.go` 文件
  - [x] 1.1.2 定义 `WorkspaceContext` 结构体
  - [x] 1.1.3 实现 `NewWorkspaceContext` 函数
  - [x] 1.1.4 实现 `NewWorkspaceContextWithUser` 函数
  - [x] 1.1.5 实现 `GetWorkspaceContext` 函数
  - [x] 1.1.6 实现 `GetDefaultWorkspaceContext` 函数
  - [x] 1.1.7 实现 `SetWorkspaceContext` 函数
  - [x] 1.1.8 实现 Context 辅助方法（GetDataDir, GetConfDir 等）

- [x] 1.2 修改认证中间件
  - [x] 1.2.1 在 `CheckWebAuth` 中创建 `WorkspaceContext`
  - [x] 1.2.2 将 `WorkspaceContext` 存储到 Gin Context
  - [x] 1.2.3 保留现有的 workspace 切换逻辑（向后兼容）

- [x] 1.3 测试验证
  - [x] 1.3.1 编译测试
  - [x] 1.3.2 服务启动测试
  - [x] 1.3.3 登录功能测试
  - [x] 1.3.4 查看笔记列表测试

- [x] 1.4 提交到 Git
  - [x] 1.4.1 提交代码
  - [x] 1.4.2 推送到 GitHub

## 阶段 2：API 层重构

### 2.1 文件树 API（高优先级）

- [x] 2.1.1 listDocsByPath - 列出文档
  - [x] 2.1.1.1 修改 `kernel/api/filetree.go` 中的 `listDocsByPath` 函数
  - [x] 2.1.1.2 修改 `kernel/model/file.go` 中的 `ListDocTree` 函数
  - [x] 2.1.1.3 修改所有调用 `ListDocTree` 的地方
  - [x] 2.1.1.4 编译测试
  - [x] 2.1.1.5 功能测试
  - [x] 2.1.1.6 提交到 Git

- [x] 2.1.2 getDoc - 获取文档内容
  - [x] 2.1.2.1 在 `kernel/filesys/tree.go` 中添加 `LoadTreeWithDataDir` 函数
  - [x] 2.1.2.2 在 `kernel/model/tree.go` 中添加 `LoadTreeByBlockIDWithContext` 函数
  - [x] 2.1.2.3 在 `kernel/model/file.go` 中添加 `GetDocWithContext` 函数
  - [x] 2.1.2.4 修改 `kernel/api/filetree.go` 中的 `getDoc` 函数
  - [x] 2.1.2.5 保持向后兼容（旧函数调用新函数）
  - [x] 2.1.2.6 编译测试
  - [x] 2.1.2.7 功能测试
  - [x] 2.1.2.8 提交到 Git

- [x] 2.1.3 createDocWithMd - 创建文档
  - [x] 2.1.3.1 在 `kernel/model/file.go` 中添加 `createDocWithContext` 函数
  - [x] 2.1.3.2 在 `kernel/model/file.go` 中添加 `CreateWithMarkdownWithContext` 函数
  - [x] 2.1.3.3 在 `kernel/model/path.go` 中添加 `createDocsByHPathWithContext` 函数
  - [x] 2.1.3.4 修改 `kernel/api/filetree.go` 中的 `createDocWithMd` 函数
  - [x] 2.1.3.5 使用 `ctx.GetDataDir()` 替代 `util.DataDir`
  - [x] 2.1.3.6 编译测试
  - [x] 2.1.3.7 功能测试
  - [x] 2.1.3.8 提交到 Git

- [x] 2.1.4 renameDoc - 重命名文档
  - [x] 2.1.4.1 在 `kernel/model/file.go` 中添加 `RenameDocWithContext` 函数
  - [x] 2.1.4.2 修改 `kernel/api/filetree.go` 中的 `renameDoc` 函数
  - [x] 2.1.4.3 使用 `filesys.LoadTreeWithDataDir` 替代 `filesys.LoadTree`
  - [x] 2.1.4.4 编译测试
  - [x] 2.1.4.5 功能测试
  - [x] 2.1.4.6 提交到 Git

- [x] 2.1.5 removeDoc - 删除文档
  - [x] 2.1.5.1 在 `kernel/model/file.go` 中添加 `RemoveDocWithContext` 函数
  - [x] 2.1.5.2 在 `kernel/model/file.go` 中添加 `removeDocWithContext` 函数
  - [x] 2.1.5.3 修改 `kernel/api/filetree.go` 中的 `removeDoc` 函数
  - [x] 2.1.5.4 使用 `filesys.LoadTreeWithDataDir` 和 `LoadTreeByBlockIDWithContext`
  - [x] 2.1.5.5 编译测试
  - [x] 2.1.5.6 功能测试
  - [x] 2.1.5.7 提交到 Git

### 2.2 块操作 API（高优先级）

- [x] 2.2.1 getDocInfo 和 getBlockInfo - 获取块信息
  - [x] 2.2.1.1 在 `kernel/model/blockinfo.go` 中添加 `GetDocInfoWithContext` 函数
  - [x] 2.2.1.2 在 `kernel/model/tree.go` 中添加 `LoadTreeByBlockIDWithReindexAndContext` 函数
  - [x] 2.2.1.3 修改 `kernel/api/block.go` 中的 `getDocInfo` 函数
  - [x] 2.2.1.4 修改 `kernel/api/block.go` 中的 `getBlockInfo` 函数
  - [x] 2.2.1.5 使用 `filesys.LoadTreeWithDataDir` 替代 `filesys.LoadTree`
  - [x] 2.2.1.6 编译测试
  - [x] 2.2.1.7 功能测试
  - [x] 2.2.1.8 提交到 Git

- [x] 2.2.2 insertBlock - 插入块
  - [x] 2.2.2.1 读取 `kernel/api/block_op.go` 中的 `insertBlock` 实现
  - [x] 2.2.2.2 分析调用的 Model 层函数（如 `PerformTransactions`）
  - [x] 2.2.2.3 为相关 Model 层函数添加带 Context 的版本
  - [x] 2.2.2.4 修改 API 层的 `insertBlock` 函数
  - [x] 2.2.2.5 编译测试
  - [x] 2.2.2.6 功能测试（插入块）
  - [x] 2.2.2.7 提交到 Git

- [x] 2.2.3 updateBlock - 更新块
  - [x] 2.2.3.1 读取 `kernel/api/block_op.go` 中的 `updateBlock` 实现
  - [x] 2.2.3.2 分析调用的 Model 层函数
  - [x] 2.2.3.3 为相关 Model 层函数添加带 Context 的版本
  - [x] 2.2.3.4 修改 API 层的 `updateBlock` 函数
  - [x] 2.2.3.5 编译测试
  - [x] 2.2.3.6 功能测试（更新块）
  - [x] 2.2.3.7 提交到 Git

- [x] 2.2.4 deleteBlock - 删除块
  - [x] 2.2.4.1 读取 `kernel/api/block_op.go` 中的 `deleteBlock` 实现
  - [x] 2.2.4.2 分析调用的 Model 层函数
  - [x] 2.2.4.3 为相关 Model 层函数添加带 Context 的版本
  - [x] 2.2.4.4 修改 API 层的 `deleteBlock` 函数
  - [x] 2.2.4.5 编译测试
  - [x] 2.2.4.6 功能测试（删除块）
  - [x] 2.2.4.7 提交到 Git

### 2.3 笔记本 API（高优先级）

- [x] 2.3.1 lsNotebooks - 列出笔记本
  - [x] 2.3.1.1 修改 `kernel/api/notebook.go` 中的 `lsNotebooks` 函数
  - [x] 2.3.1.2 修改 `kernel/model/box.go` 中的 `ListNotebooks` 函数
  - [x] 2.3.1.3 修改所有调用 `ListNotebooks` 的地方
  - [x] 2.3.1.4 编译测试
  - [x] 2.3.1.5 功能测试
  - [x] 2.3.1.6 提交到 Git

- [x] 2.3.2 openNotebook - 打开笔记本
  - [x] 2.3.2.1 读取 `kernel/api/notebook.go` 中的 `openNotebook` 实现
  - [x] 2.3.2.2 分析调用的 Model 层函数
  - [x] 2.3.2.3 为相关 Model 层函数添加带 Context 的版本
  - [x] 2.3.2.4 修改 API 层的 `openNotebook` 函数
  - [x] 2.3.2.5 编译测试
  - [x] 2.3.2.6 功能测试
  - [x] 2.3.2.7 提交到 Git

- [x] 2.3.3 closeNotebook - 关闭笔记本
  - [x] 2.3.3.1 读取 `kernel/api/notebook.go` 中的 `closeNotebook` 实现
  - [x] 2.3.3.2 分析调用的 Model 层函数
  - [x] 2.3.3.3 为相关 Model 层函数添加带 Context 的版本
  - [x] 2.3.3.4 修改 API 层的 `closeNotebook` 函数
  - [x] 2.3.3.5 编译测试
  - [x] 2.3.3.6 功能测试
  - [x] 2.3.3.7 提交到 Git

- [x] 2.3.4 createNotebook - 创建笔记本
  - [x] 2.3.4.1 读取 `kernel/api/notebook.go` 中的 `createNotebook` 实现
  - [x] 2.3.4.2 分析调用的 Model 层函数
  - [x] 2.3.4.3 为相关 Model 层函数添加带 Context 的版本
  - [x] 2.3.4.4 修改 API 层的 `createNotebook` 函数
  - [x] 2.3.4.5 编译测试
  - [x] 2.3.4.6 功能测试
  - [x] 2.3.4.7 提交到 Git
  - [ ] 2.3.4.5 编译测试
  - [ ] 2.3.4.6 功能测试
  - [ ] 2.3.4.7 提交到 Git

### 2.4 搜索和资产 API（中优先级）

- [x] 2.4.1 搜索 API
  - [x] 2.4.1.1 分析 `kernel/api/search.go` 中的搜索函数
  - [x] 2.4.1.2 为相关 Model 层函数添加带 Context 的版本
  - [x] 2.4.1.3 修改 API 层函数
  - [x] 2.4.1.4 编译测试
  - [x] 2.4.1.5 功能测试
  - [x] 2.4.1.6 提交到 Git

- [x] 2.4.2 资产 API
  - [x] 2.4.2.1 分析 `kernel/api/asset.go` 中的资产函数
  - [x] 2.4.2.2 为相关 Model 层函数添加带 Context 的版本
  - [x] 2.4.2.3 修改 API 层函数
  - [x] 2.4.2.4 编译测试
  - [x] 2.4.2.5 功能测试（上传图片、附件）
  - [x] 2.4.2.6 提交到 Git

### 2.5 模板和标签 API（中优先级）

- [x] 2.5.1 模板 API
  - [x] 2.5.1.1 分析 `kernel/api/template.go` 中的模板函数
  - [x] 2.5.1.2 为相关 Model 层函数添加带 Context 的版本
  - [x] 2.5.1.3 修改 API 层函数
  - [x] 2.5.1.4 编译测试
  - [x] 2.5.1.5 功能测试
  - [x] 2.5.1.6 提交到 Git

- [x] 2.5.2 标签 API
  - [x] 2.5.2.1 分析 `kernel/api/tag.go` 中的标签函数
  - [x] 2.5.2.2 为相关 Model 层函数添加带 Context 的版本
  - [x] 2.5.2.3 修改 API 层函数
  - [x] 2.5.2.4 编译测试
  - [x] 2.5.2.5 功能测试
  - [x] 2.5.2.6 提交到 Git

### 2.6 导出、导入和历史 API（低优先级）

- [ ] 2.6.1 导出 API
  - [ ] 2.6.1.1 分析 `kernel/api/export.go` 中的导出函数
  - [ ] 2.6.1.2 为相关 Model 层函数添加带 Context 的版本
  - [ ] 2.6.1.3 修改 API 层函数
  - [ ] 2.6.1.4 编译测试
  - [ ] 2.6.1.5 功能测试
  - [ ] 2.6.1.6 提交到 Git

- [ ] 2.6.2 导入 API
  - [ ] 2.6.2.1 分析 `kernel/api/import.go` 中的导入函数
  - [ ] 2.6.2.2 为相关 Model 层函数添加带 Context 的版本
  - [ ] 2.6.2.3 修改 API 层函数
  - [ ] 2.6.2.4 编译测试
  - [ ] 2.6.2.5 功能测试
  - [ ] 2.6.2.6 提交到 Git

- [ ] 2.6.3 历史 API
  - [ ] 2.6.3.1 分析 `kernel/api/history.go` 中的历史函数
  - [ ] 2.6.3.2 为相关 Model 层函数添加带 Context 的版本
  - [ ] 2.6.3.3 修改 API 层函数
  - [ ] 2.6.3.4 编译测试
  - [ ] 2.6.3.5 功能测试
  - [ ] 2.6.3.6 提交到 Git

## 阶段 3：数据库层重构

- [x] 3.1 创建数据库连接池
  - [x] 3.1.1 创建 `kernel/sql/db_pool.go` 文件
  - [x] 3.1.2 定义 `DBPool` 结构体
  - [x] 3.1.3 实现 `GetDB(ctx *WorkspaceContext)` 方法
  - [x] 3.1.4 实现连接复用机制
  - [x] 3.1.5 实现连接清理机制
  - [x] 3.1.6 实现最大连接数限制

- [x] 3.2 重构数据库访问层
  - [x] 3.2.1 修改 `kernel/sql/database.go`
  - [x] 3.2.2 为核心查询函数添加带 `ctx WorkspaceContext` 参数的版本
  - [x] 3.2.3 使用连接池获取数据库连接
  - [x] 3.2.4 编译测试
  - [x] 3.2.5 功能测试

- [ ] 3.3 重构索引和搜索
  - [ ] 3.3.1 修改 `kernel/search/` 下的文件
  - [ ] 3.3.2 支持按用户隔离的索引
  - [ ] 3.3.3 编译测试
  - [ ] 3.3.4 功能测试（搜索功能）

- [ ] 3.4 测试验证
  - [ ] 3.4.1 多用户并发测试
  - [ ] 3.4.2 数据隔离验证
  - [ ] 3.4.3 数据库连接数监控
  - [ ] 3.4.4 性能测试

- [ ] 3.5 提交到 Git
  - [ ] 3.5.1 提交代码
  - [ ] 3.5.2 推送到 GitHub

## 阶段 4：缓存层重构

- [x] 4.1 创建用户缓存管理器
  - [x] 4.1.1 创建 `kernel/cache/user_cache.go` 文件
  - [x] 4.1.2 定义 `UserCache` 结构体
  - [x] 4.1.3 实现 `GetCache(ctx *WorkspaceContext)` 方法
  - [x] 4.1.4 实现缓存隔离机制
  - [x] 4.1.5 实现缓存清理机制

- [x] 4.2 重构缓存访问
  - [x] 4.2.1 修改所有缓存访问代码
  - [x] 4.2.2 添加 `ctx *WorkspaceContext` 参数
  - [x] 4.2.3 使用用户缓存管理器
  - [x] 4.2.4 编译测试
  - [x] 4.2.5 功能测试

- [ ] 4.3 测试验证
  - [ ] 4.3.1 多用户缓存隔离测试
  - [ ] 4.3.2 缓存命中率测试
  - [ ] 4.3.3 内存使用监控

- [x] 4.4 提交到 Git
  - [x] 4.4.1 提交代码
  - [x] 4.4.2 推送到 GitHub

## 阶段 5：移除全局 workspace 切换

- [x] 5.1 移除 workspace 切换逻辑
  - [x] 5.1.1 删除 `CheckWebAuth` 中的 workspace 切换代码
  - [x] 5.1.2 删除 `workspaceMutex` 互斥锁
  - [x] 5.1.3 删除 `defer` 恢复逻辑
  - [x] 5.1.4 编译测试

- [ ] 5.2 清理全局变量使用
  - [ ] 5.2.1 搜索所有 `util.DataDir` 使用
  - [ ] 5.2.2 确认都已改为使用 `WorkspaceContext`
  - [ ] 5.2.3 清理未使用的代码
  - [ ] 5.2.4 编译测试

- [ ] 5.3 测试验证
  - [ ] 5.3.1 并发性能测试
  - [ ] 5.3.2 响应时间测试
  - [ ] 5.3.3 数据正确性验证
  - [ ] 5.3.4 无互斥锁阻塞验证

- [x] 5.4 提交到 Git
  - [x] 5.4.1 提交代码
  - [ ] 5.4.2 推送到 GitHub

## 阶段 6：全面测试和优化

### 6.1 功能测试

- [ ] 6.1.1 基础功能测试
  - [ ] 6.1.1.1 用户登录/登出
  - [ ] 6.1.1.2 笔记本列表
  - [ ] 6.1.1.3 文档列表
  - [ ] 6.1.1.4 打开文档
  - [ ] 6.1.1.5 编辑文档
  - [ ] 6.1.1.6 创建文档
  - [ ] 6.1.1.7 删除文档
  - [ ] 6.1.1.8 重命名文档

- [ ] 6.1.2 高级功能测试
  - [ ] 6.1.2.1 搜索功能
  - [ ] 6.1.2.2 标签功能
  - [ ] 6.1.2.3 书签功能
  - [ ] 6.1.2.4 模板功能
  - [ ] 6.1.2.5 资产上传
  - [ ] 6.1.2.6 导出功能
  - [ ] 6.1.2.7 历史记录

### 6.2 多用户测试

- [ ] 6.2.1 多用户并发测试
  - [ ] 6.2.1.1 两个用户同时登录
  - [ ] 6.2.1.2 同时编辑不同文档
  - [ ] 6.2.1.3 数据隔离验证
  - [ ] 6.2.1.4 缓存隔离验证
  - [ ] 6.2.1.5 数据库隔离验证

- [ ] 6.2.2 压力测试
  - [ ] 6.2.2.1 10 个并发用户测试
  - [ ] 6.2.2.2 100 个并发用户测试
  - [ ] 6.2.2.3 持续 1 小时负载测试
  - [ ] 6.2.2.4 内存泄漏检测

### 6.3 性能测试

- [ ] 6.3.1 响应时间测试
  - [ ] 6.3.1.1 单用户响应时间
  - [ ] 6.3.1.2 10 个并发用户响应时间
  - [ ] 6.3.1.3 100 个并发用户响应时间
  - [ ] 6.3.1.4 性能对比（重构前后）

- [ ] 6.3.2 资源使用测试
  - [ ] 6.3.2.1 内存使用监控
  - [ ] 6.3.2.2 CPU 使用监控
  - [ ] 6.3.2.3 数据库连接数监控
  - [ ] 6.3.2.4 磁盘 I/O 监控

### 6.4 正确性属性测试

- [ ] 6.4.1 数据隔离属性测试
  - [ ] 6.4.1.1 Property 1.1: 用户数据完全隔离
  - [ ] 6.4.1.2 Property 1.2: WorkspaceContext 不可变性

- [ ] 6.4.2 并发安全属性测试
  - [ ] 6.4.2.1 Property 2.1: 无数据竞争
  - [ ] 6.4.2.2 Property 2.2: 请求独立性

- [ ] 6.4.3 数据一致性属性测试
  - [ ] 6.4.3.1 Property 3.1: 数据库连接正确性
  - [ ] 6.4.3.2 Property 3.2: 缓存一致性

- [ ] 6.4.4 性能属性测试
  - [ ] 6.4.4.1 Property 4.1: 响应时间上界
  - [ ] 6.4.4.2 Property 4.2: 并发性能

- [ ] 6.4.5 资源管理属性测试
  - [ ] 6.4.5.1 Property 5.1: 数据库连接数上界
  - [ ] 6.4.5.2 Property 5.2: 内存使用上界

### 6.5 优化

- [ ] 6.5.1 数据库连接池优化
  - [ ] 6.5.1.1 调整连接池大小
  - [ ] 6.5.1.2 调整空闲连接超时
  - [ ] 6.5.1.3 调整最大连接数

- [ ] 6.5.2 缓存优化
  - [ ] 6.5.2.1 调整缓存大小
  - [ ] 6.5.2.2 调整缓存过期策略
  - [ ] 6.5.2.3 实现 LRU 缓存淘汰

- [ ] 6.5.3 性能优化
  - [ ] 6.5.3.1 热点代码优化
  - [ ] 6.5.3.2 减少锁竞争
  - [ ] 6.5.3.3 减少内存分配

### 6.6 文档

- [ ] 6.6.1 更新架构文档
- [ ] 6.6.2 更新 API 文档
- [ ] 6.6.3 编写部署文档
- [ ] 6.6.4 编写运维文档

### 6.7 最终验证

- [ ] 6.7.1 所有功能正常
- [ ] 6.7.2 所有测试通过
- [ ] 6.7.3 性能达标
- [ ] 6.7.4 代码审查通过

### 6.8 提交到 Git

- [ ] 6.8.1 提交代码
- [ ] 6.8.2 推送到 GitHub
- [ ] 6.8.3 创建 Release

## 备注

- 每完成一个子任务都要测试
- 每完成一个阶段都要提交到 Git
- 出现问题立即回滚到上一个稳定版本
- 保持现有功能正常运行
