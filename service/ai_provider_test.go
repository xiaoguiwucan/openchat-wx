package service

import (
	"encoding/json"
	"testing"

	"github.com/xiaoguiwucan/openchat-wx/interface/settings"
	"github.com/xiaoguiwucan/openchat-wx/model"
)

func TestApplyAIProvider(t *testing.T) {
	config := settings.AIConfig{BaseURL: "https://legacy.example/v1", APIKey: "legacy", Model: "legacy-model"}
	provider := &model.AIProvider{
		BaseURL: "https://provider.example/v1", APIKey: "secret", ChatModel: "chat-model",
		ImageRecognitionModel: "vision-model", ImageGenerationModel: "image-model",
		ImageSize: "1536x1024", ImageQuality: "high",
	}

	ApplyAIProvider(&config, provider)

	if config.BaseURL != provider.BaseURL || config.APIKey != provider.APIKey || config.Model != provider.ChatModel {
		t.Fatalf("chat provider was not applied: %#v", config)
	}
	if config.ImageRecognitionModel != provider.ImageRecognitionModel {
		t.Fatalf("vision model = %q, want %q", config.ImageRecognitionModel, provider.ImageRecognitionModel)
	}
	var image map[string]string
	if err := json.Unmarshal(config.ImageAISettings, &image); err != nil {
		t.Fatalf("image settings are invalid JSON: %v", err)
	}
	if image["model"] != provider.ImageGenerationModel || image["size"] != provider.ImageSize || image["quality"] != provider.ImageQuality {
		t.Fatalf("image provider was not applied: %#v", image)
	}
}

func TestMaskAPIKey(t *testing.T) {
	if got := maskAPIKey("sk-1234567890"); got != "sk-1...7890" {
		t.Fatalf("maskAPIKey() = %q", got)
	}
	if got := maskAPIKey("short"); got != "********" {
		t.Fatalf("short mask = %q", got)
	}
}
