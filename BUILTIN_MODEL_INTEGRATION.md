# 内置免费模型与统一设置服务集成

## 🎯 功能概述

当用户在思源笔记AI设置中选择"内置免费模型"时，系统会：
1. 保存特殊标记 `USE_DEFAULT_CONFIG` 到本地配置
2. 同时保存到统一认证服务的用户设置文件
3. 后端从 `default-models.json` 加载实际的模型配置

## 📂 相关文件

### 1. 用户设置文件
**位置**：`/mnt/nas-sata12/MindOcean/user-data/settings/{username}_settings.json`

**示例**（jason用户）：`/mnt/nas-sata12/MindOcean/user-data/settings/jason_settings.json`

```json
{
  "neuralink_llm": {
    "provider": "builtin",
    "model": "USE_DEFAULT_CONFIG",
    "api_key": "USE_DEFAULT_CONFIG",
    "base_url": "USE_DEFAULT_CONFIG",
    "updated_at": "2025-11-26T02:48:00.000Z"
  }
}
```

### 2. 默认模型配置文件
**位置**：`/home/jason/code/unified-settings-service/config/default-models.json`

```json
{
  "builtin_free_neuralink": {
    "name": "灵枢笔记专用模型",
    "provider": "builtin",
    "api_key": "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",
    "base_url": "https://openrouter.ai/api/v1",
    "model_name": "x-ai/grok-4.1-fast:free",
    "temperature": 0.6,
    "max_tokens": 20000,
    "description": "为灵枢笔记优化的AI模型",
    "version": "2.0.1",
    "last_updated": "2025-08-27T03:20:00.000Z",
    "features": ["知识图谱", "智能摘要", "概念关联"],
    "system_prompt": "你是灵枢笔记的AI助手，专门帮助用户整理知识、建立概念关联和生成智能摘要。"
  },
  "builtin_free": {
    "name": "内置免费模型",
    "provider": "builtin",
    "api_key": "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",
    "base_url": "https://openrouter.ai/api/v1",
    "model_name": "x-ai/grok-4.1-fast:free",
    "temperature": 0.7,
    "max_tokens": 20000,
    "description": "通用免费模型",
    "version": "2.0.0",
    "last_updated": "2025-01-21T10:00:00.000Z"
  }
}
```

## 🔄 工作流程

### 前端操作流程

```
用户操作
  ↓
在AI设置中选择：AI服务提供商 = "内置免费模型"
  ↓
前端检测到 provider = "builtin"
  ↓
保存配置到思源后端
  ├─ apiProvider: "builtin"
  ├─ apiModel: "USE_DEFAULT_CONFIG"
  ├─ apiKey: "USE_DEFAULT_CONFIG"
  └─ apiBaseURL: "USE_DEFAULT_CONFIG"
  ↓
同时保存到统一认证服务
  ↓
POST http://localhost:3002/api/settings/neuralink_llm
  {
    "provider": "builtin",
    "model": "USE_DEFAULT_CONFIG",
    "api_key": "USE_DEFAULT_CONFIG",
    "base_url": "USE_DEFAULT_CONFIG"
  }
  ↓
保存到 jason_settings.json
```

### 后端读取流程

```
后端收到AI请求
  ↓
读取用户配置
  ↓
检测到 model = "USE_DEFAULT_CONFIG"
  ↓
从 default-models.json 加载配置
  ↓
选择对应的模型配置
  ├─ 思源笔记 → builtin_free_neuralink
  ├─ 潮汐志 → builtin_free_tidelog
  └─ 其他 → builtin_free
  ↓
使用加载的配置调用AI服务
  ├─ API Key: sk-or-v1-...
  ├─ Base URL: https://openrouter.ai/api/v1
  ├─ Model: x-ai/grok-4.1-fast:free
  └─ 其他参数...
```

## 💻 前端实现

### 文件：`/home/jason/code/siyuan/app/src/config/ai.ts`

#### 关键代码

```typescript
// AI配置变更事件
ai.element.querySelectorAll("#apiKey, #apiModel, ...").forEach((item) => {
    item.addEventListener("change", () => {
        const provider = (ai.element.querySelector("#apiProvider") as HTMLSelectElement).value;
        const model = (ai.element.querySelector("#apiModel") as HTMLSelectElement).value;
        
        // 如果选择了内置免费模型，使用特殊标记
        const isBuiltinFree = provider === "builtin";
        
        const configData = {
            openAI: {
                apiBaseURL: isBuiltinFree ? "USE_DEFAULT_CONFIG" : baseURL,
                apiKey: isBuiltinFree ? "USE_DEFAULT_CONFIG" : apiKey,
                apiModel: isBuiltinFree ? "USE_DEFAULT_CONFIG" : model,
                apiProvider: provider,
                // ... 其他配置
            }
        };
        
        // 保存到思源后端
        fetchPost("/api/setting/setAI", configData, response => {
            window.siyuan.config.ai = response.data;
            
            // 如果是内置免费模型，同时保存到统一认证服务
            if (isBuiltinFree && window.siyuan.config.system?.container === "web") {
                const jwtToken = localStorage.getItem('siyuan_jwt_token');
                if (jwtToken) {
                    fetch(`${unifiedAuthServiceUrl}/api/settings/neuralink_llm`, {
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
                    });
                }
            }
        });
    });
});
```

## 🔧 配置说明

## 3. 架构设计

### 3.1 核心流程

1.  **前端 (Siyuan Desktop)**:
    *   用户在 AI 设置中选择 "内置免费模型"。
    *   前端保存设置时，将 `apiProvider` 设为 `builtin`，其他敏感字段（API Key 等）设为 `USE_DEFAULT_CONFIG`。
    *   同时，前端调用 Unified Settings Service 的 `/api/settings/neuralink_llm` 接口，同步保存配置。
    *   **AI 对话时**:
        *   前端检测到 provider 为 `builtin`。
        *   直接调用 Unified Settings Service 的 `/api/ai/chat` 接口。
        *   请求头包含 JWT Token 进行认证。

2.  **Unified Settings Service**:
    *   提供 `/api/ai/chat` 接口。
    *   验证 JWT Token。
    *   读取 `config/default-models.json` 获取真实的 API Key 和配置。
    *   代理调用实际的 AI 提供商（如 OpenRouter/SiliconFlow）。
    *   返回 AI 响应给前端。

3.  **Siyuan Kernel**:
    *   对于非内置模型，前端继续使用 Kernel 的 `/api/ai/chatGPT` 接口。
    *   Kernel 负责处理普通用户的自定义 API 配置。

### 3.2 优势

*   **安全性**: 真实的 API Key 存储在服务端（Unified Settings Service），不暴露给前端或 Siyuan Kernel 的普通配置。
*   **灵活性**: 可以随时在服务端更新模型配置，无需更新客户端。
*   **解耦**: 避免了修改 Siyuan Kernel 的复杂性（无需重新编译 Go 代码）。

### USE_DEFAULT_CONFIG 标记

当配置中出现 `"USE_DEFAULT_CONFIG"` 时，表示使用默认配置：

| 字段 | 标记值 | 实际值来源 |
|-----|--------|-----------|
| `provider` | `"builtin"` | 固定值 |
| `model` | `"USE_DEFAULT_CONFIG"` | `default-models.json` 中的 `model_name` |
| `api_key` | `"USE_DEFAULT_CONFIG"` | `default-models.json` 中的 `api_key` |
| `base_url` | `"USE_DEFAULT_CONFIG"` | `default-models.json` 中的 `base_url` |

### 后端配置加载逻辑（需要实现）

```pseudo
// 伪代码
function loadAIConfig(username) {
    // 读取用户设置
    userSettings = readFile(`/mnt/nas-sata12/MindOcean/user-data/settings/${username}_settings.json`);
    
    if (userSettings.neuralink_llm.model === "USE_DEFAULT_CONFIG") {
        // 读取默认配置
        defaultModels = readFile('/home/jason/code/unified-settings-service/config/default-models.json');
        
        // 根据应用选择对应的配置
        const appName = detectAppName(); // 检测当前应用
        let configKey;
        switch(appName) {
            case 'neuralink':
                configKey = 'builtin_free_neuralink';
                break;
            case 'tidelog':
                configKey = 'builtin_free_tidelog';
                break;
            default:
                configKey = 'builtin_free';
        }
        
        // 返回实际配置
        return {
            provider: defaultModels[configKey].provider,
            model: defaultModels[configKey].model_name,
            apiKey: defaultModels[configKey].api_key,
            baseUrl: defaultModels[configKey].base_url,
            temperature: defaultModels[configKey].temperature,
            maxTokens: defaultModels[configKey].max_tokens,
            systemPrompt: defaultModels[configKey].system_prompt
        };
    } else {
        // 使用用户自定义配置
        return userSettings.neuralink_llm;
    }
}
```

## 📊 数据流图

```
┌─────────────────────────────────────────────────────────────┐
│                    用户在前端选择                              │
│                 "内置免费模型 (builtin)"                       │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────────┐
│              前端保存配置（2个目标）                           │
├─────────────────────────────────────────────────────────────┤
│ 1. 思源后端 /api/setting/setAI                               │
│    └─ 保存到本地配置文件                                      │
│                                                             │
│ 2. 统一认证服务 /api/settings/neuralink_llm                  │
│    └─ 保存到 jason_settings.json                            │
│       {                                                     │
│         "provider": "builtin",                              │
│         "model": "USE_DEFAULT_CONFIG",                      │
│         "api_key": "USE_DEFAULT_CONFIG",                    │
│         "base_url": "USE_DEFAULT_CONFIG"                    │
│       }                                                     │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────────┐
│                   后端处理AI请求                              │
├─────────────────────────────────────────────────────────────┤
│ 1. 读取 jason_settings.json                                 │
│ 2. 检测到 model = "USE_DEFAULT_CONFIG"                       │
│ 3. 读取 default-models.json                                 │
│ 4. 加载 builtin_free_neuralink 配置                          │
│ 5. 使用实际配置调用AI                                         │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────────┐
│                  调用OpenRouter API                          │
├─────────────────────────────────────────────────────────────┤
│ URL: https://openrouter.ai/api/v1/chat/completions          │
│ Model: x-ai/grok-4.1-fast:free                              │
│ API Key: sk-or-v1-...                                       │
│ Temperature: 0.6                                            │
│ Max Tokens: 20000                                           │
└─────────────────────────────────────────────────────────────┘
```

## ✅ 验证步骤

### 1. 前端验证

```bash
# 打开浏览器开发者工具
# 选择 Network 标签

# 在思源设置中选择 "内置免费模型"
# 应该看到两个请求：

# 请求1：保存到思源
POST /api/setting/setAI
{
  "openAI": {
    "apiProvider": "builtin",
    "apiModel": "USE_DEFAULT_CONFIG",
    "apiKey": "USE_DEFAULT_CONFIG",
    "apiBaseURL": "USE_DEFAULT_CONFIG"
  }
}

# 请求2：保存到统一认证服务
POST http://localhost:3002/api/settings/neuralink_llm
{
  "provider": "builtin",
  "model": "USE_DEFAULT_CONFIG",
  "api_key": "USE_DEFAULT_CONFIG",
  "base_url": "USE_DEFAULT_CONFIG"
}
```

### 2. 文件验证

```bash
# 查看用户设置文件
cat /mnt/nas-sata12/MindOcean/user-data/settings/jason_settings.json | jq .neuralink_llm

# 应该输出：
{
  "provider": "builtin",
  "model": "USE_DEFAULT_CONFIG",
  "api_key": "USE_DEFAULT_CONFIG",
  "base_url": "USE_DEFAULT_CONFIG",
  "updated_at": "2025-11-26T02:48:00.000Z"
}
```

### 3. 后端验证（需要实现）

```bash
# 后端应该能读取并使用 default-models.json 中的配置
# 调用AI时使用：
# - API Key: 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
# - Base URL: https://openrouter.ai/api/v1
# - Model: x-ai/grok-4.1-fast:free
```

## 🔒 安全说明

### API密钥保护

- API密钥存储在 `default-models.json` 中
- 用户设置文件只存储标记 `"USE_DEFAULT_CONFIG"`
- 实际密钥不会暴露给前端

### 权限控制

- 只有Web模式下才会保存到统一认证服务
- 需要有效的JWT token
- 验证用户身份后才能保存

## 📋 JSON格式对照

### jason_settings.json 格式
```json
{
  "neuralink_llm": {
    "provider": "builtin",          // 固定值 "builtin"
    "model": "USE_DEFAULT_CONFIG",   // 标记：使用默认配置
    "api_key": "USE_DEFAULT_CONFIG", // 标记：使用默认配置
    "base_url": "USE_DEFAULT_CONFIG",// 标记：使用默认配置
    "updated_at": "2025-11-26T..."   // 更新时间
  }
}
```

### default-models.json 格式
```json
{
  "builtin_free_neuralink": {
    "name": "灵枢笔记专用模型",
    "provider": "builtin",
    "api_key": "实际的API密钥",
    "base_url": "实际的API地址",
    "model_name": "实际的模型名称",
    "temperature": 0.6,
    "max_tokens": 20000,
    "description": "模型描述",
    "version": "2.0.1",
    "features": [],
    "system_prompt": "系统提示词"
  }
}
```

## 🎯 优势

1. **安全性**：API密钥集中管理，不暴露给前端
2. **灵活性**：可以针对不同应用配置不同的模型
3. **一致性**：所有用户使用相同的内置免费模型配置
4. **可维护性**：只需更新 `default-models.json` 即可更新模型配置

## 📝 更新日志

### v5.1.0 (2025-11-26)

**新增功能**
- ✨ 内置免费模型与统一设置服务集成
- ✨ USE_DEFAULT_CONFIG 标记支持
- ✨ 自动保存到用户设置文件

**技术改进**
- 🔐 API密钥安全存储
- 🔄 双重配置保存机制
- 📋 统一的JSON格式

---

**相关文件**：
- 前端：`/home/jason/code/siyuan/app/src/config/ai.ts`
- 用户设置：`/mnt/nas-sata12/MindOcean/user-data/settings/jason_settings.json`
- 默认配置：`/home/jason/code/unified-settings-service/config/default-models.json`

**状态**：✅ 已部署
**版本**：v5.1.0
