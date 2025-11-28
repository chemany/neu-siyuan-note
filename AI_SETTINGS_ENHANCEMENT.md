# AI设置优化 - 多模型支持与内置免费模型

## ✨ 核心改进

### 1. 新增AI服务提供商 🚀

从原来只有2个选项扩展到**9个主流AI服务商**：

| 序号 | 服务商 | 特点 | 推荐场景 |
|-----|--------|------|---------|
| 🎁 1 | **内置免费模型** | 完全免费，无需API密钥 | 日常使用、文档分析 |
| 2 | OpenAI | GPT-4o等最强模型 | 专业任务、复杂推理 |
| 3 | Azure OpenAI | 企业级服务 | 企业用户、合规要求 |
| 4 | 硅基流动 (SiliconFlow) | 多模型低价 | 高性价比选择 |
| 5 | 阿里通义千问 | 中文能力强 | 中文内容处理 |
| 6 | 智谱AI (ChatGLM) | GLM系列模型 | 中文对话、代码 |
| 7 | DeepSeek | 推理能力强 | 代码、逻辑推理 |
| 8 | 月之暗面 (Kimi) | 超长上下文 | 长文档分析 |
| 9 | 自定义API | 兼容OpenAI格式 | 自部署模型 |

### 2. 丰富的模型选择 🔢

每个服务商都提供了精心挑选的模型：

#### 🎁 内置免费模型
- **思源内置免费模型**：完全免费，适合日常对话

#### OpenAI 模型
- **GPT-4o**：最新最强，支持图像等多模态
- **GPT-4o Mini**：轻量快速，性价比高
- **GPT-4 Turbo**：强大的GPT-4优化版
- **GPT-4**：经典GPT-4
- **GPT-3.5 Turbo**：快速经济

#### 硅基流动 (SiliconFlow)
- **通义千问 2.5 (72B)**：强大的中文模型
- **通义千问 2.5 (7B)**：轻量快速
- **GLM-4 (9B)**：智谱最新模型
- **DeepSeek V2.5**：推理能力强
- **Llama 3.1 (70B/8B)**：Meta开源大模型

#### 阿里通义千问
- **qwen-max**：最强性能
- **qwen-plus**：平衡性能
- **qwen-turbo**：快速响应

#### 智谱AI
- **GLM-4 Plus**：超大规模模型
- **GLM-4**：综合性能强
- **GLM-3 Turbo**：快速高效

#### DeepSeek
- **deepseek-chat**：通用对话
- **deepseek-coder**：代码专用

#### 月之暗面 (Kimi)
- **moonshot-v1-128k**：超长上下文(128K)
- **moonshot-v1-32k**：长上下文(32K)
- **moonshot-v1-8k**：标准上下文(8K)

### 3. 智能界面优化 🎨

#### 自动适配
- 切换服务商时，**自动更新可用模型列表**
- 自动显示/隐藏相关配置项
- 自动填充默认API地址

#### 模型描述
每个模型都有详细说明：
```
GPT-4o: "OpenAI最新最强模型，支持视觉理解和多模态输入"
Llama 3.1 (70B): "Meta开源大模型，强大的通用能力"
moonshot-v1-128k: "支持128K超长上下文，适合长文档分析"
```

#### 友好提示
- API密钥获取地址提示
- 默认API地址自动填充
- 字段说明清晰

## 🆚 改进对比

### 之前的设置（简陋）

```
AI服务提供商
├─ OpenAI
└─ Azure

配置项
├─ API密钥
├─ 模型名（手动输入）
├─ Base URL（手动输入）
├─ 其他参数...
```

**问题**：
- ❌ 只有2个选项，不够灵活
- ❌ 需要手动输入模型名
- ❌ 需要手动输入API地址
- ❌ 没有免费选项
- ❌ 不支持国产AI模型

### 现在的设置（强大）

```
AI服务提供商
├─ 🎁 内置免费模型 ⭐ 推荐
├─ OpenAI
│   ├─ GPT-4o
│   ├─ GPT-4o Mini
│   ├─ GPT-4 Turbo
│   ├─ GPT-4
│   └─ GPT-3.5 Turbo
├─ Azure OpenAI
├─ 硅基流动
│   ├─ 通义千问 2.5 (72B)
│   ├─ GLM-4 (9B)
│   ├─ DeepSeek V2.5
│   └─ Llama 3.1...
├─ 阿里通义千问
│   ├─ qwen-max
│   ├─ qwen-plus
│   └─ qwen-turbo
├─ 智谱AI
├─ DeepSeek
├─ 月之暗面
└─ 自定义API
```

**优势**：
- ✅ 9个服务商，40+模型可选
- ✅ 下拉选择模型，无需手动输入
- ✅ 自动填充API地址
- ✅ 内置免费模型
- ✅ 全面支持国产AI

## 📖 使用指南

### 快速开始（推荐）

#### 步骤1：选择内置免费模型
```
1. 打开设置 → AI → AI对话
2. AI服务提供商 选择 "🎁 内置免费模型"
3. AI模型 选择 "思源内置免费模型"
4. ✅ 完成！无需任何API密钥
```

### 使用OpenAI

#### 步骤1：获取API密钥
```
访问：https://platform.openai.com
注册账号 → API keys → Create new key
```

#### 步骤2：配置
```
1. AI服务提供商：OpenAI
2. API密钥：粘贴刚才获取的密钥
3. AI模型：选择 GPT-4o 或其他模型
4. ✅ 保存后即可使用
```

### 使用国产AI（硅基流动示例）

#### 步骤1：获取免费API密钥
```
访问：https://siliconflow.cn
注册即送免费额度！
```

#### 步骤2：配置
```
1. AI服务提供商：硅基流动
2. API密钥：粘贴密钥（sk-xxx）
3. AI模型：选择"通义千问 2.5 (72B)"等模型
4. API地址：https://api.siliconflow.cn/v1
   （会自动填充，无需手动输入）
5. ✅ 开始使用
```

### 使用月之暗面（Kimi）- 长文档分析

```
1. AI服务提供商：月之暗面
2. AI模型：moonshot-v1-128k (支持128K超长上下文)
3. 适用场景：分析长篇论文、书籍、长文档
```

## 🎯 推荐配置

### 配置1：免费日常使用
```yaml
服务提供商: 内置免费模型
模型: 思源内置免费模型
适用: 日常对话、文档摘要
成本: 完全免费
```

### 配置2：高性价比
```yaml
服务提供商: 硅基流动
模型: 通义千问 2.5 (72B)
API密钥: 注册即送免费额度
适用: 中文内容处理
成本: 低（有免费额度）
```

### 配置3：最强性能
```yaml
服务提供商: OpenAI
模型: GPT-4o
适用: 复杂推理、多模态任务
成本: 较高（专业用户）
```

### 配置4：超长文档
```yaml
服务提供商: 月之暗面
模型: moonshot-v1-128k
适用: 长文档分析、书籍总结
成本: 中等
```

### 配置5：代码专用
```yaml
服务提供商: DeepSeek
模型: deepseek-coder
适用: 代码生成、代码分析
成本: 低
```

## 💡 各服务商API密钥获取

| 服务商 | 获取地址 | 免费额度 |
|-------|---------|---------|
| **内置免费** | 无需密钥 | ✅ 完全免费 |
| OpenAI | https://platform.openai.com | ❌ 需付费 |
| 硅基流动 | https://siliconflow.cn | ✅ 注册送 |
| 通义千问 | https://dashscope.aliyuncs.com | ✅ 有免费额度 |
| 智谱AI | https://open.bigmodel.cn | ✅ 注册送 |
| DeepSeek | https://platform.deepseek.com | ✅ 注册送 |
| 月之暗面 | https://platform.moonshot.cn | ✅ 有免费额度 |

## 🔧 技术实现

### 核心功能

1. **动态模型列表**
```typescript
getModelOptions: () => {
    const provider = window.siyuan.config.ai.openAI.apiProvider;
    const modelGroups = {
        "builtin": [...],
        "OpenAI": [...],
        "SiliconFlow": [...],
        // ...
    };
    return modelGroups[provider] || [];
}
```

2. **智能字段显示**
```typescript
// 内置模型隐藏API密钥字段
if (provider === "builtin") {
    fieldElement.classList.add("fn__none");
}
// Azure显示版本字段
if (provider === "Azure") {
    versionField.classList.remove("fn__none");
}
```

3. **自动填充API地址**
```typescript
const defaultURLs = {
    "OpenAI": "https://api.openai.com/v1",
    "SiliconFlow": "https://api.siliconflow.cn/v1",
    "Qwen": "https://dashscope.aliyuncs.com/compatible-mode/v1",
    // ...
};
```

### 数据流

```
用户选择服务商
    ↓
更新模型列表
    ↓
更新模型描述
    ↓
自动填充API地址
    ↓
显示/隐藏相关字段
    ↓
保存配置到后端
```

## 📋 配置说明

### 通用参数

| 参数 | 说明 | 默认值 | 范围 |
|-----|------|--------|------|
| 超时时间 | API请求超时 | 30秒 | 5-600秒 |
| 最大Token数 | 生成的最大长度 | 2048 | 0-无限 |
| 温度 | 随机性控制 | 1.0 | 0-2.0 |
| 上下文数量 | 保留的对话轮数 | 8 | 1-64 |

### 特殊参数

- **API Version**（仅Azure）：Azure API版本号
- **代理地址**（仅自定义API）：HTTP代理设置
- **User-Agent**：自定义请求头

## ⚙️ 配置文件位置

配置保存在：
```
~/.config/siyuan/conf.json
```

其中 `ai.openAI` 部分：
```json
{
  "ai": {
    "openAI": {
      "apiProvider": "builtin",
      "apiModel": "builtin-free",
      "apiKey": "",
      "apiBaseURL": "",
      "apiTimeout": 30,
      "apiMaxTokens": 2048,
      "apiTemperature": 1.0,
      "apiMaxContexts": 8
    }
  }
}
```

## 🔄 迁移指南

### 从旧版本升级

**自动兼容**：
- ✅ 原有OpenAI配置自动保留
- ✅ 原有Azure配置自动保留
- ⚠️ 建议切换到"内置免费模型"或"硅基流动"节省成本

**手动调整**：
```
1. 打开设置 → AI → AI对话
2. 查看当前配置是否正常
3. 如需切换，选择新服务商
4. 更新API密钥和模型
5. 保存
```

## 🎨 界面预览

### 服务商选择下拉框
```
┌─────────────────────────────────┐
│ AI服务提供商                      │
├─────────────────────────────────┤
│ 🎁 内置免费模型      ◀ 默认推荐   │
│ OpenAI                          │
│ Azure OpenAI                    │
│ 硅基流动                         │
│ 阿里通义千问                     │
│ 智谱AI                          │
│ DeepSeek                        │
│ 月之暗面                         │
│ 自定义API                       │
└─────────────────────────────────┘
```

### 模型选择（以硅基流动为例）
```
┌─────────────────────────────────────────┐
│ AI模型                                   │
├─────────────────────────────────────────┤
│ 通义千问 2.5 (72B)                       │
│ 通义千问 2.5 (7B)                        │
│ GLM-4 (9B)                              │
│ DeepSeek V2.5                           │
│ Llama 3.1 (70B)                         │
│ Llama 3.1 (8B)                          │
└─────────────────────────────────────────┘

说明：阿里最强中文模型，理解能力出色
```

## 📝 更新日志

### v5.0.0 (2025-11-26)

**新增**
- ✨ 内置免费AI模型（无需API密钥）
- ✨ 8个主流AI服务商支持
- ✨ 40+精选AI模型
- ✨ 智能界面自适应
- ✨ 模型详细描述
- ✨ 自动填充API地址

**改进**
- 🎨 全新的服务商选择界面
- 🎨 模型下拉列表（无需手动输入）
- 🎨 动态显示/隐藏配置项
- 📚 完善的使用文档

**优化**
- ⚡ 更智能的配置验证
- ⚡ 更好的用户体验
- ⚡ 支持更多国产AI模型

## 🔗 相关链接

- **代码文件**：`/home/jason/code/siyuan/app/src/config/ai.ts`
- **配置文件**：`~/.config/siyuan/conf.json`

---

**访问地址**：http://localhost:6806  
**状态**：✅ 已部署运行  
**版本**：v5.0.0 - Multi-Model AI Support
