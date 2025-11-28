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
	"encoding/json"
	"fmt"
	"math"
	"os"
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

	if apiKey == "USE_DEFAULT_CONFIG" || apiModel == "USE_DEFAULT_CONFIG" {
		configPath := "/home/jason/code/unified-settings-service/config/default-models.json"
		data, err := os.ReadFile(configPath)
		if err == nil {
			var models map[string]DefaultModelConfig
			if err := json.Unmarshal(data, &models); err == nil {
				targetConfig := models["builtin_free"]
				if val, ok := models["builtin_free_neuralink"]; ok {
					targetConfig = val
				}

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
	return
}

func Chat(messages []openai.ChatCompletionMessage) (ret string, err error) {
	if !isOpenAIAPIEnabled() {
		return "", fmt.Errorf("AI not enabled")
	}

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

func isOpenAIAPIEnabled() bool {
	if "" == Conf.AI.OpenAI.APIKey {
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

// NewEmbeddingService 创建向量化服务
func NewEmbeddingService() EmbeddingService {
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
