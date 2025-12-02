#!/usr/bin/env python3
"""
修改 Nginx 配置，将 /notepads 重定向到思源笔记
"""

import re
import sys
from datetime import datetime

NGINX_CONF = "/etc/nginx/sites-available/nginx-server.conf"

def backup_config():
    """备份配置文件"""
    timestamp = datetime.now().strftime("%Y%m%d-%H%M%S")
    backup_file = f"/tmp/nginx-server-backup-{timestamp}.conf"
    
    with open(NGINX_CONF, 'r') as f:
        content = f.read()
    
    with open(backup_file, 'w') as f:
        f.write(content)
    
    print(f"✅ 已备份到: {backup_file}")
    return backup_file

def modify_config():
    """修改配置文件"""
    with open(NGINX_CONF, 'r') as f:
        lines = f.readlines()
    
    new_lines = []
    skip_until_brace = 0
    siyuan_added = False
    
    i = 0
    while i < len(lines):
        line = lines[i]
        
        # 跳过旧的 notepads 配置块
        if re.search(r'location.*notepads', line):
            skip_until_brace = 1
            i += 1
            continue
        
        # 计数大括号
        if skip_until_brace > 0:
            if '{' in line:
                skip_until_brace += line.count('{')
            if '}' in line:
                skip_until_brace -= line.count('}')
            i += 1
            continue
        
        # 在 location ~* ^/api/ 之前添加思源笔记配置
        if not siyuan_added and re.search(r'location ~\* \^/api/', line):
            new_lines.append("    # =======================================================================\n")
            new_lines.append("    # 思源笔记 (SiYuan) - 个人知识管理系统\n")
            new_lines.append("    # 主服务: 6806, 路径: / 和 /notepads\n")
            new_lines.append("    # =======================================================================\n")
            new_lines.append("\n")
            
            # 思源笔记 API
            new_lines.append("    # 思源笔记的 API 请求\n")
            new_lines.append("    location ~ ^/api/(system|notebook|filetree|block|file|asset|storage|search|export|import|template|setting|sync|repo|riff|snippet|av|ai|petal|network|broadcast|archive|ui|web)/ {\n")
            new_lines.append("        proxy_pass http://127.0.0.1:6806;\n")
            new_lines.append("        proxy_http_version 1.1;\n")
            new_lines.append("        proxy_set_header Upgrade $http_upgrade;\n")
            new_lines.append("        proxy_set_header Connection \"upgrade\";\n")
            new_lines.append("        proxy_set_header Host $host;\n")
            new_lines.append("        proxy_set_header X-Real-IP $remote_addr;\n")
            new_lines.append("        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;\n")
            new_lines.append("        proxy_set_header X-Forwarded-Proto $scheme;\n")
            new_lines.append("        proxy_connect_timeout 60s;\n")
            new_lines.append("        proxy_send_timeout 60s;\n")
            new_lines.append("        proxy_read_timeout 60s;\n")
            new_lines.append("        proxy_buffering off;\n")
            new_lines.append("    }\n")
            new_lines.append("\n")
            
            # WebSocket
            new_lines.append("    # 思源笔记的 WebSocket 连接\n")
            new_lines.append("    location /ws {\n")
            new_lines.append("        proxy_pass http://127.0.0.1:6806;\n")
            new_lines.append("        proxy_http_version 1.1;\n")
            new_lines.append("        proxy_set_header Upgrade $http_upgrade;\n")
            new_lines.append("        proxy_set_header Connection \"upgrade\";\n")
            new_lines.append("        proxy_set_header Host $host;\n")
            new_lines.append("        proxy_set_header X-Real-IP $remote_addr;\n")
            new_lines.append("        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;\n")
            new_lines.append("        proxy_set_header X-Forwarded-Proto $scheme;\n")
            new_lines.append("        proxy_buffering off;\n")
            new_lines.append("    }\n")
            new_lines.append("\n")
            
            # 上传
            new_lines.append("    # 思源笔记的上传端点\n")
            new_lines.append("    location /upload {\n")
            new_lines.append("        proxy_pass http://127.0.0.1:6806;\n")
            new_lines.append("        proxy_http_version 1.1;\n")
            new_lines.append("        proxy_set_header Host $host;\n")
            new_lines.append("        proxy_set_header X-Real-IP $remote_addr;\n")
            new_lines.append("        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;\n")
            new_lines.append("        proxy_set_header X-Forwarded-Proto $scheme;\n")
            new_lines.append("        client_max_body_size 100M;\n")
            new_lines.append("        proxy_request_buffering off;\n")
            new_lines.append("    }\n")
            new_lines.append("\n")
            
            # 静态资源
            new_lines.append("    # 思源笔记的静态资源\n")
            new_lines.append("    location ~ ^/(stage|appearance|assets|widgets|plugins|emojis|templates|public|snippets|export|history|repo)/ {\n")
            new_lines.append("        proxy_pass http://127.0.0.1:6806;\n")
            new_lines.append("        proxy_http_version 1.1;\n")
            new_lines.append("        proxy_set_header Host $host;\n")
            new_lines.append("        proxy_set_header X-Real-IP $remote_addr;\n")
            new_lines.append("        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;\n")
            new_lines.append("        proxy_set_header X-Forwarded-Proto $scheme;\n")
            new_lines.append("    }\n")
            new_lines.append("\n")
            
            # notepads 重定向
            new_lines.append("    # /notepads 路径重定向到思源笔记根路径\n")
            new_lines.append("    location = /notepads {\n")
            new_lines.append("        return 301 $scheme://$host/;\n")
            new_lines.append("    }\n")
            new_lines.append("\n")
            new_lines.append("    location /notepads/ {\n")
            new_lines.append("        return 301 $scheme://$host/;\n")
            new_lines.append("    }\n")
            new_lines.append("\n")
            
            siyuan_added = True
        
        # 修改根路径 location /
        if re.match(r'^    location / \{', line):
            # 跳过原来的 location / 块
            skip_until_brace = 1
            # 添加新的配置
            new_lines.append("    # 根路径指向思源笔记\n")
            new_lines.append("    location / {\n")
            new_lines.append("        proxy_pass http://127.0.0.1:6806;\n")
            new_lines.append("        proxy_http_version 1.1;\n")
            new_lines.append("        proxy_set_header Upgrade $http_upgrade;\n")
            new_lines.append("        proxy_set_header Connection \"upgrade\";\n")
            new_lines.append("        proxy_set_header Host $host;\n")
            new_lines.append("        proxy_set_header X-Real-IP $remote_addr;\n")
            new_lines.append("        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;\n")
            new_lines.append("        proxy_set_header X-Forwarded-Proto $scheme;\n")
            new_lines.append("        proxy_connect_timeout 60s;\n")
            new_lines.append("        proxy_send_timeout 60s;\n")
            new_lines.append("        proxy_read_timeout 60s;\n")
            new_lines.append("        proxy_buffering off;\n")
            new_lines.append("    }\n")
            i += 1
            continue
        
        new_lines.append(line)
        i += 1
    
    # 写回文件
    with open(NGINX_CONF, 'w') as f:
        f.writelines(new_lines)
    
    print("✅ 配置已更新")

def main():
    print("📋 备份当前配置...")
    backup_file = backup_config()
    
    print("\n🔧 修改 Nginx 配置...")
    try:
        modify_config()
    except Exception as e:
        print(f"❌ 修改失败: {e}")
        print(f"请手动恢复备份: sudo cp {backup_file} {NGINX_CONF}")
        sys.exit(1)
    
    print("\n✅ 配置修改完成")
    print("\n请运行以下命令测试并重新加载 Nginx:")
    print("  sudo nginx -t && sudo systemctl reload nginx")

if __name__ == "__main__":
    main()
