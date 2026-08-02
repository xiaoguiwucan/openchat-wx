package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/openai/openai-go/v3"
	"github.com/xiaoguiwucan/openchat-wx/interface/settings"
	"github.com/xiaoguiwucan/openchat-wx/model"
	"github.com/xiaoguiwucan/openchat-wx/repository"
	"github.com/xiaoguiwucan/openchat-wx/utils"
	"github.com/xiaoguiwucan/openchat-wx/vars"
	"gorm.io/datatypes"
)

const (
	AIProviderScopeGlobal   = "global"
	AIProviderScopeChatRoom = "chat_room"
	AIProviderScopeFriend   = "friend"
)

type AIProviderInput struct {
	Name                  string `json:"name" binding:"required"`
	BaseURL               string `json:"base_url" binding:"required"`
	APIKey                string `json:"api_key"`
	ChatModel             string `json:"chat_model" binding:"required"`
	ImageRecognitionModel string `json:"image_recognition_model"`
	ImageGenerationModel  string `json:"image_generation_model"`
	SummaryModel          string `json:"summary_model"`
	ImageSize             string `json:"image_size"`
	ImageQuality          string `json:"image_quality"`
	Enabled               *bool  `json:"enabled"`
}

type AIProviderSelection struct {
	ProviderID int64  `json:"provider_id" binding:"required"`
	Scope      string `json:"scope" binding:"required"`
	TargetID   string `json:"target_id"`
}

type AIProviderView struct {
	ID                    int64     `json:"id"`
	Name                  string    `json:"name"`
	BaseURL               string    `json:"base_url"`
	APIKeyMasked          string    `json:"api_key_masked"`
	HasAPIKey             bool      `json:"has_api_key"`
	ChatModel             string    `json:"chat_model"`
	ImageRecognitionModel string    `json:"image_recognition_model"`
	ImageGenerationModel  string    `json:"image_generation_model"`
	SummaryModel          string    `json:"summary_model"`
	ImageSize             string    `json:"image_size"`
	ImageQuality          string    `json:"image_quality"`
	Enabled               bool      `json:"enabled"`
	GlobalSelected        bool      `json:"global_selected"`
	TargetSelected        bool      `json:"target_selected"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

type AIProviderTestResult struct {
	Success   bool   `json:"success"`
	LatencyMS int64  `json:"latency_ms"`
	Model     string `json:"model"`
	Message   string `json:"message"`
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
	for _, table := range []string{"global_settings", "chat_room_settings", "friend_settings"} {
		var count int64
		if err := vars.DB.WithContext(s.ctx).Table(table).Where("ai_provider_id = ?", id).Count(&count).Error; err != nil {
			return err
		}
		references += count
	}
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
		result := vars.DB.WithContext(s.ctx).Model(&model.ChatRoomSettings{}).
			Where("chat_room_id = ?", selection.TargetID).Update("ai_provider_id", provider.ID)
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
		result := vars.DB.WithContext(s.ctx).Model(&model.FriendSettings{}).
			Where("wechat_id = ?", selection.TargetID).Update("ai_provider_id", provider.ID)
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

func ApplyAIProvider(config *settings.AIConfig, provider *model.AIProvider) {
	if provider == nil {
		return
	}
	config.BaseURL = utils.NormalizeAIBaseURL(provider.BaseURL)
	config.APIKey = provider.APIKey
	config.Model = provider.ChatModel
	config.ImageRecognitionModel = provider.ImageRecognitionModel
	imageConfig, _ := json.Marshal(map[string]string{
		"base_url": provider.BaseURL, "api_key": provider.APIKey, "model": provider.ImageGenerationModel,
		"size": provider.ImageSize, "quality": provider.ImageQuality,
	})
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
	return &model.AIProvider{
		ID: id, Name: name, BaseURL: baseURL, APIKey: strings.TrimSpace(input.APIKey), ChatModel: chatModel,
		ImageRecognitionModel: strings.TrimSpace(input.ImageRecognitionModel),
		ImageGenerationModel:  strings.TrimSpace(input.ImageGenerationModel), SummaryModel: strings.TrimSpace(input.SummaryModel),
		ImageSize: imageSize, ImageQuality: strings.TrimSpace(input.ImageQuality), Enabled: enabled,
	}, nil
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
	imageConfig, _ := json.Marshal(map[string]string{
		"base_url": provider.BaseURL, "api_key": provider.APIKey, "model": provider.ImageGenerationModel,
		"size": provider.ImageSize, "quality": provider.ImageQuality,
	})
	updates := map[string]any{
		"ai_provider_id": provider.ID, "chat_base_url": provider.BaseURL, "chat_api_key": provider.APIKey,
		"chat_model": provider.ChatModel, "image_recognition_model": provider.ImageRecognitionModel,
		"image_ai_settings": imageConfig,
	}
	if provider.SummaryModel != "" {
		updates["chat_room_summary_model"] = provider.SummaryModel
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
			return 0, 0, err
		}
		targetProviderID = row.AIProviderID
	case AIProviderScopeFriend:
		var row model.FriendSettings
		if err := vars.DB.WithContext(s.ctx).Select("ai_provider_id").Where("wechat_id = ?", targetID).First(&row).Error; err != nil {
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
	return AIProviderView{
		ID: provider.ID, Name: provider.Name, BaseURL: provider.BaseURL, APIKeyMasked: maskAPIKey(provider.APIKey), HasAPIKey: provider.APIKey != "",
		ChatModel: provider.ChatModel, ImageRecognitionModel: provider.ImageRecognitionModel,
		ImageGenerationModel: provider.ImageGenerationModel, SummaryModel: provider.SummaryModel,
		ImageSize: provider.ImageSize, ImageQuality: provider.ImageQuality, Enabled: provider.Enabled,
		GlobalSelected: provider.ID == globalID, TargetSelected: provider.ID == targetID,
		CreatedAt: provider.CreatedAt, UpdatedAt: provider.UpdatedAt,
	}
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
