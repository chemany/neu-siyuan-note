# 资源文件自动向量化功能 - 使用手册

## 📋 最小版本功能说明

已成功实现**手动向量化资源文件**的基础功能。

### ✅ 已实现功能

1. **资源文件向量化**
   - 支持手动触发单个资源文件的向量化
   - 支持多种文档格式：PDF, DOC, DOCX, XLS, XLSX, PPT, PPTX, MD, TXT 等
   - 自动解析文档内容并生成向量
   - 向量数据持久化存储

2. **向量数据管理**
   - 独立的向量数据库（`asset_vectors.json`）
   - 查看已向量化的资源列表
   - 按时间排序显示

3. **API接口**
   - `POST /api/ai/vectorizeAsset` - 向量化单个资源文件
   - `POST /api/ai/getVectorizedAssets` - 获取已向量化的资源列表

### 🎯 使用方法

#### 方法1：通过 API 测试（推荐用于首次测试）

1. 确保向量化服务已配置并测试成功
2. 使用以下API测试向量化功能

**向量化单个资源文件：**
```bash
curl -X POST http://localhost:6806/api/ai/vectorizeAsset \
  -H "Content-Type: application/json" \
  -d '{"assetPath": "assets/document-xxx.pdf"}'
```

**获取已向量化的资源列表：**
```bash
curl -X POST http://localhost:6806/api/ai/getVectorizedAssets \
  -H "Content-Type: application/json" \
  -d '{}'
```

#### 方法2：添加到 AI 功能测试界面（下一步实现）

在 **设置 → AI → AI功能** 标签页中添加测试按钮。

### 📂 数据存储

向量化数据存储在：
```
workspace/users/{username}/data/asset_vectors.json
```

数据结构示例：
```json
{
  "5d41402abc4b2a76b9719d911017c592": {
    "id": "5d41402abc4b2a76b9719d911017c592",
    "assetPath": "assets/report-20251203.pdf",
    "fileName": "report-20251203.pdf",
    "fileType": "pdf",
    "content": "解析后的文档内容（前500字符）...",
    "vector": [0.123, -0.456, 0.789, ...],
    "updatedAt": "2025-12-03T02:30:00Z",
    "metadata": {
      "contentLength": 15000,
      "vectorDim": 1024
    }
  }
}
```

### 🔧 技术细节

1. **文档解析**
   - 使用 `ParseAttachment()` 函数解析各种格式
   - 支持的格式：PDF, DOC, DOCX, XLS, XLSX, PPT, PPTX, MD, TXT等

2. **向量化处理**
   - 自动截取前8000字符进行向量化（避免超过模型限制）
   - 使用配置的向量化服务（SiliconFlow/OpenAI）
   - 生成 1024 维向量（具体维度取决于模型）

3. **资源ID生成**
   - 使用MD5哈希确保唯一性
   - 基于资源文件路径生成

### 🐛 故障排除

**问题1：向量化失败**
- 检查向量化服务是否已配置（设置 → AI → 向量化）
- 确认 API 密钥正确
- 查看 API BaseURL 是否为 `https://api.siliconflow.cn/v1`

**问题2：文档解析失败**
- 确认文件格式是否支持
- 检查文件路径是否正确
- 对于PDF需要安装 `pdftotext`

**问题3：找不到资源文件**
- 确认资源路径格式：`assets/xxx.pdf`
- 资源文件应该在工作空间的 assets 目录中

### 📝 下一步计划

**Phase 2：自动向量化（待实现）**
- [ ] 文件上传时自动触发向量化
- [ ] 后台队列处理
- [ ] 进度提示

**Phase 3：RAG增强聊天（待实现）**
- [ ] AI聊天时自动加载相关文档
- [ ] 语义搜索集成
- [ ] 增强Prompt构建

**Phase 4：用户界面（待实现）**
- [ ] AI设置中添加资源向量化测试
- [ ] 显示已向量化资源列表
- [ ] 手动触发向量化按钮

### 💡 使用建议

1. **先测试小文件**：第一次使用建议先用小的文本文件或PDF测试
2. **监控API配额**：注意SiliconFlow的API调用次数限制
3. **内容预览**：向量化成功后可以查看解析的内容预览（前500字符）

### 🎉 成功案例

```json
{
  "success": true,
  "id": "abc123...",
  "assetPath": "assets/report.pdf",
  "fileName": "report.pdf",
  "fileType": "pdf",
  "vectorDim": 1024,
  "updatedAt": "2025-12-03T02:30:00Z",
  "message": "成功向量化资源文件: report.pdf"
}
```

## 📚 相关文档

- 完整设计文档：`/root/code/siyuan/AUTO_VECTORIZATION_DESIGN.md`
- 向量化配置：设置 → AI → 向量化
- API文档：`/root/code/siyuan/API.md`

---

**版本**：v0.1.0 - 最小可用版本  
**更新时间**：2025-12-03  
**状态**：✅ 已实现并测试
