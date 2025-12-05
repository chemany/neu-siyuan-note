# 思源笔记外网访问配置完成总结

## ✅ 已完成的配置

### 1. Nginx 配置

**配置文件**: `/etc/nginx/sites-enabled/nginx-server.conf`

#### 思源笔记路由
```nginx
location = /notepads {
    return 301 $scheme://$host/notepads/;
}

location /notepads/ {
    proxy_pass http://127.0.0.1:6806/;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "upgrade";
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_buffering off;
    proxy_redirect / /notepads/;
}
```

#### 统一认证服务路由
```nginx
location ~* ^/api/ {
    rewrite ^/api/(.*)$ /api/$1 break;
    proxy_pass http://127.0.0.1:3002;
    # ... CORS 和其他配置
}
```

### 2. 登录页面配置

**文件**: `/root/code/siyuan/kernel/stage/login.html`

**关键修改**:
```javascript
// 修改前：硬编码 localhost
const UNIFIED_AUTH_SERVICE_URL = 'http://localhost:3002';

// 修改后：动态使用当前域名
const UNIFIED_AUTH_SERVICE_URL = window.location.origin;
const SIYUAN_API_URL = window.location.origin;
```

### 3. 服务状态

所有服务通过 PM2 管理：

```
┌────┬────────────────────┬──────────┬───────────┐
│ id │ name               │ mode     │ status    │
├────┼────────────────────┼──────────┼───────────┤
│ 0  │ tidelog-frontend   │ fork     │ online    │
│ 1  │ tidelog-backend    │ fork     │ online    │
│ 2  │ unified-settings   │ fork     │ online    │
│ 3  │ siyuan-kernel      │ fork     │ online    │
└────┴────────────────────┴──────────┴───────────┘
```

## 🌐 访问地址

### 外网访问
- **思源笔记主页**: `https://www.cheman.top/notepads/`
- **登录页面**: `https://www.cheman.top/notepads/stage/login.html`
- **注册页面**: `https://www.cheman.top/notepads/stage/register.html`

### 其他服务
- **潮汐志**: `https://www.cheman.top/calendars/`
- **官网**: `https://www.cheman.top/`

## 🧪 测试验证步骤

### 步骤 1：访问登录页面

打开浏览器访问：
```
https://www.cheman.top/notepads/stage/login.html
```

**预期结果**:
- ✅ 页面正常加载
- ✅ 显示登录表单

### 步骤 2：测试登录功能

1. 输入你的邮箱和密码
2. 打开浏览器开发者工具（F12）
3. 切换到 **Network（网络）** 标签
4. 点击 **登录** 按钮

**预期结果**:
- ✅ 看到 POST 请求发送到 `https://www.cheman.top/api/auth/login`
- ✅ NOT `http://localhost:3002/api/auth/login`
- ✅ 收到服务器响应（200 成功 或 401 认证失败等）
- ❌ 不应出现 `net::ERR_CONNECTION_REFUSED`

### 步骤 3：检查控制台

在开发者工具的 **Console（控制台）** 标签中：

**预期结果**:
```
[登录] 调用统一注册服务登录...
```

**不应出现**:
```
POST http://localhost:3002/api/auth/login net::ERR_CONNECTION_REFUSED
```

### 步骤 4：成功登录后

如果使用正确的凭据登录，应该：
- ✅ 自动跳转到思源笔记主界面
- ✅ 可以看到笔记内容
- ✅ WebSocket 连接成功（实时同步功能）

## 📋 完整的服务架构

```
用户浏览器（任何网络）
    ↓
HTTPS (443) - www.cheman.top
    ↓
Nginx 反向代理
    ↓
    ├─→ /calendars/        → 潮汐志服务 (11000, 11001)
    ├─→ /notepads/         → 思源笔记 (6806)
    │   ├─→ /notepads/stage/login.html
    │   ├─→ /notepads/stage/register.html
    │   └─→ /notepads/ws (WebSocket)
    │
    ├─→ /api/auth/         → 统一认证服务 (3002)
    ├─→ /api/unified/      → 统一设置服务 (3002)
    └─→ /                  → 官网静态文件
```

## 🔄 认证流程

```
1. 访问 https://www.cheman.top/notepads/
   ↓
2. 思源检查 JWT token (Cookie)
   ↓
3. 未登录 → 302 重定向到 /stage/login.html
   ↓
4. 用户输入邮箱密码，点击登录
   ↓
5. 前端 POST https://www.cheman.top/api/auth/login
   ↓
6. Nginx 代理到统一认证服务 (3002)
   ↓
7. 统一认证服务验证凭据
   ↓
8. 返回 JWT token (设置在 Cookie 中)
   ↓
9. 前端使用 token 访问思源 API
   ↓
10. 思源验证 token
   ↓
11. ✅ 登录成功，显示笔记界面
```

## 🛠️ 常用管理命令

### 服务管理
```bash
# 查看所有服务状态
pm2 list

# 查看思源日志
pm2 logs siyuan-kernel

# 重启思源服务
pm2 restart siyuan-kernel

# 重启所有服务
pm2 restart all

# 一键重新构建和重启思源
/root/code/siyuan/rebuild-and-restart.sh
```

### Nginx 管理
```bash
# 测试配置
nginx -t

# 重新加载配置（不中断服务）
systemctl reload nginx

# 重启 Nginx
systemctl restart nginx

# 查看 Nginx 状态
systemctl status nginx

# 查看访问日志
tail -f /var/log/nginx/cheman.top-access.log

# 查看错误日志
tail -f /var/log/nginx/cheman.top-error.log
```

### 端口检查
```bash
# 检查所有服务端口
netstat -tlnp | grep -E '6806|3002|11000|11001'

# 或使用 ss
ss -tlnp | grep -E '6806|3002|11000|11001'
```

## 🐛 故障排查

### 问题 1：仍然出现 localhost:3002 错误

**原因**: 浏览器缓存了旧的登录页面

**解决**:
```
1. 硬刷新页面: Ctrl + Shift + R (Windows/Linux) 或 Cmd + Shift + R (Mac)
2. 或清除浏览器缓存
3. 或使用无痕模式访问
```

### 问题 2：登录后显示 401 Unauthorized

**原因**: 凭据错误或用户不存在

**检查**:
```bash
# 查看统一认证服务日志
pm2 logs unified-settings --lines 50

# 验证用户是否存在于数据库
# (需要根据你的数据库类型检查)
```

### 问题 3：WebSocket 连接失败

**检查**:
```bash
# 查看 Nginx 配置是否包含 WebSocket 支持
grep -A 3 "Upgrade" /etc/nginx/sites-enabled/nginx-server.conf

# 应该看到：
# proxy_set_header Upgrade $http_upgrade;
# proxy_set_header Connection "upgrade";
```

### 问题 4：404 Not Found

**可能原因**:
1. 思源服务未运行
2. Nginx 配置未生效
3. 路径错误

**检查**:
```bash
# 1. 检查思源服务
pm2 list | grep siyuan
curl -I http://localhost:6806/stage/login.html

# 2. 检查 Nginx 配置
nginx -t
systemctl status nginx

# 3. 测试完整路径
curl -I https://www.cheman.top/notepads/stage/login.html
```

## 📊 监控建议

### 实时监控所有服务
```bash
# 使用 PM2 Monit
pm2 monit

# 或使用 htop
htop
```

### 查看综合日志
```bash
# 在一个终端中同时查看多个日志
pm2 logs --lines 50
```

### 定期检查
```bash
# 创建健康检查脚本
cat > /root/code/health-check.sh << 'EOF'
#!/bin/bash
echo "=== 服务状态 ==="
pm2 list

echo -e "\n=== 端口监听 ==="
ss -tlnp | grep -E '6806|3002|11000|11001'

echo -e "\n=== Nginx 状态 ==="
systemctl status nginx --no-pager -l

echo -e "\n=== 测试思源访问 ==="
curl -I http://localhost:6806/stage/login.html 2>&1 | head -5

echo -e "\n=== 测试外网访问 ==="
curl -I https://www.cheman.top/notepads/ 2>&1 | head -5
EOF

chmod +x /root/code/health-check.sh
```

## 📝 配置文件清单

| 文件 | 路径 | 说明 |
|------|------|------|
| Nginx 配置 | `/etc/nginx/sites-enabled/nginx-server.conf` | 主配置文件 |
| 配置备份 | `/root/code/current_nginx.conf` | 当前配置备份 |
| 登录页面 | `/root/code/siyuan/kernel/stage/login.html` | 修改了认证 URL |
| 重启脚本 | `/root/code/siyuan/rebuild-and-restart.sh` | 一键重建重启 |
| 配置指南 | `/root/code/siyuan/NGINX_CONFIG_GUIDE.md` | Nginx 配置说明 |
| 修复文档 | `/root/code/siyuan/EXTERNAL_ACCESS_FIX.md` | 外网访问修复记录 |

## 🎯 下一步建议

### 1. 测试完整流程
- [ ] 从外网访问登录页面
- [ ] 测试登录功能
- [ ] 测试注册功能
- [ ] 测试笔记编辑
- [ ] 测试文件上传
- [ ] 测试 WebSocket 实时同步

### 2. 性能优化（可选）
- [ ] 配置静态资源缓存
- [ ] 启用 Gzip 压缩
- [ ] 配置访问日志轮转

### 3. 备份策略
- [ ] 定期备份思源笔记数据
- [ ] 备份 Nginx 配置
- [ ] 备份数据库

### 4. 安全加固（可选）
- [ ] 配置 fail2ban 防止暴力破解
- [ ] 添加访问速率限制
- [ ] 定期更新 SSL 证书

## ✨ 总结

配置已完成，现在可以通过以下方式访问思源笔记：

1. ✅ **外网访问**: `https://www.cheman.top/notepads/`
2. ✅ **局域网访问**: `https://www.cheman.top/notepads/`
3. ✅ **本地访问**: `http://localhost:6806/`

所有登录和 API 请求都会自动使用正确的域名，无需额外配置！

**请现在测试外网访问，如果遇到任何问题，请查看上面的故障排查部分或查看日志。**
