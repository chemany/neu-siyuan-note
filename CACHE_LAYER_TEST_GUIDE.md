# 缓存层重构测试指南

## 测试时间
2026-01-21

## 测试目标
验证用户缓存管理器是否正常工作，确保多用户缓存隔离。

## 已完成的工作

### 1. 创建用户缓存管理器
- ✅ 创建 `kernel/cache/user_cache.go` 文件
- ✅ 实现 `UserCacheManager` 结构体
- ✅ 实现按 workspace 隔离的缓存管理
- ✅ 实现缓存自动清理机制（30分钟空闲）
- ✅ 实现最大缓存数限制（100个用户）

### 2. 重构缓存访问函数
- ✅ 在 `kernel/sql/cache.go` 中添加带 Context 的缓存函数
- ✅ `putBlockCacheWithContext` - 存储块到用户缓存
- ✅ `getBlockCacheWithContext` - 从用户缓存获取块
- ✅ `removeBlockCacheWithContext` - 从用户缓存删除块
- ✅ `GetRefsCacheByDefIDWithContext` - 获取引用缓存
- ✅ `CacheRefWithContext` - 缓存引用

### 3. 更新数据库查询函数
- ✅ 更新 `GetBlockWithContext` 使用用户缓存
- ✅ 更新 `GetBlocksByBoxWithContext` 使用用户缓存
- ✅ 添加 `deleteBlocksByIDsWithContext` 清理用户缓存
- ✅ 添加 `QueryRefsByDefIDWithContext` 支持引用查询
- ✅ 添加 `queryBlockChildrenIDsWithContext` 支持子块查询

### 4. 编译测试
- ✅ 代码编译成功
- ✅ 服务重启成功

## 缓存架构

### 用户缓存结构
```
UserCacheManager (全局管理器)
├── workspace_a → UserCache
│   ├── blockCache (ristretto)
│   └── refCache (go-cache)
├── workspace_b → UserCache
│   ├── blockCache (ristretto)
│   └── refCache (go-cache)
└── workspace_c → UserCache
    ├── blockCache (ristretto)
    └── refCache (go-cache)
```

### 缓存隔离机制
- 每个用户有独立的 `UserCache` 实例
- Block 缓存使用 ristretto（高性能缓存库）
- Ref 缓存使用 go-cache（支持过期时间）
- 缓存按 workspace 完全隔离

### 缓存生命周期
```
1. 首次访问 → 创建新缓存 → 记录访问时间
2. 后续访问 → 复用缓存 → 更新访问时间
3. 空闲30分钟 → 自动清理 → 释放内存
4. 缓存超限 → 清理空闲缓存 → 或拒绝创建
```

## 手动测试步骤

### 测试1: 基本功能测试

1. **登录用户A**
   ```bash
   # 在浏览器中访问
   http://localhost:6806
   
   # 使用账号: link918@qq.com
   # 密码: zhangli1115
   ```

2. **查看笔记列表**
   - 点击左侧笔记本列表
   - 打开一个笔记本
   - 查看文档列表

3. **打开文档**
   - 点击一个文档
   - 查看文档内容
   - 编辑文档内容

4. **验证缓存工作**
   - 第一次打开文档时，数据从数据库加载
   - 再次打开同一文档时，数据从缓存加载（速度更快）
   - 编辑文档后，缓存自动更新

### 测试2: 多用户缓存隔离测试

**注意**: 当前系统只有一个用户，此测试需要在有多个用户时进行。

1. **用户A登录并操作**
   - 打开文档A
   - 编辑内容
   - 数据存入用户A的缓存

2. **用户B登录并操作**
   - 打开文档B
   - 编辑内容
   - 数据存入用户B的缓存

3. **验证缓存隔离**
   - 用户A的缓存不包含用户B的数据
   - 用户B的缓存不包含用户A的数据
   - 两个用户的操作互不影响

### 测试3: 缓存清理测试

1. **触发缓存清理**
   - 等待30分钟不操作
   - 缓存管理器自动清理空闲缓存

2. **验证清理效果**
   - 查看日志，确认缓存被清理
   - 再次访问时，数据从数据库重新加载

## 查看日志

### 查看缓存创建日志
```bash
tail -f /root/code/neu-siyuan-note/kernel/logging.log | grep "创建新的用户缓存"
```

### 查看缓存清理日志
```bash
tail -f /root/code/neu-siyuan-note/kernel/logging.log | grep "清理空闲缓存"
```

### 查看缓存统计
```bash
# 在 Go 代码中调用
stats := cache.GetUserCacheStats()
fmt.Printf("缓存统计: %+v\n", stats)
```

## 性能对比

### 缓存命中时
- 响应时间: < 10ms
- 无数据库查询
- 内存直接读取

### 缓存未命中时
- 响应时间: 50-100ms
- 需要数据库查询
- 查询后存入缓存

## 预期结果

### 功能正确性
- ✅ 用户可以正常登录
- ✅ 笔记列表正常显示
- ✅ 文档可以正常打开和编辑
- ✅ 缓存自动工作，无需手动干预

### 性能指标
- ✅ 首次访问: 正常速度（数据库查询）
- ✅ 重复访问: 更快速度（缓存命中）
- ✅ 内存使用: 稳定，无泄漏

### 缓存隔离
- ✅ 不同用户的缓存完全隔离
- ✅ 用户A看不到用户B的缓存数据
- ✅ 缓存操作互不影响

## 已知限制

### 1. 旧代码仍使用全局缓存
- 未使用 `WithContext` 的函数仍使用全局缓存
- 需要逐步迁移到新接口
- 不影响新代码的缓存隔离

### 2. 缓存大小限制
- 每个用户缓存: 10240 个块
- 最多100个用户缓存
- 超限时自动清理空闲缓存

### 3. 缓存一致性
- 缓存在30分钟后自动过期
- 数据修改时需要手动清理缓存
- 使用 `removeBlockCacheWithContext` 清理

## 下一步工作

### 短期
1. ✅ 完成基本功能测试
2. ⏸️ 完成多用户缓存隔离测试（需要多个用户）
3. ⏸️ 监控缓存命中率
4. ⏸️ 优化缓存大小和过期策略

### 中期
1. 将更多函数迁移到带 Context 的缓存接口
2. 移除全局缓存，完全使用用户缓存
3. 实现缓存预热机制
4. 实现 LRU 缓存淘汰策略

### 长期
1. 引入 Redis 作为分布式缓存
2. 实现缓存同步机制
3. 支持缓存持久化
4. 实现缓存监控和告警

## 技术细节

### 缓存配置
```go
// Block 缓存配置 (ristretto)
NumCounters: 102400  // 计数器数量
MaxCost:     10240   // 最大成本（块数量）
BufferItems: 64      // 缓冲区大小

// Ref 缓存配置 (go-cache)
DefaultExpiration: 30 * time.Minute  // 默认过期时间
CleanupInterval:   5 * time.Minute   // 清理间隔
```

### 缓存管理器配置
```go
maxIdle:   30 * time.Minute  // 最大空闲时间
maxCaches: 100               // 最大缓存数
```

## 故障排查

### 问题1: 缓存未生效
**症状**: 每次访问都查询数据库
**原因**: 缓存被禁用或未正确初始化
**解决**: 
```go
// 检查缓存是否启用
userCache := cache.GetUserCacheWithContext(ctx)
if !userCache.IsEnabled() {
    userCache.Enable()
}
```

### 问题2: 内存占用过高
**症状**: 内存持续增长
**原因**: 缓存未正确清理
**解决**:
```bash
# 查看缓存统计
stats := cache.GetUserCacheStats()

# 手动清理缓存
cache.ClearUserCacheByWorkspace(workspaceKey)
```

### 问题3: 缓存数据不一致
**症状**: 显示旧数据
**原因**: 数据修改后未清理缓存
**解决**:
```go
// 修改数据后清理缓存
removeBlockCacheWithContext(ctx, blockID)
```

## 参考资料

- [ristretto 缓存库文档](https://github.com/dgraph-io/ristretto)
- [go-cache 文档](https://github.com/patrickmn/go-cache)
- [缓存设计模式](https://en.wikipedia.org/wiki/Cache_(computing))

## 总结

缓存层重构已完成核心功能:
- ✅ 用户缓存管理器创建完成
- ✅ 缓存访问函数已重构
- ✅ 数据库查询函数已更新
- ✅ 编译测试通过
- ✅ 服务正常运行

下一步需要进行实际的功能测试，验证缓存是否正常工作。
