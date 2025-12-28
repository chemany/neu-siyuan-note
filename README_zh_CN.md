# NeuLink Notes

<p align="center">
<a title="GitHub Release" target="_blank" href="https://github.com/neurallink-ai/neu-siyuan-note/releases"><img src="https://img.shields.io/github/release/neurallink-ai/neu-siyuan-note.svg?style=flat-square&color=9CF"></a>
<a title="AGPLv3" target="_blank" href="https://www.gnu.org/licenses/agpl-3.0.txt"><img src="http://img.shields.io/badge/license-AGPLv3-orange.svg?style=flat-square"></a>
</p>

基于思源笔记的 AI 增强版本，专注于智能会议纪要和知识管理。

## ✨ 核心特性

### 🎙️ 智能会议纪要

自动记录会议内容，实时语音转文字，生成结构化会议纪要。

**工作流程：**
```
录制会议 → FunASR语音识别 → 文本累积去重 → AI智能总结 → 正式会议纪要
```

**技术栈：**
- **语音识别**：FunASR（阿里达摩院开源模型）
- **AI 总结**：本地 VLLM 部署（Qwen3-32B）
- **文本处理**：智能增量累积算法，避免重复

### 📚 RAG 知识库

基于向量检索的智能问答系统，让你的笔记会"思考"。

- 支持语义搜索和智能问答
- 自动向量化文档内容
- 与思源笔记深度集成

### 🌐 思源笔记生态

继承思源笔记所有优秀特性：
- 块级引用和双向链接
- Markdown 所见即所得
- 数据库视图
- 丰富的导出格式

## 🚀 快速开始

### 前置要求

- 思源笔记（kernel）
- FunASR 语音识别服务
- 本地 VLLM 模型服务（Qwen3-32B 推荐）

### 安装

```bash
# 克隆仓库
git clone https://github.com/neurallink-ai/neu-siyuan-note.git
cd neu-siyuan-note

# 构建
bash rebuild-and-restart.sh
```

### 配置

在 `kernel/model/meeting.go` 中配置 AI 服务地址：

```go
llmURL := "http://your-vllm-server:8001/v1/chat/completions"
modelName := "tclf90/Qwen3-32B-GPTQ-Int4"
```

## 📁 项目结构

```
neu-siyuan-note/
├── app/                    # 前端（Electron桌面应用）
│   └── src/
│       └── meeting/        # 会议录制模块
├── kernel/                 # 后端（Go）
│   └── model/
│       └── meeting.go      # 会议纪要核心逻辑
├── rebuild-and-restart.sh  # 一键构建脚本
└── RAG_USER_MANUAL.md      # RAG 使用手册
```

## 🛠️ 开发

### 构建命令

```bash
# 一键构建和重启服务
bash rebuild-and-restart.sh

# 单独构建前端
cd app && npm run build:desktop

# 单独构建后端
cd kernel && go build
```

### 日志查看

```bash
tail -f workspace/temp/siyuan.log | grep "ASR:"
```

## 📝 功能说明

### 会议录制

1. 启动会议录制
2. 系统自动采集音频并实时转写
3. 录制结束后自动生成 AI 总结

### AI 会议纪要格式

```
> **会议主题**：[一句话概括]
> **关键讨论**：[2-3句话专业概述]
> **重要决议**：[决策和行动要点]
```

## 🔧 技术细节

### 语音识别优化

- **增量累积算法**：FunASR 返回增量文本片段，智能合并避免重复
- **超时处理**：60秒超时机制，确保长音频也能完整处理
- **详细日志**：每一步处理都有日志记录，便于调试

### AI 提示词设计

- 温度 0.3，平衡创造性和稳定性
- 明确的三段式输出格式
- 禁止思考过程和开场白

## 📄 许可证

本项目基于 [AGPL-3.0](https://www.gnu.org/licenses/agpl-3.0.txt) 开源。

## 🙏 致谢

- [思源笔记](https://github.com/siyuan-note/siyuan) - 优秀的知识管理基础
- [FunASR](https://github.com/modelscope/FunASR) - 阿里达摩院开源语音识别
- [Qwen3](https://github.com/QwenLM/Qwen3) - 通义千问大语言模型

---

<p align="center">
基于思源笔记重构 | 用 AI 增强你的思维
</p>
