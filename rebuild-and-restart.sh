#!/bin/bash

# 思源笔记一键重新构建和重启脚本
# 注意：前端使用 build:desktop 构建（Web版使用desktop目录）
set -e  # 遇到错误立即退出

echo "🔄 开始重新构建思源笔记..."
echo ""

# 切换到项目目录
cd /root/code/neu-siyuan-note

# 1. 构建前端 (使用 desktop 构建，Web版访问 /stage/build/desktop/)
echo "📦 [1/3] 构建前端 (desktop)..."
cd app
# 限制 Node 内存，降低优先级，防止抢占 SSH 资源
export NODE_OPTIONS="--max-old-space-size=2048"
nice -n 19 npm run build:desktop
if [ $? -ne 0 ]; then
    echo "❌ 前端构建失败！"
    exit 1
fi
echo "✅ 前端构建成功 (输出目录: stage/build/desktop/)"
echo ""

# 释放内存缓冲
sync
sleep 2

# 2. 构建后端
echo "🔧 [2/3] 构建后端..."
cd ../kernel
go mod tidy
# 限制 Go 编译并发核心数为 2，降低优先级
CGO_ENABLED=1 nice -n 19 go build -p 2 -v -o siyuan-kernel -tags "fts5" -ldflags "-s -w" .
if [ $? -ne 0 ]; then
    echo "❌ 后端构建失败！"
    exit 1
fi
echo "✅ 后端构建成功"
echo ""

sync
sleep 2

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

echo ""
echo "✅ 服务重启成功，请访问 http://localhost:6806 进行测试"
