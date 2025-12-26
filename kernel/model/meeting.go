package model

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/siyuan-note/logging"
)

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

	// 1. 调用 ASR 服务 (假设使用本地 FunASR REST API)
	transcription, err := s.callASR(audioData)
	if err != nil {
		logging.LogErrorf("ASR failed: %v", err)
		return nil, err
	}

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

	// 2. 发送二进制音频数据
	if err := conn.WriteMessage(websocket.BinaryMessage, audioData); err != nil {
		return "", fmt.Errorf("failed to send audio data: %v", err)
	}

	// 3. 发送结束信号 (is_speaking 为 false)
	endConfig := map[string]interface{}{
		"is_speaking": false,
	}
	if err := conn.WriteJSON(endConfig); err != nil {
		return "", fmt.Errorf("failed to send end signal: %v", err)
	}

	// 4. 循环读取识别结果，直到 is_final 为 true
	finalText := ""
	timeout := time.After(30 * time.Second)

	for {
		select {
		case <-timeout:
			return finalText, fmt.Errorf("ASR timed out waiting for final result")
		default:
			_, message, err := conn.ReadMessage()
			if err != nil {
				return finalText, fmt.Errorf("failed to read ASR result: %v", err)
			}

			var result struct {
				Text    string `json:"text"`
				IsFinal bool   `json:"is_final"`
			}
			if err := json.Unmarshal(message, &result); err != nil {
				logging.LogWarnf("Failed to parse partial ASR result: %v. Message: %s", err, string(message))
				continue
			}

			// 累积或更新文本内容
			if result.Text != "" {
				finalText = result.Text
			}

			// 如果收到最终标识，跳出循环
			if result.IsFinal {
				logging.LogDebugf("Received final ASR result: %s", finalText)
				return finalText, nil
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

	prompt := fmt.Sprintf("你是一个专业的会议速记员和摘要助手。请对以下会议转录文本进行深度整理：\n1. 修正明显的语音识别错误。\n2. 总结核心要点和决策事项。\n3. 使用明晰的中文列表呈现。\n\n待处理文本：\n%s", text)

	payload := map[string]interface{}{
		"model": modelName,
		"messages": []map[string]string{
			{"role": "system", "content": "你是思源笔记的AI助手，专门帮助用户整理知识、建立概念关联和生成智能摘要。"},
			{"role": "user", "content": prompt},
		},
		"stream":      false,
		"temperature": 0.7,
		"max_tokens":  1000,
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

	client := &http.Client{Timeout: 120 * time.Second}
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
		return result.Choices[0].Message.Content, nil
	}
	return "", fmt.Errorf("no summary generated")
}
