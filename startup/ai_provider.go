package startup

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/xiaoguiwucan/openchat-wx/model"
	"github.com/xiaoguiwucan/openchat-wx/repository"
	"github.com/xiaoguiwucan/openchat-wx/vars"
)

// SeedAIProviders copies distinct legacy AI connection settings into reusable
// channels without changing which configuration is currently selected.
func SeedAIProviders() error {
	var count int64
	if err := vars.DB.Model(&model.AIProvider{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	repo := repository.NewAIProviderRepo(context.Background(), vars.DB)
	seen := map[string]struct{}{}
	create := func(name, baseURL, apiKey, chatModel, visionModel, summaryModel string, imageSettings []byte) error {
		baseURL, apiKey, chatModel = strings.TrimSpace(baseURL), strings.TrimSpace(apiKey), strings.TrimSpace(chatModel)
		if baseURL == "" || apiKey == "" || chatModel == "" {
			return nil
		}
		fingerprint := baseURL + "\x00" + apiKey + "\x00" + chatModel
		if _, ok := seen[fingerprint]; ok {
			return nil
		}
		seen[fingerprint] = struct{}{}
		imageModel, imageSize, imageQuality := legacyImageConfig(imageSettings)
		provider := &model.AIProvider{
			Name: name, BaseURL: baseURL, APIKey: apiKey, ChatModel: chatModel,
			ImageRecognitionModel: strings.TrimSpace(visionModel), ImageGenerationModel: imageModel,
			SummaryModel: strings.TrimSpace(summaryModel), ImageSize: imageSize, ImageQuality: imageQuality, Enabled: true,
		}
		if err := repo.Create(provider); err != nil {
			return err
		}
		log.Printf("[AIProvider] 已从旧配置迁移渠道: %s", name)
		return nil
	}

	var global model.GlobalSettings
	if err := vars.DB.First(&global).Error; err == nil {
		if err := create("默认渠道（旧配置迁移）", global.ChatBaseURL, global.ChatAPIKey, global.ChatModel,
			global.ImageRecognitionModel, global.ChatRoomSummaryModel, global.ImageAISettings); err != nil {
			return err
		}
	}
	var rooms []model.ChatRoomSettings
	if err := vars.DB.Find(&rooms).Error; err != nil {
		return err
	}
	for _, room := range rooms {
		if room.ChatBaseURL == nil || room.ChatAPIKey == nil || room.ChatModel == nil {
			continue
		}
		vision, summary := "", ""
		if room.ImageRecognitionModel != nil {
			vision = *room.ImageRecognitionModel
		}
		if room.ChatRoomSummaryModel != nil {
			summary = *room.ChatRoomSummaryModel
		}
		if err := create(fmt.Sprintf("群聊渠道 %s", room.ChatRoomID), *room.ChatBaseURL, *room.ChatAPIKey,
			*room.ChatModel, vision, summary, room.ImageAISettings); err != nil {
			return err
		}
	}
	return nil
}

func legacyImageConfig(raw []byte) (modelName, size, quality string) {
	size = "1024x1024"
	if len(raw) == 0 {
		return
	}
	var config map[string]any
	if json.Unmarshal(raw, &config) != nil {
		return
	}
	modelName, _ = config["model"].(string)
	if value, ok := config["size"].(string); ok && value != "" {
		size = value
	}
	quality, _ = config["quality"].(string)
	return
}
