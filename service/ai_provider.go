package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/openai/openai-go/v3"
	"github.com/xiaoguiwucan/openchat-wx/interface/settings"
	"github.com/xiaoguiwucan/openchat-wx/model"
	"github.com/xiaoguiwucan/openchat-wx/repository"
	"github.com/xiaoguiwucan/openchat-wx/utils"
	"github.com/xiaoguiwucan/openchat-wx/vars"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	AIProviderScopeGlobal   = "global"
	AIProviderScopeChatRoom = "chat_room"
	AIProviderScopeFriend   = "friend"
)

type AIProviderInput struct {
	Name                  string    `json:"name" binding:"required"`
	BaseURL               string    `json:"base_url" binding:"required"`
	APIKey                string    `json:"api_key"`
	ChatModel             string    `json:"chat_model" binding:"required"`
	ImageRecognitionModel string    `json:"image_recognition_model"`
	ImageGenerationModel  string    `json:"image_generation_model"`
	SummaryModel          string    `json:"summary_model"`
	ImageSize             string    `json:"image_size"`
	ImageQuality          string    `json:"image_quality"`
	AvailableModels       *[]string `json:"available_models"`
	Enabled               *bool     `json:"enabled"`
}

type AIProviderSelection struct {
	ProviderID int64  `json:"provider_id" binding:"required"`
	Scope      string `json:"scope" binding:"required"`
	TargetID   string `json:"target_id"`
}

type AIProviderView struct {
	ID                    int64      `json:"id"`
	Name                  string     `json:"name"`
	BaseURL               string     `json:"base_url"`
	APIKeyMasked          string     `json:"api_key_masked"`
	HasAPIKey             bool       `json:"has_api_key"`
	ChatModel             string     `json:"chat_model"`
	ImageRecognitionModel string     `json:"image_recognition_model"`
	ImageGenerationModel  string     `json:"image_generation_model"`
	SummaryModel          string     `json:"summary_model"`
	ImageSize             string     `json:"image_size"`
	ImageQuality          string     `json:"image_quality"`
	AvailableModels       []string   `json:"available_models"`
	ModelsRefreshedAt     *time.Time `json:"models_refreshed_at"`
	Enabled               bool       `json:"enabled"`
	GlobalSelected        bool       `json:"global_selected"`
	TargetSelected        bool       `json:"target_selected"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
}

type AIProviderTestResult struct {
	Success   bool   `json:"success"`
	LatencyMS int64  `json:"latency_ms"`
	Model     string `json:"model"`
	Message   string `json:"message"`
}

type AIProviderModelsInput struct {
	ProviderID int64  `json:"provider_id"`
	BaseURL    string `json:"base_url"`
	APIKey     string `json:"api_key"`
}

type AIProviderModelsResult struct {
	Models      []string  `json:"models"`
	Count       int       `json:"count"`
	RefreshedAt time.Time `json:"refreshed_at"`
}

type AIProviderService struct {
	ctx  context.Context
	repo *repository.AIProvider
}

func NewAIProviderService(ctx context.Context) *AIProviderService {
	return &AIProviderService{ctx: ctx, repo: repository.NewAIProviderRepo(ctx, vars.DB)}
}

func (s *AIProviderService) List(scope, targetID string) ([]AIProviderView, error) {
	providers, err := s.repo.List()
	if err != nil {
		return nil, err
	}
	globalID, targetProviderID, err := s.selectedProviderIDs(scope, targetID)
	if err != nil {
		return nil, err
	}
	views := make([]AIProviderView, 0, len(providers))
	for _, provider := range providers {
		views = append(views, providerView(provider, globalID, targetProviderID))
	}
	return views, nil
}

func (s *AIProviderService) Create(input AIProviderInput) (*AIProviderView, error) {
	provider, err := s.providerFromInput(0, input)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(input.APIKey) == "" {
		return nil, errors.New("API Key 不能为空")
	}
	if err := s.ensureUniqueName(provider.Name, 0); err != nil {
		return nil, err
	}
	if err := s.repo.Create(provider); err != nil {
		return nil, fmt.Errorf("创建模型渠道失败: %w", err)
	}
	view := providerView(provider, 0, 0)
	return &view, nil
}

func (s *AIProviderService) Update(id int64, input AIProviderInput) (*AIProviderView, error) {
	existing, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, errors.New("模型渠道不存在")
	}
	provider, err := s.providerFromInput(id, input)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(input.APIKey) == "" {
		provider.APIKey = existing.APIKey
	}
	if input.AvailableModels == nil {
		provider.AvailableModels = existing.AvailableModels
		provider.ModelsRefreshedAt = existing.ModelsRefreshedAt
	}
	if err := s.ensureUniqueName(provider.Name, id); err != nil {
		return nil, err
	}
	if err := s.repo.Update(provider); err != nil {
		return nil, fmt.Errorf("更新模型渠道失败: %w", err)
	}
	provider, err = s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	view := providerView(provider, 0, 0)
	return &view, nil
}

func (s *AIProviderService) Delete(id int64) error {
	provider, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}
	if provider == nil {
		return errors.New("模型渠道不存在")
	}
	var references int64
	if err := vars.DB.WithContext(s.ctx).Table("chat_room_settings").Where(
		"ai_provider_id = ? OR chat_ai_provider_id = ? OR image_recognition_provider_id = ? OR image_generation_provider_id = ? OR summary_ai_provider_id = ?",
		id, id, id, id, id,
	).Count(&references).Error; err != nil {
		return err
	}
	var friendReferences int64
	if err := vars.DB.WithContext(s.ctx).Table("friend_settings").Where("ai_provider_id = ?", id).Count(&friendReferences).Error; err != nil {
		return err
	}
	references += friendReferences
	var globalReferences int64
	if err := vars.DB.WithContext(s.ctx).Table("global_settings").Where(
		"ai_provider_id = ? OR chat_ai_provider_id = ? OR image_recognition_provider_id = ? OR image_generation_provider_id = ? OR summary_ai_provider_id = ? OR text_embedding_provider_id = ?",
		id, id, id, id, id, id,
	).Count(&globalReferences).Error; err != nil {
		return err
	}
	references += globalReferences
	if references > 0 {
		return errors.New("该渠道仍被全局、群聊或好友配置使用，请先切换后再删除")
	}
	return s.repo.Delete(id)
}

func (s *AIProviderService) Select(selection AIProviderSelection) error {
	provider, err := s.repo.GetByID(selection.ProviderID)
	if err != nil {
		return err
	}
	if provider == nil || !provider.Enabled {
		return errors.New("模型渠道不存在或已停用")
	}
	switch selection.Scope {
	case AIProviderScopeGlobal:
		return s.selectGlobal(provider)
	case AIProviderScopeChatRoom:
		if strings.TrimSpace(selection.TargetID) == "" {
			return errors.New("群聊 ID 不能为空")
		}
		imageConfig, _ := json.Marshal(map[string]string{
			"base_url": provider.BaseURL, "api_key": provider.APIKey, "model": provider.ImageGenerationModel,
			"size": provider.ImageSize, "quality": provider.ImageQuality,
		})
		updates := map[string]any{
			"ai_provider_id": provider.ID, "chat_model": provider.ChatModel,
			"image_recognition_model": provider.ImageRecognitionModel, "image_ai_settings": imageConfig,
		}
		if provider.SummaryModel != "" {
			updates["chat_room_summary_model"] = provider.SummaryModel
		}
		result := vars.DB.WithContext(s.ctx).Model(&model.ChatRoomSettings{}).
			Where("chat_room_id = ?", selection.TargetID).Updates(updates)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return errors.New("群聊配置不存在，请先保存该群的基础设置")
		}
		return nil
	case AIProviderScopeFriend:
		if strings.TrimSpace(selection.TargetID) == "" {
			return errors.New("好友微信 ID 不能为空")
		}
		imageConfig, _ := json.Marshal(map[string]string{
			"base_url": provider.BaseURL, "api_key": provider.APIKey, "model": provider.ImageGenerationModel,
			"size": provider.ImageSize, "quality": provider.ImageQuality,
		})
		result := vars.DB.WithContext(s.ctx).Model(&model.FriendSettings{}).
			Where("wechat_id = ?", selection.TargetID).Updates(map[string]any{
			"ai_provider_id": provider.ID, "chat_model": provider.ChatModel,
			"image_recognition_model": provider.ImageRecognitionModel, "image_ai_settings": imageConfig,
		})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return errors.New("好友配置不存在，请先保存该好友的基础设置")
		}
		return nil
	default:
		return errors.New("scope 仅支持 global、chat_room 或 friend")
	}
}

func (s *AIProviderService) Test(id int64) (*AIProviderTestResult, error) {
	provider, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if provider == nil {
		return nil, errors.New("模型渠道不存在")
	}
	ctx, cancel := context.WithTimeout(s.ctx, 45*time.Second)
	defer cancel()
	client := newOpenAIClient(provider.APIKey, provider.BaseURL)
	started := time.Now()
	message, err := streamChatCompletionMessage(ctx, &client, openai.ChatCompletionNewParams{
		Model:               provider.ChatModel,
		Messages:            []openai.ChatCompletionMessageParamUnion{openai.UserMessage("Reply with OK only.")},
		MaxCompletionTokens: openai.Int(16),
	})
	latency := time.Since(started).Milliseconds()
	if err != nil {
		return &AIProviderTestResult{Success: false, LatencyMS: latency, Model: provider.ChatModel, Message: err.Error()}, nil
	}
	if strings.TrimSpace(message.Content) == "" {
		return &AIProviderTestResult{Success: false, LatencyMS: latency, Model: provider.ChatModel, Message: "渠道返回了空内容"}, nil
	}
	return &AIProviderTestResult{Success: true, LatencyMS: latency, Model: provider.ChatModel, Message: "渠道连接和对话模型验证通过"}, nil
}

func (s *AIProviderService) FetchModels(input AIProviderModelsInput) (*AIProviderModelsResult, error) {
	baseURL := strings.TrimSpace(input.BaseURL)
	apiKey := strings.TrimSpace(input.APIKey)
	var savedProvider *model.AIProvider
	if input.ProviderID > 0 {
		provider, err := s.repo.GetByID(input.ProviderID)
		if err != nil {
			return nil, err
		}
		if provider == nil {
			return nil, errors.New("模型渠道不存在")
		}
		savedProvider = provider
		if baseURL == "" {
			baseURL = provider.BaseURL
		}
		if apiKey == "" {
			apiKey = provider.APIKey
		}
	}
	if baseURL == "" || apiKey == "" {
		return nil, errors.New("请先填写 Base URL 和 API Key")
	}
	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		return nil, errors.New("Base URL 必须以 http:// 或 https:// 开头")
	}

	ctx, cancel := context.WithTimeout(s.ctx, 20*time.Second)
	defer cancel()
	endpoint := utils.NormalizeAIBaseURL(baseURL) + "/models"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("创建模型列表请求失败: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "openchat-wx/1.1")
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("获取模型列表失败: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("获取模型列表失败: HTTP %d", response.StatusCode)
	}
	var payload struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("解析模型列表失败: %w", err)
	}
	unique := make(map[string]struct{}, len(payload.Data))
	models := make([]string, 0, len(payload.Data))
	for _, item := range payload.Data {
		modelID := strings.TrimSpace(item.ID)
		if modelID == "" {
			continue
		}
		if _, exists := unique[modelID]; exists {
			continue
		}
		unique[modelID] = struct{}{}
		models = append(models, modelID)
	}
	if len(models) == 0 {
		return nil, errors.New("渠道没有返回可用模型")
	}
	sort.Strings(models)
	refreshedAt := time.Now()
	if savedProvider != nil && strings.TrimRight(baseURL, "/") == strings.TrimRight(savedProvider.BaseURL, "/") && strings.TrimSpace(input.APIKey) == "" {
		encodedModels, err := json.Marshal(models)
		if err != nil {
			return nil, fmt.Errorf("保存模型列表失败: %w", err)
		}
		if err := s.repo.UpdateModelCache(savedProvider.ID, encodedModels, refreshedAt); err != nil {
			return nil, fmt.Errorf("保存模型列表失败: %w", err)
		}
	}
	return &AIProviderModelsResult{Models: models, Count: len(models), RefreshedAt: refreshedAt}, nil
}

func (s *AIProviderService) GetEnabledByID(id *int64) (*model.AIProvider, error) {
	if id == nil || *id <= 0 {
		return nil, nil
	}
	provider, err := s.repo.GetByID(*id)
	if err != nil {
		return nil, err
	}
	if provider == nil || !provider.Enabled {
		return nil, errors.New("所选模型渠道不存在或已停用")
	}
	return provider, nil
}

func providerIDOrFallback(capabilityID, fallbackID *int64) *int64 {
	if capabilityID != nil && *capabilityID > 0 {
		return capabilityID
	}
	return fallbackID
}

func resolveAIProviderIDs(global *model.GlobalSettings, overrideID *int64) (chatID, visionID, imageID *int64) {
	chatID = providerIDOrFallback(global.ChatAIProviderID, global.AIProviderID)
	visionID = providerIDOrFallback(global.ImageRecognitionProviderID, global.AIProviderID)
	imageID = providerIDOrFallback(global.ImageGenerationProviderID, global.AIProviderID)
	if overrideID == nil || *overrideID <= 0 {
		return chatID, visionID, imageID
	}

	// The legacy room/friend channel remains the chat override. Before capability
	// bindings are saved it also keeps its original all-capability behavior.
	chatID = overrideID
	if global.ImageRecognitionProviderID == nil || *global.ImageRecognitionProviderID <= 0 {
		visionID = overrideID
	}
	if global.ImageGenerationProviderID == nil || *global.ImageGenerationProviderID <= 0 {
		imageID = overrideID
	}
	return chatID, visionID, imageID
}

func resolveAIProviderIDsForTarget(
	global *model.GlobalSettings,
	legacyOverrideID, chatOverrideID, visionOverrideID, imageOverrideID *int64,
) (chatID, visionID, imageID *int64) {
	chatID, visionID, imageID = resolveAIProviderIDs(global, legacyOverrideID)
	globalChatID, globalVisionID, globalImageID := resolveAIProviderIDs(global, nil)
	if chatOverrideID != nil {
		chatID = globalChatID
		if *chatOverrideID > 0 {
			chatID = chatOverrideID
		}
	}
	if visionOverrideID != nil {
		visionID = globalVisionID
		if *visionOverrideID > 0 {
			visionID = visionOverrideID
		}
	}
	if imageOverrideID != nil {
		imageID = globalImageID
		if *imageOverrideID > 0 {
			imageID = imageOverrideID
		}
	}
	return chatID, visionID, imageID
}

// ResolveAIProviders returns independent providers for chat, vision and image generation.
// Legacy room/friend selections cover capabilities that have no explicit global binding.
func ResolveAIProviders(ctx context.Context, global *model.GlobalSettings, overrideID *int64) (*model.AIProvider, *model.AIProvider, *model.AIProvider, error) {
	if global == nil {
		return nil, nil, nil, nil
	}
	chatID, visionID, imageID := resolveAIProviderIDs(global, overrideID)
	providerService := NewAIProviderService(ctx)
	chatProvider, err := providerService.GetEnabledByID(chatID)
	if err != nil {
		return nil, nil, nil, err
	}
	visionProvider, err := providerService.GetEnabledByID(visionID)
	if err != nil {
		return nil, nil, nil, err
	}
	imageProvider, err := providerService.GetEnabledByID(imageID)
	if err != nil {
		return nil, nil, nil, err
	}
	return chatProvider, visionProvider, imageProvider, nil
}

// ResolveAIProvidersForTarget applies independent per-target capability overrides.
func ResolveAIProvidersForTarget(
	ctx context.Context,
	global *model.GlobalSettings,
	legacyOverrideID, chatOverrideID, visionOverrideID, imageOverrideID *int64,
) (*model.AIProvider, *model.AIProvider, *model.AIProvider, error) {
	if global == nil {
		return nil, nil, nil, nil
	}
	chatID, visionID, imageID := resolveAIProviderIDsForTarget(
		global, legacyOverrideID, chatOverrideID, visionOverrideID, imageOverrideID,
	)
	providerService := NewAIProviderService(ctx)
	chatProvider, err := providerService.GetEnabledByID(chatID)
	if err != nil {
		return nil, nil, nil, err
	}
	visionProvider, err := providerService.GetEnabledByID(visionID)
	if err != nil {
		return nil, nil, nil, err
	}
	imageProvider, err := providerService.GetEnabledByID(imageID)
	if err != nil {
		return nil, nil, nil, err
	}
	return chatProvider, visionProvider, imageProvider, nil
}

func ApplyAIProvider(config *settings.AIConfig, provider *model.AIProvider) {
	ApplyAIProviders(config, provider, provider, provider)
}

// ApplyAIProviders resolves credentials independently for chat, vision, and image generation.
// A nil capability provider keeps the legacy values already present in AIConfig.
func ApplyAIProviders(config *settings.AIConfig, chatProvider, visionProvider, imageProvider *model.AIProvider) {
	if chatProvider != nil {
		config.BaseURL = utils.NormalizeAIBaseURL(chatProvider.BaseURL)
		config.APIKey = chatProvider.APIKey
		if config.Model == "" {
			config.Model = chatProvider.ChatModel
		}
	}
	if visionProvider != nil {
		config.ImageRecognitionBaseURL = utils.NormalizeAIBaseURL(visionProvider.BaseURL)
		config.ImageRecognitionAPIKey = visionProvider.APIKey
		if config.ImageRecognitionModel == "" {
			config.ImageRecognitionModel = visionProvider.ImageRecognitionModel
		}
	}
	if imageProvider == nil {
		return
	}
	imageSettings := map[string]any{}
	_ = json.Unmarshal(config.ImageAISettings, &imageSettings)
	imageSettings["base_url"] = imageProvider.BaseURL
	imageSettings["api_key"] = imageProvider.APIKey
	if value, _ := imageSettings["model"].(string); strings.TrimSpace(value) == "" {
		imageSettings["model"] = imageProvider.ImageGenerationModel
	}
	if value, _ := imageSettings["size"].(string); strings.TrimSpace(value) == "" {
		imageSettings["size"] = imageProvider.ImageSize
	}
	if value, _ := imageSettings["quality"].(string); strings.TrimSpace(value) == "" {
		imageSettings["quality"] = imageProvider.ImageQuality
	}
	imageConfig, _ := json.Marshal(imageSettings)
	config.ImageAISettings = datatypes.JSON(imageConfig)
}

func (s *AIProviderService) providerFromInput(id int64, input AIProviderInput) (*model.AIProvider, error) {
	name := strings.TrimSpace(input.Name)
	baseURL := strings.TrimRight(strings.TrimSpace(input.BaseURL), "/")
	chatModel := strings.TrimSpace(input.ChatModel)
	if name == "" || baseURL == "" || chatModel == "" {
		return nil, errors.New("渠道名称、Base URL 和对话模型不能为空")
	}
	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		return nil, errors.New("Base URL 必须以 http:// 或 https:// 开头")
	}
	enabled := true
	if input.Enabled != nil {
		enabled = *input.Enabled
	}
	imageSize := strings.TrimSpace(input.ImageSize)
	if imageSize == "" {
		imageSize = "1024x1024"
	}
	provider := &model.AIProvider{
		ID: id, Name: name, BaseURL: baseURL, APIKey: strings.TrimSpace(input.APIKey), ChatModel: chatModel,
		ImageRecognitionModel: strings.TrimSpace(input.ImageRecognitionModel),
		ImageGenerationModel:  strings.TrimSpace(input.ImageGenerationModel), SummaryModel: strings.TrimSpace(input.SummaryModel),
		ImageSize: imageSize, ImageQuality: strings.TrimSpace(input.ImageQuality), Enabled: enabled,
	}
	if input.AvailableModels != nil {
		models := normalizeModelIDs(*input.AvailableModels)
		encodedModels, err := json.Marshal(models)
		if err != nil {
			return nil, fmt.Errorf("保存模型列表失败: %w", err)
		}
		provider.AvailableModels = datatypes.JSON(encodedModels)
		refreshedAt := time.Now()
		provider.ModelsRefreshedAt = &refreshedAt
	}
	return provider, nil
}

func (s *AIProviderService) ensureUniqueName(name string, exceptID int64) error {
	existing, err := s.repo.GetByName(name)
	if err != nil {
		return err
	}
	if existing != nil && existing.ID != exceptID {
		return errors.New("渠道名称已存在")
	}
	return nil
}

func (s *AIProviderService) selectGlobal(provider *model.AIProvider) error {
	updates := map[string]any{
		"ai_provider_id": provider.ID, "chat_base_url": provider.BaseURL, "chat_api_key": provider.APIKey,
	}
	if err := vars.DB.WithContext(s.ctx).Model(&model.GlobalSettings{}).Where("id > 0").Updates(updates).Error; err != nil {
		return err
	}
	newSettings, err := NewGlobalSettingsService(s.ctx).GetGlobalSettings()
	if err == nil && newSettings != nil {
		vars.SettingsObserver.NotifyAll(newSettings)
	}
	return err
}

func (s *AIProviderService) selectedProviderIDs(scope, targetID string) (int64, int64, error) {
	var global model.GlobalSettings
	if err := vars.DB.WithContext(s.ctx).Select("ai_provider_id").First(&global).Error; err != nil {
		return 0, 0, err
	}
	var globalID int64
	if global.AIProviderID != nil {
		globalID = *global.AIProviderID
	}
	if targetID == "" || scope == "" || scope == AIProviderScopeGlobal {
		return globalID, globalID, nil
	}
	var targetProviderID *int64
	switch scope {
	case AIProviderScopeChatRoom:
		var row model.ChatRoomSettings
		if err := vars.DB.WithContext(s.ctx).Select("ai_provider_id").Where("chat_room_id = ?", targetID).First(&row).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return globalID, 0, nil
			}
			return 0, 0, err
		}
		targetProviderID = row.AIProviderID
	case AIProviderScopeFriend:
		var row model.FriendSettings
		if err := vars.DB.WithContext(s.ctx).Select("ai_provider_id").Where("wechat_id = ?", targetID).First(&row).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return globalID, 0, nil
			}
			return 0, 0, err
		}
		targetProviderID = row.AIProviderID
	default:
		return 0, 0, errors.New("scope 仅支持 global、chat_room 或 friend")
	}
	if targetProviderID == nil {
		return globalID, 0, nil
	}
	return globalID, *targetProviderID, nil
}

func providerView(provider *model.AIProvider, globalID, targetID int64) AIProviderView {
	availableModels := make([]string, 0)
	if len(provider.AvailableModels) > 0 {
		_ = json.Unmarshal(provider.AvailableModels, &availableModels)
	}
	return AIProviderView{
		ID: provider.ID, Name: provider.Name, BaseURL: provider.BaseURL, APIKeyMasked: maskAPIKey(provider.APIKey), HasAPIKey: provider.APIKey != "",
		ChatModel: provider.ChatModel, ImageRecognitionModel: provider.ImageRecognitionModel,
		ImageGenerationModel: provider.ImageGenerationModel, SummaryModel: provider.SummaryModel,
		ImageSize: provider.ImageSize, ImageQuality: provider.ImageQuality,
		AvailableModels: normalizeModelIDs(availableModels), ModelsRefreshedAt: provider.ModelsRefreshedAt,
		Enabled:        provider.Enabled,
		GlobalSelected: provider.ID == globalID, TargetSelected: provider.ID == targetID,
		CreatedAt: provider.CreatedAt, UpdatedAt: provider.UpdatedAt,
	}
}

// ValidateAIProviderModel prevents a model cached for one channel from being
// paired with another channel's credentials.
func ValidateAIProviderModel(provider *model.AIProvider, modelName, label string) error {
	if provider == nil {
		return nil
	}
	modelName = strings.TrimSpace(modelName)
	if modelName == "" {
		return nil
	}
	availableModels := make([]string, 0)
	if len(provider.AvailableModels) > 0 {
		_ = json.Unmarshal(provider.AvailableModels, &availableModels)
	}
	availableModels = normalizeModelIDs(append(availableModels,
		provider.ChatModel,
		provider.ImageRecognitionModel,
		provider.ImageGenerationModel,
		provider.SummaryModel,
	))
	for _, availableModel := range availableModels {
		if availableModel == modelName {
			return nil
		}
	}
	return fmt.Errorf("%s模型 %q 不属于渠道 %q，请从该渠道的模型列表中选择", label, modelName, provider.Name)
}

func (s *AIProviderService) FindUniqueEnabledProviderForModel(modelName string) (*model.AIProvider, error) {
	modelName = strings.TrimSpace(modelName)
	if modelName == "" {
		return nil, nil
	}
	providers, err := s.repo.List()
	if err != nil {
		return nil, err
	}
	var matched *model.AIProvider
	for _, provider := range providers {
		if provider == nil || !provider.Enabled || ValidateAIProviderModel(provider, modelName, "") != nil {
			continue
		}
		if matched != nil {
			return nil, fmt.Errorf("模型 %q 同时存在于多个渠道，请明确选择渠道", modelName)
		}
		matched = provider
	}
	return matched, nil
}

func normalizeModelIDs(models []string) []string {
	unique := make(map[string]struct{}, len(models))
	result := make([]string, 0, len(models))
	for _, modelID := range models {
		modelID = strings.TrimSpace(modelID)
		if modelID == "" {
			continue
		}
		if _, exists := unique[modelID]; exists {
			continue
		}
		unique[modelID] = struct{}{}
		result = append(result, modelID)
	}
	sort.Strings(result)
	return result
}

func maskAPIKey(key string) string {
	if len(key) <= 8 {
		if key == "" {
			return ""
		}
		return "********"
	}
	return key[:4] + "..." + key[len(key)-4:]
}
