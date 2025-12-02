#!/usr/bin/env python3
"""
最终修复 notepads 配置
"""

import re

NGINX_CONF = "/etc/nginx/sites-available/nginx-server.conf"

# 读取配置
with open(NGINX_CONF, 'r') as f:
    content = f.read()

# 找到并删除所有旧的 notepads 配置块
# 使用正则表达式匹配 location 块
patterns_to_remove = [
    r'    # 笔记 API.*?\n    location ~ \^/notepads/\[Aa\]\[Pp\]\[Ii\]/.*?\n    \}',
    r'    # 笔记前端页面.*?\n    location = /notepads.*?\n    \}',
    r'    # 笔记静态资源.*?\n    location ~\* \^/notepads/_next/.*?\n    \}',
    r'    # 笔记前端页面和其他静态资源.*?\n    location /notepads/.*?\n    \}',
]

for pattern in patterns_to_remove:
    content = re.sub(pattern, '', content, flags=re.DOTALL)

# 在统一设置服务之前添加新的 notepads 配置
unified_service_marker = '    # ======================================================================='
if '# 统一设置服务' in content:
    # 找到统一设置服务的位置
    pos = content.find('    # =======================================================================\n    # 统一设置服务')
    
    # 在这之前插入新配置
    new_config = '''    # /notepads 路径代理到思源笔记（去掉 /notepads 前缀）
    location = /notepads {
        return 301 $scheme://$host/notepads/;
    }

    location /notepads/ {
        rewrite ^/notepads/(.*)$ /$1 break;
        proxy_pass http://127.0.0.1:6806;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_buffering off;
    }

'''
    content = content[:pos] + new_config + content[pos:]

# 写回文件
with open(NGINX_CONF, 'w') as f:
    f.write(content)

print("✅ 配置已更新")
print("\n请运行: sudo nginx -t && sudo systemctl reload nginx")
