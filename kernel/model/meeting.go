package model

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/siyuan-note/logging"
)

// ASR 配置
var asrEndpoint = "ws://jason.cheman.top:10096/"

// LLM 配置
var llmEndpoint = "http://jason.cheman.top:8001/v1"
var llmAPIKey = "vllm-token"
var llmModelName = "tclf90/Qwen3-32B-GPTQ-Int4"

func init() {
	// 加载 ASR 配置
	asrConfigPath := "/root/code/unified-settings-service/config/asr-config.json"
	asrData, err := os.ReadFile(asrConfigPath)
	if err == nil {
		var asrConfig struct {
			Endpoint string `json:"endpoint"`
		}
		if json.Unmarshal(asrData, &asrConfig) == nil && asrConfig.Endpoint != "" {
			asrEndpoint = asrConfig.Endpoint
			if !strings.HasSuffix(asrEndpoint, "/") {
				asrEndpoint += "/"
			}
			logging.LogInfof("ASR 配置已加载: %s", asrEndpoint)
		}
	} else {
		logging.LogWarnf("ASR 配置文件不存在，使用默认配置: %s", asrConfigPath)
	}

	// 加载 LLM 默认模型配置
	llmConfigPath := "/root/code/unified-settings-service/config/default-models.json"
	llmData, err := os.ReadFile(llmConfigPath)
	if err == nil {
		var models map[string]struct {
			BaseURL   string `json:"base_url"`
			APIKey    string `json:"api_key"`
			ModelName string `json:"model_name"`
		}
		if json.Unmarshal(llmData, &models) == nil {
			// 优先使用 siyuan 模型
			if model, ok := models["builtin_free_siyuan"]; ok {
				llmEndpoint = model.BaseURL
				llmAPIKey = model.APIKey
				llmModelName = model.ModelName
				if !strings.HasSuffix(llmEndpoint, "/") {
					llmEndpoint += "/"
				}
				logging.LogInfof("LLM 配置已加载 (siyuan): %s %s", llmEndpoint, llmModelName)
			}
		}
	} else {
		logging.LogWarnf("LLM 配置文件不存在，使用默认配置: %s", llmConfigPath)
	}
}

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

func containsValidUTF8Multibyte(s string) bool {
	for i := 0; i < len(s); i++ {
		b := s[i]
		if b >= 0xE0 && b <= 0xEF {
			if i+2 < len(s) && s[i+1] >= 0x80 && s[i+1] <= 0xBF && s[i+2] >= 0x80 && s[i+2] <= 0xBF {
				return true
			}
		}
		if b >= 0xF0 && b <= 0xF7 {
			if i+3 < len(s) && s[i+1] >= 0x80 && s[i+1] <= 0xBF && s[i+2] >= 0x80 && s[i+2] <= 0xBF && s[i+3] >= 0x80 && s[i+3] <= 0xBF {
				return true
			}
		}
	}
	return false
}

var gbkToUnicode = [0x10000]rune{
	0x00: 0x0000, 0x01: 0x0001, 0x02: 0x0002, 0x03: 0x0003, 0x04: 0x0004, 0x05: 0x0005, 0x06: 0x0006, 0x07: 0x0007,
	0x08: 0x0008, 0x09: 0x0009, 0x0A: 0x000A, 0x0B: 0x000B, 0x0C: 0x000C, 0x0D: 0x000D, 0x0E: 0x000E, 0x0F: 0x000F,
	0x10: 0x0010, 0x11: 0x0011, 0x12: 0x0012, 0x13: 0x0013, 0x14: 0x0014, 0x15: 0x0015, 0x16: 0x0016, 0x17: 0x0017,
	0x18: 0x0018, 0x19: 0x0019, 0x1A: 0x001A, 0x1B: 0x001B, 0x1C: 0x001C, 0x1D: 0x001D, 0x1E: 0x001E, 0x1F: 0x001F,
	0x20: 0x0020, 0x21: 0x0021, 0x22: 0x0022, 0x23: 0x0023, 0x24: 0x0024, 0x25: 0x0025, 0x26: 0x0026, 0x27: 0x0027,
	0x28: 0x0028, 0x29: 0x0029, 0x2A: 0x002A, 0x2B: 0x002B, 0x2C: 0x002C, 0x2D: 0x002D, 0x2E: 0x002E, 0x2F: 0x002F,
	0x30: 0x0030, 0x31: 0x0031, 0x32: 0x0032, 0x33: 0x0033, 0x34: 0x0034, 0x35: 0x0035, 0x36: 0x0036, 0x37: 0x0037,
	0x38: 0x0038, 0x39: 0x0039, 0x3A: 0x003A, 0x3B: 0x003B, 0x3C: 0x003C, 0x3D: 0x003D, 0x3E: 0x003E, 0x3F: 0x003F,
	0x40: 0x0040, 0x41: 0x0041, 0x42: 0x0042, 0x43: 0x0043, 0x44: 0x0044, 0x45: 0x0045, 0x46: 0x0046, 0x47: 0x0047,
	0x48: 0x0048, 0x49: 0x0049, 0x4A: 0x004A, 0x4B: 0x004B, 0x4C: 0x004C, 0x4D: 0x004D, 0x4E: 0x004E, 0x4F: 0x004F,
	0x50: 0x0050, 0x51: 0x0051, 0x52: 0x0052, 0x53: 0x0053, 0x54: 0x0054, 0x55: 0x0055, 0x56: 0x0056, 0x57: 0x0057,
	0x58: 0x0058, 0x59: 0x0059, 0x5A: 0x005A, 0x5B: 0x005B, 0x5C: 0x005C, 0x5D: 0x005D, 0x5E: 0x005E, 0x5F: 0x005F,
	0x60: 0x0060, 0x61: 0x0061, 0x62: 0x0062, 0x63: 0x0063, 0x64: 0x0064, 0x65: 0x0065, 0x66: 0x0066, 0x67: 0x0067,
	0x68: 0x0068, 0x69: 0x0069, 0x6A: 0x006A, 0x6B: 0x006B, 0x6C: 0x006C, 0x6D: 0x006D, 0x6E: 0x006E, 0x6F: 0x006F,
	0x70: 0x0070, 0x71: 0x0071, 0x72: 0x0072, 0x73: 0x0073, 0x74: 0x0074, 0x75: 0x0075, 0x76: 0x0076, 0x77: 0x0077,
	0x78: 0x0078, 0x79: 0x0079, 0x7A: 0x007A, 0x7B: 0x007B, 0x7C: 0x007C, 0x7D: 0x007D, 0x7E: 0x007E, 0x7F: 0x007F,
}

func initGBKTable() {
	commonGBK := []struct {
		gbk     byte
		unicode rune
	}{
		{0xA1, 0x3000}, {0xA2, 0x3001}, {0xA3, 0x3002}, {0xA4, 0x00B7}, {0xA5, 0x02C7}, {0xA6, 0x00B8}, {0xA7, 0x00A8}, {0xA8, 0x02CB},
		{0xA9, 0x00B0}, {0xAA, 0x00A4}, {0xAB, 0x00B6}, {0xAC, 0x00A7}, {0xAD, 0x00F7}, {0xAE, 0x00B1}, {0xAF, 0x2015},
		{0xB0, 0x4E00}, {0xB1, 0x4E01}, {0xB2, 0x4E02}, {0xB3, 0x4E03}, {0xB4, 0x4E04}, {0xB5, 0x4E05}, {0xB6, 0x4E06}, {0xB7, 0x4E07},
		{0xB8, 0x4E08}, {0xB9, 0x4E09}, {0xBA, 0x4E0A}, {0xBB, 0x4E0B}, {0xBC, 0x4E0C}, {0xBD, 0x4E0D}, {0xBE, 0x4E0E}, {0xBF, 0x4E0F},
		{0xC0, 0x4E10}, {0xC1, 0x4E11}, {0xC2, 0x4E12}, {0xC3, 0x4E13}, {0xC4, 0x4E14}, {0xC5, 0x4E15}, {0xC6, 0x4E16}, {0xC7, 0x4E17},
		{0xC8, 0x4E18}, {0xC9, 0x4E19}, {0xCA, 0x4E1A}, {0xCB, 0x4E1B}, {0xCC, 0x4E1C}, {0xCD, 0x4E1D}, {0xCE, 0x4E1E}, {0xCF, 0x4E1F},
		{0xD0, 0x4E20}, {0xD1, 0x4E21}, {0xD2, 0x4E22}, {0xD3, 0x4E23}, {0xD4, 0x4E24}, {0xD5, 0x4E25}, {0xD6, 0x4E26}, {0xD7, 0x4E27},
		{0xD8, 0x4E28}, {0xD9, 0x4E29}, {0xDA, 0x4E2A}, {0xDB, 0x4E2B}, {0xDC, 0x4E2C}, {0xDD, 0x4E2D}, {0xDE, 0x4E2E}, {0xDF, 0x4E2F},
		{0xE0, 0x4E30}, {0xE1, 0x4E31}, {0xE2, 0x4E32}, {0xE3, 0x4E34}, {0xE4, 0x4E35}, {0xE5, 0x4E36}, {0xE6, 0x4E37}, {0xE7, 0x4E38},
		{0xE8, 0x4E39}, {0xE9, 0x4E3A}, {0xEA, 0x4E3B}, {0xEB, 0x4E3C}, {0xEC, 0x4E3D}, {0xED, 0x4E3E}, {0xEE, 0x4E3F}, {0xEF, 0x4E40},
		{0xF0, 0x4E41}, {0xF1, 0x4E42}, {0xF2, 0x4E43}, {0xF3, 0x4E44}, {0xF4, 0x4E45}, {0xF5, 0x4E46}, {0xF6, 0x4E47}, {0xF7, 0x4E48},
		{0xF8, 0x4E49}, {0xF9, 0x4E4A}, {0xFA, 0x4E4B}, {0xFB, 0x4E4C}, {0xFC, 0x4E4D}, {0xFD, 0x4E4E}, {0xFE, 0x4E4F},
	}
	for _, v := range commonGBK {
		gbkToUnicode[v.gbk] = v.unicode
	}
}

func gbkDecode(byte1, byte2 byte) rune {
	if byte1 >= 0x81 && byte1 <= 0xFE && byte2 >= 0x40 && byte2 <= 0xFE {
		offset := (int(byte1-0x81) * 190) + int(byte2-0x40)
		if offset >= 0 && offset < 0x8000 {
			row := byte1
			cell := byte2
			if row >= 0xA1 && row <= 0xA9 {
				row -= 0xA1
				if cell >= 0xA1 && cell <= 0xFE {
					return gbkToUnicode[int(row)*94+int(cell-0xA1)+0x100]
				}
			} else if row >= 0xB0 && row <= 0xF7 {
				row -= 0xA6
				if cell >= 0xA1 && cell <= 0xFE {
					return gbkToUnicode[94+int(row)*94+int(cell-0xA1)+0x100]
				}
			}
		}
	}
	return rune(byte2)
}

func convertGBKToUTF8(text string) string {
	if text == "" {
		return text
	}

	hasHighBytes := false
	for i := 0; i < len(text); i++ {
		if text[i] >= 0x80 {
			hasHighBytes = true
			break
		}
	}

	if !hasHighBytes {
		return text
	}

	if containsValidUTF8Multibyte(text) {
		return text
	}

	var result strings.Builder
	result.Grow(len(text) * 2)

	i := 0
	for i < len(text) {
		b := text[i]
		if b < 0x80 {
			result.WriteByte(b)
			i++
		} else if b >= 0x81 && b <= 0xFE && i+1 < len(text) {
			gbkChar := gbkDecode(b, text[i+1])
			if gbkChar > 0 {
				result.WriteRune(gbkChar)
			} else {
				result.WriteByte(b)
				result.WriteByte(text[i+1])
			}
			i += 2
		} else if b >= 0xC0 && b <= 0xDF && i+1 < len(text) {
			result.WriteByte(b)
			result.WriteByte(text[i+1])
			i += 2
		} else {
			result.WriteByte(b)
			i++
		}
	}

	return result.String()
}

func convertToUTF8(text string) string {
	if text == "" {
		return text
	}

	if isValidUTF8(text) {
		return text
	}

	converted := convertGBKToUTF8(text)
	if converted != text {
		logging.LogWarnf("Converted text from GBK to UTF-8 (original length: %d, converted length: %d)", len(text), len(converted))
		return converted
	}

	logging.LogWarnf("Text has invalid encoding and cannot be converted (length: %d)", len(text))
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

	// 移除 convertToUTF8 调用，JSON Unmarshal 得到的已经是 UTF-8 字符串
	// 强制转换会导致偶发的乱码问题 (False Positive GBK detection)
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
	asrURL := asrEndpoint

	// 1. 去除 WAV 头部 (44 bytes)，只发送纯 PCM 数据
	// 如果不去除，头部信息会被识别为刺耳噪音，导致开头乱码或识别错误
	pcmData := audioData
	if len(audioData) > 44 && string(audioData[:4]) == "RIFF" {
		logging.LogDebugf("ASR: Detected WAV header, stripping first 44 bytes.")
		pcmData = audioData[44:]
	}

	logging.LogDebugf("ASR: Connecting to %s, PCM data size: %d bytes", asrURL, len(pcmData))

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

	// 2. 发送开始配置 (根据文档使用 2pass 模式)
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

	// 准备收集结果
	var fullTranscriptBuilder bytes.Buffer
	// 暂存最新的流式中间结果，防止最后一句因未 finalize 而丢失
	var latestPartialText string

	// 用于同步发送端状态的通道
	sendErrChan := make(chan error, 1)

	// 3. 启动协程并发发送音频数据
	go func() {
		defer close(sendErrChan)

		logging.LogDebugf("ASR: Start sending PCM data (%d bytes)...", len(pcmData))

		// 16000Hz, 16bit (2bytes), Mono => 32000 bytes/sec
		// 分块大小：0.2秒的数据 => 6400 bytes
		const chunkSize = 6400 // ~200ms
		// 为了保证 FunASR 能够有足够的时间处理 VAD，我们使用 1:1 实时速度发送
		// 或者稍微快一点点 (0.9x)。这里设为 200ms 对应 200ms 数据。
		const sleepInterval = 200 * time.Millisecond

		for i := 0; i < len(pcmData); i += chunkSize {
			end := i + chunkSize
			if end > len(pcmData) {
				end = len(pcmData)
			}

			if err := conn.WriteMessage(websocket.BinaryMessage, pcmData[i:end]); err != nil {
				sendErrChan <- fmt.Errorf("failed to send audio chunk: %v", err)
				return
			}

			time.Sleep(sleepInterval)
		}

		logging.LogDebugf("ASR: Audio data sent, sending end signal...")

		// 4. 发送结束信号 (is_speaking 为 false)
		endConfig := map[string]interface{}{
			"is_speaking": false,
		}
		if err := conn.WriteJSON(endConfig); err != nil {
			sendErrChan <- fmt.Errorf("failed to send end signal: %v", err)
			return
		}
		logging.LogDebugf("ASR: End signal sent.")
	}()

	// 5. 主协程循环读取识别结果
	// 读取循环会在 socket 关闭或出错时退出

	messageCount := 0
	timeout := time.After(180 * time.Second) // 延长超时到180秒
	logging.LogDebugf("ASR: Waiting for recognition results...")

	for {
		// 检查发送端是否有错误
		select {
		case err := <-sendErrChan:
			if err != nil {
				// 发送失败直接返回目前已识别的内容+错误
				if latestPartialText != "" {
					fullTranscriptBuilder.WriteString(latestPartialText)
				}
				return fullTranscriptBuilder.String(), err
			}
		case <-timeout:
			logging.LogWarnf("ASR: Timeout waiting for result.")
			// 超时返回当前结果
			if latestPartialText != "" {
				fullTranscriptBuilder.WriteString(latestPartialText)
			}
			return fullTranscriptBuilder.String(), nil
		default:
		}

		// 设置读取超时，防止服务端假死
		// conn.SetReadDeadline(time.Now().Add(10 * time.Second)) // Removed as per instruction
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure) || err == io.EOF {
				logging.LogDebugf("ASR connection closed normally.")
			} else {
				// 忽略 ReadDeadline 产生的超时错误，这在流式传输间隙是正常的吗？
				// 不，对于 continuous streaming，read 应该 block 直到有数据。
				// 但如果对方一直不发数据（比如 VAD silence），可能会超时。
				// 这里我们需要区分是网络断了还是只是静音。
				// 简单起见，如果超时且发送端还没结束，我们继续等待?
				// 上面使用了 SetReadDeadline，如果超时会返回 error。
				// 为了稳妥，这里如果只是读超时，我们不退出的逻辑比较复杂。
				// 暂时去掉 SetReadDeadline，依赖外层总 timeout 和 socket close。
				logging.LogDebugf("ASR connection read stop: %v", err)
			}

			// 连接关闭意味着转录结束，将最后未 finalize 的内容拼上去
			if latestPartialText != "" {
				logging.LogDebugf("ASR: Appending unfinalized tail: '%s'", latestPartialText)
				fullTranscriptBuilder.WriteString(latestPartialText)
			}
			return fullTranscriptBuilder.String(), nil
		}

		messageCount++

		var result struct {
			Text    string `json:"text"`
			IsFinal bool   `json:"is_final"`
			Mode    string `json:"mode"`
		}
		if err := json.Unmarshal(message, &result); err != nil {
			logging.LogWarnf("Failed to parse ASR result: %v. Message: %s", err, string(message))
			continue
		}

		if result.IsFinal {
			logging.LogDebugf("ASR: Sentence finalized: '%s'", result.Text)
			fullTranscriptBuilder.WriteString(result.Text)
			// 此句话已经确定，清空暂存区
			latestPartialText = ""
		} else {
			// 将中间结果暂存
			// FunASR 的 partial text 通常是当前正在识别的完整短句
			latestPartialText = result.Text
			// logging.LogDebugf("ASR Partial: %s", result.Text)
		}
	}
}

// GenerateSummary 生成摘要
func (s *MeetingService) GenerateSummary(text string) (string, error) {
	// 使用配置中的 LLM 设置
	llmURL := llmEndpoint + "/chat/completions"
	apiKey := llmAPIKey
	modelName := llmModelName

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
		// 移除 convertToUTF8 调用
		return summary, nil
	}
	return "", fmt.Errorf("no summary generated")
}
