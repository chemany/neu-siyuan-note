# 灵枢笔记构建和部署指南

## 构建路径说明

**重要**：所有构建脚本必须将内核编译到 `kernel/siyuan-kernel`，因为 pm2 配置运行的是这个路径。

### PM2 配置

```javascript
{
  name: 'siyuan-kernel',
  cwd: '/root/code/neu-siyuan-note/kernel',
  script: 'siyuan-kernel',  // 运行 /root/code/neu-siyuan-note/kernel/siyuan-kernel
  // ...
}
```

### 构建脚本

所有构建脚本都应该使用以下编译命令（在 `kernel` 目录下执行）：

```bash
cd /root/code/neu-siyuan-note/kernel
CGO_ENABLED=1 go build -v -o ./siyuan-kernel -tags "fts5" -ldflags "-s -w" .
```

**注意**：
- ✅ 正确：`-o ./siyuan-kernel` 或 `-o siyuan-kernel`（编译到当前目录，即 kernel 目录）
- ❌ 错误：`-o ../siyuan-kernel`（编译到父目录，pm2 找不到）

## 构建脚本列表

### 1. rebuild-kernel-only.sh
仅重新编译后端内核，不构建前端。

```bash
bash /root/code/neu-siyuan-note/rebuild-kernel-only.sh
```

**用途**：快速修复后端 bug，不需要重新构建前端。

### 2. rebuild-and-restart.sh
完整重新构建前端和后端，然后重启服务。

```bash
bash /root/code/neu-siyuan-note/rebuild-and-restart.sh
```

**用途**：前端或后端有重大更新时使用。

### 3. rebuild-and-test-websocket.sh
重新编译内核并测试 WebSocket 功能。

```bash
bash /root/code/neu-siyuan-note/rebuild-and-test-websocket.sh
```

**用途**：测试 WebSocket 相关功能。

## 验证构建

构建完成后，验证二进制文件位置：

```bash
ls -lh /root/code/neu-siyuan-note/kernel/siyuan-kernel
```

应该看到最新的编译时间。

## 常见问题

### 问题：pm2 运行的是旧版本内核

**症状**：修改代码后重新编译，但是运行时仍然是旧的行为。

**原因**：构建脚本将内核编译到了错误的位置（如 `neu-siyuan-note/siyuan-kernel`），而 pm2 运行的是 `neu-siyuan-note/kernel/siyuan-kernel`。

**解决方案**：
1. 检查构建脚本的 `-o` 参数
2. 确保编译到 `./siyuan-kernel`（在 kernel 目录下）
3. 手动复制：`cp /root/code/neu-siyuan-note/siyuan-kernel /root/code/neu-siyuan-note/kernel/siyuan-kernel`
4. 重启 pm2：`pm2 restart siyuan-kernel`

### 问题：编译失败

**常见原因**：
1. Go 模块依赖问题：运行 `go mod tidy`
2. CGO 未启用：确保 `CGO_ENABLED=1`
3. 缺少构建标签：确保包含 `-tags "fts5"`

## 部署流程

### 开发环境

1. 修改代码
2. 运行 `rebuild-kernel-only.sh`（仅后端）或 `rebuild-and-restart.sh`（前后端）
3. 测试功能
4. 查看日志：`tail -f /root/code/pm2-apps/logs/siyuan-out.log`

### 生产环境

1. 在开发环境测试通过
2. 提交代码到 Git
3. 在生产服务器拉取最新代码
4. 运行 `rebuild-and-restart.sh`
5. 验证服务状态：`pm2 status siyuan-kernel`
6. 监控日志确认无错误

## 服务管理

### 启动服务
```bash
pm2 start ecosystem.config.js --only siyuan-kernel
```

### 停止服务
```bash
pm2 stop siyuan-kernel
```

### 重启服务
```bash
pm2 restart siyuan-kernel
```

### 查看日志
```bash
# 实时日志
pm2 logs siyuan-kernel

# 最近 100 行
tail -100 /root/code/pm2-apps/logs/siyuan-out.log

# 错误日志
tail -100 /root/code/pm2-apps/logs/siyuan-error.log
```

### 查看服务状态
```bash
pm2 status siyuan-kernel
```

## 目录结构

```
/root/code/neu-siyuan-note/
├── kernel/
│   ├── siyuan-kernel          # ← pm2 运行这个文件
│   ├── main.go
│   ├── model/
│   ├── api/
│   └── ...
├── app/                        # 前端源码
├── stage/                      # 前端构建产物
├── rebuild-kernel-only.sh      # 仅构建后端
├── rebuild-and-restart.sh      # 构建前后端
└── rebuild-and-test-websocket.sh
```

## 注意事项

1. **构建路径统一**：所有构建脚本必须编译到 `kernel/siyuan-kernel`
2. **pm2 配置不变**：不要修改 pm2 的 `cwd` 和 `script` 配置
3. **验证构建**：每次构建后检查文件时间戳
4. **测试充分**：修改后端代码后务必测试核心功能
5. **备份重要数据**：重大更新前备份用户数据

修复日期：2026-01-26
