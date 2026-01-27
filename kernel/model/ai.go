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

package model

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/88250/gulu"
	"github.com/88250/lute/ast"
	"github.com/88250/lute/parse"
	"github.com/sashabaranov/go-openai"
	"github.com/siyuan-note/logging"
	"github.com/siyuan-note/siyuan/kernel/sql"
	"github.com/siyuan-note/siyuan/kernel/treenode"
	"github.com/siyuan-note/siyuan/kernel/util"
)

func init() {
	StartAssetVectorizeWorker()
	// 暂时禁用全局向量化服务，避免索引队列堵塞
	// StartGlobalVectorizeService()
}

func ChatGPT(msg string) (ret string) {
	if !isOpenAIAPIEnabled() {
		return
	}

	return chatGPT(msg, false)
}

func ChatGPTWithAction(ids []string, action string) (ret string) {
	if !isOpenAIAPIEnabled() {
		return
	}

	if "Clear context" == action {
		// AI clear context action https://github.com/siyuan-note/siyuan/issues/10255
		cachedContextMsg = nil
		return
	}

	msg := getBlocksContent(ids)
	ret = chatGPTWithAction(msg, action, false)
	return
}

var cachedContextMsg []string

func chatGPT(msg string, cloud bool) (ret string) {
	if "Clear context" == strings.TrimSpace(msg) {
		// AI clear context action https://github.com/siyuan-note/siyuan/issues/10255
		cachedContextMsg = nil
		return
	}

	ret, retCtxMsgs, err := chatGPTContinueWrite(msg, cachedContextMsg, cloud)
	if err != nil {
		return
	}
	cachedContextMsg = append(cachedContextMsg, retCtxMsgs...)
	return
}

func chatGPTWithAction(msg string, action string, cloud bool) (ret string) {
	action = strings.TrimSpace(action)
	if "" != action {
		msg = action + ":\n\n" + msg
	}
	ret, _, err := chatGPTContinueWrite(msg, nil, cloud)
	if err != nil {
		return
	}
	return
}

func chatGPTContinueWrite(msg string, contextMsgs []string, cloud bool) (ret string, retContextMsgs []string, err error) {
	util.PushEndlessProgress("Requesting...")
	defer util.ClearPushProgress(100)

	// RAG 增强：搜索相关文档
	embeddingService := NewEmbeddingService()
	if embeddingService != nil && embeddingService.IsEnabled() {
		// 搜索最相关的10个分块
		chunks, err := SemanticSearchAssetChunks(util.DataDir, msg, 10, nil)
		if err == nil && len(chunks) > 0 {
			var contextBuilder strings.Builder
			contextBuilder.WriteString("以下是相关的文档内容供参考：\n\n")
			for i, chunk := range chunks {
				contextBuilder.WriteString(fmt.Sprintf("【片段%d】\n%s\n\n", i+1, chunk.Content))
			}
			contextBuilder.WriteString("请基于以上文档内容回答用户的问题：\n")
			contextBuilder.WriteString(msg)

			// 更新 msg
			msg = contextBuilder.String()
			logging.LogInfof("RAG 增强已启用，加载了 %d 个相关分块", len(chunks))
		}
	}

	if Conf.AI.OpenAI.APIMaxContexts < len(contextMsgs) {
		contextMsgs = contextMsgs[len(contextMsgs)-Conf.AI.OpenAI.APIMaxContexts:]
	}

	apiKey, apiBaseURL, apiModel, _, _ := getEffectiveAIConfig()

	var gpt GPT
	if cloud {
		gpt = &CloudGPT{}
	} else {
		gpt = &OpenAIGPT{c: util.NewOpenAIClient(apiKey, Conf.AI.OpenAI.APIProxy, apiBaseURL, Conf.AI.OpenAI.APIUserAgent, Conf.AI.OpenAI.APIVersion, Conf.AI.OpenAI.APIProvider), model: apiModel}
	}

	buf := &bytes.Buffer{}
	for i := 0; i < Conf.AI.OpenAI.APIMaxContexts; i++ {
		part, stop, chatErr := gpt.chat(msg, contextMsgs)
		buf.WriteString(part)

		if stop || nil != chatErr {
			break
		}

		util.PushEndlessProgress("Continue requesting...")
	}

	ret = buf.String()
	ret = strings.TrimSpace(ret)
	if "" != ret {
		retContextMsgs = append(retContextMsgs, msg, ret)
	}
	return
}

type DefaultModelConfig struct {
	Provider    string  `json:"provider"`
	APIKey      string  `json:"api_key"`
	BaseURL     string  `json:"base_url"`
	ModelName   string  `json:"model_name"`
	Temperature float64 `json:"temperature"`
	MaxTokens   int     `json:"max_tokens"`
}

func getEffectiveAIConfig() (apiKey, apiBaseURL, apiModel string, maxTokens int, temperature float64) {
	apiKey = Conf.AI.OpenAI.APIKey
	apiBaseURL = Conf.AI.OpenAI.APIBaseURL
	apiModel = Conf.AI.OpenAI.APIModel
	maxTokens = Conf.AI.OpenAI.APIMaxTokens
	temperature = Conf.AI.OpenAI.APITemperature

	// 当APIKey为空、为USE_DEFAULT_CONFIG，或者APIProvider为builtin时，读取默认配置
	needDefaultConfig := apiKey == "" || apiKey == "USE_DEFAULT_CONFIG" || apiModel == "USE_DEFAULT_CONFIG" || Conf.AI.OpenAI.APIProvider == "builtin"

	if needDefaultConfig {
		configPath := "/root/code/unified-settings-service/config/default-models.json"
		data, err := os.ReadFile(configPath)
		if err == nil {
			var models map[string]DefaultModelConfig
			if err := json.Unmarshal(data, &models); err == nil {
				// 优先使用思源笔记专用模型配置
				var targetConfig DefaultModelConfig
				if val, ok := models["builtin_free_siyuan"]; ok {
					targetConfig = val
				} else if val, ok := models["builtin_free_neuralink"]; ok {
					// 兼容旧配置
					targetConfig = val
				} else if val, ok := models["builtin_free"]; ok {
					targetConfig = val
				}

				if targetConfig.APIKey != "" {
					apiKey = targetConfig.APIKey
					apiBaseURL = targetConfig.BaseURL
					apiModel = targetConfig.ModelName
					if targetConfig.MaxTokens > 0 {
						maxTokens = targetConfig.MaxTokens
					}
					if targetConfig.Temperature > 0 {
						temperature = targetConfig.Temperature
					}
				}
			}
		}
	}
	return
}

// ChatWithContext 聊天（支持用户上下文）
func ChatWithContext(ctx *WorkspaceContext, messages []openai.ChatCompletionMessage, allowedAssets []string) (ret string, err error) {
	if !isOpenAIAPIEnabled() {
		return "", fmt.Errorf("AI not enabled")
	}

	// RAG 增强：从用户消息中提取查询，搜索相关文档
	messages = EnhanceMessagesWithRAGContext(ctx, messages, allowedAssets)

	apiKey, apiBaseURL, apiModel, maxTokens, temperature := getEffectiveAIConfig()

	client := util.NewOpenAIClient(apiKey, Conf.AI.OpenAI.APIProxy, apiBaseURL, Conf.AI.OpenAI.APIUserAgent, Conf.AI.OpenAI.APIVersion, Conf.AI.OpenAI.APIProvider)

	resp, err := client.CreateChatCompletion(context.Background(), openai.ChatCompletionRequest{
		Model:       apiModel,
		Messages:    messages,
		MaxTokens:   maxTokens,
		Temperature: float32(temperature),
	})

	if err != nil {
		return "", err
	}

	if len(resp.Choices) > 0 {
		return resp.Choices[0].Message.Content, nil
	}

	return "", fmt.Errorf("no response from AI")
}

// Chat 聊天（兼容旧版本）
func Chat(messages []openai.ChatCompletionMessage, allowedAssets []string) (ret string, err error) {
	if !isOpenAIAPIEnabled() {
		return "", fmt.Errorf("AI not enabled")
	}

	// RAG 增强：从用户消息中提取查询，搜索相关文档
	messages = EnhanceMessagesWithRAG(messages, allowedAssets)

	apiKey, apiBaseURL, apiModel, maxTokens, temperature := getEffectiveAIConfig()

	client := util.NewOpenAIClient(apiKey, Conf.AI.OpenAI.APIProxy, apiBaseURL, Conf.AI.OpenAI.APIUserAgent, Conf.AI.OpenAI.APIVersion, Conf.AI.OpenAI.APIProvider)

	resp, err := client.CreateChatCompletion(context.Background(), openai.ChatCompletionRequest{
		Model:       apiModel,
		Messages:    messages,
		MaxTokens:   maxTokens,
		Temperature: float32(temperature),
	})

	if err != nil {
		return "", err
	}

	if len(resp.Choices) > 0 {
		return resp.Choices[0].Message.Content, nil
	}

	return "", fmt.Errorf("no response from AI")
}

// ChatStreamWithContext 流式聊天，通过 channel 返回每个 token（支持用户上下文）
func ChatStreamWithContext(ctx *WorkspaceContext, messages []openai.ChatCompletionMessage, allowedAssets []string, onToken func(token string) error) error {
	if !isOpenAIAPIEnabled() {
		return fmt.Errorf("AI not enabled")
	}

	// RAG 增强
	messages = EnhanceMessagesWithRAGContext(ctx, messages, allowedAssets)

	apiKey, apiBaseURL, apiModel, maxTokens, temperature := getEffectiveAIConfig()

	client := util.NewOpenAIClient(apiKey, Conf.AI.OpenAI.APIProxy, apiBaseURL, Conf.AI.OpenAI.APIUserAgent, Conf.AI.OpenAI.APIVersion, Conf.AI.OpenAI.APIProvider)

	req := openai.ChatCompletionRequest{
		Model:       apiModel,
		Messages:    messages,
		MaxTokens:   maxTokens,
		Temperature: float32(temperature),
		Stream:      true,
	}

	stream, err := client.CreateChatCompletionStream(context.Background(), req)
	if err != nil {
		return fmt.Errorf("创建流式请求失败: %v", err)
	}
	defer stream.Close()

	for {
		response, err := stream.Recv()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return fmt.Errorf("接收流式响应失败: %v", err)
		}

		if len(response.Choices) > 0 {
			token := response.Choices[0].Delta.Content
			if token != "" {
				if err := onToken(token); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// ChatStream 流式聊天，通过 channel 返回每个 token（兼容旧版本）
func ChatStream(messages []openai.ChatCompletionMessage, allowedAssets []string, onToken func(token string) error) error {
	if !isOpenAIAPIEnabled() {
		return fmt.Errorf("AI not enabled")
	}

	// RAG 增强
	messages = EnhanceMessagesWithRAG(messages, allowedAssets)

	apiKey, apiBaseURL, apiModel, maxTokens, temperature := getEffectiveAIConfig()

	client := util.NewOpenAIClient(apiKey, Conf.AI.OpenAI.APIProxy, apiBaseURL, Conf.AI.OpenAI.APIUserAgent, Conf.AI.OpenAI.APIVersion, Conf.AI.OpenAI.APIProvider)

	req := openai.ChatCompletionRequest{
		Model:       apiModel,
		Messages:    messages,
		MaxTokens:   maxTokens,
		Temperature: float32(temperature),
		Stream:      true,
	}

	stream, err := client.CreateChatCompletionStream(context.Background(), req)
	if err != nil {
		return fmt.Errorf("创建流式请求失败: %v", err)
	}
	defer stream.Close()

	for {
		response, err := stream.Recv()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return fmt.Errorf("接收流式响应失败: %v", err)
		}

		if len(response.Choices) > 0 {
			token := response.Choices[0].Delta.Content
			if token != "" {
				if err := onToken(token); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// EnhanceMessagesWithRAGContext 使用 RAG 增强消息（支持用户上下文）
func EnhanceMessagesWithRAGContext(ctx *WorkspaceContext, messages []openai.ChatCompletionMessage, allowedAssets []string) []openai.ChatCompletionMessage {
	if ctx == nil {
		logging.LogWarnf("RAG: 用户上下文为空，跳过 RAG 增强")
		return messages
	}
	
	embeddingService := NewEmbeddingService()
	if embeddingService == nil || !embeddingService.IsEnabled() {
		return messages
	}

	// 从最后一条用户消息中提取查询
	var userQuery string
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			userQuery = messages[i].Content
			break
		}
	}

	if userQuery == "" {
		return messages
	}

	// 如果消息中已经包含了"【文档正文内容】"，说明前端已经提供了具体的文档上下文，此时跳过 RAG，避免干扰
	for _, m := range messages {
		if m.Role == "system" && strings.Contains(m.Content, "【文档正文内容】") {
			return messages
		}
	}

	// 检测是否为总结类请求
	isSummaryRequest := false
	summaryKeywords := []string{"总结", "摘要", "概览", "概括", "所有文档", "文档集", "整体", "总结一下"}
	for _, kw := range summaryKeywords {
		if strings.Contains(userQuery, kw) {
			isSummaryRequest = true
			break
		}
	}

	var ragContext strings.Builder
	if isSummaryRequest {
		logging.LogInfof("RAG: 检测到总结请求，执行全量注入模式")
		// 全量模式：从所有向量化资产中提取所有内容
		assets, err := GetVectorizedAssetsWithContext(ctx)
		if err == nil && len(assets) > 0 {
			ragContext.WriteString("以下是所有相关附件的完整内容总结参考（请以此为准，严禁强行关联不同来源的技术）：\n\n")
			totalLen := 0
			// 适配 72k Token 模型：将总结上限提升至 100,000 字符 (约 30k+ Token)
			const maxTotalLen = 100000 
			for _, asset := range assets {
				if totalLen > maxTotalLen { break }
				// 跨笔记本隔离：如果指定了允许的附件列表，则只包含列表中的文件
				if len(allowedAssets) > 0 {
					found := false
					for _, allowed := range allowedAssets {
						if asset.FileName == allowed || strings.Contains(asset.AssetPath, allowed) {
							found = true
							break
						}
					}
					if !found { continue }
				}
				ragContext.WriteString(fmt.Sprintf("--- 文档来源: %s ---\n", asset.FileName))
				for _, chunk := range asset.Chunks {
					if totalLen+len(chunk.Content) > maxTotalLen {
						ragContext.WriteString("...(由于长度限制，后续内容已截断)\n")
						totalLen = maxTotalLen + 1
						break
					}
					ragContext.WriteString(chunk.Content)
					ragContext.WriteString("\n")
					totalLen += len(chunk.Content)
				}
				ragContext.WriteString("\n")
			}
		}
	} else {
		// 问答模式：执行高配版分块检索 (Top-30)
		chunks, err := SemanticSearchAssetChunksWithContext(ctx, userQuery, 30, allowedAssets)
		if err == nil && len(chunks) > 0 {
			logging.LogInfof("RAG: 问答模式，找到 %d 个相关分块", len(chunks))
			ragContext.WriteString("以下是与用户问题高度相关的文档片段，请据此回答。注意区分不同来源，不要强行拼凑逻辑：\n\n")
			ragContext.WriteString("以下是与用户问题高度相关的文档片段，请据此回答。注意区分不同来源，不要强行拼凑逻辑：\n\n")
			totalLen := 0
			const maxQALen = 30000 // 问答模式大幅扩容至 30,000 字符，提供深层上下文
			for i, chunk := range chunks {
				if totalLen+len(chunk.Content) > maxQALen {
					ragContext.WriteString("...(由于长度限制，后续片段已忽略)\n")
					break
				}
				ragContext.WriteString(fmt.Sprintf("【片段%d - 来源: %s】\n%s\n\n", i+1, chunk.Source, chunk.Content))
				totalLen += len(chunk.Content)
			}
		}
	}

	if ragContext.Len() == 0 {
		return messages
	}

	// 将 RAG 上下文添加到 system 消息中
	ragSystemMsg := openai.ChatCompletionMessage{
		Role:    "system",
		Content: "你是一个严谨的文档分析专家。请分别参考以下不同来源的文档内容回答。\n" +
				"**绝对规则：**\n" +
				"1. 严禁在没有明确文档支持的情况下，强行关联或合并不同文档或不同领域的技术逻辑。\n" +
				"2. 如果不同文档讨论的是不相关的领域（如某种化学工艺 vs 另一种无关中间体），请分段分别陈述，严禁进行逻辑拼凑。\n" +
				"3. 必须在回答中明确提到信息来源（如\"根据XXX文档...\"）。\n\n" +
				ragContext.String(),
	}

	// 在消息列表开头插入 RAG 上下文
	enhancedMessages := make([]openai.ChatCompletionMessage, 0, len(messages)+1)
	enhancedMessages = append(enhancedMessages, ragSystemMsg)
	enhancedMessages = append(enhancedMessages, messages...)

	return enhancedMessages
}

// EnhanceMessagesWithRAG 使用 RAG 增强消息（兼容旧版本，使用全局 util.DataDir）
func EnhanceMessagesWithRAG(messages []openai.ChatCompletionMessage, allowedAssets []string) []openai.ChatCompletionMessage {
	embeddingService := NewEmbeddingService()
	if embeddingService == nil || !embeddingService.IsEnabled() {
		return messages
	}

	// 从最后一条用户消息中提取查询
	var userQuery string
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			userQuery = messages[i].Content
			break
		}
	}

	if userQuery == "" {
		return messages
	}

	// 如果消息中已经包含了"【文档正文内容】"，说明前端已经提供了具体的文档上下文，此时跳过 RAG，避免干扰
	for _, m := range messages {
		if m.Role == "system" && strings.Contains(m.Content, "【文档正文内容】") {
			return messages
		}
	}

	// 检测是否为总结类请求
	isSummaryRequest := false
	summaryKeywords := []string{"总结", "摘要", "概览", "概括", "所有文档", "文档集", "整体", "总结一下"}
	for _, kw := range summaryKeywords {
		if strings.Contains(userQuery, kw) {
			isSummaryRequest = true
			break
		}
	}

	var ragContext strings.Builder
	if isSummaryRequest {
		logging.LogInfof("RAG: 检测到总结请求，执行全量注入模式")
		// 全量模式：从所有向量化资产中提取所有内容
		assets, err := GetVectorizedAssets(util.DataDir)
		if err == nil && len(assets) > 0 {
			ragContext.WriteString("以下是所有相关附件的完整内容总结参考（请以此为准，严禁强行关联不同来源的技术）：\n\n")
			totalLen := 0
			// 适配 72k Token 模型：将总结上限提升至 100,000 字符 (约 30k+ Token)
			const maxTotalLen = 100000 
			for _, asset := range assets {
				if totalLen > maxTotalLen { break }
				// 跨笔记本隔离：如果指定了允许的附件列表，则只包含列表中的文件
				if len(allowedAssets) > 0 {
					found := false
					for _, allowed := range allowedAssets {
						if asset.FileName == allowed || strings.Contains(asset.AssetPath, allowed) {
							found = true
							break
						}
					}
					if !found { continue }
				}
				ragContext.WriteString(fmt.Sprintf("--- 文档来源: %s ---\n", asset.FileName))
				for _, chunk := range asset.Chunks {
					if totalLen+len(chunk.Content) > maxTotalLen {
						ragContext.WriteString("...(由于长度限制，后续内容已截断)\n")
						totalLen = maxTotalLen + 1
						break
					}
					ragContext.WriteString(chunk.Content)
					ragContext.WriteString("\n")
					totalLen += len(chunk.Content)
				}
				ragContext.WriteString("\n")
			}
		}
	} else {
		// 问答模式：执行高配版分块检索 (Top-30)
		chunks, err := SemanticSearchAssetChunks(util.DataDir, userQuery, 30, allowedAssets)
		if err == nil && len(chunks) > 0 {
			logging.LogInfof("RAG: 问答模式，找到 %d 个相关分块", len(chunks))
			ragContext.WriteString("以下是与用户问题高度相关的文档片段，请据此回答。注意区分不同来源，不要强行拼凑逻辑：\n\n")
			ragContext.WriteString("以下是与用户问题高度相关的文档片段，请据此回答。注意区分不同来源，不要强行拼凑逻辑：\n\n")
			totalLen := 0
			const maxQALen = 30000 // 问答模式大幅扩容至 30,000 字符，提供深层上下文
			for i, chunk := range chunks {
				if totalLen+len(chunk.Content) > maxQALen {
					ragContext.WriteString("...(由于长度限制，后续片段已忽略)\n")
					break
				}
				ragContext.WriteString(fmt.Sprintf("【片段%d - 来源: %s】\n%s\n\n", i+1, chunk.Source, chunk.Content))
				totalLen += len(chunk.Content)
			}
		}
	}

	if ragContext.Len() == 0 {
		return messages
	}

	// 将 RAG 上下文添加到 system 消息中
	ragSystemMsg := openai.ChatCompletionMessage{
		Role:    "system",
		Content: "你是一个严谨的文档分析专家。请分别参考以下不同来源的文档内容回答。\n" +
				"**绝对规则：**\n" +
				"1. 严禁在没有明确文档支持的情况下，强行关联或合并不同文档或不同领域的技术逻辑。\n" +
				"2. 如果不同文档讨论的是不相关的领域（如某种化学工艺 vs 另一种无关中间体），请分段分别陈述，严禁进行逻辑拼凑。\n" +
				"3. 必须在回答中明确提到信息来源（如“根据XXX文档...”）。\n\n" +
				ragContext.String(),
	}

	// 在消息列表开头插入 RAG 上下文
	enhancedMessages := make([]openai.ChatCompletionMessage, 0, len(messages)+1)
	enhancedMessages = append(enhancedMessages, ragSystemMsg)
	enhancedMessages = append(enhancedMessages, messages...)

	return enhancedMessages
}

func isOpenAIAPIEnabled() bool {
	// 如果配置了USE_DEFAULT_CONFIG或builtin provider，说明使用内置模型，也应该启用
	if Conf.AI.OpenAI.APIKey == "USE_DEFAULT_CONFIG" || Conf.AI.OpenAI.APIModel == "USE_DEFAULT_CONFIG" || Conf.AI.OpenAI.APIProvider == "builtin" {
		return true
	}
	// 如果APIKey为空，尝试检查是否有默认配置可用
	if "" == Conf.AI.OpenAI.APIKey {
		// 检查默认配置文件是否存在
		configPath := "/root/code/unified-settings-service/config/default-models.json"
		if _, err := os.Stat(configPath); err == nil {
			return true // 有默认配置文件，允许使用
		}
		util.PushMsg(Conf.Language(193), 5000)
		return false
	}
	return true
}

func getBlocksContent(ids []string) string {
	var nodes []*ast.Node
	trees := map[string]*parse.Tree{}
	for _, id := range ids {
		bt := treenode.GetBlockTree(id)
		if nil == bt {
			continue
		}

		var tree *parse.Tree
		if tree = trees[bt.RootID]; nil == tree {
			tree, _ = LoadTreeByBlockID(bt.RootID)
			if nil == tree {
				continue
			}

			trees[bt.RootID] = tree
		}

		if node := treenode.GetNodeInTree(tree, id); nil != node {
			if ast.NodeDocument == node.Type {
				for child := node.FirstChild; nil != child; child = child.Next {
					nodes = append(nodes, child)
				}
			} else {
				nodes = append(nodes, node)
			}
		}
	}

	luteEngine := util.NewLute()
	buf := bytes.Buffer{}
	for _, node := range nodes {
		md := treenode.ExportNodeStdMd(node, luteEngine)
		buf.WriteString(md)
		buf.WriteString("\n\n")
	}
	return buf.String()
}

type GPT interface {
	chat(msg string, contextMsgs []string) (partRet string, stop bool, err error)
}

type OpenAIGPT struct {
	c     *openai.Client
	model string
}

func (gpt *OpenAIGPT) chat(msg string, contextMsgs []string) (partRet string, stop bool, err error) {
	modelName := gpt.model
	if modelName == "" {
		modelName = Conf.AI.OpenAI.APIModel
	}
	return util.ChatGPT(msg, contextMsgs, gpt.c, modelName, Conf.AI.OpenAI.APIMaxTokens, Conf.AI.OpenAI.APITemperature, Conf.AI.OpenAI.APITimeout)
}

type CloudGPT struct {
}

func (gpt *CloudGPT) chat(msg string, contextMsgs []string) (partRet string, stop bool, err error) {
	return CloudChatGPT(msg, contextMsgs)
}

// ===== 新增向量化和AI文档分析功能 =====

// AIService AI服务接口
type AIService interface {
	VectorizeText(text string) ([]float64, error)
	SemanticSearch(query string, notebookID string, limit int) ([]*BlockVector, error)
	GenerateNotebookSummary(notebookID string) (*NotebookSummary, error)
}

// BlockVector 块向量数据
type BlockVector struct {
	ID         string    `json:"id"`
	NotebookID string    `json:"notebookId"`
	Content    string    `json:"content"`
	Vector     []float64 `json:"vector"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

// NotebookSummary 笔记本摘要
type NotebookSummary struct {
	NotebookID string    `json:"notebookId"`
	Summary    string    `json:"summary"`
	WordCount  int       `json:"wordCount"`
	Topics     []string  `json:"topics"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

// EmbeddingService 向量化服务接口
type EmbeddingService interface {
	VectorizeText(text string) ([]float64, error)
	IsEnabled() bool
}

// OpenAIService OpenAI LLM服务实现
type OpenAIService struct {
	client *openai.Client
}

// SiliconFlowEmbeddingService SiliconFlow向量化服务实现
type SiliconFlowEmbeddingService struct {
	apiKey         string
	baseURL        string
	model          string
	encodingFormat string
	timeout        int
}

// OpenAIEmbeddingService OpenAI向量化服务实现
type OpenAIEmbeddingService struct {
	client *openai.Client
}

// NewOpenAIService 创建OpenAI LLM服务
func NewOpenAIService() *OpenAIService {
	return &OpenAIService{
		client: util.NewOpenAIClient(Conf.AI.OpenAI.APIKey, Conf.AI.OpenAI.APIProxy, Conf.AI.OpenAI.APIBaseURL, Conf.AI.OpenAI.APIUserAgent, Conf.AI.OpenAI.APIVersion, Conf.AI.OpenAI.APIProvider),
	}
}

// GlobalEmbeddingConfig 全局向量化配置
type GlobalEmbeddingConfig struct {
	Provider       string `json:"provider"`
	APIKey         string `json:"apiKey"`
	Model          string `json:"model"`
	APIBaseURL     string `json:"apiBaseUrl"`
	EncodingFormat string `json:"encodingFormat"`
	Timeout        int    `json:"timeout"`
	Enabled        bool   `json:"enabled"`
}

// 全局配置文件路径
const globalEmbeddingConfigPath = "/root/code/unified-settings-service/config/embedding-config.json"

// loadGlobalEmbeddingConfig 加载全局向量化配置
func loadGlobalEmbeddingConfig() *GlobalEmbeddingConfig {
	data, err := os.ReadFile(globalEmbeddingConfigPath)
	if err != nil {
		logging.LogWarnf("读取全局向量化配置失败: %v", err)
		return nil
	}

	var config GlobalEmbeddingConfig
	if err := json.Unmarshal(data, &config); err != nil {
		logging.LogWarnf("解析全局向量化配置失败: %v", err)
		return nil
	}

	return &config
}

// NewEmbeddingService 创建向量化服务
// 优先使用全局配置，如果全局配置不存在或未启用，则使用用户配置
func NewEmbeddingService() EmbeddingService {
	// 优先尝试全局配置
	globalConfig := loadGlobalEmbeddingConfig()
	if globalConfig != nil && globalConfig.Enabled && globalConfig.APIKey != "" {
		logging.LogInfof("使用全局向量化配置: provider=%s, model=%s", globalConfig.Provider, globalConfig.Model)
		switch globalConfig.Provider {
		case "siliconflow":
			return &SiliconFlowEmbeddingService{
				apiKey:         globalConfig.APIKey,
				baseURL:        globalConfig.APIBaseURL,
				model:          globalConfig.Model,
				encodingFormat: globalConfig.EncodingFormat,
				timeout:        globalConfig.Timeout,
			}
		case "openai":
			return &OpenAIEmbeddingService{
				client: util.NewOpenAIClient(globalConfig.APIKey, "", globalConfig.APIBaseURL, "", "", "openai"),
			}
		}
	}

	// 回退到用户配置
	if Conf.AI.Embedding == nil || !Conf.AI.Embedding.Enabled {
		return nil
	}

	switch Conf.AI.Embedding.Provider {
	case "siliconflow":
		return &SiliconFlowEmbeddingService{
			apiKey:         Conf.AI.Embedding.APIKey,
			baseURL:        Conf.AI.Embedding.APIBaseURL,
			model:          Conf.AI.Embedding.Model,
			encodingFormat: Conf.AI.Embedding.EncodingFormat,
			timeout:        Conf.AI.Embedding.Timeout,
		}
	case "openai":
		return &OpenAIEmbeddingService{
			client: util.NewOpenAIClient(Conf.AI.Embedding.APIKey, "", Conf.AI.Embedding.APIBaseURL, "", "", "openai"),
		}
	default:
		return nil
	}
}

// IsEnabled 检查向量化服务是否启用
func (s *SiliconFlowEmbeddingService) IsEnabled() bool {
	return s.apiKey != "" && s.model != ""
}

// IsEnabled 检查向量化服务是否启用
func (s *OpenAIEmbeddingService) IsEnabled() bool {
	return s.client != nil
}

// VectorizeText 向量化文本 (SiliconFlow)
func (s *SiliconFlowEmbeddingService) VectorizeText(text string) ([]float64, error) {
	if text == "" {
		return nil, fmt.Errorf("文本为空")
	}

	// Create a custom client for SiliconFlow
	client := openai.NewClient(s.apiKey)
	if s.baseURL != "" {
		config := openai.DefaultConfig(s.apiKey)
		config.BaseURL = s.baseURL
		client = openai.NewClientWithConfig(config)
	}

	resp, err := client.CreateEmbeddings(context.Background(), openai.EmbeddingRequest{
		Model: openai.EmbeddingModel(s.model),
		Input: []string{text},
	})
	if err != nil {
		return nil, fmt.Errorf("SiliconFlow向量化失败: %v", err)
	}

	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("未获取到向量数据")
	}

	// Convert []float32 to []float64
	embedding := make([]float64, len(resp.Data[0].Embedding))
	for i, v := range resp.Data[0].Embedding {
		embedding[i] = float64(v)
	}
	return embedding, nil
}

// VectorizeText 向量化文本 (OpenAI)
func (s *OpenAIEmbeddingService) VectorizeText(text string) ([]float64, error) {
	if text == "" {
		return nil, fmt.Errorf("文本为空")
	}

	resp, err := s.client.CreateEmbeddings(context.Background(), openai.EmbeddingRequest{
		Model: openai.AdaEmbeddingV2,
		Input: []string{text},
	})
	if err != nil {
		return nil, fmt.Errorf("OpenAI向量化失败: %v", err)
	}

	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("未获取到向量数据")
	}

	// Convert []float32 to []float64
	embedding := make([]float64, len(resp.Data[0].Embedding))
	for i, v := range resp.Data[0].Embedding {
		embedding[i] = float64(v)
	}
	return embedding, nil
}

// SemanticSearch 语义搜索
func SemanticSearch(query string, notebookID string, limit int) ([]*BlockVector, error) {
	embeddingService := NewEmbeddingService()
	if embeddingService == nil || !embeddingService.IsEnabled() {
		return nil, fmt.Errorf("向量化服务未启用或未配置")
	}

	// 获取查询向量
	queryVector, err := embeddingService.VectorizeText(query)
	if err != nil {
		return nil, fmt.Errorf("向量化查询失败: %v", err)
	}

	// 加载所有块向量
	vectors, err := loadBlockVectors()
	if err != nil {
		return nil, fmt.Errorf("加载向量数据失败: %v", err)
	}

	// 计算相似度并排序
	type result struct {
		vector     *BlockVector
		similarity float64
	}

	var results []result
	for _, vector := range vectors {
		if notebookID != "" && vector.NotebookID != notebookID {
			continue
		}

		similarity := cosineSimilarity(queryVector, vector.Vector)
		if similarity > 0.7 { // 相似度阈值
			results = append(results, result{
				vector:     vector,
				similarity: similarity,
			})
		}
	}

	// 按相似度排序
	sort.Slice(results, func(i, j int) bool {
		return results[i].similarity > results[j].similarity
	})

	// 限制结果数量
	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	var blockVectors []*BlockVector
	for _, r := range results {
		blockVectors = append(blockVectors, r.vector)
	}

	return blockVectors, nil
}

// truncateAtSentence 在句子边界截断文本
func truncateAtSentence(text string, maxChars int) string {
	if len(text) <= maxChars {
		return text
	}

	// 截取到 maxChars
	truncated := text[:maxChars]

	// 尝试在句子结束符处截断
	sentenceEnders := []string{"。", "！", "？", ".", "!", "?", "\n"}
	lastPos := -1
	for _, ender := range sentenceEnders {
		if pos := strings.LastIndex(truncated, ender); pos > lastPos {
			lastPos = pos
		}
	}

	// 如果找到句子结束符，在那里截断
	if lastPos > maxChars/2 {
		return truncated[:lastPos+1]
	}

	// 否则直接截断
	return truncated
}

// cosineSimilarity 计算余弦相似度
func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct, normA, normB float64
	for i := 0; i < len(a); i++ {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

// GenerateNotebookSummary 生成笔记本摘要
func GenerateNotebookSummary(notebookID string) (*NotebookSummary, error) {
	if !isOpenAIAPIEnabled() {
		return nil, fmt.Errorf("AI功能未启用")
	}

	// 获取笔记本中的所有块内容
	blocks, err := sql.GetBlocksByBox(notebookID)
	if err != nil {
		return nil, fmt.Errorf("获取笔记本块失败: %v", err)
	}

	var contents []string
	var wordCount int

	for _, block := range blocks {
		if block.Content != "" && len(block.Content) > 10 {
			contents = append(contents, block.Content)
			wordCount += len(strings.Fields(block.Content))
		}
	}

	if len(contents) == 0 {
		return nil, fmt.Errorf("笔记本内容为空")
	}

	// 限制内容长度
	allContent := strings.Join(contents[:min(20, len(contents))], "\n\n")
	if len(allContent) > 8000 {
		allContent = allContent[:8000] + "..."
	}

	// 使用AI生成摘要
	summary, topics, err := generateSummaryWithAI(allContent)
	if err != nil {
		return nil, fmt.Errorf("生成摘要失败: %v", err)
	}

	notebookSummary := &NotebookSummary{
		NotebookID: notebookID,
		Summary:    summary,
		WordCount:  wordCount,
		Topics:     topics,
		UpdatedAt:  time.Now(),
	}

	// 保存摘要
	if err := saveNotebookSummary(notebookSummary); err != nil {
		logging.LogWarnf("保存笔记本摘要失败: %v", err)
	}

	return notebookSummary, nil
}

// generateSummaryWithAI 使用AI生成摘要
func generateSummaryWithAI(content string) (string, []string, error) {
	client := util.NewOpenAIClient(Conf.AI.OpenAI.APIKey, Conf.AI.OpenAI.APIProxy, Conf.AI.OpenAI.APIBaseURL, Conf.AI.OpenAI.APIUserAgent, Conf.AI.OpenAI.APIVersion, Conf.AI.OpenAI.APIProvider)

	prompt := `你是一个专业的内容分析师。请对以下内容进行总结，并提取主要话题。
请用JSON格式返回，包含以下字段：
- summary: 详细的摘要（200-500字）
- topics: 主要话题列表（3-5个关键词）

内容：
` + content

	resp, err := client.CreateChatCompletion(context.Background(), openai.ChatCompletionRequest{
		Model: Conf.AI.OpenAI.APIModel,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: prompt,
			},
		},
		MaxTokens:   2000,
		Temperature: 0.7,
	})

	if err != nil {
		return "", nil, err
	}

	if len(resp.Choices) == 0 {
		return "", nil, fmt.Errorf("未获取到AI回复")
	}

	content = resp.Choices[0].Message.Content

	// 尝试解析JSON格式的回复
	var aiResponse struct {
		Summary string   `json:"summary"`
		Topics  []string `json:"topics"`
	}

	if err := json.Unmarshal([]byte(content), &aiResponse); err != nil {
		// 如果解析失败，使用原始内容作为摘要
		return content, []string{}, nil
	}

	return aiResponse.Summary, aiResponse.Topics, nil
}

// VectorizeBlock 向量化单个块
func VectorizeBlock(blockID string) error {
	embeddingService := NewEmbeddingService()
	if embeddingService == nil || !embeddingService.IsEnabled() {
		return fmt.Errorf("向量化服务未启用或未配置")
	}

	block := sql.GetBlock(blockID)
	if block == nil {
		return fmt.Errorf("获取块失败: 块不存在")
	}

	if block.Content == "" {
		return fmt.Errorf("块内容为空")
	}

	vector, err := embeddingService.VectorizeText(block.Content)
	if err != nil {
		return fmt.Errorf("向量化失败: %v", err)
	}

	blockVector := &BlockVector{
		ID:         blockID,
		NotebookID: block.Box,
		Content:    block.Content,
		Vector:     vector,
		UpdatedAt:  time.Now(),
	}

	return saveBlockVector(blockVector)
}

// VectorChunk 单个内容块及其向量
type VectorChunk struct {
	ID      string    `json:"id"`
	Source  string    `json:"source"`  // 片段来源文件名
	Content string    `json:"content"`
	Vector  []float64 `json:"vector"`
}

// AssetVector 资源文件向量数据
type AssetVector struct {
	ID        string                 `json:"id"`        // 资源文件ID（基于路径生成）
	AssetPath string                 `json:"assetPath"` // 资源文件路径
	FileName  string                 `json:"fileName"`  // 文件名
	FileType  string                 `json:"fileType"`  // 文件类型
	Content   string                 `json:"content"`   // 解析后的全文预览（或第一块内容）
	Vector    []float64              `json:"vector"`    // 整体向量（或第一块向量），用于快速初步搜索
	Chunks    []*VectorChunk         `json:"chunks"`    // 分块向量数据
	UpdatedAt time.Time              `json:"updatedAt"` // 更新时间
	Metadata  map[string]interface{} `json:"metadata"`  // 额外元数据
}

// VectorizeAsset 向量化单个资源文件
// assetPath 必须是绝对路径
func VectorizeAsset(assetPath string) (*AssetVector, error) {
	embeddingService := NewEmbeddingService()
	if embeddingService == nil || !embeddingService.IsEnabled() {
		return nil, fmt.Errorf("向量化服务未启用或未配置")
	}

	// 解析资源文件内容
	content, err := ParseAttachment(assetPath)
	if err != nil {
		return nil, fmt.Errorf("解析资源文件失败: %v", err)
	}

	if strings.TrimSpace(content) == "" || strings.Contains(content, "找到的 PDF 文件没有找到") || strings.Contains(content, "解析失败") {
		return nil, fmt.Errorf("资源文件内容无效或解析失败，跳过向量化")
	}

	// 分块逻辑：每 2000 字符一块，重叠 200 字符
	const chunkSize = 2000
	const overlap = 200
	var chunks []*VectorChunk
	runes := []rune(content)
	
	logging.LogInfof("开始分块向量化资源文件: %s, 总长度: %d", filepath.Base(assetPath), len(runes))

	for i := 0; i < len(runes); i += (chunkSize - overlap) {
		end := i + chunkSize
		if end > len(runes) {
			end = len(runes)
		}
		
		chunkText := string(runes[i:end])
		if len(strings.TrimSpace(chunkText)) < 10 {
			if end == len(runes) { break }
			continue
		}

		// 向量化文本
		vector, err := embeddingService.VectorizeText(chunkText)
		if err != nil {
			logging.LogErrorf("分块向量化失败 (块 %d): %v", len(chunks), err)
			continue
		}

		chunks = append(chunks, &VectorChunk{
			ID:      fmt.Sprintf("%s_c%d", assetPath, len(chunks)),
			Source:  filepath.Base(assetPath),
			Content: chunkText,
			Vector:  vector,
		})
		
		if end == len(runes) {
			break
		}
	}

	if len(chunks) == 0 {
		return nil, fmt.Errorf("分块向量化失败，没有生成任何有效分块")
	}

	// 生成资源ID（基于路径的哈希）
	h := md5.New()
	h.Write([]byte(assetPath))
	assetID := fmt.Sprintf("%x", h.Sum(nil))

	// 获取文件名和类型
	fileName := filepath.Base(assetPath)
	fileType := strings.ToLower(strings.TrimPrefix(filepath.Ext(assetPath), "."))

	// 创建资源向量对象
	assetVector := &AssetVector{
		ID:        assetID,
		AssetPath: assetPath,
		FileName:  fileName,
		FileType:  fileType,
		Content:   chunks[0].Content, // 第一块内容作为预览
		Vector:    chunks[0].Vector,  // 第一块向量作为主向量
		Chunks:    chunks,
		UpdatedAt: time.Now(),
		Metadata: map[string]interface{}{
			"contentLength": len(content),
			"chunkCount":    len(chunks),
			"vectorDim":     len(chunks[0].Vector),
		},
	}

	// 保存向量文件（与资源文件同目录）
	if err := saveAssetVector(assetVector); err != nil {
		return nil, fmt.Errorf("保存向量数据失败: %v", err)
	}

	logging.LogInfof("资源文件 %s 分块向量化成功，共 %d 块，向量文件: %s", fileName, len(chunks), getVectorFilePath(assetPath))
	return assetVector, nil
}

// saveBlockVector 保存块向量
func saveBlockVector(blockVector *BlockVector) error {
	vectors, err := loadBlockVectors()
	if err != nil {
		vectors = make(map[string]*BlockVector)
	}

	vectors[blockVector.ID] = blockVector

	data, err := json.MarshalIndent(vectors, "", "  ")
	if err != nil {
		return err
	}

	vectorPath := filepath.Join(util.DataDir, "block_vectors.json")
	return os.WriteFile(vectorPath, data, 0644)
}

// loadBlockVectors 加载块向量
func loadBlockVectors() (map[string]*BlockVector, error) {
	vectors := make(map[string]*BlockVector)

	vectorPath := filepath.Join(util.DataDir, "block_vectors.json")
	if !gulu.File.IsExist(vectorPath) {
		return vectors, nil
	}

	data, err := os.ReadFile(vectorPath)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(data, &vectors)
	return vectors, err
}

// getVectorFilePath 根据资源文件路径获取对应的向量文件路径
// 例如: /path/to/assets/doc.pdf -> /path/to/assets/doc.pdf.vectors.json
func getVectorFilePath(assetPath string) string {
	return assetPath + ".vectors.json"
}

// saveAssetVector 保存资源文件向量（存储在资源文件同目录）
func saveAssetVector(assetVector *AssetVector) error {
	vectorPath := getVectorFilePath(assetVector.AssetPath)

	data, err := json.MarshalIndent(assetVector, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(vectorPath, data, 0644)
}

// loadAssetVector 加载单个资源文件的向量
func loadAssetVector(assetPath string) (*AssetVector, error) {
	vectorPath := getVectorFilePath(assetPath)

	if !gulu.File.IsExist(vectorPath) {
		return nil, fmt.Errorf("向量文件不存在: %s", vectorPath)
	}

	data, err := os.ReadFile(vectorPath)
	if err != nil {
		return nil, err
	}

	var vector AssetVector
	err = json.Unmarshal(data, &vector)
	if err != nil {
		return nil, err
	}

	return &vector, nil
}

// loadAllAssetVectors 扫描工作空间下所有向量文件
func loadAllAssetVectors(dataDir string) ([]*AssetVector, error) {
	var vectors []*AssetVector

	// 扫描 assets 目录下所有 .vectors.json 文件
	assetsDir := filepath.Join(dataDir, "assets")
	if !gulu.File.IsDir(assetsDir) {
		return vectors, nil
	}

	err := filepath.Walk(assetsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // 忽略错误，继续扫描
		}

		if info.IsDir() {
			return nil
		}

		if strings.HasSuffix(path, ".vectors.json") {
			data, err := os.ReadFile(path)
			if err != nil {
				logging.LogWarnf("读取向量文件失败 [%s]: %v", path, err)
				return nil
			}

			var vector AssetVector
			if err := json.Unmarshal(data, &vector); err != nil {
				logging.LogWarnf("解析向量文件失败 [%s]: %v", path, err)
				return nil
			}

			// 兼容性逻辑：如果是旧版索引（没有 Chunks）
			if len(vector.Chunks) == 0 && vector.Content != "" {
				logging.LogInfof("检测到旧版向量索引 [%s]，正在进行兼容处理并排期升级", vector.FileName)
				// 1. 临时封装为一个 Chunk，确保当前功能不落空
				vector.Chunks = append(vector.Chunks, &VectorChunk{
					ID:      vector.ID + "_legacy",
					Source:  vector.FileName,
					Content: vector.Content,
					Vector:  vector.Vector,
				})
				// 2. 将其加入异步队列，静默生成全量分块索引
				EnqueueAssetVectorize(vector.AssetPath)
			}

			vectors = append(vectors, &vector)
		}

		return nil
	})

	if err != nil {
		logging.LogWarnf("扫描向量文件时出错: %v", err)
	}

	return vectors, nil
}

// GetVectorizedAssetsWithContext 获取已向量化的资源文件列表（支持用户上下文）
func GetVectorizedAssetsWithContext(ctx *WorkspaceContext) ([]*AssetVector, error) {
	if ctx == nil {
		return nil, fmt.Errorf("用户上下文不能为空")
	}
	return GetVectorizedAssets(ctx.GetDataDir())
}

// GetVectorizedAssets 获取已向量化的资源文件列表
func GetVectorizedAssets(dataDir string) ([]*AssetVector, error) {
	vectors, err := loadAllAssetVectors(dataDir)
	if err != nil {
		return nil, err
	}

	// 按更新时间倒序排序
	sort.Slice(vectors, func(i, j int) bool {
		return vectors[i].UpdatedAt.After(vectors[j].UpdatedAt)
	})

	return vectors, nil
}

type assetSearchResult struct {
	chunk      *VectorChunk
	similarity float64
	rerankerScore float64  // 重排序分数
}

// RerankerService 重排序服务
type RerankerService struct {
	provider string
	model    string
	apiKey   string
	baseURL  string
	enabled  bool
}

// NewRerankerService 创建重排序服务实例
func NewRerankerService() *RerankerService {
	// 使用全局向量化配置
	config := loadGlobalEmbeddingConfig()
	if config == nil || !config.Enabled {
		return &RerankerService{enabled: false}
	}

	return &RerankerService{
		provider: config.Provider,
		model:    "BAAI/bge-reranker-v2-m3", // 固定使用重排序模型
		apiKey:   config.APIKey,
		baseURL:  config.APIBaseURL,
		enabled:  true,
	}
}

// IsEnabled 检查重排序服务是否启用
func (s *RerankerService) IsEnabled() bool {
	return s != nil && s.enabled
}

// RerankRequest 重排序请求
type RerankRequest struct {
	Model string   `json:"model"`
	Query string   `json:"query"`
	Documents []string `json:"documents"`
}

// RerankResponse 重排序响应
type RerankResponse struct {
	Results []RerankResult `json:"results"`
}

// RerankResult 单个重排序结果
type RerankResult struct {
	Index int     `json:"index"`
	Score float64 `json:"relevance_score"`
}

// Rerank 对文档列表进行重排序
func (s *RerankerService) Rerank(query string, documents []string) ([]float64, error) {
	if !s.IsEnabled() {
		return nil, fmt.Errorf("重排序服务未启用")
	}

	if len(documents) == 0 {
		return []float64{}, nil
	}

	reqBody := RerankRequest{
		Model:     s.model,
		Query:     query,
		Documents: documents,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %v", err)
	}

	// 构建 API URL
	apiURL := s.baseURL
	if !strings.HasSuffix(apiURL, "/") {
		apiURL += "/"
	}
	apiURL += "rerank"

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API 返回错误 %d: %s", resp.StatusCode, string(body))
	}

	var rerankResp RerankResponse
	if err := json.Unmarshal(body, &rerankResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	// 构建分数数组（按原始索引顺序）
	scores := make([]float64, len(documents))
	for _, result := range rerankResp.Results {
		if result.Index >= 0 && result.Index < len(scores) {
			scores[result.Index] = result.Score
		}
	}

	return scores, nil
}

// SemanticSearchAssetChunksWithContext 资源文件分块语义搜索（带重排序，支持用户上下文）
func SemanticSearchAssetChunksWithContext(ctx *WorkspaceContext, query string, limit int, allowedAssets []string) ([]*VectorChunk, error) {
	if ctx == nil {
		return nil, fmt.Errorf("用户上下文不能为空")
	}
	return SemanticSearchAssetChunks(ctx.GetDataDir(), query, limit, allowedAssets)
}

// SemanticSearchAssetChunks 资源文件分块语义搜索（带重排序）
func SemanticSearchAssetChunks(dataDir, query string, limit int, allowedAssets []string) ([]*VectorChunk, error) {
	embeddingService := NewEmbeddingService()
	if embeddingService == nil || !embeddingService.IsEnabled() {
		return nil, fmt.Errorf("向量化服务未启用或未配置")
	}

	// 获取查询向量
	queryVector, err := embeddingService.VectorizeText(query)
	if err != nil {
		return nil, fmt.Errorf("向量化查询失败: %v", err)
	}

	// 加载工作空间下所有资源向量
	assets, err := loadAllAssetVectors(dataDir)
	if err != nil {
		return nil, fmt.Errorf("加载资源向量数据失败: %v", err)
	}

	var results []assetSearchResult
	for _, asset := range assets {
		// 跨笔记本隔离
		if len(allowedAssets) > 0 {
			found := false
			for _, allowed := range allowedAssets {
				if asset.FileName == allowed || strings.Contains(asset.AssetPath, allowed) {
					found = true
					break
				}
			}
			if !found { continue }
		}

		for _, chunk := range asset.Chunks {
			sim := cosineSimilarity(queryVector, chunk.Vector)
			if sim > 0.4 {
				results = append(results, assetSearchResult{
					chunk:      chunk,
					similarity: sim,
					rerankerScore: 0, // 初始化为 0
				})
			}
		}
	}

	// 按向量相似度排序
	sort.Slice(results, func(i, j int) bool {
		return results[i].similarity > results[j].similarity
	})

	// 第一阶段：向量检索，取 Top-K（K = limit * 3，为重排序提供更多候选）
	firstStageLimit := limit * 3
	if firstStageLimit > 100 {
		firstStageLimit = 100 // 最多 100 个候选
	}
	if len(results) > firstStageLimit {
		results = results[:firstStageLimit]
	}

	// 第二阶段：重排序（如果启用）
	rerankerService := NewRerankerService()
	if rerankerService != nil && rerankerService.IsEnabled() && len(results) > 0 {
		logging.LogInfof("RAG: 使用重排序模型优化检索结果，候选数: %d", len(results))
		
		// 准备文档列表
		documents := make([]string, len(results))
		for i, r := range results {
			documents[i] = r.chunk.Content
		}

		// 调用重排序 API
		scores, err := rerankerService.Rerank(query, documents)
		if err != nil {
			logging.LogWarnf("重排序失败，使用原始向量相似度排序: %v", err)
		} else {
			// 更新重排序分数
			for i := range results {
				if i < len(scores) {
					results[i].rerankerScore = scores[i]
				}
			}

			// 按重排序分数重新排序
			sort.Slice(results, func(i, j int) bool {
				return results[i].rerankerScore > results[j].rerankerScore
			})

			logging.LogInfof("RAG: 重排序完成，Top-3 分数: %.4f, %.4f, %.4f", 
				results[0].rerankerScore,
				getScoreOrZero(results, 1),
				getScoreOrZero(results, 2))
		}
	}

	// 返回最终的 Top-N 结果
	if len(results) > limit {
		results = results[:limit]
	}

	var finalResults []*VectorChunk
	for _, r := range results {
		finalResults = append(finalResults, r.chunk)
	}

	return finalResults, nil
}

// getScoreOrZero 安全获取分数，避免越界
func getScoreOrZero(results []assetSearchResult, index int) float64 {
	if index < len(results) {
		return results[index].rerankerScore
	}
	return 0.0
}

// 异步向量化队列（只需要 assetPath，向量文件存储在同目录）
var assetVectorizeQueue = make(chan string, 1000)

// StartAssetVectorizeWorker 启动资源向量化工作协程
func StartAssetVectorizeWorker() {
	go func() {
		for assetPath := range assetVectorizeQueue {
			// 简单的防抖或延迟，等待文件完全写入
			time.Sleep(2 * time.Second)

			// 检查是否是支持的文档类型
			ext := strings.ToLower(filepath.Ext(assetPath))
			switch ext {
			case ".pdf", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx", ".md", ".txt":
				logging.LogInfof("开始自动向量化资源文件: %s", assetPath)
				if _, err := VectorizeAsset(assetPath); err != nil {
					logging.LogErrorf("自动向量化失败 [%s]: %v", assetPath, err)
				} else {
					logging.LogInfof("自动向量化成功: %s", assetPath)
				}
			}
		}
	}()
}

// EnqueueAssetVectorize 将资源文件加入向量化队列
// assetPath 必须是绝对路径
func EnqueueAssetVectorize(assetPath string) {
	logging.LogInfof("尝试加入向量化队列: %s", assetPath)

	// 检查向量化服务是否启用
	embeddingService := NewEmbeddingService()
	if embeddingService == nil || !embeddingService.IsEnabled() {
		logging.LogWarnf("向量化服务未启用，跳过: %s", assetPath)
		return
	}

	select {
	case assetVectorizeQueue <- assetPath:
		logging.LogInfof("已加入向量化队列: %s", assetPath)
	default:
		logging.LogWarnf("向量化队列已满，丢弃任务: %s", assetPath)
	}
}

// saveNotebookSummary 保存笔记本摘要
func saveNotebookSummary(summary *NotebookSummary) error {
	summaryPath := filepath.Join(util.DataDir, "notebook_summaries.json")

	summaries := make(map[string]*NotebookSummary)
	if gulu.File.IsExist(summaryPath) {
		data, err := os.ReadFile(summaryPath)
		if err == nil {
			json.Unmarshal(data, &summaries)
		}
	}

	summaries[summary.NotebookID] = summary

	data, err := json.MarshalIndent(summaries, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(summaryPath, data, 0644)
}

// BatchVectorizeNotebook 批量向量化笔记本
func BatchVectorizeNotebook(notebookID string) error {
	embeddingService := NewEmbeddingService()
	if embeddingService == nil || !embeddingService.IsEnabled() {
		return fmt.Errorf("向量化服务未启用或未配置")
	}

	blocks, err := sql.GetBlocksByBox(notebookID)
	if err != nil {
		return fmt.Errorf("获取笔记本块失败: %v", err)
	}

	vectorized := 0
	for _, block := range blocks {
		if block.Content != "" && len(block.Content) > 10 {
			if err := VectorizeBlock(block.ID); err == nil {
				vectorized++
			}
		}

		// 限制处理速度，避免API限流
		if vectorized > 0 && vectorized%5 == 0 {
			time.Sleep(1 * time.Second)
		}
	}

	logging.LogInfof("笔记本 %s 完成向量化 %d 个块", notebookID, vectorized)
	return nil
}

// ParseAttachmentWithContext 解析附件内容（支持用户上下文）
func ParseAttachmentWithContext(ctx *WorkspaceContext, assetPath string) (string, error) {
	if ctx == nil {
		return "", fmt.Errorf("用户上下文不能为空")
	}
	
	// 处理assets路径
	var fullPath string
	if strings.HasPrefix(assetPath, "assets/") {
		// 相对于用户数据目录的assets路径
		fullPath = filepath.Join(ctx.GetDataDir(), assetPath)
	} else if strings.HasPrefix(assetPath, "/") {
		// 绝对路径
		fullPath = assetPath
	} else {
		// 其他情况，尝试在用户数据目录的assets下查找
		fullPath = filepath.Join(ctx.GetDataDir(), "assets", assetPath)
	}

	// 检查文件是否存在
	if !gulu.File.IsExist(fullPath) {
		return "", fmt.Errorf("文件不存在: %s", fullPath)
	}

	// 获取文件扩展名
	ext := strings.ToLower(filepath.Ext(fullPath))

	switch ext {
	case ".pdf":
		return parsePDF(fullPath)
	case ".txt", ".md", ".markdown", ".json", ".xml", ".html", ".htm", ".css", ".js", ".ts", ".go", ".py", ".java", ".c", ".cpp", ".h", ".sh", ".yaml", ".yml", ".toml", ".ini", ".conf", ".log":
		return parseTextFile(fullPath)
	case ".docx":
		return parseDocx(fullPath)
	case ".doc":
		return parseDoc(fullPath)
	case ".xlsx", ".xls":
		return parseExcel(fullPath)
	case ".pptx":
		return parsePptx(fullPath)
	case ".rtf":
		return parseRtf(fullPath)
	case ".odt":
		return parseOdt(fullPath)
	case ".csv":
		return parseCsv(fullPath)
	default:
		return "", fmt.Errorf("不支持的文件格式: %s (支持: pdf, doc, docx, xls, xlsx, pptx, txt, md, csv, rtf, odt等)", ext)
	}
}

// ParseAttachment 解析附件内容（兼容旧版本，使用全局 util.DataDir）
func ParseAttachment(assetPath string) (string, error) {
	// 处理assets路径
	var fullPath string
	if strings.HasPrefix(assetPath, "assets/") {
		// 相对于data目录的assets路径
		fullPath = filepath.Join(util.DataDir, assetPath)
	} else if strings.HasPrefix(assetPath, "/") {
		// 绝对路径
		fullPath = assetPath
	} else {
		// 其他情况，尝试在data/assets下查找
		fullPath = filepath.Join(util.DataDir, "assets", assetPath)
	}

	// 检查文件是否存在
	if !gulu.File.IsExist(fullPath) {
		return "", fmt.Errorf("文件不存在: %s", fullPath)
	}

	// 获取文件扩展名
	ext := strings.ToLower(filepath.Ext(fullPath))

	switch ext {
	case ".pdf":
		return parsePDF(fullPath)
	case ".txt", ".md", ".markdown", ".json", ".xml", ".html", ".htm", ".css", ".js", ".ts", ".go", ".py", ".java", ".c", ".cpp", ".h", ".sh", ".yaml", ".yml", ".toml", ".ini", ".conf", ".log":
		return parseTextFile(fullPath)
	case ".docx":
		return parseDocx(fullPath)
	case ".doc":
		return parseDoc(fullPath)
	case ".xlsx", ".xls":
		return parseExcel(fullPath)
	case ".pptx":
		return parsePptx(fullPath)
	case ".rtf":
		return parseRtf(fullPath)
	case ".odt":
		return parseOdt(fullPath)
	case ".csv":
		return parseCsv(fullPath)
	default:
		return "", fmt.Errorf("不支持的文件格式: %s (支持: pdf, doc, docx, xls, xlsx, pptx, txt, md, csv, rtf, odt等)", ext)
	}
}

// parsePDF 使用pdftotext解析PDF文件，如果是扫描版则自动调用OCR
// 优先使用已有的 Markdown 文件（OCR 生成的格式化文档）
func parsePDF(filePath string) (string, error) {
	// 1. 优先检查是否有 Markdown 文件（OCR 生成的格式化文档）
	mdPath := filePath + ".md"
	if gulu.File.IsExist(mdPath) {
		logging.LogInfof("发现 Markdown 文件，直接使用: %s", mdPath)
		mdContent, err := os.ReadFile(mdPath)
		if err == nil && len(mdContent) > 100 {
			content := string(mdContent)
			// 移除 Markdown 元数据头部
			if strings.HasPrefix(content, "#") {
				// 跳过标题和元数据
				lines := strings.Split(content, "\n")
				var contentLines []string
				skipHeader := true
				for _, line := range lines {
					if skipHeader {
						if strings.HasPrefix(line, "---") || strings.HasPrefix(line, ">") || strings.HasPrefix(line, "#") {
							continue
						}
						if strings.TrimSpace(line) == "" {
							continue
						}
						skipHeader = false
					}
					contentLines = append(contentLines, line)
				}
				content = strings.Join(contentLines, "\n")
			}
			
			content = strings.TrimSpace(content)
			if len(content) > 100 {
				logging.LogInfof("使用 Markdown 文件内容，长度: %d 字符", len(content))
				return content, nil
			}
		}
	}
	
	// 2. 检查是否有 OCR JSON 文件（可以快速读取）
	ocrJSONPath := filePath + ".ocr.json"
	if gulu.File.IsExist(ocrJSONPath) {
		logging.LogInfof("发现 OCR JSON 文件，读取文本: %s", ocrJSONPath)
		text, err := GetOCRText(filePath)
		if err == nil && len(strings.TrimSpace(text)) > 100 {
			logging.LogInfof("使用 OCR JSON 文本，长度: %d 字符", len(text))
			return text, nil
		}
	}
	
	// 3. 使用 pdftotext 命令行工具
	cmd := exec.Command("pdftotext", "-enc", "UTF-8", "-layout", filePath, "-")
	output, err := cmd.Output()
	if err != nil {
		// 尝试不带layout参数
		cmd = exec.Command("pdftotext", "-enc", "UTF-8", filePath, "-")
		output, err = cmd.Output()
		if err != nil {
			// pdftotext 失败，尝试 OCR
			logging.LogWarnf("pdftotext 解析失败，尝试 OCR: %v", err)
			return tryOCRForPDF(filePath)
		}
	}

	content := string(output)
	content = strings.TrimSpace(content)

	// 检查提取的文本是否足够（判断是否为扫描版PDF）
	cleanContent := strings.ReplaceAll(content, "\n", "")
	cleanContent = strings.ReplaceAll(cleanContent, " ", "")
	cleanContent = strings.TrimSpace(cleanContent)

	if len(cleanContent) < 50 {
		// 文本内容过少，可能是扫描版PDF，尝试OCR
		logging.LogInfof("PDF 文本内容过少 (%d 字符)，尝试 OCR: %s", len(cleanContent), filePath)
		ocrContent, ocrErr := tryOCRForPDF(filePath)
		if ocrErr == nil && len(ocrContent) > len(content) {
			return ocrContent, nil
		}
		// OCR 失败或结果不如原始提取，返回原始内容
		if content != "" {
			return content, nil
		}
		if ocrErr != nil {
			return "", fmt.Errorf("PDF内容为空且OCR失败: %v", ocrErr)
		}
	}

	// 限制返回内容长度，避免过大
	if len(content) > 50000 {
		content = content[:50000] + "\n...(内容已截断)"
	}

	return content, nil
}

// tryOCRForPDF 尝试对PDF进行OCR
func tryOCRForPDF(filePath string) (string, error) {
	// 检查 OCR 服务是否可用
	healthy, msg := PaddleOCRHealthCheck()
	if !healthy {
		return "", fmt.Errorf("OCR 服务不可用: %s", msg)
	}

	// 执行 OCR
	result, err := OCRAsset(filePath)
	if err != nil {
		return "", err
	}

	content := result.FullText
	if len(content) > 50000 {
		content = content[:50000] + "\n...(内容已截断)"
	}

	return content, nil
}

// parseTextFile 解析纯文本文件
func parseTextFile(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("读取文件失败: %v", err)
	}

	content := string(data)
	if len(content) > 50000 {
		content = content[:50000] + "\n...(内容已截断)"
	}

	return content, nil
}

// parseDocx 解析Word文档（简单实现，提取纯文本）
func parseDocx(filePath string) (string, error) {
	// 使用unzip提取document.xml然后解析
	// 这是一个简化实现，实际可能需要更完善的docx解析库
	cmd := exec.Command("unzip", "-p", filePath, "word/document.xml")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("DOCX解析失败: %v", err)
	}

	// 简单提取文本内容（去除XML标签）
	content := string(output)
	// 移除XML标签，保留文本
	content = removeXMLTags(content)
	content = strings.TrimSpace(content)

	if len(content) > 50000 {
		content = content[:50000] + "\n...(内容已截断)"
	}

	return content, nil
}

// removeXMLTags 移除XML标签
func removeXMLTags(s string) string {
	var result strings.Builder
	inTag := false
	for _, r := range s {
		if r == '<' {
			inTag = true
			continue
		}
		if r == '>' {
			inTag = false
			result.WriteRune(' ')
			continue
		}
		if !inTag {
			result.WriteRune(r)
		}
	}
	// 清理多余空白
	return strings.Join(strings.Fields(result.String()), " ")
}

// parseDoc 解析旧版Word文档(.doc)
func parseDoc(filePath string) (string, error) {
	// 优先尝试antiword
	cmd := exec.Command("antiword", "-m", "UTF-8", filePath)
	output, err := cmd.Output()
	if err == nil && len(output) > 0 {
		content := strings.TrimSpace(string(output))
		if len(content) > 50000 {
			content = content[:50000] + "\n...(内容已截断)"
		}
		return content, nil
	}

	// 备选：尝试catdoc
	cmd = exec.Command("catdoc", "-d", "utf-8", filePath)
	output, err = cmd.Output()
	if err != nil {
		return "", fmt.Errorf("DOC解析失败: %v (请确保安装了antiword或catdoc)", err)
	}

	content := strings.TrimSpace(string(output))
	if len(content) > 50000 {
		content = content[:50000] + "\n...(内容已截断)"
	}
	return content, nil
}

// parseExcel 解析Excel文件(.xlsx, .xls)
func parseExcel(filePath string) (string, error) {
	ext := strings.ToLower(filepath.Ext(filePath))

	if ext == ".xlsx" {
		// 解析xlsx (Office Open XML格式)
		cmd := exec.Command("unzip", "-p", filePath, "xl/sharedStrings.xml")
		output, err := cmd.Output()
		if err == nil && len(output) > 0 {
			content := removeXMLTags(string(output))
			content = strings.TrimSpace(content)
			if len(content) > 50000 {
				content = content[:50000] + "\n...(内容已截断)"
			}
			return content, nil
		}
	}

	// 尝试使用ssconvert (gnumeric)转换为CSV
	tmpFile := filepath.Join(os.TempDir(), "excel_temp.csv")
	defer os.Remove(tmpFile)

	cmd := exec.Command("ssconvert", filePath, tmpFile)
	if err := cmd.Run(); err == nil {
		data, err := os.ReadFile(tmpFile)
		if err == nil {
			content := string(data)
			if len(content) > 50000 {
				content = content[:50000] + "\n...(内容已截断)"
			}
			return content, nil
		}
	}

	return "", fmt.Errorf("Excel解析失败 (支持有限，建议转换为CSV)")
}

// parsePptx 解析PowerPoint文件(.pptx)
func parsePptx(filePath string) (string, error) {
	// 提取所有slide的文本
	cmd := exec.Command("unzip", "-p", filePath)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("PPTX解析失败: %v", err)
	}

	// 从输出中提取文本
	content := removeXMLTags(string(output))
	content = strings.TrimSpace(content)

	if content == "" {
		return "", fmt.Errorf("PPTX内容为空或无法提取文本")
	}

	if len(content) > 50000 {
		content = content[:50000] + "\n...(内容已截断)"
	}
	return content, nil
}

// parseRtf 解析RTF文件
func parseRtf(filePath string) (string, error) {
	// 尝试使用unrtf
	cmd := exec.Command("unrtf", "--text", filePath)
	output, err := cmd.Output()
	if err != nil {
		// 备选：直接读取并简单清理
		data, err := os.ReadFile(filePath)
		if err != nil {
			return "", fmt.Errorf("RTF解析失败: %v", err)
		}
		// 简单移除RTF控制字符
		content := string(data)
		// 移除 {\rtf1 ... } 等控制序列
		content = strings.ReplaceAll(content, "\\par", "\n")
		// 这是简化处理，可能不完美
		if len(content) > 50000 {
			content = content[:50000] + "\n...(内容已截断)"
		}
		return content, nil
	}

	content := strings.TrimSpace(string(output))
	if len(content) > 50000 {
		content = content[:50000] + "\n...(内容已截断)"
	}
	return content, nil
}

// parseOdt 解析OpenDocument文本文件(.odt)
func parseOdt(filePath string) (string, error) {
	// ODT是ZIP格式，内容在content.xml中
	cmd := exec.Command("unzip", "-p", filePath, "content.xml")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("ODT解析失败: %v", err)
	}

	content := removeXMLTags(string(output))
	content = strings.TrimSpace(content)

	if len(content) > 50000 {
		content = content[:50000] + "\n...(内容已截断)"
	}
	return content, nil
}

// parseCsv 解析CSV文件
func parseCsv(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("CSV读取失败: %v", err)
	}

	content := string(data)
	if len(content) > 50000 {
		content = content[:50000] + "\n...(内容已截断)"
	}
	return content, nil
}


// 批量向量化进度跟踪
type VectorizeProgress struct {
	IsRunning       bool      `json:"isRunning"`
	TotalFiles      int       `json:"totalFiles"`
	ProcessedFiles  int       `json:"processedFiles"`
	SuccessCount    int       `json:"successCount"`
	FailedCount     int       `json:"failedCount"`
	CurrentFile     string    `json:"currentFile"`
	StartTime       time.Time `json:"startTime"`
	LastUpdateTime  time.Time `json:"lastUpdateTime"`
	EstimatedTimeLeft string  `json:"estimatedTimeLeft"`
}

var (
	vectorizeProgress     VectorizeProgress
	vectorizeProgressLock sync.RWMutex
)

// GetVectorizeProgress 获取向量化进度
func GetVectorizeProgress() VectorizeProgress {
	vectorizeProgressLock.RLock()
	defer vectorizeProgressLock.RUnlock()
	
	progress := vectorizeProgress
	
	// 计算预计剩余时间
	if progress.IsRunning && progress.ProcessedFiles > 0 {
		elapsed := time.Since(progress.StartTime)
		avgTimePerFile := elapsed / time.Duration(progress.ProcessedFiles)
		remaining := progress.TotalFiles - progress.ProcessedFiles
		estimatedTime := avgTimePerFile * time.Duration(remaining)
		
		if estimatedTime < time.Minute {
			progress.EstimatedTimeLeft = fmt.Sprintf("%d 秒", int(estimatedTime.Seconds()))
		} else if estimatedTime < time.Hour {
			progress.EstimatedTimeLeft = fmt.Sprintf("%d 分钟", int(estimatedTime.Minutes()))
		} else {
			progress.EstimatedTimeLeft = fmt.Sprintf("%.1f 小时", estimatedTime.Hours())
		}
	}
	
	return progress
}

// updateVectorizeProgress 更新向量化进度
func updateVectorizeProgress(currentFile string, success bool) {
	vectorizeProgressLock.Lock()
	defer vectorizeProgressLock.Unlock()
	
	vectorizeProgress.ProcessedFiles++
	if success {
		vectorizeProgress.SuccessCount++
	} else {
		vectorizeProgress.FailedCount++
	}
	vectorizeProgress.CurrentFile = currentFile
	vectorizeProgress.LastUpdateTime = time.Now()
}

// BatchVectorizeAllAssets 批量向量化所有未向量化的资源文件
func BatchVectorizeAllAssets(dataDir string) {
	// 检查是否已经在运行
	vectorizeProgressLock.Lock()
	if vectorizeProgress.IsRunning {
		vectorizeProgressLock.Unlock()
		logging.LogWarnf("批量向量化任务已在运行中")
		return
	}
	
	// 初始化进度
	vectorizeProgress = VectorizeProgress{
		IsRunning:      true,
		StartTime:      time.Now(),
		LastUpdateTime: time.Now(),
	}
	vectorizeProgressLock.Unlock()
	
	logging.LogInfof("开始批量向量化所有资源文件...")
	
	// 检查向量化服务是否启用
	embeddingService := NewEmbeddingService()
	if embeddingService == nil || !embeddingService.IsEnabled() {
		logging.LogErrorf("向量化服务未启用或未配置")
		vectorizeProgressLock.Lock()
		vectorizeProgress.IsRunning = false
		vectorizeProgressLock.Unlock()
		return
	}
	
	// 支持的文档格式
	supportedExts := []string{".pdf", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx", ".md", ".txt"}
	
	// 扫描 assets 目录
	assetsDir := filepath.Join(dataDir, "assets")
	if !gulu.File.IsDir(assetsDir) {
		logging.LogWarnf("Assets 目录不存在: %s", assetsDir)
		vectorizeProgressLock.Lock()
		vectorizeProgress.IsRunning = false
		vectorizeProgressLock.Unlock()
		return
	}
	
	// 收集需要向量化的文件
	var filesToVectorize []string
	
	err := filepath.Walk(assetsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // 忽略错误，继续扫描
		}
		
		if info.IsDir() {
			return nil
		}
		
		// 检查文件扩展名
		ext := strings.ToLower(filepath.Ext(path))
		isSupported := false
		for _, supportedExt := range supportedExts {
			if ext == supportedExt {
				isSupported = true
				break
			}
		}
		
		if !isSupported {
			return nil
		}
		
		// 检查是否已有向量文件
		vectorFile := path + ".vectors.json"
		if gulu.File.IsExist(vectorFile) {
			return nil // 已向量化，跳过
		}
		
		filesToVectorize = append(filesToVectorize, path)
		return nil
	})
	
	if err != nil {
		logging.LogErrorf("扫描 assets 目录失败: %v", err)
	}
	
	// 更新总文件数
	vectorizeProgressLock.Lock()
	vectorizeProgress.TotalFiles = len(filesToVectorize)
	vectorizeProgressLock.Unlock()
	
	logging.LogInfof("发现 %d 个需要向量化的文件", len(filesToVectorize))
	
	if len(filesToVectorize) == 0 {
		logging.LogInfof("没有需要向量化的文件")
		vectorizeProgressLock.Lock()
		vectorizeProgress.IsRunning = false
		vectorizeProgressLock.Unlock()
		return
	}
	
	// 开始向量化
	for _, filePath := range filesToVectorize {
		fileName := filepath.Base(filePath)
		logging.LogInfof("正在向量化: %s", fileName)
		
		// 执行向量化
		_, err := VectorizeAsset(filePath)
		success := err == nil
		
		if success {
			logging.LogInfof("✓ 向量化成功: %s", fileName)
		} else {
			logging.LogErrorf("✗ 向量化失败: %s, 错误: %v", fileName, err)
		}
		
		// 更新进度
		updateVectorizeProgress(fileName, success)
		
		// 避免请求过快，每个文件间隔 2 秒
		time.Sleep(2 * time.Second)
	}
	
	// 完成
	vectorizeProgressLock.Lock()
	vectorizeProgress.IsRunning = false
	vectorizeProgress.CurrentFile = ""
	vectorizeProgressLock.Unlock()
	
	logging.LogInfof("批量向量化完成: 总计 %d 个文件, 成功 %d 个, 失败 %d 个",
		vectorizeProgress.TotalFiles,
		vectorizeProgress.SuccessCount,
		vectorizeProgress.FailedCount)
}


// 全局速率限制器
var (
	vectorizeRateLimiter *time.Ticker
	vectorizeSemaphore   chan struct{}
)

// StartGlobalVectorizeService 启动全局向量化服务
// 定期扫描所有用户的文档，自动向量化未处理的文件
// 优化版本：最大化利用 API 配额（RPM 2000, TPM 500,000）
func StartGlobalVectorizeService() {
	// 检查是否启用向量化服务
	embeddingService := NewEmbeddingService()
	if embeddingService == nil || !embeddingService.IsEnabled() {
		logging.LogWarnf("向量化服务未启用，跳过全局向量化服务")
		return
	}
	
	logging.LogInfof("启动全局向量化服务（高性能模式）...")
	logging.LogInfof("API 配额: RPM 2000, TPM 500,000")
	
	// 初始化速率限制器
	// 目标：每分钟 1800 次请求（留 10% 余量）
	// 每秒约 30 次请求
	vectorizeRateLimiter = time.NewTicker(33 * time.Millisecond) // 约 30 req/s
	
	// 初始化并发控制（同时处理 20 个文档）
	vectorizeSemaphore = make(chan struct{}, 20)
	
	go func() {
		// 启动后等待 30 秒，让系统完全启动
		time.Sleep(30 * time.Second)
		
		// 定期扫描间隔：每 10 分钟（更频繁的扫描）
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()
		
		// 立即执行一次
		scanAndVectorizeAllUsers()
		
		// 定期执行
		for range ticker.C {
			scanAndVectorizeAllUsers()
		}
	}()
}

// scanAndVectorizeAllUsers 扫描并向量化所有用户的文档
// 优化版本：并发处理，最大化利用 API 配额
func scanAndVectorizeAllUsers() {
	startTime := time.Now()
	logging.LogInfof("[全局向量化] 开始扫描所有用户的文档（高性能模式）...")
	
	// 获取用户数据根目录
	userDataRoot := "/root/code/MindOcean/user-data/notes"
	
	if !gulu.File.IsDir(userDataRoot) {
		logging.LogWarnf("[全局向量化] 用户数据目录不存在: %s", userDataRoot)
		return
	}
	
	// 扫描所有用户目录
	userDirs, err := os.ReadDir(userDataRoot)
	if err != nil {
		logging.LogErrorf("[全局向量化] 读取用户目录失败: %v", err)
		return
	}
	
	// 收集所有需要处理的文件
	type FileTask struct {
		Path     string
		Username string
		FileName string
	}
	
	var allTasks []FileTask
	supportedExts := []string{".pdf", ".doc", ".docx", ".xls", ".xlsx", ".pptx", ".md", ".txt"}
	
	for _, userDir := range userDirs {
		if !userDir.IsDir() {
			continue
		}
		
		username := userDir.Name()
		userAssetsDir := filepath.Join(userDataRoot, username, "assets")
		
		if !gulu.File.IsDir(userAssetsDir) {
			continue
		}
		
		// 扫描该用户的文档
		filepath.Walk(userAssetsDir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			
			// 检查文件扩展名
			ext := strings.ToLower(filepath.Ext(path))
			isSupported := false
			for _, supportedExt := range supportedExts {
				if ext == supportedExt {
					isSupported = true
					break
				}
			}
			
			if !isSupported {
				return nil
			}
			
			// 检查是否已有向量文件
			vectorFile := path + ".vectors.json"
			if gulu.File.IsExist(vectorFile) {
				return nil // 已向量化，跳过
			}
			
			allTasks = append(allTasks, FileTask{
				Path:     path,
				Username: username,
				FileName: filepath.Base(path),
			})
			
			return nil
		})
	}
	
	totalFiles := len(allTasks)
	if totalFiles == 0 {
		logging.LogInfof("[全局向量化] 本轮扫描完成: 没有需要向量化的文档")
		return
	}
	
	logging.LogInfof("[全局向量化] 发现 %d 个待处理文档，开始并发处理...", totalFiles)
	
	// 并发处理所有文件
	var wg sync.WaitGroup
	successCount := 0
	failedCount := 0
	var countMutex sync.Mutex
	
	for i, task := range allTasks {
		wg.Add(1)
		
		go func(task FileTask, index int) {
			defer wg.Done()
			
			// 获取信号量（限制并发数）
			vectorizeSemaphore <- struct{}{}
			defer func() { <-vectorizeSemaphore }()
			
			// 速率限制
			<-vectorizeRateLimiter.C
			
			// 执行向量化
			_, err := VectorizeAsset(task.Path)
			
			countMutex.Lock()
			if err == nil {
				successCount++
				if (index+1)%10 == 0 || index+1 == totalFiles {
					logging.LogInfof("[全局向量化] 进度: %d/%d (成功: %d, 失败: %d)",
						index+1, totalFiles, successCount, failedCount)
				}
			} else {
				failedCount++
				logging.LogErrorf("[全局向量化] [%s] ✗ 向量化失败: %s, 错误: %v",
					task.Username, task.FileName, err)
			}
			countMutex.Unlock()
			
		}(task, i)
	}
	
	// 等待所有任务完成
	wg.Wait()
	
	elapsed := time.Since(startTime)
	logging.LogInfof("[全局向量化] 本轮扫描完成: 总计 %d 个文档, 成功 %d 个, 失败 %d 个, 耗时 %.1f 分钟",
		totalFiles, successCount, failedCount, elapsed.Minutes())
	
	if successCount > 0 {
		avgTime := elapsed.Seconds() / float64(successCount)
		logging.LogInfof("[全局向量化] 平均处理速度: %.2f 秒/文档, 约 %.1f 文档/分钟",
			avgTime, 60.0/avgTime)
	}
}

// vectorizeUserAssets 向量化单个用户的资源文件
// 返回: (处理数量, 成功数量, 失败数量)
func vectorizeUserAssets(assetsDir string, username string) (int, int, int) {
	// 支持的文档格式
	supportedExts := []string{".pdf", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx", ".md", ".txt"}
	
	processed := 0
	success := 0
	failed := 0
	
	// 扫描文档
	err := filepath.Walk(assetsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // 忽略错误，继续扫描
		}
		
		if info.IsDir() {
			return nil
		}
		
		// 检查文件扩展名
		ext := strings.ToLower(filepath.Ext(path))
		isSupported := false
		for _, supportedExt := range supportedExts {
			if ext == supportedExt {
				isSupported = true
				break
			}
		}
		
		if !isSupported {
			return nil
		}
		
		// 检查是否已有向量文件
		vectorFile := path + ".vectors.json"
		if gulu.File.IsExist(vectorFile) {
			return nil // 已向量化，跳过
		}
		
		// 限制每轮最多处理 10 个文档（避免占用太多资源）
		if processed >= 10 {
			return filepath.SkipDir
		}
		
		fileName := filepath.Base(path)
		logging.LogInfof("[全局向量化] [%s] 正在向量化: %s", username, fileName)
		
		// 执行向量化
		_, err = VectorizeAsset(path)
		processed++
		
		if err == nil {
			success++
			logging.LogInfof("[全局向量化] [%s] ✓ 向量化成功: %s", username, fileName)
		} else {
			failed++
			logging.LogErrorf("[全局向量化] [%s] ✗ 向量化失败: %s, 错误: %v", username, fileName, err)
		}
		
		// 每个文档间隔 3 秒
		time.Sleep(3 * time.Second)
		
		return nil
	})
	
	if err != nil {
		logging.LogErrorf("[全局向量化] [%s] 扫描目录失败: %v", username, err)
	}
	
	return processed, success, failed
}
