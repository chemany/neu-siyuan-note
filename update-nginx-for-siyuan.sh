#!/bin/bash

# 更新 Nginx 配置以支持思源笔记
# 这个脚本会在现有配置中添加思源笔记的路由规则

set -e

NGINX_CONF="/etc/nginx/sites-available/nginx-server.conf"
BACKUP_CONF="/tmp/nginx-server-backup-$(date +%Y%m%d-%H%M%S).conf"

echo "📋 备份当前配置..."
cp "$NGINX_CONF" "$BACKUP_CONF"
echo "✅ 已备份到: $BACKUP_CONF"

echo ""
echo "🔧 修改 Nginx 配置..."

# 找到 location ~* ^/api/ 这一行的行号
LINE_NUM=$(grep -n "location ~\* \^/api/" "$NGINX_CONF" | cut -d: -f1)

if [ -z "$LINE_NUM" ]; then
    echo "❌ 错误: 找不到 'location ~* ^/api/' 规则"
    exit 1
fi

echo "找到 /api/ 规则在第 $LINE_NUM 行"

# 在这一行之前插入思源笔记的规则
# 使用 sed 在指定行之前插入内容
sudo sed -i "${LINE_NUM}i\\    # =======================================================================\\
    # 思源笔记 (SiYuan) - 个人知识管理系统\\
    # 主服务: 6806\\
    # =======================================================================\\
    \\
    # 思源笔记的 API 请求 (排除 auth 和 unified，它们由统一认证服务处理)\\
    # 这个规则必须在通用的 /api/ 规则之前\\
    location ~ ^/api/(system|notebook|filetree|block|file|asset|storage|search|export|import|template|setting|sync|repo|riff|snippet|av|ai|petal|network|broadcast|archive|ui|web)/ {\\
        proxy_pass http://127.0.0.1:6806;\\
        proxy_http_version 1.1;\\
        \\
        # WebSocket 支持\\
        proxy_set_header Upgrade \$http_upgrade;\\
        proxy_set_header Connection \"upgrade\";\\
        \\
        # 传递真实客户端信息\\
        proxy_set_header Host \$host;\\
        proxy_set_header X-Real-IP \$remote_addr;\\
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;\\
        proxy_set_header X-Forwarded-Proto \$scheme;\\
        \\
        # 超时设置\\
        proxy_connect_timeout 60s;\\
        proxy_send_timeout 60s;\\
        proxy_read_timeout 60s;\\
        \\
        # 禁用缓冲以支持实时推送\\
        proxy_buffering off;\\
    }\\
    \\
    # 思源笔记的 WebSocket 连接\\
    location /ws {\\
        proxy_pass http://127.0.0.1:6806;\\
        proxy_http_version 1.1;\\
        \\
        # WebSocket 支持\\
        proxy_set_header Upgrade \$http_upgrade;\\
        proxy_set_header Connection \"upgrade\";\\
        \\
        proxy_set_header Host \$host;\\
        proxy_set_header X-Real-IP \$remote_addr;\\
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;\\
        proxy_set_header X-Forwarded-Proto \$scheme;\\
        \\
        proxy_connect_timeout 60s;\\
        proxy_send_timeout 60s;\\
        proxy_read_timeout 60s;\\
        proxy_buffering off;\\
    }\\
    \\
    # 思源笔记的上传端点\\
    location /upload {\\
        proxy_pass http://127.0.0.1:6806;\\
        proxy_http_version 1.1;\\
        \\
        proxy_set_header Host \$host;\\
        proxy_set_header X-Real-IP \$remote_addr;\\
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;\\
        proxy_set_header X-Forwarded-Proto \$scheme;\\
        \\
        # 支持大文件上传\\
        client_max_body_size 100M;\\
        proxy_request_buffering off;\\
    }\\
    \\
    # 思源笔记的静态资源\\
    location ~ ^/(stage|appearance|assets|widgets|plugins|emojis|templates|public|snippets|export|history|repo)/ {\\
        proxy_pass http://127.0.0.1:6806;\\
        proxy_http_version 1.1;\\
        \\
        proxy_set_header Host \$host;\\
        proxy_set_header X-Real-IP \$remote_addr;\\
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;\\
        proxy_set_header X-Forwarded-Proto \$scheme;\\
    }\\
    \\
" "$NGINX_CONF"

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
    echo "✅ 配置更新成功！"
    echo ""
    echo "现在可以通过以下方式访问："
    echo "  - 思源笔记: http://your-domain/"
    echo "  - 潮汐志: http://your-domain/calendars/"
    echo "  - 笔记本: http://your-domain/notepads/"
    echo ""
    echo "备份文件: $BACKUP_CONF"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
else
    echo "❌ Nginx 配置测试失败"
    echo "正在恢复备份..."
    sudo cp "$BACKUP_CONF" "$NGINX_CONF"
    echo "✅ 已恢复备份"
    exit 1
fi
