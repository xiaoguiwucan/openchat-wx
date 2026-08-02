package service

import (
	"encoding/json"
	"testing"

	"github.com/xiaoguiwucan/openchat-wx/interface/settings"
	"github.com/xiaoguiwucan/openchat-wx/model"
	"gorm.io/datatypes"
)

func TestApplyAIProvider(t *testing.T) {
	config := settings.AIConfig{BaseURL: "https://legacy.example/v1", APIKey: "legacy", Model: "legacy-model"}
	provider := &model.AIProvider{
		BaseURL: "https://provider.example/v1", APIKey: "secret", ChatModel: "chat-model",
		ImageRecognitionModel: "vision-model", ImageGenerationModel: "image-model",
		ImageSize: "1536x1024", ImageQuality: "high",
	}

	ApplyAIProvider(&config, provider)

	if config.BaseURL != provider.BaseURL || config.APIKey != provider.APIKey || config.Model != "legacy-model" {
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

func TestApplyAIProviderUsesDefaultModelsWhenSettingsAreEmpty(t *testing.T) {
	provider := &model.AIProvider{ChatModel: "chat-model", ImageRecognitionModel: "vision-model"}
	config := settings.AIConfig{}
	ApplyAIProvider(&config, provider)
	if config.Model != provider.ChatModel || config.ImageRecognitionModel != provider.ImageRecognitionModel {
		t.Fatalf("provider defaults were not applied: %#v", config)
	}
}

func TestApplyAIProviderKeepsSelectedImageModel(t *testing.T) {
	provider := &model.AIProvider{
		BaseURL: "https://provider.example/v1", APIKey: "provider-key", ImageGenerationModel: "provider-image-model",
		ImageSize: "1024x1024", ImageQuality: "high",
	}
	config := settings.AIConfig{ImageAISettings: datatypes.JSON(`{"model":"selected-image-model","size":"1536x1024"}`)}

	ApplyAIProvider(&config, provider)

	var image map[string]string
	if err := json.Unmarshal(config.ImageAISettings, &image); err != nil {
		t.Fatalf("image settings are invalid JSON: %v", err)
	}
	if image["model"] != "selected-image-model" || image["size"] != "1536x1024" {
		t.Fatalf("selected image settings were overwritten: %#v", image)
	}
	if image["base_url"] != provider.BaseURL || image["api_key"] != provider.APIKey {
		t.Fatalf("provider credentials were not applied: %#v", image)
	}
}

func TestApplyAIProvidersUsesIndependentCapabilityCredentials(t *testing.T) {
	chatProvider := &model.AIProvider{
		BaseURL: "https://chat.example/v1", APIKey: "chat-key", ChatModel: "chat-model",
	}
	visionProvider := &model.AIProvider{
		BaseURL: "https://vision.example/v1", APIKey: "vision-key", ImageRecognitionModel: "vision-model",
	}
	imageProvider := &model.AIProvider{
		BaseURL: "https://image.example/v1", APIKey: "image-key", ImageGenerationModel: "image-model",
		ImageSize: "1536x1024", ImageQuality: "high",
	}
	config := settings.AIConfig{}

	ApplyAIProviders(&config, chatProvider, visionProvider, imageProvider)

	if config.BaseURL != chatProvider.BaseURL || config.APIKey != chatProvider.APIKey || config.Model != chatProvider.ChatModel {
		t.Fatalf("chat capability did not use its provider: %#v", config)
	}
	if config.ImageRecognitionBaseURL != visionProvider.BaseURL || config.ImageRecognitionAPIKey != visionProvider.APIKey || config.ImageRecognitionModel != visionProvider.ImageRecognitionModel {
		t.Fatalf("vision capability did not use its provider: %#v", config)
	}
	var image map[string]string
	if err := json.Unmarshal(config.ImageAISettings, &image); err != nil {
		t.Fatalf("image settings are invalid JSON: %v", err)
	}
	if image["base_url"] != imageProvider.BaseURL || image["api_key"] != imageProvider.APIKey || image["model"] != imageProvider.ImageGenerationModel {
		t.Fatalf("image capability did not use its provider: %#v", image)
	}
}

func TestResolveAIProviderIDsKeepsLegacyOverrideOnlyForUnboundCapabilities(t *testing.T) {
	legacyID, chatID, visionID, imageID, overrideID := int64(1), int64(2), int64(3), int64(4), int64(9)
	global := &model.GlobalSettings{
		AIProviderID:               &legacyID,
		ChatAIProviderID:           &chatID,
		ImageRecognitionProviderID: &visionID,
		ImageGenerationProviderID:  &imageID,
	}

	resolvedChatID, resolvedVisionID, resolvedImageID := resolveAIProviderIDs(global, &overrideID)
	if *resolvedChatID != overrideID || *resolvedVisionID != visionID || *resolvedImageID != imageID {
		t.Fatalf("resolved IDs = chat:%d vision:%d image:%d", *resolvedChatID, *resolvedVisionID, *resolvedImageID)
	}

	global.ImageRecognitionProviderID = nil
	global.ImageGenerationProviderID = nil
	_, resolvedVisionID, resolvedImageID = resolveAIProviderIDs(global, &overrideID)
	if *resolvedVisionID != overrideID || *resolvedImageID != overrideID {
		t.Fatalf("legacy capability override was not preserved: vision:%d image:%d", *resolvedVisionID, *resolvedImageID)
	}
}

func TestResolveAIProviderIDsForTargetUsesIndependentOverrides(t *testing.T) {
	legacyID, globalChatID, globalVisionID, globalImageID := int64(1), int64(2), int64(3), int64(4)
	roomChatID, roomVisionID, roomImageID := int64(5), int64(6), int64(7)
	global := &model.GlobalSettings{
		AIProviderID:               &legacyID,
		ChatAIProviderID:           &globalChatID,
		ImageRecognitionProviderID: &globalVisionID,
		ImageGenerationProviderID:  &globalImageID,
	}

	chatID, visionID, imageID := resolveAIProviderIDsForTarget(
		global, &legacyID, &roomChatID, &roomVisionID, &roomImageID,
	)
	if *chatID != roomChatID || *visionID != roomVisionID || *imageID != roomImageID {
		t.Fatalf("room overrides = chat:%d vision:%d image:%d", *chatID, *visionID, *imageID)
	}

	chatID, visionID, imageID = resolveAIProviderIDsForTarget(global, nil, nil, &roomVisionID, nil)
	if *chatID != globalChatID || *visionID != roomVisionID || *imageID != globalImageID {
		t.Fatalf("mixed inheritance = chat:%d vision:%d image:%d", *chatID, *visionID, *imageID)
	}

	inherit := int64(0)
	chatID, visionID, imageID = resolveAIProviderIDsForTarget(global, &legacyID, &inherit, &inherit, &inherit)
	if *chatID != globalChatID || *visionID != globalVisionID || *imageID != globalImageID {
		t.Fatalf("explicit inheritance kept legacy override = chat:%d vision:%d image:%d", *chatID, *visionID, *imageID)
	}
}

func TestChatRoomAIConfigUsesExplicitImageGenerationModel(t *testing.T) {
	selectedModel := "room-image-model"
	service := &ChatRoomSettingsService{
		globalSettings: &model.GlobalSettings{},
		chatRoomSettings: &model.ChatRoomSettings{
			ImageGenerationModel: &selectedModel,
			ImageAISettings:      datatypes.JSON(`{"model":"legacy-model","size":"1536x1024"}`),
		},
	}

	config := service.GetAIConfig()
	var image map[string]string
	if err := json.Unmarshal(config.ImageAISettings, &image); err != nil {
		t.Fatalf("image settings are invalid JSON: %v", err)
	}
	if image["model"] != selectedModel || image["size"] != "1536x1024" {
		t.Fatalf("room image model was not applied: %#v", image)
	}
}

func TestProviderFromInputStoresAvailableModels(t *testing.T) {
	models := []string{"vision-model", "chat-model", "chat-model", " "}
	provider, err := (&AIProviderService{}).providerFromInput(0, AIProviderInput{
		Name: "test", BaseURL: "https://provider.example/v1", ChatModel: "chat-model", AvailableModels: &models,
	})
	if err != nil {
		t.Fatalf("providerFromInput returned error: %v", err)
	}
	var stored []string
	if err := json.Unmarshal(provider.AvailableModels, &stored); err != nil {
		t.Fatalf("available models are invalid JSON: %v", err)
	}
	want := []string{"chat-model", "vision-model"}
	if len(stored) != len(want) || stored[0] != want[0] || stored[1] != want[1] {
		t.Fatalf("stored models = %#v, want %#v", stored, want)
	}
	if provider.ModelsRefreshedAt == nil {
		t.Fatal("models refresh time was not stored")
	}
}

func TestValidateAIProviderModelRejectsCrossProviderModel(t *testing.T) {
	provider := &model.AIProvider{
		Name:            "BigSea",
		ChatModel:       "gpt-chat",
		AvailableModels: datatypes.JSON(`["gpt-chat","gpt-image"]`),
	}
	if err := ValidateAIProviderModel(provider, "gpt-chat", "AI回复"); err != nil {
		t.Fatalf("valid model was rejected: %v", err)
	}
	if err := ValidateAIProviderModel(provider, "grok-chat", "AI回复"); err == nil {
		t.Fatal("cross-provider model was accepted")
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
