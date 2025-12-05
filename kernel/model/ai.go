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
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
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
		// 搜索最相关的3个文档
		assets, err := SemanticSearchAssets(util.DataDir, msg, 3)
		if err == nil && len(assets) > 0 {
			var contextBuilder strings.Builder
			contextBuilder.WriteString("以下是相关的文档内容供参考：\n\n")
			for i, asset := range assets {
				// 截取内容以避免 Prompt 过长
				content := asset.Content
				if len(content) > 2000 {
					content = content[:2000] + "..."
				}
				contextBuilder.WriteString(fmt.Sprintf("【文档%d: %s】\n%s\n\n", i+1, asset.FileName, content))
			}
			contextBuilder.WriteString("请基于以上文档内容回答用户的问题：\n")
			contextBuilder.WriteString(msg)

			// 更新 msg
			msg = contextBuilder.String()
			logging.LogInfof("RAG 增强已启用，加载了 %d 个相关文档", len(assets))
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
		configPath := "/home/jason/code/unified-settings-service/config/default-models.json"
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

func Chat(messages []openai.ChatCompletionMessage) (ret string, err error) {
	if !isOpenAIAPIEnabled() {
		return "", fmt.Errorf("AI not enabled")
	}

	// RAG 增强：从用户消息中提取查询，搜索相关文档
	messages = enhanceMessagesWithRAG(messages)

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

// ChatStream 流式聊天，通过 channel 返回每个 token
func ChatStream(messages []openai.ChatCompletionMessage, onToken func(token string) error) error {
	if !isOpenAIAPIEnabled() {
		return fmt.Errorf("AI not enabled")
	}

	// RAG 增强
	messages = enhanceMessagesWithRAG(messages)

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

// enhanceMessagesWithRAG 使用 RAG 增强消息
func enhanceMessagesWithRAG(messages []openai.ChatCompletionMessage) []openai.ChatCompletionMessage {
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

	// 搜索相关文档
	assets, err := SemanticSearchAssets(util.DataDir, userQuery, 3)
	if err != nil || len(assets) == 0 {
		logging.LogInfof("RAG: 未找到相关文档或搜索失败: %v", err)
		return messages
	}

	// 构建 RAG 上下文
	var ragContext strings.Builder
	ragContext.WriteString("以下是与用户问题相关的文档内容，请参考这些内容回答：\n\n")
	for i, asset := range assets {
		content := asset.Content
		if len(content) > 1500 {
			content = content[:1500] + "..."
		}
		ragContext.WriteString(fmt.Sprintf("【相关文档%d: %s】\n%s\n\n", i+1, asset.FileName, content))
	}

	logging.LogInfof("RAG 增强已启用，找到 %d 个相关文档", len(assets))

	// 将 RAG 上下文添加到 system 消息中
	ragSystemMsg := openai.ChatCompletionMessage{
		Role:    "system",
		Content: ragContext.String(),
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
		configPath := "/home/jason/code/unified-settings-service/config/default-models.json"
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
const globalEmbeddingConfigPath = "/home/jason/code/unified-settings-service/config/embedding-config.json"

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

// AssetVector 资源文件向量数据
type AssetVector struct {
	ID        string                 `json:"id"`        // 资源文件ID（基于路径生成）
	AssetPath string                 `json:"assetPath"` // 资源文件路径
	FileName  string                 `json:"fileName"`  // 文件名
	FileType  string                 `json:"fileType"`  // 文件类型
	Content   string                 `json:"content"`   // 解析后的文本内容（用于展示，可截断）
	Vector    []float64              `json:"vector"`    // 向量数据
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

	if strings.TrimSpace(content) == "" {
		return nil, fmt.Errorf("资源文件内容为空")
	}

	// 限制内容长度用于向量化
	// 支持 8000 tokens 的模型，中文约 1 字符 = 1-2 tokens
	// 为安全起见，限制到 7000 字符
	vectorizeContent := content
	maxChars := 7000
	if len(vectorizeContent) > maxChars {
		// 尝试在句子边界截断
		vectorizeContent = truncateAtSentence(vectorizeContent, maxChars)
	}

	// 向量化文本
	vector, err := embeddingService.VectorizeText(vectorizeContent)
	if err != nil {
		return nil, fmt.Errorf("向量化失败: %v", err)
	}

	// 生成资源ID（基于路径的哈希）
	h := md5.New()
	h.Write([]byte(assetPath))
	assetID := fmt.Sprintf("%x", h.Sum(nil))

	// 获取文件名和类型
	fileName := filepath.Base(assetPath)
	fileType := strings.ToLower(strings.TrimPrefix(filepath.Ext(assetPath), "."))

	// 准备展示内容（截断到500字符）
	displayContent := content
	if len(displayContent) > 500 {
		displayContent = displayContent[:500] + "..."
	}

	// 创建资源向量对象
	assetVector := &AssetVector{
		ID:        assetID,
		AssetPath: assetPath,
		FileName:  fileName,
		FileType:  fileType,
		Content:   displayContent,
		Vector:    vector,
		UpdatedAt: time.Now(),
		Metadata: map[string]interface{}{
			"contentLength": len(content),
			"vectorDim":     len(vector),
		},
	}

	// 保存向量文件（与资源文件同目录）
	if err := saveAssetVector(assetVector); err != nil {
		return nil, fmt.Errorf("保存向量数据失败: %v", err)
	}

	logging.LogInfof("资源文件 %s 向量化成功，向量文件: %s", fileName, getVectorFilePath(assetPath))
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

			vectors = append(vectors, &vector)
		}

		return nil
	})

	if err != nil {
		logging.LogWarnf("扫描向量文件时出错: %v", err)
	}

	return vectors, nil
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

// SemanticSearchAssets 资源文件语义搜索
func SemanticSearchAssets(dataDir, query string, limit int) ([]*AssetVector, error) {
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
	vectors, err := loadAllAssetVectors(dataDir)
	if err != nil {
		return nil, fmt.Errorf("加载资源向量数据失败: %v", err)
	}

	// 计算相似度并排序
	type result struct {
		vector     *AssetVector
		similarity float64
	}

	var results []result
	for _, vector := range vectors {
		sim := cosineSimilarity(queryVector, vector.Vector)
		if sim > 0.5 { // 相似度阈值
			results = append(results, result{vector, sim})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].similarity > results[j].similarity
	})

	if len(results) > limit {
		results = results[:limit]
	}

	var finalResults []*AssetVector
	for _, r := range results {
		finalResults = append(finalResults, r.vector)
	}

	return finalResults, nil
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

// ParseAttachment 解析附件内容（支持PDF等格式）
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

// parsePDF 使用pdftotext解析PDF文件
func parsePDF(filePath string) (string, error) {
	// 使用pdftotext命令行工具
	cmd := exec.Command("pdftotext", "-enc", "UTF-8", "-layout", filePath, "-")
	output, err := cmd.Output()
	if err != nil {
		// 尝试不带layout参数
		cmd = exec.Command("pdftotext", "-enc", "UTF-8", filePath, "-")
		output, err = cmd.Output()
		if err != nil {
			return "", fmt.Errorf("PDF解析失败: %v", err)
		}
	}

	content := string(output)
	content = strings.TrimSpace(content)

	if content == "" {
		return "", fmt.Errorf("PDF内容为空或无法提取文本")
	}

	// 限制返回内容长度，避免过大
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
