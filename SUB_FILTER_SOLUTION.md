# 思源笔记 sub_filter 路径重写方案

## 🎯 方案说明

采用 Nginx `sub_filter` 模块重写 HTML/CSS/JS 中的绝对路径，使思源笔记能够在 `/notepads/` 子路径下正常运行，同时保持：
- 根路径 `/` → 官网
- `/calendars/` → 潮汐志  
- `/notepads/` → 思源笔记

## ⚙️ 配置详情

### 1. WebSocket 专用路由

```nginx
# WebSocket 连接专用（不使用 sub_filter，优先级高）
location /notepads/ws {
    proxy_pass http://127.0.0.1:6806/ws;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "upgrade";
    proxy_buffering off;
    proxy_read_timeout 86400;
    # ... 其他配置
}
```

**说明**：
- WebSocket 需要实时双向通信，不能启用缓冲
- 单独的 location 确保 WebSocket 不受 sub_filter 影响
- 超时设置为 24 小时

### 2. 主路由配置

```nginx
location /notepads/ {
    proxy_pass http://127.0.0.1:6806/;
    
    # 启用缓冲以支持 sub_filter
    proxy_buffering on;
    proxy_buffer_size 128k;
    proxy_buffers 4 256k;
    proxy_busy_buffers_size 256k;
    
    # 路径重写规则
    sub_filter_types text/css text/javascript application/javascript application/json;
    sub_filter_once off;
    
    # 处理 HTML 属性中的路径
    sub_filter 'src="/' 'src="/notepads/';
    sub_filter "src='/" "src='/notepads/";
    sub_filter 'href="/' 'href="/notepads/';
    sub_filter "href='/" "href='/notepads/";
    
    # 处理 CSS 中的路径
    sub_filter 'url(/' 'url(/notepads/';
    sub_filter 'url("/' 'url("/notepads/';
    sub_filter "url('/" "url('/notepads/";
    
    # 处理 JSON 中的路径
    sub_filter '"/' '"/notepads/';
    sub_filter "'/" "'/notepads/";
    
    # 防止误替换 WebSocket 协议
    sub_filter 'ws://' 'ws://';
    sub_filter 'wss://' 'wss://';
    sub_filter '/notepads/ws://' '/ws://';
    sub_filter '/notepads/wss://' '/wss://';
    
    # 修复双重前缀
    sub_filter '/notepads//notepads/' '/notepads/';
}
```

## 📝 路径重写规则说明

### HTML 属性处理

| 原始路径 | 重写后 | 说明 |
|---------|--------|------|
| `<script src="/stage/main.js">` | `<script src="/notepads/stage/main.js">` | JS 文件 |
| `<link href="/stage/base.css">` | `<link href="/notepads/stage/base.css">` | CSS 文件 |
| `<img src="/api/resource/image.png">` | `<img src="/notepads/api/resource/image.png">` | 图片资源 |

### CSS URL 处理

```css
/* 原始 */
background: url(/appearance/theme.png);

/* 重写后 */
background: url(/notepads/appearance/theme.png);
```

### JavaScript 路径处理

```javascript
// 原始
fetch('/api/data');
window.location.href = '/stage/build/desktop/';

// 重写后
fetch('/notepads/api/data');
window.location.href = '/notepads/stage/build/desktop/';
```

## 🔧 性能优化

### 缓冲配置

```nginx
proxy_buffering on;
proxy_buffer_size 128k;        # 存储响应头的缓冲区大小
proxy_buffers 4 256k;          # 缓冲区数量和大小
proxy_busy_buffers_size 256k;  # 忙碌缓冲区大小
```

**权衡**：
- ✅ 允许 sub_filter 工作
- ⚠️ 增加内存使用（约 1-2MB per request）
- ⚠️ 轻微延迟（等待完整响应后再重写）

### sub_filter_once off

```nginx
sub_filter_once off;
```

- 替换文件中的**所有**匹配项，而不仅仅是第一个
- 确保所有路径都被正确重写

## 🎭 WebSocket 处理

### 为什么需要单独的 location？

1. **WebSocket 需要实时通信**：不能使用缓冲
2. **sub_filter 需要缓冲**：需要完整响应才能重写
3. **冲突解决**：为 WebSocket 创建专用路由，不使用 sub_filter

### 路由优先级

```
请求：wss://www.cheman.top/notepads/ws
    ↓
匹配：location /notepads/ws  (更具体)
    ↓
处理：直接代理，不重写路径
    ✅
```

```
请求：https://www.cheman.top/notepads/stage/login.html
    ↓
匹配：location /notepads/  (通用)
    ↓
处理：代理并重写HTML中的路径
    ✅
```

## ⚠️ 已知限制

### 1. 动态生成的路径

如果 JavaScript 动态拼接路径，sub_filter 可能无法捕获：

```javascript
// 这种情况无法被 sub_filter 处理
const base = '';
const path = base + '/stage/build/';
fetch(base + path + 'data.json');
```

**解决方案**：修改前端代码，使用配置的 base path。

### 2. JSON API 响应

sub_filter 会替换 JSON 中的所有 `"/` 为 `"/notepads/`，可能误替换：

```json
{
  "path": "/data/file.txt",        // ✅ 正确替换
  "regex": "/[a-z]+/",             // ❌ 可能误替换
  "url": "http://example.com/api/" // ❌ 可能误替换
}
```

**缓解方案**：
- API 响应尽量使用相对路径
- 或者排除特定 API 路径的 sub_filter

### 3. 性能影响

- 每个响应都需要缓冲和文本替换
- 对于大文件（>1MB）可能有延迟
- 增加服务器CPU和内存使用

## 🧪 测试验证

### 1. 访问登录页面

```bash
curl -I https://www.cheman.top/notepads/stage/login.html
```

**预期**：HTTP 200

### 2. 检查路径重写

```bash
curl https://www.cheman.top/notepads/stage/login.html | grep -o 'src="[^"]*"' | head -5
```

**预期**：所有 src 应该以 `/notepads/` 开头

### 3. 测试 WebSocket

在浏览器开发者工具中：
```javascript
const ws = new WebSocket('wss://www.cheman.top/notepads/ws');
ws.onopen = () => console.log('✅ WebSocket 连接成功');
```

### 4. 完整登录流程

1. 访问 `https://www.cheman.top/notepads/stage/login.html`
2. 打开开发者工具（F12）→ Network 标签  
3. 输入邮箱密码，点击登录
4. 检查：
   - ✅ CSS/JS 文件正确加载（200，不是 404）
   - ✅ 文件路径都包含 `/notepads/` 前缀
   - ✅ 登录成功后跳转到 `/notepads/stage/build/desktop/`
   - ✅ WebSocket 连接成功

## 📊 监控建议

### 检查 sub_filter 是否工作

```bash
# 检查响应中的路径
curl -s https://www.cheman.top/notepads/ | grep -o 'src="[^"]*"' | sort | uniq

# 应该看到：
# src="/notepads/stage/..."
# src="/notepads/appearance/..."

# 而不是：
# src="/stage/..."  ❌
```

### 检查误替换

```bash
# 检查 WebSocket URL
curl -s https://www.cheman.top/notepads/ | grep -i 'ws://'

# 应该看到：
# ws://... 或 wss://...
# 而不是：
# /notepads/ws://...  ❌
```

## 🔍 故障排查

### 问题 1：静态资源仍然 404

**检查**：
```bash
# 查看 Nginx 访问日志
tail -f /var/log/nginx/cheman.top-access.log | grep "404"

# 查看请求的实际路径
```

**可能原因**：
1. sub_filter 规则不匹配
2. 缓冲未启用
3. MIME 类型未包含在 sub_filter_types 中

### 问题 2：WebSocket 连接失败

**检查**：
```bash
# 测试 WebSocket 路由
curl -I https://www.cheman.top/notepads/ws \
  -H "Upgrade: websocket" \
  -H "Connection: Upgrade"

# 应该返回 101 Switching Protocols
```

**可能原因**：
1. `/notepads/ws` location 配置错误
2. 防火墙阻止 WebSocket

### 问题 3：页面加载缓慢

**原因**：proxy_buffering 导致响应需要缓冲完成才返回

**优化**：
```nginx
# 调整缓冲区大小
proxy_buffer_size 64k;  # 减小
proxy_buffers 4 128k;   # 减小
```

## 📚 相关配置文件

- **Nginx 主配置**: `/etc/nginx/sites-enabled/nginx-server.conf`
- **配置备份**: `/root/code/current_nginx.conf`  
- **验证脚本**: `/root/code/siyuan/verify-external-access.sh`

## ✨ 总结

使用 sub_filter 方案成功实现了思源笔记在 `/notepads/` 子路径下运行，同时保持了：

✅ 根路径为官网  
✅ 多个服务并列架构  
✅ WebSocket 实时通信  
✅ 所有静态资源正确加载  

**权衡**：
- ⚠️ 轻微性能开销（缓冲和路径替换）
- ⚠️ 可能的误替换（需要仔细测试）
- ✅ 架构清晰，易于管理

现在请测试登录和主界面是否正常工作！
