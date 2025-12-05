# 自动向量化与 RAG 系统设计文档

## 功能目标

实现一个完整的 RAG（检索增强生成）系统：
1. 当用户插入/上传文档（PDF, DOC, XLS, PPT, MD 等）时，**自动在后台进行向量化**
2. 当用户使用 AI 聊天时，**自动加载相关的向量化数据**，实现精确的文档分析

## 架构设计

### 1. 自动向量化流程

```
文件上传/插入 → 检测文档类型 → 解析文档内容 → 向量化 → 存储向量数据
     ↓              ↓                ↓             ↓            ↓
  Upload()    支持的文档格式    ParseAttachment  Embedding   Vector DB
```

### 2. RAG 检索流程

```
用户提问 → 向量化问题 → 语义搜索相关文档 → 构建增强Prompt → AI生成回答
    ↓          ↓              ↓                  ↓               ↓
  Chat     Embedding    SemanticSearch    Context+Question   Response
```

## 实现计划

### Phase 1: 自动向量化 Hook（已实现 0%）

**目标**：在文件上传后自动触发向量化

**实现位置**：
- `/root/code/siyuan/kernel/model/upload.go` - `Upload()` 函数
- `/root/code/siyuan/kernel/model/assets.go` - `InsertLocalAssets()` 函数

**实现步骤**：
1. ✅ 在 `Upload()` 函数成功上传后，添加异步向量化任务
2. ✅ 检测文件类型（PDF, DOCX, XLSX, PPTX, MD 等）
3. ✅ 调用 `ParseAttachment()` 解析文档内容
4. ✅ 调用 `VectorizeText()` 进行向量化
5. ✅ 存储向量数据到 `block_vectors.json`

### Phase 2: RAG 增强的 AI 聊天（已实现 0%）

**目标**：AI 聊天时自动加载相关文档

**实现位置**：
- `/root/code/siyuan/kernel/model/ai.go` - `chatGPTContinueWrite()` 函数
- `/root/code/siyuan/kernel/api/ai.go` - `chat()` API

**实现步骤**：
1. ✅ 在 AI 聊天前，先进行语义搜索
2. ✅ 查找与用户问题最相关的文档片段
3. ✅ 构建增强的 Prompt：`上下文 + 用户问题`
4. ✅ 将增强后的 Prompt 发送给 AI
5. ✅ 返回结果给用户

### Phase 3: 用户配置与控制（已实现 0%）

**目标**：让用户可以控制 RAG 功能

**实现位置**：
- `/root/code/siyuan/app/src/config/ai.ts` - AI 设置界面

**配置项**：
1. ✅ 启用/禁用自动向量化
2. ✅ 启用/禁用 RAG 增强聊天
3. ✅ 设置语义搜索的相似度阈值
4. ✅ 设置每次加载的文档片段数量

## 数据结构

### 向量数据存储

```json
{
  "asset_id_xxx": {
    "id": "asset_id_xxx",
    "assetPath": "assets/document-xxx.pdf",
    "fileName": "重要文档.pdf",
    "content": "解析后的文档文本内容...",
    "vector": [0.123, -0.456, ...],  // 1024维向量
    "updatedAt": "2025-12-03T02:20:00Z",
    "metadata": {
      "fileType": "pdf",
      "pageCount": 10,
      "wordCount": 5000
    }
  }
}
```

### RAG 配置（新增到 conf.ai.go）

```go
type RAGConfig struct {
    Enabled              bool    `json:"enabled"`               // 是否启用 RAG
    AutoVectorize        bool    `json:"autoVectorize"`         // 自动向量化新文档
    SimilarityThreshold  float64 `json:"similarityThreshold"`   // 相似度阈值（0.7-0.9）
    MaxContextDocuments  int     `json:"maxContextDocuments"`   // 最多加载几个文档片段
    ContextMaxTokens     int     `json:"contextMaxTokens"`      // 上下文最大 Token 数
}
```

## API 接口

### 新增 API

1. **POST /api/ai/vectorizeAsset**
   - 手动向量化单个资源文件
   - 参数：`{assetPath: string}`
   - 返回：`{success: bool, vectorId: string}`

2. **POST /api/ai/chatWithRAG**
   - RAG 增强的 AI 聊天
   - 参数：`{messages: [], useRAG: bool}`
   - 返回：`{content: string, sources: []}`

3. **GET /api/ai/getVectorizedAssets**
   - 获取已向量化的资源列表
   - 返回：`{assets: [{path, fileName, updatedAt}]}`

## 技术细节

### 1. 异步向量化队列

使用 Go channel 实现异步处理：

```go
var vectorizeQueue = make(chan VectorizeTask, 100)

type VectorizeTask struct {
    AssetPath string
    Priority  int
}

func StartVectorizeWorker() {
    for task := range vectorizeQueue {
        vectorizeAssetAsync(task)
    }
}
```

### 2. 文档内容分块

对于长文档，需要分块处理：

```go
func ChunkDocument(content string, maxChunkSize int) []string {
    // 按段落或固定长度分块
    // 每块保持语义完整性
}
```

### 3. RAG Prompt 模板

```
你是一个专业的文档分析助手。以下是相关的文档内容：

【文档1: 重要报告.pdf】
这是文档的相关内容...

【文档2: 项目计划.docx】
这是另一个文档的相关内容...

基于以上文档内容，请回答用户的问题：
{user_question}

要求：
1. 优先使用提供的文档内容回答
2. 如果文档中没有相关信息，请明确说明
3. 引用文档时请标注来源
```

## 性能优化

1. **缓存机制**：已向量化的文档不重复处理
2. **批量处理**：多个文档一起向量化，减少 API 调用
3. **延迟加载**：只在需要时才进行语义搜索
4. **索引优化**：使用更高效的向量检索算法（如FAISS、Annoy）

## 用户体验

### 用户界面提示

1. 文档上传后显示"正在向量化..."
2. AI 聊天时显示"正在查找相关文档..."
3. 回答时标注引用的文档来源

### 设置界面

在 AI 设置中新增 "RAG 设置" 标签页：
- 自动向量化开关
- RAG 聊天开关
- 相似度阈值滑块
- 上下文文档数量

## 下一步

1. 实现 Phase 1（自动向量化）
2. 测试各种文档格式的解析
3. 实现 Phase 2（RAG 聊天）
4. 优化性能和用户体验
5. 添加配置界面
