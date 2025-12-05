#!/bin/bash

# 思源笔记外网访问验证脚本

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  思源笔记外网访问配置验证"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 1. 检查服务状态
echo "1️⃣  检查服务状态..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
pm2 list | grep -E "siyuan-kernel|unified-settings"

SIYUAN_STATUS=$(pm2 jlist | jq -r '.[] | select(.name=="siyuan-kernel") | .pm2_env.status')
UNIFIED_STATUS=$(pm2 jlist | jq -r '.[] | select(.name=="unified-settings") | .pm2_env.status')

if [ "$SIYUAN_STATUS" == "online" ]; then
    echo -e "${GREEN}✅ 思源笔记服务运行中${NC}"
else
    echo -e "${RED}❌ 思源笔记服务未运行${NC}"
fi

if [ "$UNIFIED_STATUS" == "online" ]; then
    echo -e "${GREEN}✅ 统一认证服务运行中${NC}"
else
    echo -e "${RED}❌ 统一认证服务未运行${NC}"
fi
echo ""

# 2. 检查端口监听
echo "2️⃣  检查端口监听..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
if netstat -tlnp 2>/dev/null | grep -q ":6806"; then
    echo -e "${GREEN}✅ 端口 6806 (思源笔记) 正在监听${NC}"
else
    echo -e "${RED}❌ 端口 6806 未监听${NC}"
fi

if netstat -tlnp 2>/dev/null | grep -q ":3002"; then
    echo -e "${GREEN}✅ 端口 3002 (统一认证) 正在监听${NC}"
else
    echo -e "${RED}❌ 端口 3002 未监听${NC}"
fi

if netstat -tlnp 2>/dev/null | grep -q ":443"; then
    echo -e "${GREEN}✅ 端口 443 (HTTPS) 正在监听${NC}"
else
    echo -e "${RED}❌ 端口 443 未监听${NC}"
fi
echo ""

# 3. 检查 Nginx 配置
echo "3️⃣  检查 Nginx 配置..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
if nginx -t 2>&1 | grep -q "successful"; then
    echo -e "${GREEN}✅ Nginx 配置语法正确${NC}"
else
    echo -e "${RED}❌ Nginx 配置有误${NC}"
    nginx -t
fi

if systemctl is-active --quiet nginx; then
    echo -e "${GREEN}✅ Nginx 服务运行中${NC}"
else
    echo -e "${RED}❌ Nginx 服务未运行${NC}"
fi
echo ""

# 4. 检查登录页面配置
echo "4️⃣  检查登录页面配置..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
if grep -q "window.location.origin" /root/code/siyuan/kernel/stage/login.html; then
    echo -e "${GREEN}✅ 登录页面已使用动态 URL（window.location.origin）${NC}"
else
    echo -e "${RED}❌ 登录页面仍使用硬编码 localhost${NC}"
fi

if grep -q "localhost:3002" /root/code/siyuan/kernel/stage/login.html; then
    echo -e "${RED}❌ 警告: 登录页面仍包含 localhost:3002${NC}"
else
    echo -e "${GREEN}✅ 登录页面不包含硬编码的 localhost:3002${NC}"
fi
echo ""

# 5. 测试本地访问
echo "5️⃣  测试本地访问..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:6806/stage/login.html)
if [ "$HTTP_CODE" == "200" ]; then
    echo -e "${GREEN}✅ 本地访问登录页面成功 (HTTP $HTTP_CODE)${NC}"
else
    echo -e "${RED}❌ 本地访问登录页面失败 (HTTP $HTTP_CODE)${NC}"
fi

AUTH_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST http://localhost:3002/api/auth/login \
    -H "Content-Type: application/json" \
    -d '{"email":"test","password":"test"}')
if [ "$AUTH_CODE" == "401" ] || [ "$AUTH_CODE" == "400" ] || [ "$AUTH_CODE" == "200" ]; then
    echo -e "${GREEN}✅ 统一认证服务响应正常 (HTTP $AUTH_CODE)${NC}"
else
    echo -e "${YELLOW}⚠️  统一认证服务响应异常 (HTTP $AUTH_CODE)${NC}"
fi
echo ""

# 6. 测试 Nginx 代理
echo "6️⃣  测试 Nginx 代理..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
PROXY_CODE=$(curl -s -o /dev/null -w "%{http_code}" -k https://localhost/notepads/stage/login.html)
if [ "$PROXY_CODE" == "200" ]; then
    echo -e "${GREEN}✅ Nginx 代理思源笔记成功 (HTTP $PROXY_CODE)${NC}"
else
    echo -e "${RED}❌ Nginx 代理思源笔记失败 (HTTP $PROXY_CODE)${NC}"
fi

API_PROXY_CODE=$(curl -s -o /dev/null -w "%{http_code}" -k -X POST https://localhost/api/auth/login \
    -H "Content-Type: application/json" \
    -d '{"email":"test","password":"test"}')
if [ "$API_PROXY_CODE" == "401" ] || [ "$API_PROXY_CODE" == "400" ] || [ "$API_PROXY_CODE" == "200" ]; then
    echo -e "${GREEN}✅ Nginx 代理认证 API 成功 (HTTP $API_PROXY_CODE)${NC}"
else
    echo -e "${YELLOW}⚠️  Nginx 代理认证 API 异常 (HTTP $API_PROXY_CODE)${NC}"
fi
echo ""

# 7. 检查 Nginx 配置项
echo "7️⃣  检查关键 Nginx 配置项..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
NGINX_CONF="/etc/nginx/sites-enabled/nginx-server.conf"

if grep -q "location /notepads/" "$NGINX_CONF"; then
    echo -e "${GREEN}✅ /notepads/ 路由配置存在${NC}"
else
    echo -e "${RED}❌ /notepads/ 路由配置不存在${NC}"
fi

if grep -q "location ~\* \^/api/" "$NGINX_CONF"; then
    echo -e "${GREEN}✅ /api/ 路由配置存在${NC}"
else
    echo -e "${RED}❌ /api/ 路由配置不存在${NC}"
fi

if grep -A 5 "location /notepads/" "$NGINX_CONF" | grep -q "proxy_pass.*6806"; then
    echo -e "${GREEN}✅ /notepads/ 正确代理到端口 6806${NC}"
else
    echo -e "${RED}❌ /notepads/ 未正确代理到端口 6806${NC}"
fi

if grep -A 5 "location ~\* \^/api/" "$NGINX_CONF" | grep -q "proxy_pass.*3002"; then
    echo -e "${GREEN}✅ /api/ 正确代理到端口 3002${NC}"
else
    echo -e "${RED}❌ /api/ 未正确代理到端口 3002${NC}"
fi

if grep -A 10 "location /notepads/" "$NGINX_CONF" | grep -q "Upgrade"; then
    echo -e "${GREEN}✅ WebSocket 支持已配置${NC}"
else
    echo -e "${YELLOW}⚠️  WebSocket 支持可能未配置${NC}"
fi
echo ""

# 8. 总结
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  测试总结"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo -e "${YELLOW}📝 访问地址:${NC}"
echo "   - 外网: https://www.cheman.top/notepads/"
echo "   - 登录: https://www.cheman.top/notepads/stage/login.html"
echo ""
echo -e "${YELLOW}🧪 下一步:${NC}"
echo "   1. 在浏览器中访问: https://www.cheman.top/notepads/stage/login.html"
echo "   2. 打开开发者工具 (F12) -> Network 标签"
echo "   3. 输入邮箱密码，点击登录"
echo "   4. 验证请求发送到: https://www.cheman.top/api/auth/login"
echo "   5. 确认不再出现: net::ERR_CONNECTION_REFUSED"
echo ""
echo -e "${YELLOW}📚 相关文档:${NC}"
echo "   - Nginx 配置指南: /root/code/siyuan/NGINX_CONFIG_GUIDE.md"
echo "   - 外网访问修复: /root/code/siyuan/EXTERNAL_ACCESS_FIX.md"
echo "   - 部署总结: /root/code/siyuan/DEPLOYMENT_SUMMARY.md"
echo ""
echo -e "${YELLOW}🔍 日志查看:${NC}"
echo "   - 思源日志: pm2 logs siyuan-kernel"
echo "   - 认证日志: pm2 logs unified-settings"
echo "   - Nginx 访问日志: tail -f /var/log/nginx/cheman.top-access.log"
echo "   - Nginx 错误日志: tail -f /var/log/nginx/cheman.top-error.log"
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
