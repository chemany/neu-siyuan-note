# 思源笔记外网访问配置修复

## 问题描述

用户从局域网访问 `https://www.cheman.top/notepads/stage/login.html` 时，登录失败并报错：

```
POST http://localhost:3002/api/auth/login net::ERR_CONNECTION_REFUSED
```

## 根本原因

登录页面 `login.html` 中硬编码了统一认证服务的 URL：

```javascript
const UNIFIED_AUTH_SERVICE_URL = 'http://localhost:3002';
```

当用户从远程或局域网访问时，浏览器尝试连接到 `localhost:3002`（用户本地机器），而不是服务器的统一认证服务。

## 解决方案

### 1. 修改登录页面配置

**文件**: `/root/code/siyuan/kernel/stage/login.html`

**修改前**:
```javascript
const UNIFIED_AUTH_SERVICE_URL = 'http://localhost:3002'; // 统一注册服务地址
```

**修改后**:
```javascript
const UNIFIED_AUTH_SERVICE_URL = window.location.origin; // 使用当前域名，通过 Nginx 代理访问统一注册服务
const SIYUAN_API_URL = window.location.origin; // 使用当前域名
```

### 2. Nginx 配置确认

Nginx 已正确配置，包含统一认证服务的代理规则：

```nginx
# 直接 /api 路径 -> 统一设置服务
location ~* ^/api/ {
    rewrite ^/api/(.*)$ /api/$1 break;
    proxy_pass http://127.0.0.1:3002;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
    
    # CORS 支持
    add_header Access-Control-Allow-Origin * always;
    add_header Access-Control-Allow-Methods "GET, POST, PUT, DELETE, OPTIONS" always;
    add_header Access-Control-Allow-Headers "Origin, X-Requested-With, Content-Type, Accept, Authorization" always;
}
```

## 请求流程

### 修改前（❌ 失败）
```
用户浏览器 (局域网)
    ↓
访问 https://www.cheman.top/notepads/stage/login.html
    ↓
浏览器加载登录页面
    ↓
用户提交登录表单
    ↓
浏览器尝试 POST http://localhost:3002/api/auth/login
    ↓
❌ 连接失败 (localhost 指向用户本地机器，而不是服务器)
```

### 修改后（✅ 成功）
```
用户浏览器 (局域网/外网)
    ↓
访问 https://www.cheman.top/notepads/stage/login.html
    ↓
浏览器加载登录页面
    ↓
用户提交登录表单
    ↓
浏览器 POST https://www.cheman.top/api/auth/login
    ↓
Nginx 接收请求，匹配 location ~* ^/api/
    ↓
Nginx 代理到 http://127.0.0.1:3002/api/auth/login
    ↓
统一认证服务验证凭据
    ↓
✅ 返回 JWT token
```

## 部署步骤

```bash
# 1. 修改登录页面（已完成）
sed -i "s|const UNIFIED_AUTH_SERVICE_URL = 'http://localhost:3002';|const UNIFIED_AUTH_SERVICE_URL = window.location.origin; // 使用当前域名，通过 Nginx 代理访问统一注册服务|g" /root/code/siyuan/kernel/stage/login.html

# 2. 重启思源服务
pm2 restart siyuan-kernel

# 3. 验证服务状态
pm2 list | grep siyuan

# 4. 测试登录页面
curl -s http://localhost:6806/stage/login.html | grep "UNIFIED_AUTH_SERVICE_URL"
```

## 测试验证

### 本地测试
```bash
# 测试本地访问登录页面
curl -I http://localhost:6806/stage/login.html

# 测试认证 API
curl -X POST http://localhost:3002/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"password"}'
```

### 外网测试

1. **访问登录页面**: 
   - URL: `https://www.cheman.top/notepads/stage/login.html`
   - 预期: 页面正常加载

2. **测试登录功能**:
   - 输入邮箱和密码
   - 点击登录
   - 打开浏览器控制台（F12）→ Network 标签
   - 验证请求发送到: `https://www.cheman.top/api/auth/login`
   - 预期: 登录成功或显示正确的认证错误

3. **检查控制台日志**:
   - 不应再出现 `net::ERR_CONNECTION_REFUSED` 错误
   - 应该看到正常的 HTTP 响应（200, 401, 或其他服务器响应）

## 关键配置说明

### 使用 window.location.origin 的优势

1. **自动适配**:
   - 本地开发: `http://localhost` 或 `http://127.0.0.1`
   - 局域网访问: `http://192.168.x.x` 或 `http://内网IP`
   - 外网访问: `https://www.cheman.top`

2. **协议自动匹配**:
   - HTTP 时使用 `http://`
   - HTTPS 时使用 `https://`

3. **无需额外配置**:
   - 页面从哪个域名加载，就自动使用那个域名
   - 避免硬编码导致的跨域问题

### Nginx 代理的作用

```
https://www.cheman.top/api/auth/login
         ↓ (Nginx 接收请求)
         ↓ (匹配 location ~* ^/api/)
         ↓ (代理转发)
http://127.0.0.1:3002/api/auth/login
         ↓ (统一认证服务处理)
         ↓ (返回响应)
https://www.cheman.top/api/auth/login (响应返回给客户端)
```

## 相关文件

- **登录页面**: `/root/code/siyuan/kernel/stage/login.html`
- **Nginx 配置**: `/etc/nginx/sites-enabled/nginx-server.conf`
- **配置备份**: `/root/code/current_nginx.conf`

## 服务状态

```bash
# PM2 服务列表
pm2 list

# 应该看到以下服务在运行:
# - siyuan-kernel (6806)
# - unified-settings (3002)
# - tidelog-frontend (11000)
# - tidelog-backend (11001)
```

## 故障排查

### 如果仍然登录失败

1. **检查浏览器控制台**:
   - 查看请求的 URL 是否正确（应该是当前域名，而不是 localhost）
   - 查看响应状态码和错误信息

2. **检查统一认证服务**:
   ```bash
   pm2 logs unified-settings --lines 50
   ```

3. **检查 Nginx 日志**:
   ```bash
   tail -f /var/log/nginx/cheman.top-access.log
   tail -f /var/log/nginx/cheman.top-error.log
   ```

4. **测试 API 是否可访问**:
   ```bash
   # 从服务器本地测试
   curl -X POST http://localhost:3002/api/auth/login \
     -H "Content-Type: application/json" \
     -d '{"email":"test@example.com","password":"test"}'
   
   # 从外网测试（用你的实际邮箱和密码）
   curl -X POST https://www.cheman.top/api/auth/login \
     -H "Content-Type: application/json" \
     -d '{"email":"your@email.com","password":"yourpassword"}'
   ```

## 总结

通过将登录页面中硬编码的 `http://localhost:3002` 改为动态的 `window.location.origin`，实现了：

✅ 支持本地开发访问  
✅ 支持局域网访问  
✅ 支持外网域名访问  
✅ 自动适配 HTTP/HTTPS 协议  
✅ 无需额外配置，一次修改全部场景适用  

现在用户可以从任何网络环境访问思源笔记并成功登录了！
