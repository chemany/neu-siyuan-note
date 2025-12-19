// PaddleOCR API 集成
// 用于处理扫描版 PDF 和图片的 OCR 识别

package model

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/88250/gulu"
	"github.com/siyuan-note/logging"
)

// PaddleOCR 配置
const (
	DefaultPaddleOCRBaseURL = "http://127.0.0.1:8081"
	PaddleOCRTimeout        = 60 * time.Second
)

// PaddleOCRConfig OCR 服务配置
type PaddleOCRConfig struct {
	BaseURL string `json:"baseUrl"`
	Enabled bool   `json:"enabled"`
}

// PaddleOCRRequest OCR 请求结构 (适配 Umi-OCR)
type PaddleOCRRequest struct {
	Base64 string `json:"base64"`
}

// PaddleOCRURLRequest URL OCR 请求结构
type PaddleOCRURLRequest struct {
	URL string `json:"url"`
}

// PaddleOCRResult OCR 识别结果 (适配 Umi-OCR)
type PaddleOCRResult struct {
	Text       string      `json:"text"`
	Confidence float64     `json:"score"`
	Position   [][]float64 `json:"box"`
}

// PaddleOCRResponse OCR 响应结构 (适配 Umi-OCR)
type PaddleOCRResponse struct {
	Code    int               `json:"code"`
	Data    []PaddleOCRResult `json:"data"`
	Message string            `json:"msg"`
}

// OCRAssetResult OCR 资源文件结果（保存到 .ocr.json）
type OCRAssetResult struct {
	ID         string            `json:"id"`
	AssetPath  string            `json:"assetPath"`
	FileName   string            `json:"fileName"`
	FileType   string            `json:"fileType"`
	OCRResults []PaddleOCRResult `json:"ocrResults"`
	FullText   string            `json:"fullText"`
	PageCount  int               `json:"pageCount"`
	UpdatedAt  time.Time         `json:"updatedAt"`
}

// getPaddleOCRConfig 获取 PaddleOCR 配置
func getPaddleOCRConfig() *PaddleOCRConfig {
	// 从配置文件读取，如果不存在则使用默认配置
	configPath := "/root/code/NeuraLink-Notes/config/ocr-config.json"
	data, err := os.ReadFile(configPath)
	if err == nil {
		var config PaddleOCRConfig
		if err := json.Unmarshal(data, &config); err == nil && config.BaseURL != "" {
			return &config
		}
	}

	// 默认配置
	return &PaddleOCRConfig{
		BaseURL: DefaultPaddleOCRBaseURL,
		Enabled: true,
	}
}

// PaddleOCRHealthCheck 检查 PaddleOCR 服务状态
func PaddleOCRHealthCheck() (bool, string) {
	config := getPaddleOCRConfig()
	if !config.Enabled {
		return false, "OCR 服务未启用"
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(config.BaseURL + "/health")
	if err != nil {
		return false, fmt.Sprintf("OCR 服务连接失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Sprintf("OCR 服务状态异常: %d", resp.StatusCode)
	}

	return true, "OCR 服务正常"
}

// PaddleOCRFromBase64 使用 base64 图片进行 OCR
func PaddleOCRFromBase64(base64Image string) (*PaddleOCRResponse, error) {
	config := getPaddleOCRConfig()
	if !config.Enabled {
		return nil, fmt.Errorf("OCR 服务未启用")
	}

	// 构建 JSON 请求
	reqBody := PaddleOCRRequest{Base64: base64Image}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %v", err)
	}

	client := &http.Client{Timeout: PaddleOCRTimeout}
	resp, err := client.Post(
		config.BaseURL+"/api/ocr",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return nil, fmt.Errorf("OCR 请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	var ocrResp PaddleOCRResponse
	if err := json.Unmarshal(body, &ocrResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v, body: %s", err, string(body))
	}

	// Umi-OCR 状态码: 100 成功, 101 无文字
	if ocrResp.Code != 100 {
		if ocrResp.Code == 101 {
			// 返回空结果而不是错误，因为“无文字”是正常情况
			return &ocrResp, nil
		}
		return nil, fmt.Errorf("OCR 识别失败: %s (code: %d)", ocrResp.Message, ocrResp.Code)
	}

	return &ocrResp, nil
}

// PaddleOCRFromFile 从文件进行 OCR
func PaddleOCRFromFile(filePath string) (*PaddleOCRResponse, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("读取文件失败: %v", err)
	}

	base64Image := base64.StdEncoding.EncodeToString(data)
	return PaddleOCRFromBase64(base64Image)
}

// getOCRFilePath 获取 OCR 结果文件路径
func getOCRFilePath(assetPath string) string {
	return assetPath + ".ocr.json"
}

// saveOCRResult 保存 OCR 结果到文件
func saveOCRResult(result *OCRAssetResult) error {
	ocrPath := getOCRFilePath(result.AssetPath)
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化 OCR 结果失败: %v", err)
	}
	return os.WriteFile(ocrPath, data, 0644)
}

// loadOCRResult 加载 OCR 结果
func loadOCRResult(assetPath string) (*OCRAssetResult, error) {
	ocrPath := getOCRFilePath(assetPath)
	if !gulu.File.IsExist(ocrPath) {
		return nil, fmt.Errorf("OCR 结果文件不存在")
	}

	data, err := os.ReadFile(ocrPath)
	if err != nil {
		return nil, err
	}

	var result OCRAssetResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// OCRAsset 对资源文件进行 OCR 识别
// 支持图片格式：png, jpg, jpeg, bmp, gif
// 支持 PDF（会转换为图片后 OCR）
func OCRAsset(assetPath string) (*OCRAssetResult, error) {
	if !gulu.File.IsExist(assetPath) {
		return nil, fmt.Errorf("文件不存在: %s", assetPath)
	}

	ext := strings.ToLower(filepath.Ext(assetPath))
	fileName := filepath.Base(assetPath)

	var allResults []PaddleOCRResult
	var fullTextBuilder strings.Builder
	pageCount := 1

	switch ext {
	case ".png", ".jpg", ".jpeg", ".bmp", ".gif", ".webp":
		// 直接 OCR 图片
		resp, err := PaddleOCRFromFile(assetPath)
		if err != nil {
			return nil, err
		}
		allResults = resp.Data
		for _, r := range resp.Data {
			fullTextBuilder.WriteString(r.Text)
			fullTextBuilder.WriteString("\n")
		}

	case ".pdf":
		// PDF 需要先转换为图片再 OCR
		results, text, pages, err := ocrPDFFile(assetPath)
		if err != nil {
			return nil, err
		}
		allResults = results
		fullTextBuilder.WriteString(text)
		pageCount = pages

	default:
		return nil, fmt.Errorf("不支持的文件格式: %s", ext)
	}

	// 生成资源 ID
	h := gulu.Rand.String(16)

	result := &OCRAssetResult{
		ID:         h,
		AssetPath:  assetPath,
		FileName:   fileName,
		FileType:   ext,
		OCRResults: allResults,
		FullText:   strings.TrimSpace(fullTextBuilder.String()),
		PageCount:  pageCount,
		UpdatedAt:  time.Now(),
	}

	// 保存结果
	if err := saveOCRResult(result); err != nil {
		logging.LogWarnf("保存 OCR 结果失败: %v", err)
	}

	logging.LogInfof("OCR 完成: %s, 识别 %d 个文本块", fileName, len(allResults))
	return result, nil
}

// ocrPDFFile 对 PDF 文件进行 OCR
// 使用 pdftoppm 将 PDF 转换为图片，然后逐页 OCR
func ocrPDFFile(pdfPath string) ([]PaddleOCRResult, string, int, error) {
	// 创建临时目录
	tmpDir := filepath.Join(os.TempDir(), "paddle_ocr_pdf", gulu.Rand.String(8))
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return nil, "", 0, fmt.Errorf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 使用 pdftoppm 将 PDF 转换为 PNG 图片
	// pdftoppm -png -r 150 input.pdf output_prefix
	outputPrefix := filepath.Join(tmpDir, "page")
	cmd := fmt.Sprintf("pdftoppm -png -r 150 \"%s\" \"%s\"", pdfPath, outputPrefix)

	logging.LogInfof("执行 PDF 转图片: %s", cmd)

	// 执行命令
	var stdout, stderr bytes.Buffer
	execCmd := newCommand(cmd)
	execCmd.Stdout = &stdout
	execCmd.Stderr = &stderr
	if err := execCmd.Run(); err != nil {
		return nil, "", 0, fmt.Errorf("PDF 转图片失败: %v, stderr: %s", err, stderr.String())
	}

	// 查找生成的图片文件
	files, err := filepath.Glob(filepath.Join(tmpDir, "page-*.png"))
	if err != nil {
		return nil, "", 0, fmt.Errorf("查找图片文件失败: %v", err)
	}

	if len(files) == 0 {
		// 尝试其他命名格式
		files, _ = filepath.Glob(filepath.Join(tmpDir, "page*.png"))
	}

	if len(files) == 0 {
		return nil, "", 0, fmt.Errorf("PDF 转换后未生成图片文件")
	}

	logging.LogInfof("PDF 转换完成，共 %d 页", len(files))

	// 对每页图片进行 OCR
	var allResults []PaddleOCRResult
	var fullTextBuilder strings.Builder
	pageCount := len(files)

	// 按文件名排序确保页面顺序
	sortedFiles := make([]string, len(files))
	copy(sortedFiles, files)
	// 简单排序
	for i := 0; i < len(sortedFiles)-1; i++ {
		for j := i + 1; j < len(sortedFiles); j++ {
			if sortedFiles[i] > sortedFiles[j] {
				sortedFiles[i], sortedFiles[j] = sortedFiles[j], sortedFiles[i]
			}
		}
	}

	for i, imgFile := range sortedFiles {
		logging.LogInfof("OCR 第 %d/%d 页: %s", i+1, pageCount, filepath.Base(imgFile))

		resp, err := PaddleOCRFromFile(imgFile)
		if err != nil {
			logging.LogWarnf("第 %d 页 OCR 失败: %v", i+1, err)
			continue
		}

		// 添加页码标记
		fullTextBuilder.WriteString(fmt.Sprintf("\n--- 第 %d 页 ---\n", i+1))

		for _, r := range resp.Data {
			allResults = append(allResults, r)
			fullTextBuilder.WriteString(r.Text)
			fullTextBuilder.WriteString("\n")
		}
	}

	return allResults, fullTextBuilder.String(), pageCount, nil
}

// newCommand 创建命令（跨平台兼容）
func newCommand(cmd string) *exec.Cmd {
	return exec.Command("sh", "-c", cmd)
}

// NeedOCR 检查 PDF 是否需要 OCR（扫描版 PDF）
// 通过尝试提取文本来判断，如果提取的文本很少或为空，则认为需要 OCR
func NeedOCR(pdfPath string) bool {
	// 尝试使用 pdftotext 提取文本
	content, err := parsePDF(pdfPath)
	if err != nil {
		// 提取失败，可能需要 OCR
		return true
	}

	// 清理空白字符后检查内容长度
	cleanContent := strings.TrimSpace(content)
	cleanContent = strings.ReplaceAll(cleanContent, "\n", "")
	cleanContent = strings.ReplaceAll(cleanContent, " ", "")

	// 如果提取的文本少于 50 个字符，认为是扫描版 PDF
	if len(cleanContent) < 50 {
		logging.LogInfof("PDF 文本内容过少 (%d 字符)，判定为扫描版，需要 OCR", len(cleanContent))
		return true
	}

	return false
}

// ParsePDFWithOCR 解析 PDF，如果是扫描版则自动 OCR
func ParsePDFWithOCR(pdfPath string) (string, error) {
	// 首先尝试直接提取文本
	content, err := parsePDF(pdfPath)
	if err == nil {
		cleanContent := strings.TrimSpace(content)
		cleanContent = strings.ReplaceAll(cleanContent, "\n", "")
		cleanContent = strings.ReplaceAll(cleanContent, " ", "")

		// 如果提取到足够的文本，直接返回
		if len(cleanContent) >= 50 {
			return content, nil
		}
	}

	// 检查 OCR 服务是否可用
	healthy, msg := PaddleOCRHealthCheck()
	if !healthy {
		logging.LogWarnf("OCR 服务不可用: %s", msg)
		// 返回原始提取结果（可能为空）
		if content != "" {
			return content, nil
		}
		return "", fmt.Errorf("PDF 无法提取文本且 OCR 服务不可用: %s", msg)
	}

	// 执行 OCR
	logging.LogInfof("PDF 需要 OCR，开始识别: %s", pdfPath)
	result, err := OCRAsset(pdfPath)
	if err != nil {
		return "", fmt.Errorf("OCR 失败: %v", err)
	}

	return result.FullText, nil
}

// GetOCRText 获取资源文件的 OCR 文本（从缓存文件读取）
func GetOCRText(assetPath string) (string, error) {
	result, err := loadOCRResult(assetPath)
	if err != nil {
		return "", err
	}
	return result.FullText, nil
}

// HasOCRResult 检查是否已有 OCR 结果
func HasOCRResult(assetPath string) bool {
	ocrPath := getOCRFilePath(assetPath)
	return gulu.File.IsExist(ocrPath)
}
