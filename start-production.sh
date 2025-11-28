#!/bin/bash

# 思源笔记生产环境启动脚本
echo "🚀 启动思源笔记..."

# 设置工作目录
cd /root/code/siyuan

# 创建工作空间目录
mkdir -p workspace/data

# 检查构建产物
echo "📦 检查构建产物..."
if [ ! -f "kernel/siyuan-kernel" ]; then
    echo "❌ 后端未构建，请先运行: cd kernel && go build -v -o siyuan-kernel ."
    exit 1
fi

if [ ! -d "app/stage/build" ]; then
    echo "❌ 前端未构建，请先运行: cd app && npm run build:app"
    exit 1
fi

echo "✅ 构建产物检查通过"

# AI 配置状态提示
echo ""
echo "🤖 AI 配置提示:"
echo "如需启用 AI 功能，请在 ecosystem.config.js 中设置:"
echo "  - OPENAI_API_KEY: OpenAI API 密钥"
echo "  - SIYUAN_LLM_PROVIDER: LLM 提供商 (openai/anthropic等)"
echo "  - SIYUAN_LLM_MODEL: 模型名称 (gpt-4o-mini等)"
echo "  - SIYUAN_EMBEDDING_PROVIDER: 向量化提供商 (siliconflow/openai)"
echo "  - SIYUAN_EMBEDDING_MODEL: 向量化模型"
echo ""

# 停止旧服务
echo "🛑 停止旧服务..."
pm2 delete siyuan-kernel 2>/dev/null || true

# 使用 PM2 启动服务
echo "🔧 启动后端服务..."
pm2 start ecosystem.config.js

# 显示服务状态
echo ""
echo "✅ 思源笔记已启动！"
echo ""
pm2 list
echo ""
echo "🌐 访问地址: http://localhost:6806"
echo "📊 查看日志: pm2 logs siyuan-kernel"
echo "🔍 查看状态: pm2 status"
echo "🛑 停止服务: pm2 stop siyuan-kernel"
echo "🔄 重启服务: pm2 restart siyuan-kernel"
echo ""
