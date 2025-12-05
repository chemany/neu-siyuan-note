# 思源笔记 API 路由修复

## 问题描述

用户成功通过统一认证服务登录后，在调用思源笔记的 `/api/web/auth/unified-login` 接口时返回 **404 Not Found**。

### 错误日志
```
login.html:284 [登录] 统一服务登录成功
login.html:287 [登录] 使用统一token登录思源笔记...
login.html:288 POST https://www.cheman.top/api/web/auth/unified-login 404 (Not Found)
```

## 根本原因

Nginx 配置中的路由优先级问题：

1. 用户请求：`https://www.cheman.top/api/web/auth/unified-login`
2. Nginx 的 `location ~* ^/api/` 规则匹配了这个请求（通配符匹配）
3. 请求被代理到**统一认证服务**（3002 端口）
4. 统一认证服务没有 `/api/web/auth/unified-login` 接口
5. 返回 404

**实际应该**：
- `/api/web/*` 应该代理到**思源笔记**（6806 端口）
- `/api/auth/*` 和 `/api/unified/*` 代理到**统一认证服务**（3002 端口）

## 解决方案

在 Nginx 配置中，在通用的 `/api/` 规则之前添加更具体的 `/api/web/` 规则。

### Nginx 配置修改

**文件**: `/etc/nginx/sites-enabled/nginx-server.conf`

**修改内容**：

```nginx
# 思源笔记 Web API (优先级高于通用 /api/)
# 这个必须放在 /api/ 之前，因为需要更具体的匹配
location ~* ^/api/web/ {
    proxy_pass http://127.0.0.1:6806;
    proxy_http_version 1.1;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
    
    # CORS 支持
    add_header Access-Control-Allow-Origin * always;
    add_header Access-Control-Allow-Methods "GET, POST, PUT, DELETE, OPTIONS" always;
    add_header Access-Control-Allow-Headers "Origin, X-Requested-With, Content-Type, Accept, Authorization" always;
}

# 直接 /api 路径 -> 统一设置服务 (兜底路由)
location ~* ^/api/ {
    rewrite ^/api/(.*)$ /api/$1 break;
    proxy_pass http://127.0.0.1:3002;
    # ... 其他配置
}
```

### 配置说明

**Nginx location 匹配优先级**：

在 Nginx 中，当多个 location 规则可能匹配同一个请求时，Nginx 会按照以下优先级选择：

1. **精确匹配** (`location = /path`)
2. **优先前缀匹配** (`location ^~ /path`)
3. **正则匹配** (`location ~* /pattern`)（按配置文件中的顺序）
4. **普通前缀匹配** (`location /path`)

由于我们使用的都是正则匹配（`~*`），所以**配置文件中的顺序很重要**。更具体的规则必须放在前面。

**路由规则**：

```
/api/web/auth/unified-login
    ↓ 匹配 location ~* ^/api/web/
    ↓ 代理到思源笔记 (6806)
    ✅ 正确

/api/auth/login
    ↓ 不匹配 ^/api/web/ (因为路径是 /api/auth/)
    ↓ 匹配 location ~* ^/api/
    ↓ 代理到统一认证服务 (3002)
    ✅ 正确

/api/unified/settings
    ↓ 不匹配 ^/api/web/
    ↓ 匹配 location ~* ^/api/
    ↓ 代理到统一认证服务 (3002)
    ✅ 正确
```

## 部署步骤

```bash
# 1. 修改配置文件
vi /root/code/current_nginx.conf

# 2. 复制到 Nginx 配置目录
cp /root/code/current_nginx.conf /etc/nginx/sites-enabled/nginx-server.conf

# 3. 测试配置语法
nginx -t

# 4. 重新加载 Nginx（无需重启,不中断服务）
systemctl reload nginx

# 5. 验证配置
curl -X POST https://www.cheman.top/api/web/auth/unified-login \
  -H "Content-Type: application/json" \
  -d '{"unified_token":"test"}' -k
```

## 测试验证

### 测试 1: 思源笔记 Web API

```bash
curl -X POST https://www.cheman.top/api/web/auth/unified-login \
  -H "Content-Type: application/json" \
  -d '{"unified_token":"test"}' -k
```

**预期结果**（不再是 404）：
```json
{
  "code": -1,
  "msg": "统一登录失败: failed to verify unified token...",
  "data": null
}
```

这说明请求成功到达了思源笔记服务，token 验证失败是正常的（因为我们用的是测试 token）。

### 测试 2: 统一认证 API

```bash
curl -X POST https://www.cheman.top/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test","password":"test"}' -k
```

**预期结果**：仍然正常代理到统一认证服务

### 测试 3: 完整登录流程

1. 访问：`https://www.cheman.top/notepads/stage/login.html`
2. 输入邮箱密码
3. 打开浏览器开发者工具（F12）→ Network 标签
4. 点击登录

**预期请求流程**：

```
1. POST https://www.cheman.top/api/auth/login
   → 统一认证服务 (3002)
   → 返回 accessToken
   ✅

2. POST https://www.cheman.top/api/web/auth/unified-login
   → 思源笔记 (6806)
   → 验证 unified token 并创建会话
   ✅

3. 登录成功，跳转到笔记界面
   ✅
```

## API 路由总览

| 路径模式 | 目标服务 | 端口 | 说明 |
|---------|---------|------|------|
| `/api/web/*` | 思源笔记 | 6806 | 思源笔记 Web API |
| `/api/auth/*` | 统一认证服务 | 3002 | 登录、注册、令牌刷新 |
| `/api/unified/*` | 统一认证服务 | 3002 | 统一设置相关 API |
| `/api/*` | 统一认证服务 | 3002 | 兜底规则（其他所有 /api/ 请求） |
| `/notepads/*` | 思源笔记 | 6806 | 思源笔记主服务（页面和静态资源） |
| `/calendars/*` | 潮汐志 | 11000/11001 | 潮汐志服务 |

## 完整的请求流程

### 登录流程

```
1. 用户访问 https://www.cheman.top/notepads/
   ↓
2. 思源笔记检查会话（Cookie）
   ↓
3. 未登录 → 302 重定向到 /stage/login.html
   ↓
4. 用户输入邮箱密码，点击登录
   ↓
5. 前端 POST https://www.cheman.top/api/auth/login
   ↓ Nginx 匹配 location ~* ^/api/
   ↓ 代理到统一认证服务 (3002)
   ↓
6. 统一认证服务验证凭据
   ↓ 返回 { accessToken, user }
   ↓
7. 前端 POST https://www.cheman.top/api/web/auth/unified-login
   ↓ Nginx 匹配 location ~* ^/api/web/
   ↓ 代理到思源笔记 (6806)
   ↓
8. 思源笔记验证 unified token
   ↓ 调用统一认证服务验证 token
   ↓ 创建本地用户（如果首次登录）
   ↓ 创建会话，设置 Cookie
   ↓ 返回成功响应
   ↓
9. ✅ 登录成功，前端跳转到笔记主界面
```

## 故障排查

### 问题：仍然返回 404

**检查顺序**：

1. **检查 Nginx 配置是否生效**
   ```bash
   grep -A 5 "location ~\* \^/api/web/" /etc/nginx/sites-enabled/nginx-server.conf
   ```
   应该能看到 `/api/web/` 的配置

2. **检查配置顺序**
   ```bash
   grep -n "location ~\* \^/api" /etc/nginx/sites-enabled/nginx-server.conf
   ```
   `/api/web/` 的行号应该小于 `/api/` 的行号

3. **检查 Nginx 是否重载**
   ```bash
   systemctl status nginx
   ```
   应该看到最近的 reload 记录

### 问题：token 验证失败

**可能原因**：

1. accessToken 格式不正确
2. 统一认证服务无法访问
3. 时钟不同步（JWT 过期）

**检查**：

```bash
# 查看思源笔记日志
pm2 logs siyuan-kernel --lines 50

# 查看统一认证服务日志
pm2 logs unified-settings --lines 50
```

## 总结

通过在 Nginx 配置中添加更具体的 `/api/web/` 路由规则，成功解决了 API 404 问题：

✅ `/api/web/*` → 思源笔记（6806）  
✅ `/api/auth/*` → 统一认证服务（3002）  
✅ `/api/unified/*` → 统一认证服务（3002）  
✅ `/api/*` → 统一认证服务（3002，兜底）

现在用户可以正常完成登录流程了！

## 相关文件

- **Nginx 配置**: `/etc/nginx/sites-enabled/nginx-server.conf`
- **配置备份**: `/root/code/current_nginx.conf`
- **登录页面**: `/root/code/siyuan/kernel/stage/login.html`
- **API 路由**: `/root/code/siyuan/kernel/api/router.go`
