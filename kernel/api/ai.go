// SiYuan - Refactor your thinking
// Copyright (c) 2020-present, b3log.org
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/88250/gulu"
	"github.com/gin-gonic/gin"
	"github.com/sashabaranov/go-openai"
	"github.com/siyuan-note/siyuan/kernel/conf"
	"github.com/siyuan-note/siyuan/kernel/model"
	"github.com/siyuan-note/siyuan/kernel/util"
)

func chatGPT(c *gin.Context) {
	ret := gulu.Ret.NewResult()
	defer c.JSON(http.StatusOK, ret)

	arg, ok := util.JsonArg(c, ret)
	if !ok {
		return
	}

	msg := arg["msg"].(string)
	ret.Data = model.ChatGPT(msg)
}

func chatGPTWithAction(c *gin.Context) {
	ret := gulu.Ret.NewResult()
	defer c.JSON(http.StatusOK, ret)

	arg, ok := util.JsonArg(c, ret)
	if !ok {
		return
	}

	idsArg := arg["ids"].([]interface{})
	var ids []string
	for _, id := range idsArg {
		ids = append(ids, id.(string))
	}
	action := arg["action"].(string)
	ret.Data = model.ChatGPTWithAction(ids, action)
}

func chat(c *gin.Context) {
	ret := gulu.Ret.NewResult()
	defer c.JSON(http.StatusOK, ret)

	arg, ok := util.JsonArg(c, ret)
	if !ok {
		return
	}

	messagesArg, ok := arg["messages"].([]interface{})
	if !ok {
		ret.Code = -1
		ret.Msg = "messages parameter is missing or invalid"
		return
	}

	var messages []openai.ChatCompletionMessage
	for _, msg := range messagesArg {
		msgMap, ok := msg.(map[string]interface{})
		if !ok {
			continue
		}

		role, _ := msgMap["role"].(string)
		content, _ := msgMap["content"].(string)

		messages = append(messages, openai.ChatCompletionMessage{
			Role:    role,
			Content: content,
		})
	}

	content, err := model.Chat(messages)
	if err != nil {
		ret.Code = -1
		ret.Msg = err.Error()
		return
	}

	ret.Data = map[string]interface{}{
		"content": content,
	}
}

// 新增向量化和AI文档分析API

func vectorizeBlock(c *gin.Context) {
	ret := gulu.Ret.NewResult()
	defer c.JSON(http.StatusOK, ret)

	arg, ok := util.JsonArg(c, ret)
	if !ok {
		return
	}

	blockID := arg["blockId"].(string)
	err := model.VectorizeBlock(blockID)
	if err != nil {
		ret.Code = -1
		ret.Msg = err.Error()
		return
	}

	ret.Data = map[string]interface{}{
		"blockId": blockID,
		"success": true,
	}
}

func batchVectorizeNotebook(c *gin.Context) {
	ret := gulu.Ret.NewResult()
	defer c.JSON(http.StatusOK, ret)

	arg, ok := util.JsonArg(c, ret)
	if !ok {
		return
	}

	notebookID := arg["notebookId"].(string)
	err := model.BatchVectorizeNotebook(notebookID)
	if err != nil {
		ret.Code = -1
		ret.Msg = err.Error()
		return
	}

	ret.Data = map[string]interface{}{
		"notebookId": notebookID,
		"success":    true,
	}
}

func semanticSearch(c *gin.Context) {
	ret := gulu.Ret.NewResult()
	defer c.JSON(http.StatusOK, ret)

	arg, ok := util.JsonArg(c, ret)
	if !ok {
		return
	}

	query := arg["query"].(string)
	notebookID := ""
	if nbID, ok := arg["notebookId"].(string); ok {
		notebookID = nbID
	}

	limit := 10
	if l, ok := arg["limit"].(float64); ok {
		limit = int(l)
	}

	results, err := model.SemanticSearch(query, notebookID, limit)
	if err != nil {
		ret.Code = -1
		ret.Msg = err.Error()
		return
	}

	ret.Data = map[string]interface{}{
		"query":   query,
		"results": results,
	}
}

func generateNotebookSummary(c *gin.Context) {
	ret := gulu.Ret.NewResult()
	defer c.JSON(http.StatusOK, ret)

	arg, ok := util.JsonArg(c, ret)
	if !ok {
		return
	}

	notebookID := arg["notebookId"].(string)
	summary, err := model.GenerateNotebookSummary(notebookID)
	if err != nil {
		ret.Code = -1
		ret.Msg = err.Error()
		return
	}

	ret.Data = summary
}

func getNotebookSummary(c *gin.Context) {
	ret := gulu.Ret.NewResult()
	defer c.JSON(http.StatusOK, ret)

	arg, ok := util.JsonArg(c, ret)
	if !ok {
		return
	}

	notebookID := arg["notebookId"].(string)

	// 从保存的摘要文件中读取
	summaryPath := filepath.Join(util.DataDir, "notebook_summaries.json")
	if !gulu.File.IsExist(summaryPath) {
		ret.Code = -1
		ret.Msg = "摘要不存在"
		return
	}

	data, err := os.ReadFile(summaryPath)
	if err != nil {
		ret.Code = -1
		ret.Msg = err.Error()
		return
	}

	var summaries map[string]*model.NotebookSummary
	if err := json.Unmarshal(data, &summaries); err != nil {
		ret.Code = -1
		ret.Msg = err.Error()
		return
	}

	if summary, exists := summaries[notebookID]; exists {
		ret.Data = summary
	} else {
		ret.Code = -1
		ret.Msg = "该笔记本暂无摘要"
	}
}

func getEmbeddingConfig(c *gin.Context) {
	ret := gulu.Ret.NewResult()
	defer c.JSON(http.StatusOK, ret)

	if model.Conf.AI.Embedding == nil {
		ret.Data = map[string]interface{}{
			"provider":       "siliconflow",
			"model":          "BAAI/bge-large-zh-v1.5",
			"apiBaseUrl":     "https://api.siliconflow.cn/v1/embeddings",
			"encodingFormat": "float",
			"timeout":        30,
			"enabled":        false,
			"apiKey":         "",
		}
	} else {
		ret.Data = model.Conf.AI.Embedding
	}
}

func setEmbeddingConfig(c *gin.Context) {
	ret := gulu.Ret.NewResult()
	defer c.JSON(http.StatusOK, ret)

	arg, ok := util.JsonArg(c, ret)
	if !ok {
		return
	}

	// 更新配置
	if model.Conf.AI.Embedding == nil {
		model.Conf.AI.Embedding = &conf.Embedding{}
	}

	if provider, ok := arg["provider"].(string); ok {
		model.Conf.AI.Embedding.Provider = provider
	}
	if apiKey, ok := arg["apiKey"].(string); ok {
		model.Conf.AI.Embedding.APIKey = apiKey
		// 如果设置了API密钥，则启用向量化功能
		model.Conf.AI.Embedding.Enabled = apiKey != ""
	}
	if modelName, ok := arg["model"].(string); ok {
		model.Conf.AI.Embedding.Model = modelName
	}
	if apiBaseUrl, ok := arg["apiBaseUrl"].(string); ok {
		model.Conf.AI.Embedding.APIBaseURL = apiBaseUrl
	}
	if encodingFormat, ok := arg["encodingFormat"].(string); ok {
		model.Conf.AI.Embedding.EncodingFormat = encodingFormat
	}
	if timeout, ok := arg["timeout"].(float64); ok {
		model.Conf.AI.Embedding.Timeout = int(timeout)
	}
	if enabled, ok := arg["enabled"].(bool); ok {
		model.Conf.AI.Embedding.Enabled = enabled
	}

	model.Conf.Save()

	ret.Data = map[string]interface{}{
		"success": true,
		"message": "向量化配置已更新",
	}
}

func getEmbeddingModels(c *gin.Context) {
	ret := gulu.Ret.NewResult()
	defer c.JSON(http.StatusOK, ret)

	arg, ok := util.JsonArg(c, ret)
	if !ok {
		return
	}

	provider := "siliconflow"
	if p, ok := arg["provider"].(string); ok {
		provider = p
	}

	var models []map[string]interface{}

	switch provider {
	case "siliconflow":
		models = []map[string]interface{}{
			{"value": "BAAI/bge-large-zh-v1.5", "label": "BGE-Large-ZH (中文大型模型)"},
			{"value": "BAAI/bge-large-en-v1.5", "label": "BGE-Large-EN (英文大型模型)"},
			{"value": "netease-youdao/bce-embedding-base_v1", "label": "BCE-Embedding-Base"},
			{"value": "BAAI/bge-m3", "label": "BGE-M3 (通用模型)"},
			{"value": "Pro/BAAI/bge-m3", "label": "Pro/BGE-M3 (高级版)"},
		}
	case "openai":
		models = []map[string]interface{}{
			{"value": "text-embedding-ada-002", "label": "OpenAI Ada Embedding"},
			{"value": "text-embedding-3-small", "label": "OpenAI Embedding v3 Small"},
			{"value": "text-embedding-3-large", "label": "OpenAI Embedding v3 Large"},
		}
	default:
		models = []map[string]interface{}{}
	}

	ret.Data = map[string]interface{}{
		"provider": provider,
		"models":   models,
	}
}

func testEmbeddingConnection(c *gin.Context) {
	ret := gulu.Ret.NewResult()
	defer c.JSON(http.StatusOK, ret)

	embeddingService := model.NewEmbeddingService()
	if embeddingService == nil {
		ret.Code = -1
		ret.Msg = "向量化服务未配置"
		return
	}

	if !embeddingService.IsEnabled() {
		ret.Code = -1
		ret.Msg = "向量化服务未启用"
		return
	}

	// 测试向量化一段简单文本
	testText := "这是一个测试文本，用于验证向量化服务连接。"
	vector, err := embeddingService.VectorizeText(testText)
	if err != nil {
		ret.Code = -1
		ret.Msg = fmt.Sprintf("向量化测试失败: %v", err)
		return
	}

	ret.Data = map[string]interface{}{
		"success":    true,
		"message":    "向量化服务连接成功",
		"vectorSize": len(vector),
		"testText":   testText,
		"provider":   model.Conf.AI.Embedding.Provider,
		"model":      model.Conf.AI.Embedding.Model,
	}
}
