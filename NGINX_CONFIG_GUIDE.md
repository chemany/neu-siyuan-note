# 思源笔记 Nginx 配置指南

## 📋 配置概览

思源笔记已成功集成到现有的 Nginx 配置中，与潮汐志和统一认证服务共存。

## 🌐 访问方式

- **外网访问地址**: `https://www.cheman.top/notepads/`
- **服务端口**: 6806
- **登录页面**: `/stage/login.html`

## 📝 配置文件位置

- **Nginx 配置**: `/etc/nginx/sites-enabled/nginx-server.conf`
- **配置备份**: `/root/code/current_nginx.conf`

## 🔧 核心配置

### 思源笔记路由配置

```nginx
# =======================================================================
# 思源笔记 (Siyuan Notes) - 智能笔记系统
# 端口: 6806
# 路径: /notepads/
# =======================================================================

# /notepads 路径代理到思源笔记（去掉 /notepads 前缀）
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

### 关键配置说明

1. **路径重写**: 
   - 外部访问 `https://www.cheman.top/notepads/xxx`
   - Nginx 代理到 `http://127.0.0.1:6806/xxx`
   - 去掉 `/notepads` 前缀，直接转发到思源服务

2. **WebSocket 支持**:
   ```nginx
   proxy_set_header Upgrade $http_upgrade;
   proxy_set_header Connection "upgrade";
   ```
   支持思源笔记的实时同步功能

3. **禁用缓冲**:
   ```nginx
   proxy_buffering off;
   ```
   确保实时推送和大文件传输的流畅性

4. **重定向处理**:
   ```nginx
   proxy_redirect / /notepads/;
   ```
   自动将服务端的重定向路径加上 `/notepads/` 前缀

## 🔐 统一认证集成

思源笔记的登录功能依赖统一认证服务（3002端口），配置已包含：

```nginx
# 直接 /api 路径 -> 统一设置服务 (兜底路由，用于处理遗留的直接API调用)
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

### 认证流程

1. 用户访问 `https://www.cheman.top/notepads/`
2. 未登录时，思源服务返回 302 重定向到 `/stage/login.html`
3. 登录页面调用 `/api/auth/login` 进行认证
4. Nginx 将认证请求代理到统一认证服务（3002端口）
5. 认证成功后，返回 JWT token
6. 前端使用 token 访问思源笔记的其他功能

## 📦 完整服务架构

```
外网请求 (HTTPS 443)
    ↓
Nginx 反向代理
    ↓
    ├─→ /calendars/       → 潮汐志 (11000, 11001)
    ├─→ /notepads/        → 思源笔记 (6806)
    ├─→ /api/             → 统一认证服务 (3002)
    ├─→ /unified-settings/→ 统一设置 (3002)
    └─→ /                 → 瀚海渊智官网静态文件
```

## 🚀 服务管理

### 查看服务状态

```bash
# 查看所有 PM2 服务
pm2 list

# 查看思源笔记服务状态
pm2 list | grep siyuan

# 查看思源笔记日志
pm2 logs siyuan-kernel
```

### 重启服务

```bash
# 重启思源笔记
pm2 restart siyuan-kernel

# 重新加载 Nginx
systemctl reload nginx

# 使用一键脚本重新构建和重启
/root/code/siyuan/rebuild-and-restart.sh
```

### 测试配置

```bash
# 测试 Nginx 配置语法
nginx -t

# 测试本地服务可访问性
curl -I http://localhost:6806/stage/login.html

# 测试通过 Nginx 访问
curl -I https://www.cheman.top/notepads/
```

## 🔍 故障排查

### 1. 无法访问思源笔记

**检查项**:
```bash
# 1. 检查思源服务是否运行
pm2 list | grep siyuan

# 2. 检查端口是否监听
netstat -tlnp | grep 6806

# 3. 检查 Nginx 配置
nginx -t

# 4. 查看 Nginx 错误日志
tail -f /var/log/nginx/cheman.top-error.log
```

### 2. 登录失败

**检查项**:
```bash
# 1. 检查统一认证服务
pm2 list | grep unified

# 2. 检查认证服务端口
netstat -tlnp | grep 3002

# 3. 查看认证服务日志
pm2 logs unified-settings

# 4. 测试认证 API
curl -X POST https://www.cheman.top/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"password"}'
```

### 3. WebSocket 连接失败

**检查项**:
```bash
# 查看 Nginx WebSocket 相关配置
grep -A 5 "Upgrade" /etc/nginx/sites-enabled/nginx-server.conf

# 查看浏览器控制台 Network 标签中的 WS 连接状态
# 应该看到 /ws 连接成功建立
```

## 📊 监控和日志

### Nginx 访问日志
```bash
tail -f /var/log/nginx/cheman.top-access.log
```

### Nginx 错误日志
```bash
tail -f /var/log/nginx/cheman.top-error.log
```

### 思源笔记日志
```bash
pm2 logs siyuan-kernel --lines 100
```

### 统一认证服务日志
```bash
pm2 logs unified-settings --lines 100
```

## 🔒 安全配置

当前配置已包含：

1. **SSL/TLS 加密**: 使用 Let's Encrypt 证书
2. **HTTP 自动跳转 HTTPS**: 强制使用加密连接
3. **安全头设置**:
   - `X-Frame-Options: SAMEORIGIN`
   - `X-Content-Type-Options: nosniff`
   - `X-XSS-Protection: 1; mode=block`
   - `Referrer-Policy: no-referrer-when-downgrade`

4. **文件上传限制**: `client_max_body_size 100M`
5. **CORS 支持**: 已为 API 路由配置 CORS 头

## 📝 配置变更历史

### 2025-12-03
- ✅ 移除了灵枢笔记（NeuraLink-Notes）的相关配置
- ✅ 将思源笔记集成到 `/notepads/` 路径
- ✅ 确保统一认证服务正确代理
- ✅ 保留了潮汐志和其他现有服务的配置
- ✅ 配置文件简化，仅保留一个配置文件

## 🎯 下一步优化建议

1. **性能优化**:
   - 考虑为静态资源添加缓存策略
   - 启用 gzip 压缩

2. **监控增强**:
   - 配置 Prometheus + Grafana 监控
   - 添加访问统计和性能指标

3. **备份策略**:
   - 定期备份思源笔记数据
   - 配置自动化备份脚本

4. **负载均衡**:
   - 如果访问量增大，考虑增加思源笔记实例
   - 配置 Nginx 负载均衡

## 📞 相关文档

- [思源笔记官方文档](https://github.com/siyuan-note/siyuan)
- [Nginx 官方文档](https://nginx.org/en/docs/)
- [PM2 官方文档](https://pm2.keymetrics.io/)
