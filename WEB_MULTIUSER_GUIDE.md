# 思源笔记Web多用户系统 - 完整部署和测试指南

## 🎉 功能概述

思源笔记现已支持完整的Web多用户系统,包括:

- ✅ 用户注册和登录
- ✅ JWT Token认证
- ✅ 用户数据隔离
- ✅ 统一注册服务集成
- ✅ 独立workspace管理
- ✅ Web模式强制认证

## 🚀 部署步骤

### 1. 启动统一注册服务 (端口3002)

```bash
cd /home/jason/code/unified-settings-service
npm install
npm start
```

### 2. 启动思源笔记Web服务 (端口6806)

```bash
cd /home/jason/code/siyuan/kernel

# 编译 (包含FTS5支持)
/usr/local/go/bin/go build -tags "fts5" -o siyuan-kernel main.go

# 启动 (Web模式)
export SIYUAN_WORKSPACE_PATH="/home/jason/code/siyuan/workspace"
export SIYUAN_WEB_MODE=true
export SIYUAN_JWT_SECRET="your-super-secret-jwt-key-change-in-production"
export UNIFIED_AUTH_SERVICE_URL="http://localhost:3002"

./siyuan-kernel --port 6806
```

## 📋 服务端口说明

| 服务 | 端口 | 地址 | 说明 |
|------|------|------|------|
| 统一注册服务 | 3002 | http://localhost:3002 | 用户注册/登录/认证 |
| 思源笔记后端 | 6806 | http://localhost:6806 | 思源笔记API和UI |

## 🔐 用户使用流程

### 1. 注册新用户

访问: http://localhost:6806/stage/register.html

填写信息:
- 用户名: 3-20个字符,仅支持字母、数字、下划线
- 邮箱: 有效的邮箱地址
- 密码: 至少6个字符,包含字母和数字

### 2. 登录

访问: http://localhost:6806/stage/login.html

或直接访问: http://localhost:6806/ (未登录会自动重定向)

登录流程:
1. 输入邮箱和密码
2. 系统调用统一注册服务验证
3. 获取统一服务token
4. 使用统一token登录思源笔记
5. 创建用户专属workspace
6. 保存思源token到Cookie和LocalStorage
7. 跳转到主应用

### 3. 使用思源笔记

登录成功后,可以正常使用思源笔记的所有功能。

每个用户的数据独立存储在:
```
/home/jason/code/siyuan/workspace/temp/siyuan-workspaces/{username}/
```

## 🧪 测试验证

### 1. 测试用户注册

```bash
curl -X POST http://localhost:3002/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser1",
    "email": "test1@example.com",
    "password": "password123"
  }'
```

### 2. 测试统一服务登录

```bash
curl -X POST http://localhost:3002/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test1@example.com",
    "password": "password123"
  }'
```

### 3. 测试思源笔记统一登录

```bash
# 先获取统一服务token
UNIFIED_TOKEN=$(curl -s -X POST http://localhost:3002/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test1@example.com","password":"password123"}' \
  | jq -r '.accessToken')

# 使用统一token登录思源笔记
curl -X POST http://localhost:6806/api/web/auth/unified-login \
  -H "Content-Type: application/json" \
  -d "{\"unified_token\": \"$UNIFIED_TOKEN\"}"
```

### 4. 测试JWT Token验证

```bash
# 获取思源token
SIYUAN_TOKEN=$(c...保存的token...)

# 测试受保护的API
curl -X POST http://localhost:6806/api/web/auth/profile \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $SIYUAN_TOKEN"
```

### 5. 测试多用户隔离

```bash
# 注册两个用户
curl -X POST http://localhost:3002/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username":"user1","email":"user1@test.com","password":"pass123"}'

curl -X POST http://localhost:3002/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username":"user2","email":"user2@test.com","password":"pass456"}'

# 检查workspace目录
ls -la /home/jason/code/siyuan/workspace/temp/siyuan-workspaces/
# 应该看到: user1/ 和 user2/ 两个独立目录
```

## 🔒 安全配置

### JWT密钥设置

```bash
# 生成随机密钥
node -e "console.log(require('crypto').randomBytes(64).toString('hex'))"

# 设置环境变量
export SIYUAN_JWT_SECRET="your-generated-secret-key"
```

### 密码安全

- 用户密码使用bcrypt加密存储
- JWT Token 24小时有效期
- 支持Token刷新机制
- 支持Token黑名单

## 📊 API端点列表

### 公开端点 (无需认证)

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/web/auth/register | 用户注册 |
| POST | /api/web/auth/login | 用户登录 |
| POST | /api/web/auth/unified-login | 统一服务登录 |
| GET | /api/web/auth/unified-status | 统一服务状态 |
| GET | /api/web/auth/health | 健康检查 |
| POST | /api/web/auth/verify-token | 验证Token |
| GET | /stage/login.html | 登录页面 |
| GET | /stage/register.html | 注册页面 |

### 受保护端点 (需要JWT Token)

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/web/auth/profile | 获取用户信息 |
| POST | /api/web/auth/update-profile | 更新用户信息 |
| POST | /api/web/auth/change-password | 修改密码 |
| POST | /api/web/auth/logout | 登出 |
| POST | /api/web/auth/refresh-token | 刷新Token |
| GET | / | 主页 (所有笔记相关功能) |

## 🐛 故障排除

### 1. 登录后显示404

**原因**: 前端资源未构建

**解决方案**:
```bash
cd /home/jason/code/siyuan/app
npm run build:desktop
npm run build:mobile
cp -r stage/build ../kernel/stage/
```

### 2. 统一登录失败

**原因**: 统一注册服务未启动或端口错误

**检查**:
```bash
curl http://localhost:3002/health
```

**解决方案**:
```bash
cd /home/jason/code/unified-settings-service
npm start
```

### 3. Token验证失败

**原因**: JWT_SECRET不匹配

**检查**:
```bash
echo $SIYUAN_JWT_SECRET
```

**解决方案**: 确保环境变量已设置

### 4. Workspace权限问题

**检查**:
```bash
ls -la /home/jason/code/siyuan/workspace/temp/siyuan-workspaces/
```

**解决方案**:
```bash
chmod -R 755 /home/jason/code/siyuan/workspace/temp/siyuan-workspaces/
```

## 📝 注意事项

### 当前限制

1. **Workspace动态切换未完全实现**: 
   - 用户workspace已创建
   - Token中包含workspace路径
   - 但API调用尚未完全切换到用户workspace
   - 需要在后续版本中实现

2. **WebSocket认证增强**:
   - 基本的HTTP认证已实现
   - WebSocket连接的JWT认证需要进一步完善

3. **用户数据迁移**:
   - 首次启动Web模式时,原有数据仍在默认workspace
   - 新注册用户会获得独立workspace

### 未来改进

1. 实现完整的workspace动态切换
2. 增强WebSocket JWT认证
3. 添加用户配额管理
4. 实现数据导入导出功能
5. 添加管理员控制面板

## 🎯 下一步建议

1. **测试完整流程**: 从注册到登录再到使用笔记
2. **配置生产环境**: 设置强密码和密钥
3. **配置HTTPS**: 使用Nginx反向代理
4. **设置防火墙**: 限制端口访问
5. **配置备份**: 定期备份用户数据

##  联系支持

如有问题,请检查:
1. 服务日志: 查看kernel输出
2. 浏览器控制台: 检查前端错误
3. 网络请求: 使用开发者工具查看API调用

---

📅 最后更新: 2025-11-25
📝 版本: v1.0.0
🚀 思源笔记Web多用户系统
