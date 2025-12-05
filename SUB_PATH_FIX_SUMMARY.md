# 思源笔记子路径部署修复总结

## ✅ 已解决的问题

### 1. 登录后跳转错误
**问题**：登录成功后跳转到 `https://www.cheman.top/`（官网），而不是 `/notepads/`。
**原因**：`login.html` 中硬编码了 `window.location.href = '/';`。
**修复**：修改为动态路径跳转：
```javascript
window.location.href = window.location.pathname.split("/stage/")[0] + "/";
```
这样在 `/notepads/stage/login.html` 登录时，会自动跳转到 `/notepads/`。

### 2. 静态资源 404
**问题**：访问 `/notepads/` 时，HTML 引用的资源路径错误（如 `/stage/build/desktop/main.js`）。
**原因**：思源笔记生成绝对路径，不包含 `/notepads` 前缀。
**修复**：使用 Nginx `sub_filter` 模块重写响应内容。

### 3. sub_filter 不生效
**问题**：配置了 sub_filter 但页面源代码未改变。
**原因**：后端返回了 gzip 压缩的内容，Nginx 无法替换。
**修复**：添加 `proxy_set_header Accept-Encoding "";` 禁用后端压缩。

## ⚙️ 最终配置

### Nginx 配置 (`/etc/nginx/sites-enabled/nginx-server.conf`)

```nginx
# WebSocket 专用（无缓冲，无 sub_filter）
location /notepads/ws {
    proxy_pass http://127.0.0.1:6806/ws;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "upgrade";
    proxy_buffering off;
}

# 主应用（启用缓冲和 sub_filter）
location /notepads/ {
    proxy_pass http://127.0.0.1:6806/;
    
    # 禁用压缩，确保 sub_filter 生效
    proxy_set_header Accept-Encoding "";
    
    # 启用缓冲
    proxy_buffering on;
    
    # 路径重写规则
    sub_filter_types text/css text/javascript application/javascript application/json;
    sub_filter_once off;
    sub_filter 'src="/' 'src="/notepads/';
    sub_filter 'href="/' 'href="/notepads/';
    # ... 其他规则
}
```

### 登录页面 (`/root/code/siyuan/kernel/stage/login.html`)

```javascript
// 动态跳转逻辑
window.location.href = window.location.pathname.split("/stage/")[0] + "/";
```

## 🧪 验证步骤

1. **清除浏览器缓存**（非常重要，因为之前的 301 重定向可能被缓存）
2. 访问 `https://www.cheman.top/notepads/stage/login.html`
3. 登录
4. 验证跳转到 `https://www.cheman.top/notepads/`
5. 验证页面正常加载，无 404 错误

## ⚠️ 注意事项

如果仍然遇到问题，请检查：
1. **浏览器缓存**：尝试使用无痕模式
2. **Nginx 日志**：`tail -f /var/log/nginx/cheman.top-error.log`
3. **页面源码**：查看页面源码，确认 `src="/notepads/..."` 是否已替换成功
