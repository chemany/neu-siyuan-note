#!/bin/bash

# 思源笔记 + AI Web服务启动脚本
echo "🚀 启动思源笔记AI增强版..."

# 设置工作目录
cd /root/code/siyuan

# 创建工作空间目录
mkdir -p workspace/data

# 设置环境变量
export SIYUAN_WORKSPACE="/root/code/siyuan/workspace"
export SIYUAN_PORT=6806
export PATH=$PATH:/usr/local/go/bin

# 提示配置AI
if [ -z "$OPENAI_API_KEY" ]; then
    echo "⚠️  警告: 未设置 OPENAI_API_KEY 环境变量"
    echo "💡 请设置: export OPENAI_API_KEY=sk-xxx"
    echo "🔧 LLM聊天功能将不可用，但基础功能正常"
fi

if [ -z "$SIYUAN_EMBEDDING_API_KEY" ]; then
    echo "⚠️  警告: 未设置 SIYUAN_EMBEDDING_API_KEY 环境变量"
    echo "💡 请设置: export SIYUAN_EMBEDDING_API_KEY=sk-xxx"
    echo "🔧 向量化功能将不可用"
fi

# 可选配置提示
echo ""
echo "🔧 可选AI配置:"
echo ""
echo "📝 LLM/对话模型配置:"
echo "   API密钥: export OPENAI_API_KEY=sk-xxx"
echo "   LLM提供商: export SIYUAN_LLM_PROVIDER=openai"
echo "   LLM模型: export SIYUAN_LLM_MODEL=gpt-4o-mini"
echo "   LLM温度: export SIYUAN_LLM_TEMPERATURE=0.7"
echo "   最大令牌: export SIYUAN_LLM_MAX_TOKENS=4000"
echo "   最大上下文: export SIYUAN_LLM_MAX_CONTEXTS=7"
echo "   请求超时: export SIYUAN_LLM_TIMEOUT=30"
echo "   API地址: export SIYUAN_LLM_API_BASE_URL=https://api.openai.com/v1"
echo "   代理设置: export SIYUAN_LLM_PROXY=http://proxy:port"
echo "   API版本: export SIYUAN_LLM_API_VERSION=2024-01-01"
echo ""
echo "🔍 向量化模型配置:"
echo "   API密钥: export SIYUAN_EMBEDDING_API_KEY=sk-xxx"
echo "   向量化提供商: export SIYUAN_EMBEDDING_PROVIDER=siliconflow"
echo "   向量化模型: export SIYUAN_EMBEDDING_MODEL=BAAI/bge-large-zh-v1.5"
echo "   API地址: export SIYUAN_EMBEDDING_API_BASE_URL=https://api.siliconflow.cn/v1/embeddings"
echo "   编码格式: export SIYUAN_EMBEDDING_ENCODING_FORMAT=float"
echo "   请求超时: export SIYUAN_EMBEDDING_TIMEOUT=30"
echo ""
echo "🎯 推荐模型选择:"
echo "   SiliconFlow向量化: BAAI/bge-large-zh-v1.5 (中文), BAAI/bge-m3 (多语言)"
echo "   OpenAI向量化: text-embedding-3-small (经济), text-embedding-3-large (精确)"
echo "   OpenAI对话: gpt-4o-mini (快速), gpt-4o (强大), gpt-3.5-turbo (经济)"

# 显示AI配置状态
echo "🤖 AI配置状态:"
echo "   LLM服务: $([ -n "$OPENAI_API_KEY" ] && echo "已配置" || echo "未配置")"
echo "   LLM提供商: ${SIYUAN_LLM_PROVIDER:-OpenAI}"
echo "   LLM模型: ${SIYUAN_LLM_MODEL:-gpt-3.5-turbo}"
echo "   LLM温度: ${SIYUAN_LLM_TEMPERATURE:-1.0}"
echo "   LLM最大令牌: ${SIYUAN_LLM_MAX_TOKENS:-4000}"
echo "   向量化服务: $([ -n "$SIYUAN_EMBEDDING_API_KEY" ] && echo "已配置" || echo "未配置")"
echo "   向量化提供商: ${SIYUAN_EMBEDDING_PROVIDER:-siliconflow}"
echo "   向量化模型: ${SIYUAN_EMBEDDING_MODEL:-BAAI/bge-large-zh-v1.5}"
echo "   向量化超时: ${SIYUAN_EMBEDDING_TIMEOUT:-30}秒"

# 构建前端（开发模式）
echo "📦 构建前端资源..."
cd app
npm install
npm run dev &
FRONTEND_PID=$!

# 等待前端构建完成
sleep 15

# 启动后端服务
echo "🔧 启动后端服务..."
cd ../kernel
go mod tidy

# 设置web模式环境变量，禁用UI进程检测
export SIYUAN_WEB_MODE="true"

# 启动后端服务（持久运行）
nohup CGO_ENABLED=1 go run main.go --mode production --port 6806 > /tmp/siyuan-backend.log 2>&1 &
BACKEND_PID=$!

echo "✅ 思源笔记AI服务已启动！"
echo "🌐 访问地址: http://localhost:6806"
echo "📊 API地址: http://localhost:6806/api"
echo "🤖 AI功能: $([ -n "$OPENAI_API_KEY" ] && echo "已启用" || echo "未配置")"
echo ""
echo "🎯 新增功能:"
echo "   • 语义搜索: POST /api/ai/semanticSearch"
echo "   • 笔记本摘要: POST /api/ai/generateNotebookSummary"
echo "   • 批量向量化: POST /api/ai/batchVectorizeNotebook"
echo ""
echo "按 Ctrl+C 停止服务"

# 等待用户中断
trap "echo '🛑 正在停止服务...'; kill $FRONTEND_PID $BACKEND_PID 2>/dev/null; exit" INT
wait