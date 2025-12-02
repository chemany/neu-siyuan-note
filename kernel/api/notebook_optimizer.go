// SiYuan - Refactor your thinking
// Copyright (c) 2020-present, b3log.org
//
// 笔记本优化模块 - 按笔记本名称组织数据，增强AI分析功能

package api

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/siyuan-note/siyuan/kernel/util"
)

// NotebookMetadata 笔记本元数据结构
type NotebookMetadata struct {
	Name      string            `json:"name"`
	Type      string            `json:"type"`
	Created   string            `json:"created"`
	Structure map[string]string `json:"structure"`
	Stats     struct {
		DocumentCount int `json:"documentCount"`
		RichNoteCount int `json:"richNoteCount"`
		VectorCount   int `json:"vectorCount"`
	} `json:"stats"`
	AIReady bool `json:"aiReady"`
}

// NotebookAnalysisData AI分析数据结构
type NotebookAnalysisData struct {
	Notebook         string                     `json:"notebook"`
	LastUpdated      string                     `json:"lastUpdated"`
	TotalDocuments   int                        `json:"totalDocuments"`
	ContentTypes     map[string]int              `json:"contentTypes"`
	SearchableContent []NotebookContentItem      `json:"searchableContent"`
}

// NotebookContentItem 笔记本内容项
type NotebookContentItem struct {
	File     string `json:"file"`
	Type     string `json:"type"`
	Preview  string `json:"preview"`
	FullText string `json:"fullText"`
}

// NotebookCategory 笔记本分类
type NotebookCategory struct {
	Name          string `json:"name"`
	Path          string `json:"path"`
	Type          string `json:"type"`
	HasRichNotes  bool   `json:"hasRichNotes"`
	HasDocuments  bool   `json:"hasDocuments"`
}

// organizeNotebooksByCategory 按笔记本名称组织数据
func organizeNotebooksByCategory(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request"})
		return
	}
	
	if req.Username == "" {
		req.Username = "jason"
	}
	
	// 获取工作空间路径
	workspaceDir := util.WorkspaceDir
	// 使用用户数据根目录
	userDataRoot := os.Getenv("SIYUAN_USER_DATA_ROOT")
	if userDataRoot == "" {
		userDataRoot = filepath.Join(workspaceDir, "data")
	}
	notesPath := filepath.Join(userDataRoot, req.Username)
	uploadsPath := filepath.Join(userDataRoot, "..", "uploads", req.Username)
	
	// 分析上传文件结构
	categories, err := analyzeUploadsStructure(uploadsPath)
	if err != nil {
		c.JSON(500, gin.H{"error": fmt.Sprintf("Failed to analyze uploads: %v", err)})
		return
	}
	
	// 创建基于笔记本的文件夹结构
	organizedPath, err := createNotebookBasedStructure(notesPath, categories)
	if err != nil {
		c.JSON(500, gin.H{"error": fmt.Sprintf("Failed to create structure: %v", err)})
		return
	}
	
	// 迁移内容到笔记本文件夹
	err = migrateContentToNotebookFolders(organizedPath, uploadsPath, categories)
	if err != nil {
		c.JSON(500, gin.H{"error": fmt.Sprintf("Failed to migrate content: %v", err)})
		return
	}
	
	c.JSON(200, gin.H{
		"success":    true,
		"categories": categories,
		"message":    "Notebook organization completed",
	})
}

// analyzeUploadsStructure 分析上传文件目录结构
func analyzeUploadsStructure(uploadsPath string) ([]NotebookCategory, error) {
	if _, err := os.Stat(uploadsPath); os.IsNotExist(err) {
		return []NotebookCategory{{Name: "default", Type: "notebook"}}, nil
	}
	
	files, err := ioutil.ReadDir(uploadsPath)
	if err != nil {
		return nil, err
	}
	
	var categories []NotebookCategory
	
	for _, file := range files {
		if file.IsDir() {
			categoryPath := filepath.Join(uploadsPath, file.Name())
			
			hasRichNotes := false
			if _, err := os.Stat(filepath.Join(categoryPath, "rich-notes")); err == nil {
				hasRichNotes = true
			}
			
			hasDocuments := hasDocuments(categoryPath)
			
			if hasRichNotes || hasDocuments {
				categories = append(categories, NotebookCategory{
					Name:         file.Name(),
					Path:         categoryPath,
					Type:         "notebook",
					HasRichNotes: hasRichNotes,
					HasDocuments: hasDocuments,
				})
			}
		}
	}
	
	return categories, nil
}

// hasDocuments 检查目录是否包含文档文件
func hasDocuments(dirPath string) bool {
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		return false
	}
	
	files, err := ioutil.ReadDir(dirPath)
	if err != nil {
		return false
	}
	
	docExtensions := map[string]bool{
		".pdf":  true,
		".docx": true,
		".xlsx": true,
		".pptx": true,
		".txt":  true,
		".md":   true,
		".html": true,
		".htm":  true,
	}
	
	for _, file := range files {
		if !file.IsDir() {
			ext := strings.ToLower(filepath.Ext(file.Name()))
			if docExtensions[ext] {
				return true
			}
		}
	}
	
	return false
}

// createNotebookBasedStructure 创建基于笔记本的文件夹结构
func createNotebookBasedStructure(notesPath string, categories []NotebookCategory) (string, error) {
	organizedPath := filepath.Join(notesPath, "organized")
	
	// 创建organized目录
	if err := os.MkdirAll(organizedPath, 0755); err != nil {
		return "", err
	}
	
	// 为每个分类创建笔记本结构
	for _, category := range categories {
		notebookPath := filepath.Join(organizedPath, category.Name)
		
		// 创建笔记本目录结构
		dirs := []string{
			notebookPath,
			filepath.Join(notebookPath, "documents"),
			filepath.Join(notebookPath, "rich-notes"),
			filepath.Join(notebookPath, "vectors"),
			filepath.Join(notebookPath, "assets"),
			filepath.Join(notebookPath, "analysis"),
		}
		
		for _, dir := range dirs {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return "", err
			}
		}
	}
	
	return organizedPath, nil
}

// migrateContentToNotebookFolders 迁移内容到笔记本文件夹
func migrateContentToNotebookFolders(organizedPath, uploadsPath string, categories []NotebookCategory) error {
	for _, category := range categories {
		sourcePath := filepath.Join(uploadsPath, category.Name)
		targetPath := filepath.Join(organizedPath, category.Name)
		
		// 迁移rich-notes
		richNotesSource := filepath.Join(sourcePath, "rich-notes")
		richNotesTarget := filepath.Join(targetPath, "rich-notes")
		if _, err := os.Stat(richNotesSource); err == nil {
			if err := copyDirectory(richNotesSource, richNotesTarget); err != nil {
				return err
			}
		}
		
		// 迁移文档文件
		documentsTarget := filepath.Join(targetPath, "documents")
		files, err := ioutil.ReadDir(sourcePath)
		if err != nil {
			continue
		}
		
		for _, file := range files {
			if file.IsDir() && file.Name() == "vectors" {
				// 迁移vectors
				vectorsTarget := filepath.Join(targetPath, "vectors")
				if err := copyDirectory(filepath.Join(sourcePath, "vectors"), vectorsTarget); err != nil {
					continue
				}
			} else if file.IsDir() {
				continue
			} else if isDocument(file.Name()) {
				// 迁移文档文件
				sourceFile := filepath.Join(sourcePath, file.Name())
				targetFile := filepath.Join(documentsTarget, file.Name())
				
				if _, err := os.Stat(targetFile); os.IsNotExist(err) {
					if err := copyNotebookFile(sourceFile, targetFile); err != nil {
						continue
					}
				}
			}
		}
		
		// 生成笔记本元数据
		if err := generateNotebookMetadata(category, targetPath); err != nil {
			continue
		}
	}
	
	return nil
}

// isDocument 判断文件是否为文档
func isDocument(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	docExtensions := map[string]bool{
		".pdf":  true,
		".docx": true,
		".xlsx": true,
		".pptx": true,
		".txt":  true,
		".md":   true,
		".html": true,
		".htm":  true,
	}
	return docExtensions[ext]
}

// copyDirectory 递归复制目录
func copyDirectory(src, dst string) error {
	if _, err := os.Stat(src); os.IsNotExist(err) {
		return nil
	}
	
	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}
	
	files, err := ioutil.ReadDir(src)
	if err != nil {
		return err
	}
	
	for _, file := range files {
		srcPath := filepath.Join(src, file.Name())
		dstPath := filepath.Join(dst, file.Name())
		
		if file.IsDir() {
			if err := copyDirectory(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := copyNotebookFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}
	
	return nil
}

// copyFile 复制文件
func copyNotebookFile(src, dst string) error {
	data, err := ioutil.ReadFile(src)
	if err != nil {
		return err
	}
	
	if err := ioutil.WriteFile(dst, data, 0644); err != nil {
		return err
	}
	
	return nil
}

// generateNotebookMetadata 生成笔记本元数据
func generateNotebookMetadata(category NotebookCategory, notebookPath string) error {
	metadata := NotebookMetadata{
		Name:    category.Name,
		Type:    "notebook",
		Created: getISOString(),
		Structure: map[string]string{
			"documents":   filepath.Join(notebookPath, "documents"),
			"rich-notes":  filepath.Join(notebookPath, "rich-notes"),
			"vectors":     filepath.Join(notebookPath, "vectors"),
			"assets":      filepath.Join(notebookPath, "assets"),
			"analysis":    filepath.Join(notebookPath, "analysis"),
		},
		AIReady: true,
	}
	
	// 统计文件数量
	metadata.Stats.DocumentCount = countFiles(filepath.Join(notebookPath, "documents"))
	metadata.Stats.RichNoteCount = countFiles(filepath.Join(notebookPath, "rich-notes"))
	metadata.Stats.VectorCount = countFiles(filepath.Join(notebookPath, "vectors"))
	
	metadataJSON, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return err
	}
	
	metadataPath := filepath.Join(notebookPath, "metadata.json")
	return ioutil.WriteFile(metadataPath, metadataJSON, 0644)
}

// countFiles 统计文件数量
func countFiles(dirPath string) int {
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		return 0
	}
	
	var count int
	filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			count++
		}
		return nil
	})
	
	return count
}

// prepareForAIAnalysis 为AI分析准备笔记本内容
func prepareForAIAnalysis(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request"})
		return
	}
	
	if req.Username == "" {
		req.Username = "jason"
	}
	
	workspaceDir := util.WorkspaceDir
	// 使用用户数据根目录
	userDataRoot := os.Getenv("SIYUAN_USER_DATA_ROOT")
	if userDataRoot == "" {
		userDataRoot = filepath.Join(workspaceDir, "data")
	}
	organizedPath := filepath.Join(userDataRoot, req.Username, "organized")
	
	if _, err := os.Stat(organizedPath); os.IsNotExist(err) {
		c.JSON(400, gin.H{"error": "Organized notebooks directory not found. Please run organizeNotebooksByCategory first."})
		return
	}
	
	files, err := ioutil.ReadDir(organizedPath)
	if err != nil {
		c.JSON(500, gin.H{"error": fmt.Sprintf("Failed to read organized directory: %v", err)})
		return
	}
	
	var notebooks []string
	for _, file := range files {
		if file.IsDir() {
			notebooks = append(notebooks, file.Name())
			generateAIAnalysisData(organizedPath, file.Name())
		}
	}
	
	c.JSON(200, gin.H{
		"success":   true,
		"notebooks": notebooks,
		"message":   "AI analysis preparation completed",
	})
}

// generateAIAnalysisData 生成AI分析数据
func generateAIAnalysisData(organizedPath, notebookName string) {
	notebookPath := filepath.Join(organizedPath, notebookName)
	analysisPath := filepath.Join(notebookPath, "analysis")
	
	// 确保analysis目录存在
	os.MkdirAll(analysisPath, 0755)
	
	// 收集所有文本内容
	textContent := collectAllTextContent(notebookPath)
	
	// 生成分析索引
	analysisIndex := NotebookAnalysisData{
		Notebook:       notebookName,
		LastUpdated:    getISOString(),
		TotalDocuments: len(textContent),
		ContentTypes:   analyzeContentTypes(textContent),
	}
	
	for _, content := range textContent {
		preview := content.Text
		if len(preview) > 200 {
			preview = content.Text[:200] + "..."
		}
		
		analysisIndex.SearchableContent = append(analysisIndex.SearchableContent, NotebookContentItem{
			File:     content.File,
			Type:     content.Type,
			Preview:  preview,
			FullText: content.Text,
		})
	}
	
	// 保存分析索引
	indexJSON, _ := json.MarshalIndent(analysisIndex, "", "  ")
	indexPath := filepath.Join(analysisPath, "index.json")
	ioutil.WriteFile(indexPath, indexJSON, 0644)
}

// ContentItem 内容项
type ContentItem struct {
	File string `json:"file"`
	Path string `json:"path"`
	Type string `json:"type"`
	Text string `json:"text"`
}

// collectAllTextContent 收集所有文本内容
func collectAllTextContent(notebookPath string) []ContentItem {
	var contents []ContentItem
	
	// 收集rich-notes内容
	richNotesPath := filepath.Join(notebookPath, "rich-notes")
	if files, err := ioutil.ReadDir(richNotesPath); err == nil {
		for _, file := range files {
			if !file.IsDir() && strings.HasSuffix(file.Name(), ".html") {
				filePath := filepath.Join(richNotesPath, file.Name())
				if data, err := ioutil.ReadFile(filePath); err == nil {
					textContent := htmlToText(string(data))
					contents = append(contents, ContentItem{
						File: file.Name(),
						Path: filePath,
						Type: "rich-note",
						Text: textContent,
					})
				}
			}
		}
	}
	
	// 收集文档内容（简化处理）
	documentsPath := filepath.Join(notebookPath, "documents")
	if files, err := ioutil.ReadDir(documentsPath); err == nil {
		for _, file := range files {
			if !file.IsDir() {
				ext := strings.ToLower(filepath.Ext(file.Name()))
				if ext == ".txt" || ext == ".md" {
					filePath := filepath.Join(documentsPath, file.Name())
					if data, err := ioutil.ReadFile(filePath); err == nil {
						textContent := string(data)
						contents = append(contents, ContentItem{
							File: file.Name(),
							Path: filePath,
							Type: "document",
							Text: textContent,
						})
					}
				}
			}
		}
	}
	
	return contents
}

// htmlToText HTML转文本（简化版本）
func htmlToText(html string) string {
	// 简化的HTML标签移除
	text := html
	for {
		start := strings.Index(text, "<")
		if start == -1 {
			break
		}
		end := strings.Index(text[start:], ">")
		if end == -1 {
			break
		}
		text = text[:start] + text[start+end+1:]
	}
	
	// 清理多余空格
	text = strings.TrimSpace(strings.Join(strings.Fields(text), " "))
	
	return text
}

// analyzeContentTypes 分析内容类型分布
func analyzeContentTypes(contents []ContentItem) map[string]int {
	types := make(map[string]int)
	for _, content := range contents {
		types[content.Type]++
	}
	return types
}

// getISOString 获取ISO格式时间字符串
func getISOString() string {
	return fmt.Sprintf("%d-%02d-%02dT%02d:%02d:%02dZ",
		2025, 11, 28, 15, 47, 0) // 简化实现
}

// getOptimizedNotebooks 获取优化后的笔记本列表
func getOptimizedNotebooks(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request"})
		return
	}
	
	if req.Username == "" {
		req.Username = "jason"
	}
	
	workspaceDir := util.WorkspaceDir
	// 使用用户数据根目录
	userDataRoot := os.Getenv("SIYUAN_USER_DATA_ROOT")
	if userDataRoot == "" {
		userDataRoot = filepath.Join(workspaceDir, "data")
	}
	organizedPath := filepath.Join(userDataRoot, req.Username, "organized")
	
	var notebooks []map[string]interface{}
	
	if files, err := ioutil.ReadDir(organizedPath); err == nil {
		for _, file := range files {
			if file.IsDir() {
				notebookPath := filepath.Join(organizedPath, file.Name())
				metadataPath := filepath.Join(notebookPath, "metadata.json")
				
				var metadata NotebookMetadata
				if data, err := ioutil.ReadFile(metadataPath); err == nil {
					json.Unmarshal(data, &metadata)
				} else {
					// 如果没有元数据文件，创建基础信息
					metadata = NotebookMetadata{
						Name: file.Name(),
						Type: "notebook",
					}
					metadata.Stats.DocumentCount = countFiles(filepath.Join(notebookPath, "documents"))
					metadata.Stats.RichNoteCount = countFiles(filepath.Join(notebookPath, "rich-notes"))
					metadata.Stats.VectorCount = countFiles(filepath.Join(notebookPath, "vectors"))
				}
				
				notebooks = append(notebooks, map[string]interface{}{
					"name":      metadata.Name,
					"type":      metadata.Type,
					"stats":     metadata.Stats,
					"aiReady":   metadata.AIReady,
					"path":      notebookPath,
				})
			}
		}
	}
	
	c.JSON(200, gin.H{
		"success":   true,
		"notebooks": notebooks,
	})
}

// searchNotebookContent 搜索笔记本内容
func searchNotebookContent(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Notebook string `json:"notebook"`
		Query    string `json:"query"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request"})
		return
	}
	
	if req.Username == "" {
		req.Username = "jason"
	}
	
	workspaceDir := util.WorkspaceDir
	// 使用用户数据根目录
	userDataRoot := os.Getenv("SIYUAN_USER_DATA_ROOT")
	if userDataRoot == "" {
		userDataRoot = filepath.Join(workspaceDir, "data")
	}
	indexPath := filepath.Join(userDataRoot, req.Username, "organized", req.Notebook, "analysis", "index.json")
	
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		c.JSON(404, gin.H{"error": "Notebook analysis index not found. Please run prepareForAIAnalysis first."})
		return
	}
	
	data, err := ioutil.ReadFile(indexPath)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to read analysis index"})
		return
	}
	
	var analysisData NotebookAnalysisData
	if err := json.Unmarshal(data, &analysisData); err != nil {
		c.JSON(500, gin.H{"error": "Failed to parse analysis index"})
		return
	}
	
	// 搜索内容
	var results []NotebookContentItem
	query := strings.ToLower(req.Query)
	
	for _, content := range analysisData.SearchableContent {
		if strings.Contains(strings.ToLower(content.FullText), query) || 
		   strings.Contains(strings.ToLower(content.File), query) {
			results = append(results, content)
		}
	}
	
	c.JSON(200, gin.H{
		"success": true,
		"results": results,
		"total":   len(results),
	})
}