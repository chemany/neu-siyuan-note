#!/bin/bash

# 思源笔记服务检查脚本

echo "🔍 检查思源笔记服务状态..."
echo ""

# 检查 PM2 进程
echo "📊 PM2 进程状态:"
pm2 list | grep siyuan-kernel
echo ""

# 检查端口监听
echo "🌐 端口监听状态:"
if netstat -tlnp 2>/dev/null | grep ":6806" > /dev/null; then
    echo "✅ 端口 6806 正在监听"
    netstat -tlnp 2>/dev/null | grep ":6806"
else
    echo "❌ 端口 6806 未监听"
fi
echo ""

# 检查 API
echo "🔌 API 测试:"
API_RESPONSE=$(curl -s http://localhost:6806/api/system/version 2>/dev/null)
if [ $? -eq 0 ]; then
    echo "✅ API 响应正常: $API_RESPONSE"
else
    echo "❌ API 无响应"
fi
echo ""

# 检查前端
echo "🖥️  前端测试:"
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:6806/ 2>/dev/null)
if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "302" ]; then
    echo "✅ 前端访问正常 (HTTP $HTTP_CODE)"
else
    echo "⚠️  前端访问异常 (HTTP $HTTP_CODE)"
fi
echo ""

# 最近日志
echo "📝 最近日志 (最后10行):"
pm2 logs siyuan-kernel --lines 10 --nostream 2>/dev/null | tail -15
echo ""

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "🌐 访问地址: http://localhost:6806"
echo "📊 查看日志: pm2 logs siyuan-kernel"
echo "🔄 重启服务: pm2 restart siyuan-kernel"
echo "🛑 停止服务: pm2 stop siyuan-kernel"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
