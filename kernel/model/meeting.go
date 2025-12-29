package model

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/siyuan-note/logging"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

func findIncrementalContent(oldText, newText string) string {
	if oldText == "" || newText == "" {
		return newText
	}
	minLen := len(oldText)
	if len(newText) < minLen {
		minLen = len(newText)
	}
	foundPrefix := false
	prefixLen := 0
	for i := 0; i < minLen; i++ {
		if oldText[i] == newText[i] {
			foundPrefix = true
			prefixLen = i + 1
		} else {
			break
		}
	}
	if foundPrefix && prefixLen < len(newText) {
		return newText[prefixLen:]
	}
	return ""
}

func isValidUTF8(s string) bool {
	for i := 0; i < len(s); i++ {
		b := s[i]
		if b >= 0x80 {
			if b >= 0xFC && b <= 0xFD {
				for i := 1; i < 6; i++ {
					if i+i >= len(s) {
						return false
					}
					if s[i+i] < 0x80 || s[i+i] > 0xBF {
						return false
					}
				}
				i += 5
			} else if b >= 0xF8 && b <= 0xFB {
				for i := 1; i < 5; i++ {
					if i+i >= len(s) {
						return false
					}
					if s[i+i] < 0x80 || s[i+i] > 0xBF {
						return false
					}
				}
				i += 4
			} else if b >= 0xF0 && b <= 0xF7 {
				for i := 1; i < 4; i++ {
					if i+i >= len(s) {
						return false
					}
					if s[i+i] < 0x80 || s[i+i] > 0xBF {
						return false
					}
				}
				i += 3
			} else if b >= 0xE0 && b <= 0xEF {
				for i := 1; i < 3; i++ {
					if i+i >= len(s) {
						return false
					}
					if s[i+i] < 0x80 || s[i+i] > 0xBF {
						return false
					}
				}
				i += 2
			} else if b >= 0xC0 && b <= 0xDF {
				if i+1 >= len(s) {
					return false
				}
				if s[i+1] < 0x80 || s[i+1] > 0xBF {
					return false
				}
				i += 1
			} else {
				return false
			}
		}
	}
	return true
}

func convertToUTF8(text string) string {
	if isValidUTF8(text) {
		return text
	}

	encodings := []encoding.Encoding{
		simplifiedchinese.GBK,
		simplifiedchinese.GB18030,
	}

	for _, enc := range encodings {
		reader := transform.NewReader(strings.NewReader(text), enc.NewDecoder())
		result, err := io.ReadAll(reader)
		if err == nil {
			if isValidUTF8(string(result)) {
				logging.LogDebugf("Converted text to UTF-8, length: %d", len(result))
				return string(result)
			}
		}
	}

	logging.LogWarnf("Failed to convert text to UTF-8, keeping original (length: %d)", len(text))
	return text
}

// MeetingService 会议纪要服务
type MeetingService struct{}

var Meeting = &MeetingService{}

// TranscribeAudioResponse 转录响应
type TranscribeAudioResponse struct {
	Transcription string `json:"transcription"`
	Summary       string `json:"summary"`
}

// TranscribeAudio 转录音频并生成摘要
func (s *MeetingService) TranscribeAudio(audioData []byte) (*TranscribeAudioResponse, error) {
	if len(audioData) == 0 {
		return nil, fmt.Errorf("audio data is empty")
	}

	logging.LogDebugf("TranscribeAudio: received audio data, size: %d bytes", len(audioData))

	// 1. 调用 ASR 服务 (假设使用本地 FunASR REST API)
	transcription, err := s.callASR(audioData)
	if err != nil {
		logging.LogErrorf("ASR failed: %v", err)
		return nil, err
	}

	// 2. 确保转录结果是 UTF-8 编码
	transcription = convertToUTF8(transcription)

	logging.LogDebugf("ASR transcription result: '%s'", transcription)

	// 2. 调用 LLM 生成摘要
	summary := ""
	if transcription != "" {
		summary, err = s.GenerateSummary(transcription)
		if err != nil {
			logging.LogWarnf("Summary generation failed: %v", err)
		}
	}

	return &TranscribeAudioResponse{
		Transcription: transcription,
		Summary:       summary,
	}, nil
}

// callASR 调用 ASR 服务 (WebSocket 模式)
func (s *MeetingService) callASR(audioData []byte) (string, error) {
	// 补全末尾斜杠，有些服务器对路径很敏感
	asrURL := "ws://jason.cheman.top:10096/"

	logging.LogDebugf("ASR: Connecting to %s, audio data size: %d bytes", asrURL, len(audioData))

	dialer := websocket.DefaultDialer
	dialer.HandshakeTimeout = 15 * time.Second

	conn, resp, err := dialer.Dial(asrURL, nil)
	if err != nil {
		status := "unknown"
		if resp != nil {
			status = resp.Status
			body, _ := io.ReadAll(resp.Body)
			logging.LogErrorf("ASR Handshake failed. Status: %s, Body: %s", status, string(body))
		}
		return "", fmt.Errorf("ASR WebSocket connection failed: %v (Status: %s)", err, status)
	}
	defer conn.Close()

	logging.LogDebugf("ASR: WebSocket connected successfully")

	// 1. 发送开始配置 (根据文档使用 2pass 模式)
	startConfig := map[string]interface{}{
		"mode":           "2pass",
		"chunk_size":     []int{5, 10, 5},
		"chunk_interval": 10,
		"wav_name":       "meeting",
		"is_speaking":    true,
	}
	if err := conn.WriteJSON(startConfig); err != nil {
		return "", fmt.Errorf("failed to send start config: %v", err)
	}

	logging.LogDebugf("ASR: Sending audio data (%d bytes)...", len(audioData))

	// 2. 发送二进制音频数据
	if err := conn.WriteMessage(websocket.BinaryMessage, audioData); err != nil {
		return "", fmt.Errorf("failed to send audio data: %v", err)
	}

	logging.LogDebugf("ASR: Audio data sent, sending end signal...")

	// 3. 发送结束信号 (is_speaking 为 false)
	endConfig := map[string]interface{}{
		"is_speaking": false,
	}
	if err := conn.WriteJSON(endConfig); err != nil {
		return "", fmt.Errorf("failed to send end signal: %v", err)
	}

	// 4. 循环读取识别结果，直到 is_final 为 true
	// FunASR 2pass 模式：前期返回增量片段，后期返回累积完整文本
	accumulatedText := ""
	timeout := time.After(60 * time.Second) // 延长超时到60秒
	messageCount := 0

	logging.LogDebugf("ASR: Waiting for recognition results...")

	for {
		select {
		case <-timeout:
			logging.LogWarnf("ASR: Timeout waiting for result, partial text: '%s'", accumulatedText)
			return accumulatedText, fmt.Errorf("ASR timed out waiting for final result")
		default:
			_, message, err := conn.ReadMessage()
			if err != nil {
				logging.LogErrorf("ASR: Failed to read message: %v", err)
				return accumulatedText, fmt.Errorf("failed to read ASR result: %v", err)
			}

			messageCount++

			var result struct {
				Text    string `json:"text"`
				IsFinal bool   `json:"is_final"`
			}
			if err := json.Unmarshal(message, &result); err != nil {
				logging.LogWarnf("Failed to parse partial ASR result: %v. Message: %s", err, string(message))
				continue
			}

			logging.LogDebugf("ASR: Received message #%d, is_final: %v, text: '%s'", messageCount, result.IsFinal, result.Text)

			// 处理识别结果
			if result.Text != "" {
				// 判断是否为累积完整文本（以已有文本开头）
				if strings.HasPrefix(result.Text, accumulatedText) {
					// 是累积文本，提取真正的新增内容
					newContent := result.Text[len(accumulatedText):]
					if newContent != "" {
						accumulatedText = result.Text
						logging.LogDebugf("ASR: Appended new content: '%s' (total: '%s')", newContent, accumulatedText)
					}
				} else {
					// 文本不匹配，尝试找出公共前缀
					logging.LogDebugf("ASR: Text mismatch. Current: '%s', New: '%s'", accumulatedText, result.Text)
					newContent := findIncrementalContent(accumulatedText, result.Text)
					if newContent != "" {
						accumulatedText += newContent
						logging.LogDebugf("ASR: Incremental append: '%s' (total: '%s')", newContent, accumulatedText)
					} else if len(result.Text) > len(accumulatedText) {
						// 新文本更长且没有公共前缀，直接使用
						accumulatedText = result.Text
						logging.LogDebugf("ASR: Replace with longer text: '%s'", accumulatedText)
					} else {
						logging.LogDebugf("ASR: Ignore shorter/invalid text: '%s'", result.Text)
					}
				}
			} else {
				logging.LogDebugf("ASR: Empty text, skip (is_final: %v)", result.IsFinal)
			}

			// 如果收到最终标识，跳出循环
			if result.IsFinal {
				// 如果最终消息的 text 为空，accumulatedText 已经包含了所有内容
				logging.LogDebugf("ASR: Received final result: '%s'", accumulatedText)
				return accumulatedText, nil
			}
		}
	}
}

// GenerateSummary 生成摘要
func (s *MeetingService) GenerateSummary(text string) (string, error) {
	// 使用用户提供的本地 VLLM 增强模型配置
	llmURL := "http://jason.cheman.top:8001/v1/chat/completions"
	apiKey := "vllm-token"
	modelName := "tclf90/Qwen3-32B-GPTQ-Int4"

	prompt := fmt.Sprintf(`你是专业的会议纪要撰写专家，请将以下会议内容整理成正式、专业的商务会议纪要格式。

### 格式要求：
> **会议主题**：[一句话概括会议核心议题]
> **关键讨论**：[主要讨论内容的专业概述，2-3句话]
> **重要决议**：[明确的决定事项和行动要点]

### 待整理内容：
%s

### 输出要求：
- 直接输出三行会议纪要
- 不包含任何思考过程或开场白`, text)

	payload := map[string]interface{}{
		"model": modelName,
		"messages": []map[string]string{
			{"role": "system", "content": "You are a strict meeting minutes generator. OUTPUT ONLY the summary in the exact format requested. NO explanation. NO thinking process. NO preamble. JUST the three lines starting with '> **'."},
			{"role": "user", "content": prompt},
		},
		"stream":      false,
		"temperature": 0.3,
		"max_tokens":  200,
	}

	jsonPayload, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", llmURL, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		logging.LogErrorf("Failed to call LLM service at %s: %v", llmURL, err)
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		logging.LogErrorf("LLM service returned error status %d: %s", resp.StatusCode, string(bodyBytes))
		return "", fmt.Errorf("LLM service returned status %d", resp.StatusCode)
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if len(result.Choices) > 0 {
		summary := result.Choices[0].Message.Content
		summary = convertToUTF8(summary)
		return summary, nil
	}
	return "", fmt.Errorf("no summary generated")
}
