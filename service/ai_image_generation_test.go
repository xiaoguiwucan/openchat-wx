package service

import (
	"testing"

	"github.com/xiaoguiwucan/openchat-wx/interface/settings"
	"gorm.io/datatypes"
)

func TestParseImageGenerationConfig(t *testing.T) {
	config, err := parseImageGenerationConfig(settings.AIConfig{
		BaseURL: "https://chat.example/v1",
		APIKey:  "chat-key",
		ImageAISettings: datatypes.JSON(`{
			"openai_compatible": {
				"base_url": "https://image.example/v1",
				"api_key": "image-key",
				"model": "image-model",
				"quality": "high"
			}
		}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if config.BaseURL != "https://image.example/v1" || config.APIKey != "image-key" || config.Model != "image-model" {
		t.Fatalf("unexpected config: %+v", config)
	}
	if config.Size != "1024x1024" || config.Quality != "high" {
		t.Fatalf("unexpected image options: %+v", config)
	}
}

func TestParseImageGenerationConfigFallsBackToChatProvider(t *testing.T) {
	config, err := parseImageGenerationConfig(settings.AIConfig{
		BaseURL:         "https://chat.example/v1",
		APIKey:          "chat-key",
		ImageAISettings: datatypes.JSON(`{"model":"image-model"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if config.BaseURL != "https://chat.example/v1" || config.APIKey != "chat-key" {
		t.Fatalf("chat provider fallback failed: %+v", config)
	}
}
