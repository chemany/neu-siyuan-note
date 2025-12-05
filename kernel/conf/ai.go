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

package conf

import (
	"os"
	"strconv"

	"github.com/siyuan-note/siyuan/kernel/util"

	"github.com/sashabaranov/go-openai"
)

type AI struct {
	OpenAI   *OpenAI   `json:"openAI"`   // LLM/Chat models for conversation and analysis
	Embedding *Embedding `json:"embedding"` // Embedding models for vectorization
}

type OpenAI struct {
	APIKey         string  `json:"apiKey"`
	APITimeout     int     `json:"apiTimeout"`
	APIProxy       string  `json:"apiProxy"`
	APIModel       string  `json:"apiModel"`
	APIMaxTokens   int     `json:"apiMaxTokens"`
	APITemperature float64 `json:"apiTemperature"`
	APIMaxContexts int     `json:"apiMaxContexts"`
	APIBaseURL     string  `json:"apiBaseURL"`
	APIUserAgent   string  `json:"apiUserAgent"`
	APIProvider    string  `json:"apiProvider"` // OpenAI, Azure, SiliconFlow, etc.
	APIVersion     string  `json:"apiVersion"`  // Azure API version
}

type Embedding struct {
	Provider       string `json:"provider"`        // siliconflow, openai, etc.
	APIKey         string `json:"apiKey"`
	Model          string `json:"model"`           // BAAI/bge-large-zh-v1.5, text-embedding-ada-002, etc.
	APIBaseURL     string `json:"apiBaseUrl"`      // Custom endpoint URL
	EncodingFormat string `json:"encodingFormat"`  // float, base64
	Timeout        int    `json:"timeout"`         // Request timeout in seconds
	Enabled        bool   `json:"enabled"`         // Whether vectorization is enabled
}

func NewAI() *AI {
	openAI := &OpenAI{
		APITemperature: 1.0,
		APIMaxContexts: 7,
		APITimeout:     30,
		APIModel:       openai.GPT3Dot5Turbo,
		APIBaseURL:     "https://api.openai.com/v1",
		APIUserAgent:   util.UserAgent,
		APIProvider:    "OpenAI",
	}

	// 优先使用新的LLM环境变量，然后回退到旧的OPENAI变量以保证兼容性
	if apiKey := os.Getenv("OPENAI_API_KEY"); "" != apiKey {
		openAI.APIKey = apiKey
	} else if apiKey := os.Getenv("SIYUAN_OPENAI_API_KEY"); "" != apiKey {
		openAI.APIKey = apiKey
	}

	// LLM相关环境变量配置
	if provider := os.Getenv("SIYUAN_LLM_PROVIDER"); "" != provider {
		openAI.APIProvider = provider
	}

	if model := os.Getenv("SIYUAN_LLM_MODEL"); "" != model {
		openAI.APIModel = model
	} else if model := os.Getenv("SIYUAN_OPENAI_API_MODEL"); "" != model {
		openAI.APIModel = model
	}

	if temperature := os.Getenv("SIYUAN_LLM_TEMPERATURE"); "" != temperature {
		temperatureFloat, err := strconv.ParseFloat(temperature, 64)
		if err == nil {
			openAI.APITemperature = temperatureFloat
		}
	} else if temperature := os.Getenv("SIYUAN_OPENAI_API_TEMPERATURE"); "" != temperature {
		temperatureFloat, err := strconv.ParseFloat(temperature, 64)
		if err == nil {
			openAI.APITemperature = temperatureFloat
		}
	}

	if maxTokens := os.Getenv("SIYUAN_LLM_MAX_TOKENS"); "" != maxTokens {
		maxTokensInt, err := strconv.Atoi(maxTokens)
		if err == nil {
			openAI.APIMaxTokens = maxTokensInt
		}
	} else if maxTokens := os.Getenv("SIYUAN_OPENAI_API_MAX_TOKENS"); "" != maxTokens {
		maxTokensInt, err := strconv.Atoi(maxTokens)
		if err == nil {
			openAI.APIMaxTokens = maxTokensInt
		}
	}

	if timeout := os.Getenv("SIYUAN_LLM_TIMEOUT"); "" != timeout {
		timeoutInt, err := strconv.Atoi(timeout)
		if err == nil {
			openAI.APITimeout = timeoutInt
		}
	} else if timeout := os.Getenv("SIYUAN_OPENAI_API_TIMEOUT"); "" != timeout {
		timeoutInt, err := strconv.Atoi(timeout)
		if err == nil {
			openAI.APITimeout = timeoutInt
		}
	}

	if maxContexts := os.Getenv("SIYUAN_LLM_MAX_CONTEXTS"); "" != maxContexts {
		maxContextsInt, err := strconv.Atoi(maxContexts)
		if err == nil {
			openAI.APIMaxContexts = maxContextsInt
		}
	} else if maxContexts := os.Getenv("SIYUAN_OPENAI_API_MAX_CONTEXTS"); "" != maxContexts {
		maxContextsInt, err := strconv.Atoi(maxContexts)
		if err == nil {
			openAI.APIMaxContexts = maxContextsInt
		}
	}

	if baseURL := os.Getenv("SIYUAN_LLM_API_BASE_URL"); "" != baseURL {
		openAI.APIBaseURL = baseURL
	} else if baseURL := os.Getenv("SIYUAN_OPENAI_API_BASE_URL"); "" != baseURL {
		openAI.APIBaseURL = baseURL
	}

	if proxy := os.Getenv("SIYUAN_LLM_PROXY"); "" != proxy {
		openAI.APIProxy = proxy
	} else if proxy := os.Getenv("SIYUAN_OPENAI_API_PROXY"); "" != proxy {
		openAI.APIProxy = proxy
	}

	if userAgent := os.Getenv("SIYUAN_LLM_USER_AGENT"); "" != userAgent {
		openAI.APIUserAgent = userAgent
	} else if userAgent := os.Getenv("SIYUAN_OPENAI_API_USER_AGENT"); "" != userAgent {
		openAI.APIUserAgent = userAgent
	}

	if version := os.Getenv("SIYUAN_LLM_API_VERSION"); "" != version {
		openAI.APIVersion = version
	} else if version := os.Getenv("SIYUAN_OPENAI_API_VERSION"); "" != version {
		openAI.APIVersion = version
	}

	// Initialize embedding configuration with defaults
	embedding := &Embedding{
		Provider:       "siliconflow", // Default to SiliconFlow as per LingShu notes
		Model:          "BAAI/bge-large-zh-v1.5", // Default Chinese embedding model
		APIBaseURL:     "https://api.siliconflow.cn/v1", // 基础 URL，OpenAI 客户端会自动添加 /embeddings 端点
		EncodingFormat: "float",
		Timeout:        30,
		Enabled:        false, // Disabled by default until API key is configured
	}

	// Load embedding settings from environment variables
	if apiKey := os.Getenv("SIYUAN_EMBEDDING_API_KEY"); "" != apiKey {
		embedding.APIKey = apiKey
		embedding.Enabled = true
	}

	if provider := os.Getenv("SIYUAN_EMBEDDING_PROVIDER"); "" != provider {
		embedding.Provider = provider
	}

	if model := os.Getenv("SIYUAN_EMBEDDING_MODEL"); "" != model {
		embedding.Model = model
	}

	if baseURL := os.Getenv("SIYUAN_EMBEDDING_API_BASE_URL"); "" != baseURL {
		embedding.APIBaseURL = baseURL
	}

	if encodingFormat := os.Getenv("SIYUAN_EMBEDDING_ENCODING_FORMAT"); "" != encodingFormat {
		embedding.EncodingFormat = encodingFormat
	}

	if timeout := os.Getenv("SIYUAN_EMBEDDING_TIMEOUT"); "" != timeout {
		timeoutInt, err := strconv.Atoi(timeout)
		if err == nil {
			embedding.Timeout = timeoutInt
		}
	}

	return &AI{
		OpenAI:   openAI,
		Embedding: embedding,
	}
}
