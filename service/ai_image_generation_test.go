package service

import (
	"errors"
	"io"
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

func TestIsTransientImageGenerationError(t *testing.T) {
	tests := []struct {
		err  error
		want bool
	}{
		{err: io.ErrUnexpectedEOF, want: true},
		{err: errors.New("AI endpoint returned HTTP 502: bad gateway"), want: true},
		{err: errors.New("connection reset by peer"), want: true},
		{err: errors.New("AI endpoint returned HTTP 400: invalid model"), want: false},
	}
	for _, tt := range tests {
		if got := isTransientImageGenerationError(tt.err); got != tt.want {
			t.Fatalf("isTransientImageGenerationError(%q) = %v, want %v", tt.err, got, tt.want)
		}
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

func TestParseImageGenerationConfigOmitsAutoPlaceholders(t *testing.T) {
	config, err := parseImageGenerationConfig(settings.AIConfig{
		BaseURL: "https://image.example/v1",
		APIKey:  "image-key",
		ImageAISettings: datatypes.JSON(`{
			"model":"image-model",
			"quality":"auto",
			"response_format":"AUTO"
		}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if config.Quality != "" || config.ResponseFormat != "" {
		t.Fatalf("auto placeholders must be omitted, got quality=%q response_format=%q", config.Quality, config.ResponseFormat)
	}
}
