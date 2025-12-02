#!/bin/bash

# 思源笔记一键重新构建和重启脚本
# 注意：前端使用 build:desktop 构建（Web版使用desktop目录）
set -e  # 遇到错误立即退出

echo "🔄 开始重新构建思源笔记..."
echo ""

# 切换到项目目录
cd /root/code/siyuan

# 1. 构建前端 (使用 desktop 构建，Web版访问 /stage/build/desktop/)
echo "📦 [1/3] 构建前端 (desktop)..."
cd app
npm run build:desktop
if [ $? -ne 0 ]; then
    echo "❌ 前端构建失败！"
    exit 1
fi
echo "✅ 前端构建成功 (输出目录: stage/build/desktop/)"
echo ""

# 2. 构建后端
echo "🔧 [2/3] 构建后端..."
cd ../kernel
go mod tidy
CGO_ENABLED=1 go build -v -o siyuan-kernel -tags "fts5" -ldflags "-s -w" .
if [ $? -ne 0 ]; then
    echo "❌ 后端构建失败！"
    exit 1
fi
echo "✅ 后端构建成功"
echo ""

# 3. 重启服务
echo "🔄 [3/3] 重启服务..."
cd ..
pm2 restart siyuan-kernel
if [ $? -ne 0 ]; then
    echo "⚠️  服务未在运行，尝试启动..."
    ./start-production.sh
fi
echo "✅ 服务重启成功"
echo ""

# 等待服务启动
echo "⏳ 等待服务启动..."
sleep 3

# 检查服务状态
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
./check-status.sh
