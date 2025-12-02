#!/bin/bash

echo "================================"
echo "思源笔记远程访问测试"
echo "================================"
echo ""

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 测试函数
test_endpoint() {
    local name=$1
    local url=$2
    local expected_code=$3
    
    echo -n "测试 $name ... "
    response=$(curl -s -o /dev/null -w "%{http_code}" "$url" 2>/dev/null)
    
    if [ "$response" = "$expected_code" ]; then
        echo -e "${GREEN}✓ 成功${NC} (HTTP $response)"
        return 0
    else
        echo -e "${RED}✗ 失败${NC} (HTTP $response, 期望 $expected_code)"
        return 1
    fi
}

# 1. 检查服务状态
echo "1. 检查服务状态"
echo "----------------------------"

echo -n "Nginx 状态: "
if systemctl is-active --quiet nginx; then
    echo -e "${GREEN}✓ 运行中${NC}"
else
    echo -e "${RED}✗ 未运行${NC}"
fi

echo -n "思源笔记服务: "
if pm2 list | grep -q "siyuan-kernel.*online"; then
    echo -e "${GREEN}✓ 运行中${NC}"
else
    echo -e "${RED}✗ 未运行${NC}"
fi

echo -n "统一认证服务: "
if pm2 list | grep -q "unified-settings.*online"; then
    echo -e "${GREEN}✓ 运行中${NC}"
else
    echo -e "${RED}✗ 未运行${NC}"
fi

echo ""

# 2. 检查端口监听
echo "2. 检查端口监听"
echo "----------------------------"

check_port() {
    local port=$1
    local name=$2
    echo -n "$name (端口 $port): "
    if netstat -tlnp 2>/dev/null | grep -q ":$port.*LISTEN"; then
        echo -e "${GREEN}✓ 监听中${NC}"
    else
        echo -e "${RED}✗ 未监听${NC}"
    fi
}

check_port 80 "Nginx"
check_port 6806 "思源笔记"
check_port 3002 "统一认证"

echo ""

# 3. 测试 HTTP 端点
echo "3. 测试 HTTP 端点"
echo "----------------------------"

# 测试主页（应该重定向到登录页）
test_endpoint "主页访问" "http://localhost/" "302"

# 测试登录页面
test_endpoint "登录页面" "http://localhost/stage/login.html" "200"

# 测试注册页面
test_endpoint "注册页面" "http://localhost/stage/register.html" "200"

# 测试 API 版本
test_endpoint "API 版本" "http://localhost/api/system/version" "200"

echo ""

# 4. 测试 Nginx 代理
echo "4. 测试 Nginx 代理"
echo "----------------------------"

# 测试认证 API 代理
echo -n "认证 API 代理: "
response=$(curl -s -X POST http://localhost/api/auth/login \
    -H "Content-Type: application/json" \
    -d '{"email":"test@test.com","password":"test"}' 2>/dev/null)

if echo "$response" | grep -q "error\|message"; then
    echo -e "${GREEN}✓ 代理正常${NC} (收到响应)"
else
    echo -e "${YELLOW}⚠ 可能有问题${NC} (响应: $response)"
fi

echo ""

# 5. 检查 Nginx 配置
echo "5. 检查 Nginx 配置"
echo "----------------------------"

echo -n "配置文件语法: "
if sudo nginx -t 2>&1 | grep -q "syntax is ok"; then
    echo -e "${GREEN}✓ 正确${NC}"
else
    echo -e "${RED}✗ 错误${NC}"
fi

echo -n "WebSocket 支持: "
if sudo nginx -T 2>/dev/null | grep -q "proxy_set_header Upgrade"; then
    echo -e "${GREEN}✓ 已配置${NC}"
else
    echo -e "${RED}✗ 未配置${NC}"
fi

echo -n "文件上传限制: "
max_size=$(sudo nginx -T 2>/dev/null | grep "client_max_body_size" | head -1 | awk '{print $2}' | tr -d ';')
if [ -n "$max_size" ]; then
    echo -e "${GREEN}✓ $max_size${NC}"
else
    echo -e "${YELLOW}⚠ 未设置${NC}"
fi

echo ""

# 6. 获取访问地址
echo "6. 访问地址"
echo "----------------------------"

# 获取本机 IP
local_ip=$(hostname -I | awk '{print $1}')
echo "本地访问: http://localhost/"
echo "局域网访问: http://$local_ip/"

# 尝试获取公网 IP
public_ip=$(curl -s ifconfig.me 2>/dev/null || curl -s icanhazip.com 2>/dev/null)
if [ -n "$public_ip" ]; then
    echo "公网访问: http://$public_ip/"
fi

echo ""

# 7. 日志位置
echo "7. 日志文件"
echo "----------------------------"
echo "Nginx 访问日志: /var/log/nginx/siyuan_access.log"
echo "Nginx 错误日志: /var/log/nginx/siyuan_error.log"
echo "思源笔记日志: pm2 logs siyuan-kernel"
echo "统一认证日志: pm2 logs unified-settings"

echo ""
echo "================================"
echo "测试完成！"
echo "================================"
