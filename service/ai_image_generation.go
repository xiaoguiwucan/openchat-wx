package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"github.com/xiaoguiwucan/openchat-wx/interface/settings"
	"github.com/xiaoguiwucan/openchat-wx/pkg/aicompat"
)

type AIImageGenerationService struct {
	client *aicompat.Client
}

func NewAIImageGenerationService() *AIImageGenerationService {
	return &AIImageGenerationService{client: aicompat.NewClient()}
}

func (s *AIImageGenerationService) Generate(ctx context.Context, aiConfig settings.AIConfig, prompt string) (*aicompat.GeneratedImage, error) {
	config, err := parseImageGenerationConfig(aiConfig)
	if err != nil {
		return nil, err
	}
	prompt = strings.TrimSpace(prompt)
	var lastErr error
	for attempt := 1; attempt <= 2; attempt++ {
		result, generateErr := s.client.GenerateImage(ctx, config, prompt)
		if generateErr == nil {
			return result, nil
		}
		lastErr = generateErr
		if attempt == 2 || !isTransientImageGenerationError(generateErr) {
			break
		}
		log.Printf("[ImageGeneration] transient upstream failure, retrying attempt=%d error=%v", attempt+1, generateErr)
		timer := time.NewTimer(2 * time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
	return nil, fmt.Errorf("生图请求失败（已重试）: %w", lastErr)
}

func isTransientImageGenerationError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	message := strings.ToLower(err.Error())
	for _, marker := range []string{
		"unexpected eof", "connection reset", "broken pipe", "timeout", "temporarily unavailable",
		"http 429", "http 500", "http 502", "http 503", "http 504",
	} {
		if strings.Contains(message, marker) {
			return true
		}
	}
	return false
}

func parseImageGenerationConfig(aiConfig settings.AIConfig) (aicompat.ImageGenerationConfig, error) {
	config := aicompat.ImageGenerationConfig{}
	if len(aiConfig.ImageAISettings) > 0 {
		var root map[string]json.RawMessage
		if err := json.Unmarshal(aiConfig.ImageAISettings, &root); err != nil {
			return config, fmt.Errorf("invalid image_ai_settings: %w", err)
		}

		_ = json.Unmarshal(aiConfig.ImageAISettings, &config)
		if config.Model == "" {
			for _, key := range []string{"openai_compatible", "openai-compatible", "custom"} {
				if raw, ok := root[key]; ok {
					_ = json.Unmarshal(raw, &config)
					if config.Model != "" {
						break
					}
				}
			}
		}
	}

	if config.BaseURL == "" {
		config.BaseURL = aiConfig.BaseURL
	}
	if config.APIKey == "" {
		config.APIKey = aiConfig.APIKey
	}
	if config.Size == "" {
		config.Size = "1024x1024"
	}
	// "auto" is a UI placeholder, not an OpenAI image API value. Some
	// compatible providers reject it instead of applying their own default.
	if strings.EqualFold(strings.TrimSpace(config.Quality), "auto") {
		config.Quality = ""
	}
	if strings.EqualFold(strings.TrimSpace(config.ResponseFormat), "auto") {
		config.ResponseFormat = ""
	}
	if config.Model == "" {
		return config, fmt.Errorf("绘图模型未配置，请在 image_ai_settings 中填写 model")
	}
	if config.BaseURL == "" || config.APIKey == "" {
		return config, fmt.Errorf("绘图中转站未配置，请填写 base_url 和 api_key，或复用聊天配置")
	}
	return config, nil
}
