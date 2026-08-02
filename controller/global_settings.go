package controller

import (
	"encoding/json"
	"errors"
	"slices"

	"github.com/gin-gonic/gin"
	"github.com/xiaoguiwucan/openchat-wx/model"
	"github.com/xiaoguiwucan/openchat-wx/pkg/appx"
	"github.com/xiaoguiwucan/openchat-wx/pkg/utils"
	"github.com/xiaoguiwucan/openchat-wx/service"
)

type FreeReplySettingsRequest struct {
	Enabled         bool   `json:"free_reply_enabled"`
	Level           string `json:"free_reply_level" binding:"required"`
	CooldownSeconds int    `json:"free_reply_cooldown_seconds"`
	DailyLimit      int    `json:"free_reply_daily_limit"`
	ChatAITrigger   string `json:"chat_ai_trigger"`
}

type GlobalSettings struct {
}

func NewGlobalSettingsController() *GlobalSettings {
	return &GlobalSettings{}
}

func (ct *GlobalSettings) GetGlobalSettings(c *gin.Context) {
	resp := appx.NewResponse(c)
	globalSettings, err := service.NewGlobalSettingsService(c).GetGlobalSettings()
	if err != nil {
		resp.ToErrorResponse(err)
		return
	}
	if globalSettings == nil {
		resp.ToErrorResponse(errors.New("获取全局设置失败"))
		return
	}
	resp.ToResponse(globalSettings)
}

func (ct *GlobalSettings) GetFreeReplySettings(c *gin.Context) {
	resp := appx.NewResponse(c)
	globalSettings, err := service.NewGlobalSettingsService(c).GetGlobalSettings()
	if err != nil || globalSettings == nil {
		if err == nil {
			err = errors.New("获取自由回复设置失败")
		}
		resp.ToErrorResponse(err)
		return
	}
	resp.ToResponse(FreeReplySettingsRequest{
		Enabled:         globalSettings.FreeReplyEnabled != nil && *globalSettings.FreeReplyEnabled,
		Level:           globalSettings.FreeReplyLevel,
		CooldownSeconds: intValueOrDefault(globalSettings.FreeReplyCooldownSeconds, 60),
		DailyLimit:      intValueOrDefault(globalSettings.FreeReplyDailyLimit, 30),
		ChatAITrigger:   stringValueOrDefault(globalSettings.ChatAITrigger),
	})
}

func (ct *GlobalSettings) SaveFreeReplySettings(c *gin.Context) {
	var req FreeReplySettingsRequest
	resp := appx.NewResponse(c)
	if ok, err := appx.BindAndValid(c, &req); !ok || err != nil ||
		!slices.Contains([]string{"active", "normal", "cautious", "crazy"}, req.Level) ||
		req.CooldownSeconds < 0 || req.CooldownSeconds > 86400 || req.DailyLimit < 0 || req.DailyLimit > 10000 {
		resp.ToErrorResponse(errors.New("自由回复设置参数错误"))
		return
	}
	if err := service.NewGlobalSettingsService(c).SaveFreeReplySettings(
		req.Enabled, req.Level, req.CooldownSeconds, req.DailyLimit, req.ChatAITrigger,
	); err != nil {
		resp.ToErrorResponse(err)
		return
	}
	resp.ToResponse(nil)
}

func stringValueOrDefault(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func intValueOrDefault(value *int, fallback int) int {
	if value == nil {
		return fallback
	}
	return *value
}

func (ct *GlobalSettings) SaveGlobalSettings(c *gin.Context) {
	var req model.GlobalSettings
	resp := appx.NewResponse(c)
	if ok, err := appx.BindAndValid(c, &req); !ok || err != nil {
		resp.ToErrorResponse(errors.New("参数错误"))
		return
	}
	currentSettings, err := service.NewGlobalSettingsService(c).GetGlobalSettings()
	if err != nil {
		resp.ToErrorResponse(err)
		return
	}
	legacyProviderID := req.AIProviderID
	if (legacyProviderID == nil || *legacyProviderID <= 0) && currentSettings != nil {
		legacyProviderID = currentSettings.AIProviderID
	}
	resolveProvider := func(requested **int64, current *int64) (*model.AIProvider, error) {
		providerID := *requested
		if (providerID == nil || *providerID <= 0) && current != nil && *current > 0 {
			providerID = current
		}
		if providerID == nil || *providerID <= 0 {
			providerID = legacyProviderID
		}
		provider, providerErr := service.NewAIProviderService(c).GetEnabledByID(providerID)
		if providerErr != nil {
			return nil, providerErr
		}
		if provider != nil {
			*requested = providerID
		}
		return provider, nil
	}
	var currentChatProviderID, currentVisionProviderID, currentImageProviderID, currentSummaryProviderID, currentEmbeddingProviderID *int64
	if currentSettings != nil {
		currentChatProviderID = currentSettings.ChatAIProviderID
		currentVisionProviderID = currentSettings.ImageRecognitionProviderID
		currentImageProviderID = currentSettings.ImageGenerationProviderID
		currentSummaryProviderID = currentSettings.SummaryAIProviderID
		currentEmbeddingProviderID = currentSettings.TextEmbeddingProviderID
	}
	if req.ChatAIEnabled != nil && *req.ChatAIEnabled {
		provider, providerErr := resolveProvider(&req.ChatAIProviderID, currentChatProviderID)
		if providerErr != nil {
			resp.ToErrorResponse(providerErr)
			return
		}
		if provider == nil && (req.ChatAPIKey == "" || req.ChatBaseURL == "" || req.ChatModel == "") {
			resp.ToErrorResponse(errors.New("请先为 AI 回复选择一个可用的模型渠道"))
			return
		}
		if providerErr = service.ValidateAIProviderModel(provider, req.ChatModel, "AI回复"); providerErr != nil {
			resp.ToErrorResponse(providerErr)
			return
		}
		visionProvider, providerErr := resolveProvider(&req.ImageRecognitionProviderID, currentVisionProviderID)
		if providerErr != nil {
			resp.ToErrorResponse(providerErr)
			return
		}
		if providerErr = service.ValidateAIProviderModel(visionProvider, req.ImageRecognitionModel, "图像识别"); providerErr != nil {
			resp.ToErrorResponse(providerErr)
			return
		}
		if req.ChatPrompt == "" {
			resp.ToErrorResponse(errors.New("AI人设不能为空"))
			return
		}
	}
	if req.ImageAIEnabled != nil && *req.ImageAIEnabled {
		if req.ImageAISettings == nil {
			resp.ToErrorResponse(errors.New("参数错误"))
			return
		}
		provider, providerErr := resolveProvider(&req.ImageGenerationProviderID, currentImageProviderID)
		if providerErr != nil || provider == nil {
			if providerErr == nil {
				providerErr = errors.New("请先为 AI 绘图选择一个可用的模型渠道")
			}
			resp.ToErrorResponse(providerErr)
			return
		}
		imageSettings := map[string]any{}
		if err := json.Unmarshal(req.ImageAISettings, &imageSettings); err != nil {
			resp.ToErrorResponse(errors.New("绘图参数格式错误"))
			return
		}
		imageModel, _ := imageSettings["model"].(string)
		if providerErr = service.ValidateAIProviderModel(provider, imageModel, "AI绘图"); providerErr != nil {
			resp.ToErrorResponse(providerErr)
			return
		}
	}
	if req.WelcomeEnabled != nil && *req.WelcomeEnabled {
		if req.WelcomeType == "" {
			resp.ToErrorResponse(errors.New("参数错误"))
			return
		}
		if req.WelcomeType == model.WelcomeTypeText {
			if req.WelcomeText == "" {
				resp.ToErrorResponse(errors.New("参数错误"))
				return
			}
		}
		if req.WelcomeType == model.WelcomeTypeEmoji {
			if req.WelcomeEmojiMD5 == "" || req.WelcomeEmojiLen == 0 {
				resp.ToErrorResponse(errors.New("参数错误"))
				return
			}
		}
		if req.WelcomeType == model.WelcomeTypeImage {
			if req.WelcomeImageURL == "" {
				resp.ToErrorResponse(errors.New("参数错误"))
				return
			}
		}
		if req.WelcomeType == model.WelcomeTypeURL {
			if req.WelcomeText == "" || req.WelcomeURL == "" {
				resp.ToErrorResponse(errors.New("参数错误"))
				return
			}
		}
	}
	if req.ChatRoomRankingEnabled != nil && *req.ChatRoomRankingEnabled {
		if req.ChatRoomRankingDailyCron == "" {
			resp.ToErrorResponse(errors.New("参数错误"))
			return
		}
		if req.ChatRoomRankingDailyCron != "" {
			if !utils.IsDailyAtHourMinute(req.ChatRoomRankingDailyCron) {
				resp.ToErrorResponse(errors.New("参数错误"))
				return
			}
		}
		if req.ChatRoomRankingWeeklyCron != nil && *req.ChatRoomRankingWeeklyCron != "" {
			if !utils.IsWeeklyMondayAtHourMinute(*req.ChatRoomRankingWeeklyCron) {
				resp.ToErrorResponse(errors.New("参数错误"))
				return
			}
		}
		if req.ChatRoomRankingMonthCron != nil && *req.ChatRoomRankingMonthCron != "" {
			if !utils.IsMonthly1stAtHourMinute(*req.ChatRoomRankingMonthCron) {
				resp.ToErrorResponse(errors.New("参数错误"))
				return
			}
		}
	}
	if req.ChatRoomSummaryEnabled != nil && *req.ChatRoomSummaryEnabled {
		if req.ChatRoomSummaryModel == "" || req.ChatRoomSummaryCron == "" {
			resp.ToErrorResponse(errors.New("参数错误"))
			return
		}
		if !utils.IsDailyAtHourMinute(req.ChatRoomSummaryCron) {
			resp.ToErrorResponse(errors.New("参数错误"))
			return
		}
		provider, providerErr := resolveProvider(&req.SummaryAIProviderID, currentSummaryProviderID)
		if providerErr != nil || provider == nil {
			if providerErr == nil {
				providerErr = errors.New("请先为群聊总结选择一个可用的模型渠道")
			}
			resp.ToErrorResponse(providerErr)
			return
		}
		if providerErr = service.ValidateAIProviderModel(provider, req.ChatRoomSummaryModel, "群聊总结"); providerErr != nil {
			resp.ToErrorResponse(providerErr)
			return
		}
	}
	if req.MemoryEnabled == nil || *req.MemoryEnabled {
		if req.TextEmbeddingModel != nil && *req.TextEmbeddingModel != "" {
			provider, providerErr := resolveProvider(&req.TextEmbeddingProviderID, currentEmbeddingProviderID)
			if providerErr != nil {
				resp.ToErrorResponse(providerErr)
				return
			}
			if providerErr = service.ValidateAIProviderModel(provider, *req.TextEmbeddingModel, "文本嵌入"); providerErr != nil {
				resp.ToErrorResponse(providerErr)
				return
			}
		}
	}
	if req.NewsEnabled != nil && *req.NewsEnabled {
		if req.NewsType == "" || req.NewsCron == "" {
			resp.ToErrorResponse(errors.New("参数错误"))
			return
		}
		if !utils.IsDailyAtHourMinute(req.NewsCron) {
			resp.ToErrorResponse(errors.New("参数错误"))
			return
		}
	}
	if req.MorningEnabled != nil && *req.MorningEnabled {
		if req.MorningCron == "" {
			resp.ToErrorResponse(errors.New("参数错误"))
			return
		}
		if !utils.IsDailyAtHourMinute(req.MorningCron) {
			resp.ToErrorResponse(errors.New("参数错误"))
			return
		}
	}
	err = service.NewGlobalSettingsService(c).SaveGlobalSettings(&req)
	if err != nil {
		resp.ToErrorResponse(err)
		return
	}
	resp.ToResponse(nil)
}
