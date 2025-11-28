#!/bin/bash

# SiYuan AI 配置示例脚本
# 使用方法: source ai-config-examples.sh

echo ""
echo "🚀 SiYuan AI 配置示例"
echo "======================="
echo ""

# 配置选项
show_examples() {
    echo "📋 可用配置示例:"
    echo "1. siliconflow-only    - 仅使用SiliconFlow向量化 (推荐中文用户)"
    echo "2. openai-only         - 仅使用OpenAI服务 (LLM + 向量化)"
    echo "3. hybrid-premium      - 混合配置: OpenAI LLM + SiliconFlow向量化"
    echo "4. economy-config      - 经济配置: GPT-3.5 + OpenAI向量化"
    echo "5. performance-config  - 性能配置: GPT-4 + 高级向量化"
    echo "6. current-config      - 显示当前配置"
    echo "clear                  - 清除所有AI配置"
    echo ""
}

# SiliconFlow 向量化配置
siliconflow_only() {
    echo "🔧 设置 SiliconFlow 向量化配置..."

    # 清除之前的配置
    unset OPENAI_API_KEY SIYUAN_OPENAI_API_KEY
    unset SIYUAN_LLM_PROVIDER SIYUAN_LLM_MODEL

    # 设置SiliconFlow向量化 (需要用户填入真实API密钥)
    export SIYUAN_EMBEDDING_PROVIDER="siliconflow"
    export SIYUAN_EMBEDDING_MODEL="BAAI/bge-large-zh-v1.5"
    export SIYUAN_EMBEDDING_API_BASE_URL="https://api.siliconflow.cn/v1/embeddings"
    export SIYUAN_EMBEDDING_ENCODING_FORMAT="float"
    export SIYUAN_EMBEDDING_TIMEOUT="30"

    # 提示用户设置API密钥
    echo "⚠️  请设置您的 SiliconFlow API 密钥:"
    echo "   export SIYUAN_EMBEDDING_API_KEY=your-siliconflow-api-key"
    echo ""
    echo "✅ SiliconFlow 向量化配置完成"
}

# 仅OpenAI配置
openai_only() {
    echo "🔧 设置 OpenAI 完整服务配置..."

    # LLM配置
    export OPENAI_API_KEY="your-openai-api-key"  # 需要用户填入真实密钥
    export SIYUAN_LLM_PROVIDER="openai"
    export SIYUAN_LLM_MODEL="gpt-4o-mini"
    export SIYUAN_LLM_TEMPERATURE="0.7"
    export SIYUAN_LLM_MAX_TOKENS="4000"
    export SIYUAN_LLM_TIMEOUT="30"
    export SIYUAN_LLM_API_BASE_URL="https://api.openai.com/v1"

    # 向量化配置
    export SIYUAN_EMBEDDING_PROVIDER="openai"
    export SIYUAN_EMBEDDING_MODEL="text-embedding-3-small"
    export SIYUAN_EMBEDDING_API_KEY="$OPENAI_API_KEY"
    export SIYUAN_EMBEDDING_API_BASE_URL="https://api.openai.com/v1/embeddings"
    export SIYUAN_EMBEDDING_ENCODING_FORMAT="float"
    export SIYUAN_EMBEDDING_TIMEOUT="30"

    echo "⚠️  请设置您的 OpenAI API 密钥:"
    echo "   export OPENAI_API_KEY=your-openai-api-key"
    echo ""
    echo "✅ OpenAI 完整服务配置完成"
}

# 混合高级配置
hybrid_premium() {
    echo "🔧 设置混合高级配置 (OpenAI LLM + SiliconFlow向量化)..."

    # OpenAI LLM配置
    export OPENAI_API_KEY="your-openai-api-key"  # 需要用户填入真实密钥
    export SIYUAN_LLM_PROVIDER="openai"
    export SIYUAN_LLM_MODEL="gpt-4o"
    export SIYUAN_LLM_TEMPERATURE="0.7"
    export SIYUAN_LLM_MAX_TOKENS="8000"
    export SIYUAN_LLM_TIMEOUT="60"
    export SIYUAN_LLM_API_BASE_URL="https://api.openai.com/v1"

    # SiliconFlow向量化配置
    export SIYUAN_EMBEDDING_PROVIDER="siliconflow"
    export SIYUAN_EMBEDDING_MODEL="BAAI/bge-m3"
    export SIYUAN_EMBEDDING_API_BASE_URL="https://api.siliconflow.cn/v1/embeddings"
    export SIYUAN_EMBEDDING_ENCODING_FORMAT="float"
    export SIYUAN_EMBEDDING_TIMEOUT="30"

    echo "⚠️  请设置API密钥:"
    echo "   export OPENAI_API_KEY=your-openai-api-key"
    echo "   export SIYUAN_EMBEDDING_API_KEY=your-siliconflow-api-key"
    echo ""
    echo "✅ 混合高级配置完成 (最佳性能)"
}

# 经济配置
economy_config() {
    echo "🔧 设置经济配置..."

    # GPT-3.5 LLM配置
    export OPENAI_API_KEY="your-openai-api-key"  # 需要用户填入真实密钥
    export SIYUAN_LLM_PROVIDER="openai"
    export SIYUAN_LLM_MODEL="gpt-3.5-turbo"
    export SIYUAN_LLM_TEMPERATURE="0.8"
    export SIYUAN_LLM_MAX_TOKENS="2000"
    export SIYUAN_LLM_TIMEOUT="20"

    # OpenAI经济向量化配置
    export SIYUAN_EMBEDDING_PROVIDER="openai"
    export SIYUAN_EMBEDDING_MODEL="text-embedding-3-small"
    export SIYUAN_EMBEDDING_API_KEY="$OPENAI_API_KEY"
    export SIYUAN_EMBEDDING_ENCODING_FORMAT="float"
    export SIYUAN_EMBEDDING_TIMEOUT="20"

    echo "⚠️  请设置您的 OpenAI API 密钥:"
    echo "   export OPENAI_API_KEY=your-openai-api-key"
    echo ""
    echo "✅ 经济配置完成 (最低成本)"
}

# 性能配置
performance_config() {
    echo "🔧 设置性能配置..."

    # GPT-4 LLM配置
    export OPENAI_API_KEY="your-openai-api-key"  # 需要用户填入真实密钥
    export SIYUAN_LLM_PROVIDER="openai"
    export SIYUAN_LLM_MODEL="gpt-4o"
    export SIYUAN_LLM_TEMPERATURE="0.5"
    export SIYUAN_LLM_MAX_TOKENS="8000"
    export SIYUAN_LLM_TIMEOUT="90"
    export SIYUAN_LLM_MAX_CONTEXTS="10"

    # 高级向量化配置
    export SIYUAN_EMBEDDING_PROVIDER="openai"
    export SIYUAN_EMBEDDING_MODEL="text-embedding-3-large"
    export SIYUAN_EMBEDDING_API_KEY="$OPENAI_API_KEY"
    export SIYUAN_EMBEDDING_ENCODING_FORMAT="float"
    export SIYUAN_EMBEDDING_TIMEOUT="60"

    echo "⚠️  请设置您的 OpenAI API 密钥:"
    echo "   export OPENAI_API_KEY=your-openai-api-key"
    echo ""
    echo "✅ 性能配置完成 (最佳质量)"
}

# 显示当前配置
show_current() {
    echo "📊 当前AI配置状态:"
    echo "=================="
    echo ""
    echo "📝 LLM/对话配置:"
    echo "   API密钥: ${OPENAI_API_KEY:+已设置} ${OPENAI_API_KEY:-未设置}"
    echo "   提供商: ${SIYUAN_LLM_PROVIDER:-OpenAI}"
    echo "   模型: ${SIYUAN_LLM_MODEL:-gpt-3.5-turbo}"
    echo "   温度: ${SIYUAN_LLM_TEMPERATURE:-1.0}"
    echo "   最大令牌: ${SIYUAN_LLM_MAX_TOKENS:-4000}"
    echo "   超时: ${SIYUAN_LLM_TIMEOUT:-30}秒"
    echo "   API地址: ${SIYUAN_LLM_API_BASE_URL:-https://api.openai.com/v1}"
    echo ""
    echo "🔍 向量化配置:"
    echo "   API密钥: ${SIYUAN_EMBEDDING_API_KEY:+已设置} ${SIYUAN_EMBEDDING_API_KEY:-未设置}"
    echo "   提供商: ${SIYUAN_EMBEDDING_PROVIDER:-siliconflow}"
    echo "   模型: ${SIYUAN_EMBEDDING_MODEL:-BAAI/bge-large-zh-v1.5}"
    echo "   API地址: ${SIYUAN_EMBEDDING_API_BASE_URL:-https://api.siliconflow.cn/v1/embeddings}"
    echo "   编码格式: ${SIYUAN_EMBEDDING_ENCODING_FORMAT:-float}"
    echo "   超时: ${SIYUAN_EMBEDDING_TIMEOUT:-30}秒"
    echo ""
}

# 清除配置
clear_config() {
    echo "🧹 清除所有AI配置..."

    # 清除LLM配置
    unset OPENAI_API_KEY SIYUAN_OPENAI_API_KEY
    unset SIYUAN_LLM_PROVIDER SIYUAN_LLM_MODEL SIYUAN_LLM_TEMPERATURE
    unset SIYUAN_LLM_MAX_TOKENS SIYUAN_LLM_TIMEOUT SIYUAN_LLM_MAX_CONTEXTS
    unset SIYUAN_LLM_API_BASE_URL SIYUAN_LLM_PROXY SIYUAN_LLM_USER_AGENT
    unset SIYUAN_LLM_API_VERSION

    # 清除向量化配置
    unset SIYUAN_EMBEDDING_API_KEY SIYUAN_EMBEDDING_PROVIDER
    unset SIYUAN_EMBEDDING_MODEL SIYUAN_EMBEDDING_API_BASE_URL
    unset SIYUAN_EMBEDDING_ENCODING_FORMAT SIYUAN_EMBEDDING_TIMEOUT

    # 清除旧的环境变量
    unset SIYUAN_OPENAI_API_TIMEOUT SIYUAN_OPENAI_API_PROXY
    unset SIYUAN_OPENAI_API_MAX_TOKENS SIYUAN_OPENAI_API_TEMPERATURE
    unset SIYUAN_OPENAI_API_MAX_CONTEXTS SIYUAN_OPENAI_API_BASE_URL
    unset SIYUAN_OPENAI_API_USER_AGENT SIYUAN_OPENAI_API_VERSION

    echo "✅ 所有AI配置已清除"
}

# 主菜单
case "${1:-}" in
    "siliconflow-only")
        siliconflow_only
        ;;
    "openai-only")
        openai_only
        ;;
    "hybrid-premium")
        hybrid_premium
        ;;
    "economy-config")
        economy_config
        ;;
    "performance-config")
        performance_config
        ;;
    "current-config"|"current")
        show_current
        ;;
    "clear")
        clear_config
        ;;
    *)
        show_examples
        echo "🎯 使用方法:"
        echo "   source ai-config-examples.sh [配置名称]"
        echo ""
        echo "📝 示例:"
        echo "   source ai-config-examples.sh hybrid-premium"
        echo "   source ai-config-examples.sh current-config"
        echo "   source ai-config-examples.sh clear"
        ;;
esac