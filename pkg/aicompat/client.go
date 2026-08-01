package aicompat

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/xiaoguiwucan/openchat-wx/utils"
)

type Config struct {
	BaseURL string `json:"base_url"`
	APIKey  string `json:"api_key"`
	Model   string `json:"model"`
}

type ImageGenerationConfig struct {
	Config
	Size           string `json:"size"`
	Quality        string `json:"quality"`
	ResponseFormat string `json:"response_format"`
}

type GeneratedImage struct {
	URL       string
	Data      []byte
	MediaType string
}

type Client struct {
	httpClient *http.Client
}

func NewClient() *Client {
	return &Client{httpClient: &http.Client{Timeout: 3 * time.Minute}}
}

func (c *Client) DescribeImage(ctx context.Context, config Config, imageURL, prompt string) (string, error) {
	if err := validateConfig(config); err != nil {
		return "", err
	}
	if strings.TrimSpace(imageURL) == "" {
		return "", fmt.Errorf("image URL is required")
	}
	if strings.TrimSpace(prompt) == "" {
		prompt = "请准确描述这张图片或表情包的主体、文字、情绪和适合的回复语境。"
	}

	payload := map[string]any{
		"model": config.Model,
		"messages": []any{map[string]any{
			"role": "user",
			"content": []any{
				map[string]any{"type": "text", "text": prompt},
				map[string]any{"type": "image_url", "image_url": map[string]any{"url": imageURL}},
			},
		}},
		"stream": false,
	}

	var response struct {
		Choices []struct {
			Message struct {
				Content any `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := c.postJSON(ctx, config, "/chat/completions", payload, &response); err != nil {
		return "", err
	}
	if len(response.Choices) == 0 {
		return "", fmt.Errorf("vision response has no choices")
	}
	content := extractTextContent(response.Choices[0].Message.Content)
	if content == "" {
		return "", fmt.Errorf("vision response is empty")
	}
	return content, nil
}

func (c *Client) GenerateImage(ctx context.Context, config ImageGenerationConfig, prompt string) (*GeneratedImage, error) {
	if err := validateConfig(config.Config); err != nil {
		return nil, err
	}
	if strings.TrimSpace(prompt) == "" {
		return nil, fmt.Errorf("image prompt is required")
	}

	payload := map[string]any{"model": config.Model, "prompt": prompt, "n": 1}
	if config.Size != "" {
		payload["size"] = config.Size
	}
	if config.Quality != "" {
		payload["quality"] = config.Quality
	}
	if config.ResponseFormat != "" {
		payload["response_format"] = config.ResponseFormat
	}

	var response struct {
		Data []struct {
			URL     string `json:"url"`
			B64JSON string `json:"b64_json"`
		} `json:"data"`
	}
	if err := c.postJSON(ctx, config.Config, "/images/generations", payload, &response); err != nil {
		return nil, err
	}
	if len(response.Data) == 0 {
		return nil, fmt.Errorf("image response has no data")
	}
	item := response.Data[0]
	if item.URL != "" {
		return &GeneratedImage{URL: utils.RewriteLoopbackURL(item.URL)}, nil
	}
	if item.B64JSON != "" {
		data, err := base64.StdEncoding.DecodeString(item.B64JSON)
		if err != nil {
			return nil, fmt.Errorf("decode generated image: %w", err)
		}
		return &GeneratedImage{Data: data, MediaType: http.DetectContentType(data)}, nil
	}
	return nil, fmt.Errorf("image response contains neither url nor b64_json")
}

func (c *Client) postJSON(ctx context.Context, config Config, path string, payload any, target any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	endpoint := utils.NormalizeAIBaseURL(config.BaseURL) + path
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+config.APIKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request AI endpoint: %w", err)
	}
	defer resp.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("AI endpoint returned HTTP %d: %s", resp.StatusCode, apiErrorMessage(responseBody))
	}
	if err := json.Unmarshal(responseBody, target); err != nil {
		return fmt.Errorf("decode AI response: %w", err)
	}
	return nil
}

func validateConfig(config Config) error {
	if strings.TrimSpace(config.BaseURL) == "" || strings.TrimSpace(config.APIKey) == "" || strings.TrimSpace(config.Model) == "" {
		return fmt.Errorf("base_url, api_key and model are required")
	}
	return nil
}

func extractTextContent(content any) string {
	switch value := content.(type) {
	case string:
		return strings.TrimSpace(value)
	case []any:
		var builder strings.Builder
		for _, raw := range value {
			part, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			if text, ok := part["text"].(string); ok {
				builder.WriteString(text)
			}
		}
		return strings.TrimSpace(builder.String())
	default:
		return ""
	}
}

func apiErrorMessage(body []byte) string {
	var payload struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
		Message string `json:"message"`
	}
	if json.Unmarshal(body, &payload) == nil {
		message := payload.Error.Message
		if message == "" {
			message = payload.Message
		}
		if message != "" {
			return truncate(message, 500)
		}
	}
	return truncate(strings.TrimSpace(string(body)), 500)
}

func truncate(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit]) + "..."
}
