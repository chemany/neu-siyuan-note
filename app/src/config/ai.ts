import { fetchPost } from "../util/fetch";

export const ai = {
    element: undefined as Element,
    genHTML: () => {
        let responsiveHTML = "";
        /// #if MOBILE
        responsiveHTML = `<div class="b3-label">
    AI服务提供商
    <div class="b3-label__text">
        选择AI对话服务提供商，支持多种主流AI模型
    </div>
    <div class="b3-label__text fn__flex config__item">
        <select id="apiProvider" class="b3-select">
            <option value="builtin" ${window.siyuan.config.ai.openAI.apiProvider === "builtin" ? "selected" : ""}>🎁 内置免费模型（推荐）</option>
            <option value="OpenAI" ${window.siyuan.config.ai.openAI.apiProvider === "OpenAI" ? "selected" : ""}>OpenAI</option>
            <option value="Azure" ${window.siyuan.config.ai.openAI.apiProvider === "Azure" ? "selected" : ""}>Azure OpenAI</option>
            <option value="SiliconFlow" ${window.siyuan.config.ai.openAI.apiProvider === "SiliconFlow" ? "selected" : ""}硅基流动 SiliconFlow</option>
            <option value="Qwen" ${window.siyuan.config.ai.openAI.apiProvider === "Qwen" ? "selected" : ""}>阿里通义千问</option>
            <option value="ZhipuAI" ${window.siyuan.config.ai.openAI.apiProvider === "ZhipuAI" ? "selected" : ""}>智谱AI (ChatGLM)</option>
            <option value="DeepSeek" ${window.siyuan.config.ai.openAI.apiProvider === "DeepSeek" ? "selected" : ""}>DeepSeek</option>
            <option value="Moonshot" ${window.siyuan.config.ai.openAI.apiProvider === "Moonshot" ? "selected" : ""}>月之暗面 (Kimi)</option>
            <option value="Custom" ${window.siyuan.config.ai.openAI.apiProvider === "Custom" ? "selected" : ""}>自定义API</option>
        </select>
    </div>
</div>
<div class="b3-label${window.siyuan.config.ai.openAI.apiProvider === "builtin" ? " fn__none" : ""}" data-field="apiKey">
    API密钥
    <div class="fn__hr"></div>
    <div class="b3-form__icona fn__block">
        <input id="apiKey" type="password" class="b3-text-field b3-form__icona-input" value="${window.siyuan.config.ai.openAI.apiKey}" placeholder="sk-xxx">
        <svg class="b3-form__icona-icon" data-action="togglePassword"><use xlink:href="#iconEye"></use></svg>
    </div>
    <div class="b3-label__text">您的API密钥${ai.getProviderKeyTip()}</div>
</div>
<div class="b3-label">
    AI模型
    <div class="b3-label__text">
        选择要使用的AI模型
    </div>
    <div class="fn__hr"></div>
    <select id="apiModel" class="b3-select fn__block">
        ${ai.getModelOptions()}
    </select>
    <div class="b3-label__text" id="modelDescription">${ai.getModelDescription()}</div>
</div>
<div class="b3-label">
    ${window.siyuan.languages.apiTimeout}
    <div class="fn__hr"></div>
    <div class="fn__flex">
        <input class="b3-text-field fn__flex-1" type="number" step="1" min="5" max="600" id="apiTimeout" value="${window.siyuan.config.ai.openAI.apiTimeout}"/>
        <span class="fn__space"></span>
        <span class="ft__on-surface fn__flex-center">s</span>
    </div>
    <div class="b3-label__text">${window.siyuan.languages.apiTimeoutTip}</div>
</div>
<div class="b3-label">
    ${window.siyuan.languages.apiMaxTokens}
    <div class="fn__hr"></div>
    <input class="b3-text-field fn__flex-center fn__block" type="number" step="1" min="0" id="apiMaxTokens" value="${window.siyuan.config.ai.openAI.apiMaxTokens}"/>
    <div class="b3-label__text">${window.siyuan.languages.apiMaxTokensTip}</div>
</div>
<div class="b3-label">
    ${window.siyuan.languages.apiTemperature}
    <div class="fn__hr"></div>
    <input class="b3-text-field fn__flex-center fn__block" type="number" step="0.1" min="0" max="2" id="apiTemperature" value="${window.siyuan.config.ai.openAI.apiTemperature}"/>
    <div class="b3-label__text">${window.siyuan.languages.apiTemperatureTip}</div>
</div>
<div class="b3-label">
    ${window.siyuan.languages.apiMaxContexts}
    <div class="fn__hr"></div>
    <input class="b3-text-field fn__flex-center fn__block" type="number" step="1" min="1" max="64" id="apiMaxContexts" value="${window.siyuan.config.ai.openAI.apiMaxContexts}"/>
    <div class="b3-label__text">${window.siyuan.languages.apiMaxContextsTip}</div>
</div>
<div class="b3-label${window.siyuan.config.ai.openAI.apiProvider === "builtin" ? " fn__none" : ""}" data-field="apiBaseURL">
    ${window.siyuan.languages.apiBaseURL}
    <div class="fn__hr"></div>
    <input class="b3-text-field fn__block" id="apiBaseURL" value="${window.siyuan.config.ai.openAI.apiBaseURL}" placeholder="${ai.getDefaultBaseURL()}"/>
    <div class="b3-label__text">${window.siyuan.languages.apiBaseURLTip}</div>
</div>
<div class="b3-label${window.siyuan.config.ai.openAI.apiProvider !== "Custom" ? " fn__none" : ""}" data-field="apiProxy">
    ${window.siyuan.languages.apiProxy}
    <div class="fn__hr"></div>
    <input class="b3-text-field fn__block" id="apiProxy" value="${window.siyuan.config.ai.openAI.apiProxy}"/>
    <div class="b3-label__text">${window.siyuan.languages.apiProxyTip}</div>
</div>
<div class="b3-label${window.siyuan.config.ai.openAI.apiProvider !== "Azure" ? " fn__none" : ""}" data-field="apiVersion">
    ${window.siyuan.languages.apiVersion}
    <div class="fn__hr"></div>
    <input class="b3-text-field fn__block" id="apiVersion" value="${window.siyuan.config.ai.openAI.apiVersion}"/>
    <div class="b3-label__text">${window.siyuan.languages.apiVersionTip}</div>
</div>
<div class="b3-label${window.siyuan.config.ai.openAI.apiProvider === "builtin" ? " fn__none" : ""}" data-field="apiUserAgent">
    User-Agent
    <div class="fn__hr"></div>
    <input class="b3-text-field fn__block" id="apiUserAgent" value="${window.siyuan.config.ai.openAI.apiUserAgent}"/>
    <div class="b3-label__text">${window.siyuan.languages.apiUserAgentTip}</div>
</div>`;
        /// #else
        responsiveHTML = `<div class="fn__flex b3-label config__item">
    <div class="fn__flex-1">
        AI服务提供商
        <div class="b3-label__text">选择AI对话服务提供商，支持多种主流AI模型</div>
    </div>
    <span class="fn__space"></span>
    <select id="apiProvider" class="b3-select fn__flex-center fn__size200">
        <option value="builtin" ${window.siyuan.config.ai.openAI.apiProvider === "builtin" ? "selected" : ""}>🎁 内置免费模型</option>
        <option value="OpenAI" ${window.siyuan.config.ai.openAI.apiProvider === "OpenAI" ? "selected" : ""}>OpenAI</option>
        <option value="Azure" ${window.siyuan.config.ai.openAI.apiProvider === "Azure" ? "selected" : ""}>Azure OpenAI</option>
        <option value="SiliconFlow" ${window.siyuan.config.ai.openAI.apiProvider === "SiliconFlow" ? "selected" : ""}>硅基流动</option>
        <option value="Qwen" ${window.siyuan.config.ai.openAI.apiProvider === "Qwen" ? "selected" : ""}>通义千问</option>
        <option value="ZhipuAI" ${window.siyuan.config.ai.openAI.apiProvider === "ZhipuAI" ? "selected" : ""}>智谱AI</option>
        <option value="DeepSeek" ${window.siyuan.config.ai.openAI.apiProvider === "DeepSeek" ? "selected" : ""}>DeepSeek</option>
        <option value="Moonshot" ${window.siyuan.config.ai.openAI.apiProvider === "Moonshot" ? "selected" : ""}>月之暗面</option>
        <option value="Custom" ${window.siyuan.config.ai.openAI.apiProvider === "Custom" ? "selected" : ""}>自定义API</option>
    </select>
</div>
<div class="fn__flex b3-label${window.siyuan.config.ai.openAI.apiProvider === "builtin" ? " fn__none" : ""}" data-field="apiKey">
    <div class="fn__block">
        API密钥
        <div class="b3-label__text">您的API密钥${ai.getProviderKeyTip()}</div>
        <div class="fn__hr"></div>
        <div class="b3-form__icona fn__block">
            <input id="apiKey" type="password" class="b3-text-field b3-form__icona-input" value="${window.siyuan.config.ai.openAI.apiKey}" placeholder="sk-xxx">
            <svg class="b3-form__icona-icon" data-action="togglePassword"><use xlink:href="#iconEye"></use></svg>
        </div>
    </div>
</div>
<div class="fn__flex b3-label">
    <div class="fn__block">
        AI模型
        <div class="b3-label__text">选择要使用的AI模型</div>
        <div class="fn__hr"></div>
        <select id="apiModel" class="b3-select fn__block">
            ${ai.getModelOptions()}
        </select>
        <div class="b3-label__text" id="modelDescription" style="margin-top: 8px;">${ai.getModelDescription()}</div>
    </div>
</div>
<div class="fn__flex b3-label">
    <div class="fn__flex-1">
        ${window.siyuan.languages.apiTimeout}
        <div class="b3-label__text">${window.siyuan.languages.apiTimeoutTip}</div>
    </div>
    <span class="fn__space"></span>
    <div class="fn__size200 fn__flex-center fn__flex">
        <input class="b3-text-field fn__flex-1" type="number" step="1" min="5" max="600" id="apiTimeout" value="${window.siyuan.config.ai.openAI.apiTimeout}"/>
        <span class="fn__space"></span>
        <span class="ft__on-surface fn__flex-center">s</span>
    </div>
</div>
<div class="fn__flex b3-label">
    <div class="fn__flex-1">
        ${window.siyuan.languages.apiMaxTokens}
        <div class="b3-label__text">${window.siyuan.languages.apiMaxTokensTip}</div>
    </div>
    <span class="fn__space"></span>
    <input class="b3-text-field fn__flex-center fn__size200" type="number" step="1" min="0" id="apiMaxTokens" value="${window.siyuan.config.ai.openAI.apiMaxTokens}"/>
</div>
<div class="fn__flex b3-label">
    <div class="fn__flex-1">
        ${window.siyuan.languages.apiTemperature}
        <div class="b3-label__text">${window.siyuan.languages.apiTemperatureTip}</div>
    </div>
    <span class="fn__space"></span>
    <input class="b3-text-field fn__flex-center fn__size200" type="number" step="0.1" min="0" max="2" id="apiTemperature" value="${window.siyuan.config.ai.openAI.apiTemperature}"/>
</div>
<div class="fn__flex b3-label">
    <div class="fn__flex-1">
        ${window.siyuan.languages.apiMaxContexts}
        <div class="b3-label__text">${window.siyuan.languages.apiMaxContextsTip}</div>
    </div>
    <span class="fn__space"></span>
    <input class="b3-text-field fn__flex-center fn__size200" type="number" step="1" min="1" max="64" id="apiMaxContexts" value="${window.siyuan.config.ai.openAI.apiMaxContexts}"/>
</div>
<div class="fn__flex b3-label${window.siyuan.config.ai.openAI.apiProvider === "builtin" ? " fn__none" : ""}" data-field="apiBaseURL">
    <div class="fn__block">
        ${window.siyuan.languages.apiBaseURL}
        <div class="b3-label__text">${window.siyuan.languages.apiBaseURLTip}</div>
        <span class="fn__hr"></span>
        <input class="b3-text-field fn__block" id="apiBaseURL" value="${window.siyuan.config.ai.openAI.apiBaseURL}" placeholder="${ai.getDefaultBaseURL()}"/>
    </div>
</div>
<div class="fn__flex b3-label${window.siyuan.config.ai.openAI.apiProvider !== "Custom" ? " fn__none" : ""}" data-field="apiProxy">
    <div class="fn__block">
        ${window.siyuan.languages.apiProxy}
        <div class="b3-label__text">${window.siyuan.languages.apiProxyTip}</div>
        <span class="fn__hr"></span>
        <input class="b3-text-field fn__block" id="apiProxy" value="${window.siyuan.config.ai.openAI.apiProxy}"/>
    </div>
</div>
<div class="fn__flex b3-label${window.siyuan.config.ai.openAI.apiProvider !== "Azure" ? " fn__none" : ""}" data-field="apiVersion">
    <div class="fn__block">
        ${window.siyuan.languages.apiVersion}
        <div class="b3-label__text">${window.siyuan.languages.apiVersionTip}</div>
        <span class="fn__hr"></span>
        <input class="b3-text-field fn__block" id="apiVersion" value="${window.siyuan.config.ai.openAI.apiVersion}"/>
    </div>
</div>
<div class="fn__flex b3-label${window.siyuan.config.ai.openAI.apiProvider === "builtin" ? " fn__none" : ""}" data-field="apiUserAgent">
    <div class="fn__block">
        User-Agent
        <div class="b3-label__text">${window.siyuan.languages.apiUserAgentTip}</div>
        <span class="fn__hr"></span>
        <input class="b3-text-field fn__block" id="apiUserAgent" value="${window.siyuan.config.ai.openAI.apiUserAgent}"/>
    </div>
</div>`;
        /// #endif
        return `<div class="fn__flex-column" style="height: 100%">
<div class="layout-tab-bar fn__flex">
    <div data-type="openai" class="item item--full item--focus"><span class="fn__flex-1"></span><span class="item__text">AI对话</span><span class="fn__flex-1"></span></div>
    <div data-type="embedding" class="item item--full"><span class="fn__flex-1"></span><span class="item__text">向量化</span><span class="fn__flex-1"></span></div>
    <div data-type="ai-features" class="item item--full"><span class="fn__flex-1"></span><span class="item__text">AI功能</span><span class="fn__flex-1"></span></div>
</div>
<div class="fn__flex-1">
    <div data-type="openai">
        ${responsiveHTML}
    </div>
    <div data-type="embedding" style="display: none;">
        ${ai.genEmbeddingHTML()}
    </div>
    <div data-type="ai-features" style="display: none;">
        ${ai.genAIFeaturesHTML()}
    </div>
</div>
</div>`;
    },
    // 获取模型选项
    getModelOptions: (providerParam?: string) => {
        const provider = providerParam || window.siyuan.config.ai.openAI.apiProvider || "builtin";
        const currentModel = window.siyuan.config.ai.openAI.apiModel || "";

        const modelGroups: { [key: string]: Array<{ value: string, label: string, desc: string }> } = {
            "builtin": [
                { value: "builtin-free", label: "思源内置免费模型", desc: "免费使用，适合日常对话" },
            ],
            "OpenAI": [
                { value: "gpt-4o", label: "GPT-4o", desc: "最新最强模型，支持多模态" },
                { value: "gpt-4o-mini", label: "GPT-4o Mini", desc: "轻量快速，性价比高" },
                { value: "gpt-4-turbo", label: "GPT-4 Turbo", desc: "强大的GPT-4模型" },
                { value: "gpt-4", label: "GPT-4", desc: "经典GPT-4模型" },
                { value: "gpt-3.5-turbo", label: "GPT-3.5 Turbo", desc: "快速且经济" },
            ],
            "Azure": [
                { value: "gpt-4", label: "GPT-4 (Azure)", desc: "Azure部署的GPT-4" },
                { value: "gpt-35-turbo", label: "GPT-3.5 Turbo (Azure)", desc: "Azure部署的GPT-3.5" },
            ],
            "SiliconFlow": [
                { value: "Qwen/Qwen2.5-72B-Instruct", label: "通义千问 2.5 (72B)", desc: "强大的中文模型" },
                { value: "Qwen/Qwen2.5-7B-Instruct", label: "通义千问 2.5 (7B)", desc: "轻量快速" },
                { value: "THUDM/glm-4-9b-chat", label: "GLM-4 (9B)", desc: "智谱最新模型" },
                { value: "deepseek-ai/DeepSeek-V2.5", label: "DeepSeek V2.5", desc: "推理能力强" },
                { value: "meta-llama/Meta-Llama-3.1-70B-Instruct", label: "Llama 3.1 (70B)", desc: "Meta开源大模型" },
                { value: "meta-llama/Meta-Llama-3.1-8B-Instruct", label: "Llama 3.1 (8B)", desc: "轻量开源模型" },
            ],
            "Qwen": [
                { value: "qwen-max", label: "通义千问 Max", desc: "最强性能" },
                { value: "qwen-plus", label: "通义千问 Plus", desc: "平衡性能" },
                { value: "qwen-turbo", label: "通义千问 Turbo", desc: "快速响应" },
            ],
            "ZhipuAI": [
                { value: "glm-4-plus", label: "GLM-4 Plus", desc: "超大规模模型" },
                { value: "glm-4", label: "GLM-4", desc: "综合性能强" },
                { value: "glm-3-turbo", label: "GLM-3 Turbo", desc: "快速高效" },
            ],
            "DeepSeek": [
                { value: "deepseek-chat", label: "DeepSeek Chat", desc: "对话模型" },
                { value: "deepseek-coder", label: "DeepSeek Coder", desc: "代码专用模型" },
            ],
            "Moonshot": [
                { value: "moonshot-v1-128k", label: "Kimi (128K)", desc: "超长上下文" },
                { value: "moonshot-v1-32k", label: "Kimi (32K)", desc: "长上下文" },
                { value: "moonshot-v1-8k", label: "Kimi (8K)", desc: "标准上下文" },
            ],
            "Custom": [
                { value: "custom-model", label: "自定义模型", desc: "兼容OpenAI API的任何模型" },
            ]
        };

        const models = modelGroups[provider] || modelGroups["builtin"];
        // 不再预选，让用户看到完整列表
        return models.map(m =>
            `<option value="${m.value}">${m.label}</option>`
        ).join("");
    },
    // 获取模型描述
    getModelDescription: (modelParam?: string, providerParam?: string) => {
        const model = modelParam || window.siyuan.config.ai.openAI.apiModel || "builtin-free";

        const descriptions: { [key: string]: string } = {
            "builtin-free": "完全免费的内置AI模型，适合日常对话和文档分析",
            "gpt-4o": "OpenAI最新最强模型，支持视觉理解和多模态输入",
            "gpt-4o-mini": "轻量版GPT-4o，响应更快，成本更低",
            "gpt-4-turbo": "GPT-4的优化版本，处理速度更快",
            "gpt-4": "经典的GPT-4模型，强大的推理能力",
            "gpt-3.5-turbo": "性价比最高的选择，响应迅速",
            "gpt-35-turbo": "Azure部署的GPT-3.5模型",
            "Qwen/Qwen2.5-72B-Instruct": "阿里最强中文模型，理解能力出色",
            "Qwen/Qwen2.5-7B-Instruct": "轻量快速，适合日常使用",
            "THUDM/glm-4-9b-chat": "智谱最新对话模型，中文表现优秀",
            "deepseek-ai/DeepSeek-V2.5": "强大的推理和代码能力",
            "qwen-max": "通义千问最强模型，综合能力突出",
            "qwen-plus": "通义千问平衡性能模型",
            "qwen-turbo": "通义千问快速响应模型",
            "glm-4-plus": "智谱AI超大模型，处理复杂任务",
            "glm-4": "智谱AI综合性能模型",
            "glm-3-turbo": "智谱AI快速模型",
            "deepseek-chat": "DeepSeek通用对话模型",
            "deepseek-coder": "DeepSeek代码专用模型",
            "moonshot-v1-128k": "支持128K超长上下文，适合长文档分析",
            "moonshot-v1-32k": "支持32K长上下文",
            "moonshot-v1-8k": "支持8K标准上下文",
            "custom-model": "兼容OpenAI API的自定义模型",
            "meta-llama/Meta-Llama-3.1-70B-Instruct": "Meta开源大模型，通用能力强",
            "meta-llama/Meta-Llama-3.1-8B-Instruct": "轻量级开源模型",
        };

        return descriptions[model] || "请选择一个AI模型";
    },
    // 获取默认的Base URL
    getDefaultBaseURL: (providerParam?: string) => {
        const provider = providerParam || window.siyuan.config.ai.openAI.apiProvider || "builtin";
        const urls: { [key: string]: string } = {
            "OpenAI": "https://api.openai.com/v1",
            "Azure": "https://YOUR_RESOURCE.openai.azure.com",
            "SiliconFlow": "https://api.siliconflow.cn/v1",
            "Qwen": "https://dashscope.aliyuncs.com/compatible-mode/v1",
            "ZhipuAI": "https://open.bigmodel.cn/api/paas/v4",
            "DeepSeek": "https://api.deepseek.com/v1",
            "Moonshot": "https://api.moonshot.cn/v1",
            "Custom": "https://your-api-endpoint.com/v1",
        };
        return urls[provider] || "";
    },
    // 获取API密钥提示
    getProviderKeyTip: (providerParam?: string) => {
        const provider = providerParam || window.siyuan.config.ai.openAI.apiProvider || "builtin";
        const tips: { [key: string]: string } = {
            "OpenAI": "，在 platform.openai.com 获取",
            "Azure": "，在Azure门户获取",
            "SiliconFlow": "，在 siliconflow.cn 免费获取",
            "Qwen": "，在阿里云控制台获取",
            "ZhipuAI": "，在 open.bigmodel.cn 获取",
            "DeepSeek": "，在 platform.deepseek.com 获取",
            "Moonshot": "，在 platform.moonshot.cn 获取",
        };
        return tips[provider] || "";
    },
    genEmbeddingHTML: () => {
        const embeddingConfig = window.siyuan.config.ai?.embedding || {
            provider: "siliconflow",
            apiKey: "",
            model: "BAAI/bge-large-zh-v1.5",
            apiBaseUrl: "https://api.siliconflow.cn/v1/embeddings",
            encodingFormat: "float",
            timeout: 30,
            enabled: false
        };

        return `<div class="fn__flex b3-label config__item">
    <div class="fn__flex-1">
        向量化提供商
        <div class="b3-label__text">选择向量化服务提供商，支持多种AI模型</div>
    </div>
    <span class="fn__space"></span>
    <select id="embeddingProvider" class="b3-select fn__flex-center fn__size200">
        <option value="siliconflow" ${embeddingConfig.provider === "siliconflow" ? "selected" : ""}>SiliconFlow</option>
        <option value="openai" ${embeddingConfig.provider === "openai" ? "selected" : ""}>OpenAI</option>
    </select>
</div>
<div class="fn__flex b3-label">
    <div class="fn__flex-1">
        启用向量化功能
        <div class="b3-label__text">开启后可使用语义搜索和AI分析功能</div>
    </div>
    <span class="fn__space"></span>
    <input type="checkbox" id="embeddingEnabled" class="b3-switch" ${embeddingConfig.enabled ? "checked" : ""}/>
</div>
<div class="fn__flex b3-label">
    <div class="fn__block">
        API密钥
        <div class="b3-label__text">向量化服务的API密钥</div>
        <div class="fn__hr"></div>
        <div class="b3-form__icona fn__block">
            <input id="embeddingApiKey" type="password" class="b3-text-field b3-form__icona-input" value="${embeddingConfig.apiKey}" placeholder="sk-xxx">
            <svg class="b3-form__icona-icon" data-action="toggleEmbeddingPassword"><use xlink:href="#iconEye"></use></svg>
        </div>
    </div>
</div>
<div class="fn__flex b3-label">
    <div class="fn__block">
        向量化模型
        <div class="b3-label__text">选择用于向量化的AI模型</div>
        <div class="fn__hr"></div>
        <select id="embeddingModel" class="b3-text-field fn__block">
            <option value="BAAI/bge-large-zh-v1.5" ${embeddingConfig.model === "BAAI/bge-large-zh-v1.5" ? "selected" : ""}>BAAI/bge-large-zh-v1.5 (中文大型)</option>
            <option value="BAAI/bge-m3" ${embeddingConfig.model === "BAAI/bge-m3" ? "selected" : ""}>BAAI/bge-m3 (多语言)</option>
            <option value="netease-youdao/bce-embedding-base_v1" ${embeddingConfig.model === "netease-youdao/bce-embedding-base_v1" ? "selected" : ""}>BCE-Embedding-Base</option>
            <option value="text-embedding-3-small" ${embeddingConfig.model === "text-embedding-3-small" ? "selected" : ""}>text-embedding-3-small (OpenAI)</option>
            <option value="text-embedding-3-large" ${embeddingConfig.model === "text-embedding-3-large" ? "selected" : ""}>text-embedding-3-large (OpenAI)</option>
            <option value="text-embedding-ada-002" ${embeddingConfig.model === "text-embedding-ada-002" ? "selected" : ""}>text-embedding-ada-002 (OpenAI)</option>
        </select>
    </div>
</div>
<div class="fn__flex b3-label">
    <div class="fn__block">
        API地址
        <div class="b3-label__text">向量化服务的API端点地址</div>
        <div class="fn__hr"></div>
        <input class="b3-text-field fn__block" id="embeddingApiBaseUrl" value="${embeddingConfig.apiBaseUrl}" placeholder="https://api.siliconflow.cn/v1/embeddings"/>
    </div>
</div>
<div class="fn__flex b3-label">
    <div class="fn__flex-1">
        编码格式
        <div class="b3-label__text">向量数据的编码格式</div>
    </div>
    <span class="fn__space"></span>
    <select id="embeddingEncodingFormat" class="b3-select fn__flex-center fn__size200">
        <option value="float" ${embeddingConfig.encodingFormat === "float" ? "selected" : ""}>float</option>
        <option value="base64" ${embeddingConfig.encodingFormat === "base64" ? "selected" : ""}>base64</option>
    </select>
</div>
<div class="fn__flex b3-label">
    <div class="fn__flex-1">
        请求超时时间
        <div class="b3-label__text">向量化请求的最大等待时间</div>
    </div>
    <span class="fn__space"></span>
    <div class="fn__size200 fn__flex-center fn__flex">
        <input class="b3-text-field fn__flex-1" type="number" step="1" min="5" max="300" id="embeddingTimeout" value="${embeddingConfig.timeout}"/>
        <span class="fn__space"></span>
        <span class="ft__on-surface fn__flex-center">秒</span>
    </div>
</div>
<div class="fn__flex b3-label">
    <div class="fn__block">
        <button id="testEmbeddingConnection" class="b3-button b3-button--outline">测试连接</button>
        <span class="fn__space"></span>
        <button id="getEmbeddingModels" class="b3-button b3-button--outline">获取可用模型</button>
    </div>
</div>`;
    },
    genAIFeaturesHTML: () => {
        return `<div class="b3-label">
    <h3>AI功能测试</h3>
    <div class="b3-label__text">测试新增的AI向量化和分析功能</div>
</div>
<div class="b3-label">
    <div class="fn__block">
        <button id="semanticSearchTest" class="b3-button b3-button--outline">语义搜索测试</button>
        <span class="fn__space"></span>
        <input id="semanticSearchQuery" class="b3-text-field fn__flex-1" placeholder="输入搜索关键词..." value="人工智能">
    </div>
</div>
<div class="b3-label">
    <div class="fn__block">
        <button id="notebookSummaryTest" class="b3-button b3-button--outline">笔记本摘要生成测试</button>
        <span class="fn__space"></span>
        <input id="notebookSummaryId" class="b3-text-field fn__flex-1" placeholder="输入笔记本ID...">
    </div>
</div>
<div class="b3-label">
    <div class="fn__block">
        <button id="batchVectorizeTest" class="b3-button b3-button--outline">批量向量化测试</button>
        <span class="fn__space"></span>
        <input id="batchVectorizeNotebook" class="b3-text-field fn__flex-1" placeholder="输入笔记本ID...">
    </div>
</div>
<div class="b3-label">
    <div class="fn__block">
        <div id="aiTestResults" class="b3-form__space-small" style="background: var(--b3-theme-background-contrast); padding: 12px; border-radius: 4px; min-height: 60px; font-family: monospace; white-space: pre-wrap;">测试结果将在这里显示...</div>
    </div>
</div>`;
    },
    bindEvent: () => {
        // 标签页切换事件
        ai.element.querySelectorAll(".layout-tab-bar .item").forEach((item) => {
            item.addEventListener("click", () => {
                const type = item.getAttribute("data-type");
                ai.element.querySelectorAll(".layout-tab-bar .item").forEach((tabItem) => {
                    tabItem.classList.remove("item--focus");
                });
                item.classList.add("item--focus");

                ai.element.querySelectorAll(".fn__flex-1 > div").forEach((contentItem) => {
                    if (contentItem.getAttribute("data-type") === type) {
                        contentItem.style.display = "block";
                    } else {
                        contentItem.style.display = "none";
                    }
                });
            });
        });

        // 服务商切换时更新界面
        const apiProviderSelect = ai.element.querySelector("#apiProvider") as HTMLSelectElement;
        if (apiProviderSelect) {
            apiProviderSelect.addEventListener("change", () => {
                const provider = apiProviderSelect.value;

                // 更新模型选项 - 传入当前provider
                const modelSelect = ai.element.querySelector("#apiModel") as HTMLSelectElement;
                if (modelSelect) {
                    modelSelect.innerHTML = ai.getModelOptions(provider);
                    // 选中第一个模型
                    if (modelSelect.options.length > 0) {
                        modelSelect.selectedIndex = 0;
                        // 更新模型描述
                        const modelDesc = ai.element.querySelector("#modelDescription");
                        if (modelDesc) {
                            modelDesc.textContent = ai.getModelDescription(modelSelect.value, provider);
                        }
                    }
                }

                // 更新Base URL占位符 - 传入当前provider
                const baseURLInput = ai.element.querySelector("#apiBaseURL") as HTMLInputElement;
                if (baseURLInput) {
                    baseURLInput.placeholder = ai.getDefaultBaseURL(provider);
                }

                // 显示/隐藏相关字段
                const fields = ["apiKey", "apiBaseURL", "apiProxy", "apiVersion", "apiUserAgent"];
                fields.forEach(field => {
                    const fieldElement = ai.element.querySelector(`[data-field="${field}"]`);
                    if (fieldElement) {
                        if (provider === "builtin") {
                            fieldElement.classList.add("fn__none");
                        } else if (field === "apiProxy" && provider !== "Custom") {
                            fieldElement.classList.add("fn__none");
                        } else if (field === "apiVersion" && provider !== "Azure") {
                            fieldElement.classList.add("fn__none");
                        } else {
                            fieldElement.classList.remove("fn__none");
                        }
                    }
                });
            });
        }

        // 模型切换时更新描述
        const apiModelSelect = ai.element.querySelector("#apiModel") as HTMLSelectElement;
        if (apiModelSelect) {
            apiModelSelect.addEventListener("change", () => {
                const modelDesc = ai.element.querySelector("#modelDescription");
                if (modelDesc) {
                    modelDesc.textContent = ai.getModelDescription(apiModelSelect.value);
                }
            });
        }

        // 密码显示/隐藏切换
        const togglePassword = ai.element.querySelector('.b3-form__icona-icon[data-action="togglePassword"]');
        if (togglePassword) {
            togglePassword.addEventListener("click", () => {
                const isEye = togglePassword.firstElementChild.getAttribute("xlink:href") === "#iconEye";
                togglePassword.firstElementChild.setAttribute("xlink:href", isEye ? "#iconEyeoff" : "#iconEye");
                togglePassword.previousElementSibling.setAttribute("type", isEye ? "text" : "password");
            });
        }

        // 向量化密码显示/隐藏切换
        const toggleEmbeddingPassword = ai.element.querySelector('.b3-form__icona-icon[data-action="toggleEmbeddingPassword"]');
        if (toggleEmbeddingPassword) {
            toggleEmbeddingPassword.addEventListener("click", () => {
                const isEye = toggleEmbeddingPassword.firstElementChild.getAttribute("xlink:href") === "#iconEye";
                toggleEmbeddingPassword.firstElementChild.setAttribute("xlink:href", isEye ? "#iconEyeoff" : "#iconEye");
                toggleEmbeddingPassword.previousElementSibling.setAttribute("type", isEye ? "text" : "password");
            });
        }

        // OpenAI配置变更事件
        ai.element.querySelectorAll("#apiKey, #apiModel, #apiMaxTokens, #apiTemperature, #apiMaxContexts, #apiProxy, #apiTimeout, #apiProvider, #apiBaseURL, #apiVersion, #apiUserAgent").forEach((item) => {
            item.addEventListener("change", () => {
                const provider = (ai.element.querySelector("#apiProvider") as HTMLSelectElement).value;
                const model = (ai.element.querySelector("#apiModel") as HTMLSelectElement).value;

                // 如果选择了内置免费模型，使用特殊标记
                const isBuiltinFree = provider === "builtin";

                const configData = {
                    openAI: {
                        apiUserAgent: (ai.element.querySelector("#apiUserAgent") as HTMLInputElement).value,
                        apiBaseURL: isBuiltinFree ? "USE_DEFAULT_CONFIG" : (ai.element.querySelector("#apiBaseURL") as HTMLInputElement).value,
                        apiVersion: (ai.element.querySelector("#apiVersion") as HTMLInputElement).value,
                        apiKey: isBuiltinFree ? "USE_DEFAULT_CONFIG" : (ai.element.querySelector("#apiKey") as HTMLInputElement).value,
                        apiModel: isBuiltinFree ? "USE_DEFAULT_CONFIG" : model,
                        apiMaxTokens: parseInt((ai.element.querySelector("#apiMaxTokens") as HTMLInputElement).value),
                        apiTemperature: parseFloat((ai.element.querySelector("#apiTemperature") as HTMLInputElement).value),
                        apiMaxContexts: parseInt((ai.element.querySelector("#apiMaxContexts") as HTMLInputElement).value),
                        apiProxy: (ai.element.querySelector("#apiProxy") as HTMLInputElement).value,
                        apiTimeout: parseInt((ai.element.querySelector("#apiTimeout") as HTMLInputElement).value),
                        apiProvider: provider,
                    }
                };

                fetchPost("/api/setting/setAI", configData, response => {
                    window.siyuan.config.ai = response.data;

                    // 如果是内置免费模型，同时保存到统一认证服务
                    if (isBuiltinFree && window.siyuan.config.system?.container === "web") {
                        // 获取JWT token
                        const jwtToken = localStorage.getItem('siyuan_jwt_token');
                        if (jwtToken) {
                            // 保存到用户设置
                            fetch(`${window.siyuan.config.system.unifiedAuthServiceUrl || 'http://localhost:3002'}/api/settings/neuralink_llm`, {
                                method: 'POST',
                                headers: {
                                    'Content-Type': 'application/json',
                                    'Authorization': `Bearer ${jwtToken}`
                                },
                                body: JSON.stringify({
                                    provider: "builtin",
                                    model: "USE_DEFAULT_CONFIG",
                                    api_key: "USE_DEFAULT_CONFIG",
                                    base_url: "USE_DEFAULT_CONFIG"
                                })
                            }).then(res => res.json()).then(data => {
                                if (data.success) {
                                    console.log('内置免费模型配置已保存到用户设置');
                                }
                            }).catch(err => {
                                console.error('保存用户设置失败:', err);
                            });
                        }
                    }
                });
            });
        });

        // 向量化配置变更事件
        ai.element.querySelectorAll("#embeddingProvider, #embeddingApiKey, #embeddingModel, #embeddingApiBaseUrl, #embeddingEncodingFormat, #embeddingTimeout, #embeddingEnabled").forEach((item) => {
            item.addEventListener("change", () => {
                const embeddingConfig = {
                    provider: (ai.element.querySelector("#embeddingProvider") as HTMLSelectElement).value,
                    apiKey: (ai.element.querySelector("#embeddingApiKey") as HTMLInputElement).value,
                    model: (ai.element.querySelector("#embeddingModel") as HTMLSelectElement).value,
                    apiBaseUrl: (ai.element.querySelector("#embeddingApiBaseUrl") as HTMLInputElement).value,
                    encodingFormat: (ai.element.querySelector("#embeddingEncodingFormat") as HTMLSelectElement).value,
                    timeout: parseInt((ai.element.querySelector("#embeddingTimeout") as HTMLInputElement).value),
                    enabled: (ai.element.querySelector("#embeddingEnabled") as HTMLInputElement).checked,
                };

                fetchPost("/api/ai/setEmbeddingConfig", embeddingConfig, response => {
                    if (response.code === 0) {
                        window.siyuan.config.ai.embedding = response.data.data;
                        ai.showMessage("向量化配置已保存", "success");
                    } else {
                        ai.showMessage("保存失败: " + response.msg, "error");
                    }
                });
            });
        });

        // AI功能测试事件
        const testEmbeddingConnection = ai.element.querySelector("#testEmbeddingConnection");
        if (testEmbeddingConnection) {
            testEmbeddingConnection.addEventListener("click", () => {
                ai.showMessage("正在测试向量化服务连接...", "info");
                fetchPost("/api/ai/testEmbeddingConnection", {}, response => {
                    const resultsDiv = ai.element.querySelector("#aiTestResults") as HTMLDivElement;
                    if (response.code === 0) {
                        resultsDiv.textContent = "✅ 连接测试成功!\n" + JSON.stringify(response.data, null, 2);
                        ai.showMessage("连接测试成功", "success");
                    } else {
                        resultsDiv.textContent = "❌ 连接测试失败: " + response.msg;
                        ai.showMessage("连接测试失败", "error");
                    }
                });
            });
        }

        const getEmbeddingModels = ai.element.querySelector("#getEmbeddingModels");
        if (getEmbeddingModels) {
            getEmbeddingModels.addEventListener("click", () => {
                const provider = (ai.element.querySelector("#embeddingProvider") as HTMLSelectElement).value;
                ai.showMessage(`正在获取${provider}的可用模型...`, "info");
                fetchPost("/api/ai/getEmbeddingModels", { provider }, response => {
                    const resultsDiv = ai.element.querySelector("#aiTestResults") as HTMLDivElement;
                    if (response.code === 0) {
                        resultsDiv.textContent = `✅ ${provider}可用模型:\n` + JSON.stringify(response.data, null, 2);
                        ai.showMessage("模型列表获取成功", "success");
                    } else {
                        resultsDiv.textContent = "❌ 获取模型列表失败: " + response.msg;
                        ai.showMessage("获取模型列表失败", "error");
                    }
                });
            });
        }

        const semanticSearchTest = ai.element.querySelector("#semanticSearchTest");
        if (semanticSearchTest) {
            semanticSearchTest.addEventListener("click", () => {
                const query = (ai.element.querySelector("#semanticSearchQuery") as HTMLInputElement).value;
                if (!query.trim()) {
                    ai.showMessage("请输入搜索查询", "warning");
                    return;
                }

                ai.showMessage("正在执行语义搜索...", "info");
                fetchPost("/api/ai/semanticSearch", { query, limit: 10 }, response => {
                    const resultsDiv = ai.element.querySelector("#aiTestResults") as HTMLDivElement;
                    if (response.code === 0) {
                        resultsDiv.textContent = "✅ 语义搜索结果:\n" + JSON.stringify(response.data, null, 2);
                        ai.showMessage("语义搜索完成", "success");
                    } else {
                        resultsDiv.textContent = "❌ 语义搜索失败: " + response.msg;
                        ai.showMessage("语义搜索失败", "error");
                    }
                });
            });
        }

        const notebookSummaryTest = ai.element.querySelector("#notebookSummaryTest");
        if (notebookSummaryTest) {
            notebookSummaryTest.addEventListener("click", () => {
                const notebookId = (ai.element.querySelector("#notebookSummaryId") as HTMLInputElement).value;
                if (!notebookId.trim()) {
                    ai.showMessage("请输入笔记本ID", "warning");
                    return;
                }

                ai.showMessage("正在生成笔记本摘要...", "info");
                fetchPost("/api/ai/generateNotebookSummary", { notebookId }, response => {
                    const resultsDiv = ai.element.querySelector("#aiTestResults") as HTMLDivElement;
                    if (response.code === 0) {
                        resultsDiv.textContent = "✅ 笔记本摘要:\n" + JSON.stringify(response.data, null, 2);
                        ai.showMessage("摘要生成完成", "success");
                    } else {
                        resultsDiv.textContent = "❌ 摘要生成失败: " + response.msg;
                        ai.showMessage("摘要生成失败", "error");
                    }
                });
            });
        }

        const batchVectorizeTest = ai.element.querySelector("#batchVectorizeTest");
        if (batchVectorizeTest) {
            batchVectorizeTest.addEventListener("click", () => {
                const notebookId = (ai.element.querySelector("#batchVectorizeNotebook") as HTMLInputElement).value;
                if (!notebookId.trim()) {
                    ai.showMessage("请输入笔记本ID", "warning");
                    return;
                }

                if (!confirm("批量向量化会消耗API额度，确定要继续吗？")) {
                    return;
                }

                ai.showMessage("正在执行批量向量化，请稍候...", "info");
                fetchPost("/api/ai/batchVectorizeNotebook", { notebookId }, response => {
                    const resultsDiv = ai.element.querySelector("#aiTestResults") as HTMLDivElement;
                    if (response.code === 0) {
                        resultsDiv.textContent = "✅ 批量向量化完成:\n" + JSON.stringify(response.data, null, 2);
                        ai.showMessage("批量向量化完成", "success");
                    } else {
                        resultsDiv.textContent = "❌ 批量向量化失败: " + response.msg;
                        ai.showMessage("批量向量化失败", "error");
                    }
                });
            });
        }
    },
    showMessage: (message: string, type: "success" | "error" | "warning" | "info" = "info") => {
        // 显示消息提示的简单实现
        const messageDiv = document.createElement("div");
        messageDiv.className = `b3-dialog__message b3-dialog__message--${type}`;
        messageDiv.textContent = message;
        messageDiv.style.position = "fixed";
        messageDiv.style.top = "20px";
        messageDiv.style.right = "20px";
        messageDiv.style.zIndex = "1000";
        document.body.appendChild(messageDiv);

        setTimeout(() => {
            if (messageDiv.parentNode) {
                messageDiv.parentNode.removeChild(messageDiv);
            }
        }, 3000);
    },
};
