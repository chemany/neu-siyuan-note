# 数据库连接池手动测试指南

## 测试目的
验证数据库连接池是否正常工作,确保多用户数据隔离功能正常。

## 前提条件
- 灵枢笔记服务正在运行 (pm2 list 显示 siyuan 为 online 状态)
- 用户账号: link918@qq.com / zhangli1115

## 测试方法

### 方法1: 浏览器测试 (推荐)

这是最简单直观的测试方法:

1. **打开浏览器**
   ```
   访问: http://localhost:6806
   ```

2. **登录账号**
   - 邮箱: link918@qq.com
   - 密码: zhangli1115

3. **验证功能**
   - ✓ 能看到笔记本列表
   - ✓ 能打开笔记本
   - ✓ 能查看文档
   - ✓ 能编辑文档
   - ✓ 能创建新文档

4. **检查日志**
   ```bash
   pm2 logs siyuan --lines 50
   ```
   
   如果看到类似以下日志,说明连接池正在工作:
   - "为 workspace [xxx] 创建新的数据库连接"
   - "当前连接数: X"

### 方法2: curl命令测试

如果浏览器测试有问题,可以用curl测试:

#### 步骤1: 测试服务状态
```bash
curl -s http://localhost:6806/api/system/version
```
应该返回版本信息。

#### 步骤2: 尝试登录
```bash
curl -X POST http://localhost:6806/api/web/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "link918@qq.com",
    "password": "zhangli1115"
  }'
```

如果返回 `"code":0` 说明登录成功,会返回一个 token。

#### 步骤3: 使用token查询笔记本
```bash
# 先从步骤2的响应中复制token
TOKEN="你的token"

curl -X POST http://localhost:6806/api/notebook/lsNotebooks \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{}'
```

如果返回 `"code":0` 并且有笔记本列表,说明数据库查询成功。

### 方法3: 查看数据库文件

检查用户的数据库文件是否存在:

```bash
# 查看用户workspace目录
ls -lh /root/code/MindOcean/user-data/notes/jason/

# 查看数据库文件
ls -lh /root/code/MindOcean/user-data/notes/jason/conf/siyuan.db
```

如果文件存在且大小正常(通常几MB到几百MB),说明数据库正常。

## 验证数据库连接池功能

### 1. 查看连接池日志

```bash
pm2 logs siyuan --lines 100 | grep -i "数据库连接\|database\|connection"
```

正常情况下应该看到:
- 创建连接的日志
- 连接数统计
- 可能的连接清理日志

### 2. 并发测试

在浏览器中:
1. 打开多个标签页
2. 同时访问不同的笔记
3. 快速切换笔记本

如果没有报错,说明连接池处理并发请求正常。

### 3. 检查服务性能

```bash
# 查看服务内存使用
pm2 info siyuan

# 查看系统资源
top -p $(pgrep -f siyuan-kernel)
```

正常情况下:
- 内存使用稳定,不会持续增长
- CPU使用在空闲时应该很低

## 常见问题排查

### 问题1: 登录失败

**可能原因:**
1. 密码错误
2. 用户不存在
3. 服务未正常启动

**解决方法:**
```bash
# 检查用户是否存在
cat /root/code/neu-siyuan-note/data/users/users.json

# 重启服务
pm2 restart siyuan

# 查看错误日志
pm2 logs siyuan --err --lines 50
```

### 问题2: 无法查询笔记本

**可能原因:**
1. 数据库文件不存在
2. 数据库文件损坏
3. 权限问题

**解决方法:**
```bash
# 检查数据库文件
ls -lh /root/code/MindOcean/user-data/notes/jason/conf/siyuan.db

# 检查权限
ls -la /root/code/MindOcean/user-data/notes/jason/conf/

# 如果权限有问题,修复权限
sudo chown -R jason:jason /root/code/MindOcean/user-data/notes/jason/
```

### 问题3: 服务崩溃或重启

**可能原因:**
1. 数据库连接数超限
2. 内存不足
3. 代码错误

**解决方法:**
```bash
# 查看崩溃日志
pm2 logs siyuan --err --lines 100

# 查看系统资源
free -h
df -h

# 重新编译和重启
cd /root/code/neu-siyuan-note/kernel
go build -o siyuan-kernel
pm2 restart siyuan
```

## 成功标志

如果以下所有测试都通过,说明数据库连接池功能正常:

- ✅ 能够正常登录
- ✅ 能够查看笔记本列表
- ✅ 能够打开和编辑笔记
- ✅ 服务日志中有数据库连接相关信息
- ✅ 并发访问时没有错误
- ✅ 内存使用稳定

## 数据库连接池工作原理

当你登录并访问笔记时:

1. **首次访问**: 系统为你的workspace创建一个新的数据库连接
2. **后续访问**: 复用已有的数据库连接,提高性能
3. **空闲清理**: 如果30分钟没有访问,连接会被自动关闭
4. **并发限制**: 最多支持100个并发数据库连接

这个过程是完全自动的,你不需要做任何配置。

## 下一步

如果所有测试都通过,说明数据库层重构成功!

接下来的工作:
- 继续完成搜索索引的重构
- 进行更全面的性能测试
- 测试多用户并发场景
