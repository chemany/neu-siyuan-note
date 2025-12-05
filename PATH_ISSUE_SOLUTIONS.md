# 思源笔记路径问题分析与解决方案

## 问题描述

登录成功后，页面跳转和静态资源加载出现路径问题：

1. **跳转问题**：登录成功后跳转到 `https://www.cheman.top` 而不是 `https://www.cheman.top/notepads/`
2. **静态资源 404**：访问 `https://www.cheman.top/notepads/stage/build/desktop/` 时，资源路径错误：
   - HTML 引用 `/stage/build/desktop/base.css` (缺少 /notepads 前缀)
   - 应该是 `/notepads/stage/build/desktop/base.css`

### 错误日志
```
GET https://www.cheman.top/stage/build/desktop/main.js 404 (Not Found)
GET https://www.cheman.top/manifest.webmanifest 404 (Not Found)
Refused to apply style from '.../base.css' because its MIME type ('text/html') is not a supported stylesheet MIME type
```

## 根本原因

思源笔记在生成 HTML 时使用的是**绝对路径**（如 `/stage/...`），在反向代理的子路径（`/notepads/`）下运行时，这些绝对路径无法正确解析。

### 技术原因

1. 思源笔记不支持 `--base-path` 或类似参数来配置基础路径
2. 生成的 HTML 中使用绝对路径：`<script src="/stage/build/desktop/main.js">`
3. Nginx 的 `proxy_redirect` 只处理 HTTP 重定向头，不处理 HTML 内容
4. `sub_filter` 方案会导致：
   - 性能问题（需要缓冲和处理所有响应）
   - 可能误替换（如 JSON 数据中的路径）
   - 与 `proxy_buffering off` 冲突（WebSocket 需要）

## 推荐解决方案

### 方案 A：思源笔记在根路径（推荐）⭐

将思源笔记部署在根路径 `/`，官网移动到子路径 `/home/` 或 `/www/`。

#### 优点
- ✅ 无路径问题，所有资源正常加载
- ✅ URL 简洁：`https://www.cheman.top/`
- ✅ 性能最佳，无需路径重写
- ✅ WebSocket 工作完美

#### Nginx 配置示例

```nginx
server {
    listen 443 ssl http2;
    server_name www.cheman.top cheman.top;
    
    # SSL 配置...
    
    # 思源笔记 Web API
    location ~* ^/api/web/ {
        proxy_pass http://127.0.0.1:6806;
        # ... 配置
    }
    
    # 统一认证 API
    location ~* ^/api/ {
        proxy_pass http://127.0.0.1:3002;
        # ... 配置
    }
    
    # 潮汐志
    location /calendars/ {
        proxy_pass http://127.0.0.1:11000;
        # ... 配置
    }
    
    # 官网（移到子路径）
    location /home/ {
        alias /home/jason/code/cheman.top/notepads/;
        index index.html;
        try_files $uri $uri/ =404;
    }
    
    # 思源笔记（根路径）
    location / {
        proxy_pass http://127.0.0.1:6806;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        # ... 其他配置
    }
}
```

#### 迁移步骤

```bash
# 1. 备份当前配置
cp /etc/nginx/sites-enabled/nginx-server.conf /root/code/nginx-server.conf.backup

# 2. 修改配置文件
# 将根路径 location / 改为 location /home/
# 将 location /notepads/ 改为 location /

# 3. 测试配置
nginx -t

# 4. 重新加载
systemctl reload nginx

# 5. 测试访问
curl -I https://www.cheman.top/
```

### 方案 B：使用 sub_filter 重写路径（不推荐）

使用 Nginx 的 `sub_filter` 模块在响应中替换路径。

#### 缺点
- ❌ 性能开销大
- ❌ 可能误替换
- ❌ 需要启用 proxy_buffering（与 WebSocket 冲突）
- ❌ 维护复杂

#### 配置示例（仅供参考）

```nginx
location /notepads/ {
    proxy_pass http://127.0.0.1:6806/;
    
    # 需要启用缓冲才能使用 sub_filter
    proxy_buffering on;
    
    # 替换 HTML 中的路径
    sub_filter_types text/html application/javascript text/css;
    sub_filter_once off;
    sub_filter '"/stage/' '"/notepads/stage/';
    sub_filter "'/stage/" "'/notepads/stage/";
    # ... 更多替换规则
}
```

### 方案 C：修改思源笔记源码（技术最佳但工作量大）

修改思源笔记的前端构建配置，支持自定义 `publicPath`。

#### 需要修改
1. `app/webpack.config.js` - 添加 `publicPath` 配置
2. HTML 模板 - 使用相对路径或配置的 base path
3. 构建脚本 - 传入环境变量

#### 缺点
- ❌ 需要深入了解思源笔记代码
- ❌ 每次更新思源笔记需要重新修改
- ❌ 维护成本高

## 当前临时方案

### 建议用户直接访问根路径

如果不想修改 Nginx 配置，可以让用户直接访问：

```
http://localhost:6806/
```

或者通过端口转发访问原始端口（但不推荐暴露到外网）。

### 快速修复（如果必须用 /notepads/）

修改登录成功后的跳转逻辑，让页面知道当前运行在 `/notepads/` 路径下。

检查登录页面代码：

```javascript
// 登录成功后，应该跳转到相对路径
window.location.href = './stage/build/desktop/';  // 相对路径
// 而不是
window.location.href = '/stage/build/desktop/';   // 绝对路径
```

## 最终推荐

### 立即采用方案 A

**原因**：
1. 思源笔记是主要应用，应该在根路径
2. 官网是静态页面，放在子路径没有任何问题
3. 方案简单、性能好、无副作用
4. 符合常见的应用部署最佳实践

### 配置调整建议

```nginx
# 新的路由结构
/                    → 思源笔记 (6806)
/api/web/*           → 思源笔记 API (6806)
/api/auth/*          → 统一认证 (3002)
/api/unified/*       → 统一认证 (3002)
/calendars/*         → 潮汐志 (11000/11001)
/home/*              → 官网静态文件
/downloads/*         → 下载文件
```

### 用户体验

- 思源笔记：`https://www.cheman.top/`
- 潮汐志：`https://www.cheman.top/calendars/`
- 官网：`https://www.cheman.top/home/`

更加直观，思源笔记作为主应用在根路径。

## 实施步骤

我可以帮你：
1. 创建新的 Nginx 配置文件
2. 测试配置
3. 平滑切换（无需停机）
4. 验证所有服务正常工作

是否现在实施方案 A？
