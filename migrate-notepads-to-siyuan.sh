#!/bin/bash

# 将 /notepads 路径迁移到思源笔记服务
# 同时将根路径 / 也指向思源笔记

set -e

NGINX_CONF="/etc/nginx/sites-available/nginx-server.conf"
BACKUP_CONF="/tmp/nginx-server-backup-$(date +%Y%m%d-%H%M%S).conf"

echo "📋 备份当前配置..."
cp "$NGINX_CONF" "$BACKUP_CONF"
echo "✅ 已备份到: $BACKUP_CONF"

echo ""
echo "🔧 修改 Nginx 配置..."

# 创建临时文件
TEMP_CONF=$(mktemp)

# 读取配置文件，进行修改
cat "$NGINX_CONF" | awk '
BEGIN {
    in_notepads_api = 0
    in_notepads_exact = 0
    in_notepads_next = 0
    in_notepads_location = 0
    in_api_location = 0
    in_root_location = 0
    skip_lines = 0
    siyuan_added = 0
}

# 跳过旧的 notepads 配置
/location ~ \^\/notepads\/\[Aa\]\[Pp\]\[Ii\]\// { in_notepads_api = 1; skip_lines = 1; next }
/location = \/notepads/ { in_notepads_exact = 1; skip_lines = 1; next }
/location ~\* \^\/notepads\/_next\// { in_notepads_next = 1; skip_lines = 1; next }
/location \/notepads\// { in_notepads_location = 1; skip_lines = 1; next }

# 检测配置块结束
/^    \}/ {
    if (in_notepads_api || in_notepads_exact || in_notepads_next || in_notepads_location) {
        in_notepads_api = 0
        in_notepads_exact = 0
        in_notepads_next = 0
        in_notepads_location = 0
        skip_lines = 0
        next
    }
}

# 在 location ~* ^/api/ 之前插入思源笔记配置
/location ~\* \^\/api\// {
    if (!siyuan_added) {
        print "    # ======================================================================="
        print "    # 思源笔记 (SiYuan) - 个人知识管理系统"
        print "    # 主服务: 6806"
        print "    # 路径: / 和 /notepads"
        print "    # ======================================================================="
        print ""
        print "    # 思源笔记的 API 请求 (排除 auth 和 unified)"
        print "    location ~ ^/api/(system|notebook|filetree|block|file|asset|storage|search|export|import|template|setting|sync|repo|riff|snippet|av|ai|petal|network|broadcast|archive|ui|web)/ {"
        print "        proxy_pass http://127.0.0.1:6806;"
        print "        proxy_http_version 1.1;"
        print ""
        print "        # WebSocket 支持"
        print "        proxy_set_header Upgrade $http_upgrade;"
        print "        proxy_set_header Connection \"upgrade\";"
        print ""
        print "        proxy_set_header Host $host;"
        print "        proxy_set_header X-Real-IP $remote_addr;"
        print "        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;"
        print "        proxy_set_header X-Forwarded-Proto $scheme;"
        print ""
        print "        proxy_connect_timeout 60s;"
        print "        proxy_send_timeout 60s;"
        print "        proxy_read_timeout 60s;"
        print "        proxy_buffering off;"
        print "    }"
        print ""
        print "    # 思源笔记的 WebSocket 连接"
        print "    location /ws {"
        print "        proxy_pass http://127.0.0.1:6806;"
        print "        proxy_http_version 1.1;"
        print ""
        print "        proxy_set_header Upgrade $http_upgrade;"
        print "        proxy_set_header Connection \"upgrade\";"
        print ""
        print "        proxy_set_header Host $host;"
        print "        proxy_set_header X-Real-IP $remote_addr;"
        print "        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;"
        print "        proxy_set_header X-Forwarded-Proto $scheme;"
        print ""
        print "        proxy_connect_timeout 60s;"
        print "        proxy_send_timeout 60s;"
        print "        proxy_read_timeout 60s;"
        print "        proxy_buffering off;"
        print "    }"
        print ""
        print "    # 思源笔记的上传端点"
        print "    location /upload {"
        print "        proxy_pass http://127.0.0.1:6806;"
        print "        proxy_http_version 1.1;"
        print ""
        print "        proxy_set_header Host $host;"
        print "        proxy_set_header X-Real-IP $remote_addr;"
        print "        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;"
        print "        proxy_set_header X-Forwarded-Proto $scheme;"
        print ""
        print "        client_max_body_size 100M;"
        print "        proxy_request_buffering off;"
        print "    }"
        print ""
        print "    # 思源笔记的静态资源"
        print "    location ~ ^/(stage|appearance|assets|widgets|plugins|emojis|templates|public|snippets|export|history|repo)/ {"
        print "        proxy_pass http://127.0.0.1:6806;"
        print "        proxy_http_version 1.1;"
        print ""
        print "        proxy_set_header Host $host;"
        print "        proxy_set_header X-Real-IP $remote_addr;"
        print "        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;"
        print "        proxy_set_header X-Forwarded-Proto $scheme;"
        print "    }"
        print ""
        print "    # /notepads 路径重定向到思源笔记"
        print "    location /notepads {"
        print "        return 301 $scheme://$host/;"
        print "    }"
        print ""
        print "    location /notepads/ {"
        print "        return 301 $scheme://$host/;"
        print "    }"
        print ""
        siyuan_added = 1
    }
}

# 修改根路径 location / 指向思源笔记
/^    location \/ \{/ {
    if (!in_root_location) {
        in_root_location = 1
        print "    # 根路径指向思源笔记"
        print "    location / {"
        print "        proxy_pass http://127.0.0.1:6806;"
        print "        proxy_http_version 1.1;"
        print ""
        print "        # WebSocket 支持"
        print "        proxy_set_header Upgrade $http_upgrade;"
        print "        proxy_set_header Connection \"upgrade\";"
        print ""
        print "        proxy_set_header Host $host;"
        print "        proxy_set_header X-Real-IP $remote_addr;"
        print "        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;"
        print "        proxy_set_header X-Forwarded-Proto $scheme;"
        print ""
        print "        proxy_connect_timeout 60s;"
        print "        proxy_send_timeout 60s;"
        print "        proxy_read_timeout 60s;"
        print "        proxy_buffering off;"
        skip_lines = 1
        next
    }
}

# 检测根路径配置块结束
/^    \}/ {
    if (in_root_location) {
        print "    }"
        in_root_location = 0
        skip_lines = 0
        next
    }
}

# 跳过需要删除的行
{
    if (!skip_lines) {
        print
    }
}
' > "$TEMP_CONF"

# 替换原配置文件
sudo cp "$TEMP_CONF" "$NGINX_CONF"
rm "$TEMP_CONF"

echo "✅ 配置已更新"

echo ""
echo "🧪 测试 Nginx 配置..."
if sudo nginx -t; then
    echo "✅ Nginx 配置测试通过"
    
    echo ""
    echo "🔄 重新加载 Nginx..."
    sudo systemctl reload nginx
    echo "✅ Nginx 已重新加载"
    
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "✅ 配置迁移成功！"
    echo ""
    echo "现在可以通过以下方式访问："
    echo "  - 思源笔记 (根路径): https://www.cheman.top/"
    echo "  - 思源笔记 (notepads): https://www.cheman.top/notepads (重定向到根路径)"
    echo "  - 潮汐志: https://www.cheman.top/calendars/"
    echo ""
    echo "备份文件: $BACKUP_CONF"
    echo ""
    echo "如果需要恢复，运行:"
    echo "  sudo cp $BACKUP_CONF $NGINX_CONF"
    echo "  sudo nginx -t && sudo systemctl reload nginx"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
else
    echo "❌ Nginx 配置测试失败"
    echo "正在恢复备份..."
    sudo cp "$BACKUP_CONF" "$NGINX_CONF"
    echo "✅ 已恢复备份"
    exit 1
fi
