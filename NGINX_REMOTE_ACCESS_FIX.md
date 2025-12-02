# Nginx 远程访问配置 - 解决登录失败问题

## 问题描述

远程访问思源笔记时，登录页面无法正常工作，浏览器控制台显示错误：

```
POST http://localhost:3002/api/auth/login net::ERR_CONNECTION_REFUSED
```

## 问题分析

### 根本原因

1. **前端硬编码了 localhost URL**：登录和注册页面中硬编码了 `http://localhost:3002` 和 `http://localhost:6806`
2. **远程访问时无法连接 localhost**：远程浏览器无法访问服务器的 localhost 地址
3. **缺少 Nginx 代理配置**：统一认证服务（3002 端口）没有通过 Nginx 代理暴露

### 架构说明

```
远程浏览器
    ↓
Nginx (80端口)
    ↓
    ├─→ 思源笔记服务 (6806端口)
    └─→ 统一认证服务 (3002端口)
```

## 解决方案

### 1. 配置 Nginx 反向代理

**文件位置**：`/etc/nginx/sites-available/siyuan-nginx.conf`

**配置内容**：

```nginx
server {
    listen 80 default_server;
    listen [::]:80 default_server;

    server_name _;

    # 增加客户端请求体大小限制（用于文件上传）
    client_max_body_size 100M;

    # 代理思源笔记主服务 (6806)
    location / {
        proxy_pass http://127.0.0.1:6806;
        proxy_http_version 1.1;
        
        # WebSocket 支持
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        
        # 传递真实客户端信息
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        
        # 超时设置
        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
        
        # 禁用缓冲以支持实时推送
        proxy_buffering off;
    }

    # 代理统一认证服务 (3002)
    location /api/auth/ {
        proxy_pass http://127.0.0.1:3002/api/auth/;
        proxy_http_version 1.1;
        
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        
        proxy_connect_timeout 30s;
        proxy_send_timeout 30s;
        proxy_read_timeout 30s;
    }

    # 代理统一认证服务的其他 API
    location /api/unified/ {
        proxy_pass http://127.0.0.1:3002/api/unified/;
        proxy_http_version 1.1;
        
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        
        proxy_connect_timeout 30s;
        proxy_send_timeout 30s;
        proxy_read_timeout 30s;
    }

    # 日志配置
    access_log /var/log/nginx/siyuan_access.log;
    error_log /var/log/nginx/siyuan_error.log;
}
```

**配置说明**：

1. **主服务代理（location /）**：
   - 代理所有请求到思源笔记服务（6806端口）
   - 支持 WebSocket 连接（用于实时推送）
   - 设置合理的超时时间

2. **认证服务代理（location /api/auth/）**：
   - 代理认证相关的 API 到统一认证服务（3002端口）
   - 包括登录、注册、令牌刷新等

3. **统一服务代理（location /api/unified/）**：
   - 代理其他统一服务的 API

### 2. 修改前端 URL 配置

#### 登录页面（login.html）

**修改前**：
```javascript
const UNIFIED_AUTH_SERVICE_URL = 'http://localhost:3002';
const SIYUAN_API_URL = 'http://localhost:6806';
```

**修改后**：
```javascript
// 配置 - 使用当前域名和协议，通过 Nginx 代理访问
const UNIFIED_AUTH_SERVICE_URL = window.location.origin; // 使用当前域名
const SIYUAN_API_URL = window.location.origin; // 使用当前域名
```

#### 注册页面（register.html）

**修改前**：
```javascript
const UNIFIED_AUTH_SERVICE_URL = 'http://localhost:3002';
```

**修改后**：
```javascript
// 配置 - 使用当前域名和协议，通过 Nginx 代理访问
const UNIFIED_AUTH_SERVICE_URL = window.location.origin;
```

**修改说明**：
- 使用 `window.location.origin` 动态获取当前访问的域名和协议
- 本地访问时：`http://localhost` 或 `http://127.0.0.1`
- 远程访问时：`http://your-server-ip` 或 `http://your-domain.com`
- 支持 HTTPS：如果配置了 SSL，会自动使用 `https://`

### 3. 部署步骤

```bash
# 1. 复制 Nginx 配置
sudo cp siyuan/siyuan-nginx.conf /etc/nginx/sites-available/siyuan-nginx.conf

# 2. 启用新配置
sudo ln -sf /etc/nginx/sites-available/siyuan-nginx.conf /etc/nginx/sites-enabled/siyuan-nginx.conf

# 3. 删除旧配置
sudo rm -f /etc/nginx/sites-enabled/simple-nginx.conf

# 4. 测试 Nginx 配置
sudo nginx -t

# 5. 重新加载 Nginx
sudo systemctl reload nginx

# 6. 检查 Nginx 状态
sudo systemctl status nginx
```

## 测试验证

### 1. 本地访问测试

```bash
# 访问主页
curl http://localhost/

# 测试认证 API
curl -X POST http://localhost/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"password"}'
```

### 2. 远程访问测试

1. **访问主页**：
   - 浏览器打开：`http://your-server-ip/`
   - 应该自动跳转到登录页面

2. **测试登录**：
   - 输入邮箱和密码
   - 点击登录按钮
   - 检查浏览器控制台，应该看到请求发送到当前域名而不是 localhost

3. **检查 WebSocket 连接**：
   - 登录成功后，打开浏览器开发者工具的 Network 标签
   - 筛选 WS（WebSocket）连接
   - 应该看到 `/ws` 连接成功建立

### 3. 日志检查

```bash
# 查看 Nginx 访问日志
sudo tail -f /var/log/nginx/siyuan_access.log

# 查看 Nginx 错误日志
sudo tail -f /var/log/nginx/siyuan_error.log

# 查看思源笔记日志
pm2 logs siyuan-kernel

# 查看统一认证服务日志
pm2 logs unified-settings
```

## 请求流程

### 登录流程

```
1. 用户访问 http://your-server-ip/
   ↓
2. Nginx 代理到思源笔记服务 (6806)
   ↓
3. 思源笔记返回登录页面
   ↓
4. 用户输入邮箱密码，点击登录
   ↓
5. 前端发送请求到 http://your-server-ip/api/auth/login
   ↓
6. Nginx 代理到统一认证服务 (3002)
   ↓
7. 统一认证服务验证用户凭据
   ↓
8. 返回 JWT token
   ↓
9. 前端使用 token 调用思源笔记 API
   ↓
10. 登录成功，进入主界面
```

### WebSocket 连接流程

```
1. 前端建立 WebSocket 连接到 ws://your-server-ip/ws
   ↓
2. Nginx 升级连接为 WebSocket
   ↓
3. 代理到思源笔记服务的 WebSocket 端点
   ↓
4. 思源笔记验证 JWT token（从 Cookie 中读取）
   ↓
5. 连接建立成功
   ↓
6. 后端可以实时推送事件到前端
```

## 关键配置说明

### Nginx 配置要点

1. **WebSocket 支持**：
   ```nginx
   proxy_set_header Upgrade $http_upgrade;
   proxy_set_header Connection "upgrade";
   ```

2. **禁用缓冲**：
   ```nginx
   proxy_buffering off;
   ```
   这对于实时推送很重要

3. **文件上传大小限制**：
   ```nginx
   client_max_body_size 100M;
   ```

4. **超时设置**：
   ```nginx
   proxy_connect_timeout 60s;
   proxy_send_timeout 60s;
   proxy_read_timeout 60s;
   ```

### 前端配置要点

1. **动态 URL**：使用 `window.location.origin` 而不是硬编码
2. **相对路径**：API 请求使用相对路径，由 Nginx 代理
3. **Cookie 共享**：所有服务在同一域名下，Cookie 可以共享

## 安全建议

### 1. 启用 HTTPS

```nginx
server {
    listen 443 ssl http2;
    listen [::]:443 ssl http2;
    
    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;
    
    # ... 其他配置
}

# HTTP 重定向到 HTTPS
server {
    listen 80;
    listen [::]:80;
    return 301 https://$host$request_uri;
}
```

### 2. 限制访问速率

```nginx
# 在 http 块中定义
limit_req_zone $binary_remote_addr zone=login:10m rate=5r/m;

# 在 location 块中使用
location /api/auth/login {
    limit_req zone=login burst=3;
    # ... 其他配置
}
```

### 3. 添加安全头

```nginx
add_header X-Frame-Options "SAMEORIGIN" always;
add_header X-Content-Type-Options "nosniff" always;
add_header X-XSS-Protection "1; mode=block" always;
```

## 故障排查

### 问题 1：登录仍然失败

**检查**：
```bash
# 检查统一认证服务是否运行
pm2 list | grep unified

# 检查端口是否监听
netstat -tlnp | grep 3002

# 测试直接访问
curl http://localhost:3002/api/auth/login
```

### 问题 2：WebSocket 连接失败

**检查**：
```bash
# 查看 Nginx 错误日志
sudo tail -f /var/log/nginx/siyuan_error.log

# 检查 WebSocket 升级配置
sudo nginx -T | grep -A 5 "Upgrade"
```

### 问题 3：文件上传失败

**检查**：
```bash
# 检查文件大小限制
sudo nginx -T | grep client_max_body_size

# 增加限制
# 在 server 块中添加：
client_max_body_size 100M;
```

## 总结

通过以下修改成功实现了远程访问：

1. ✅ 配置 Nginx 反向代理，统一入口
2. ✅ 修改前端 URL 配置，使用动态域名
3. ✅ 支持 WebSocket 连接
4. ✅ 支持文件上传
5. ✅ 保持本地访问兼容性

现在用户可以通过远程 IP 或域名访问思源笔记，登录、注册、文件上传、实时推送等功能都能正常工作。

## 相关文件

- Nginx 配置：`/etc/nginx/sites-available/siyuan-nginx.conf`
- 登录页面：`siyuan/kernel/stage/login.html`
- 注册页面：`siyuan/kernel/stage/register.html`
- 配置模板：`siyuan/siyuan-nginx.conf`
