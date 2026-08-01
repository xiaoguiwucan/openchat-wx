package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

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
	return s.client.GenerateImage(ctx, config, strings.TrimSpace(prompt))
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
	if config.Model == "" {
		return config, fmt.Errorf("绘图模型未配置，请在 image_ai_settings 中填写 model")
	}
	if config.BaseURL == "" || config.APIKey == "" {
		return config, fmt.Errorf("绘图中转站未配置，请填写 base_url 和 api_key，或复用聊天配置")
	}
	return config, nil
}
