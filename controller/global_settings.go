package controller

import (
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
	})
}

func (ct *GlobalSettings) SaveFreeReplySettings(c *gin.Context) {
	var req FreeReplySettingsRequest
	resp := appx.NewResponse(c)
	if ok, err := appx.BindAndValid(c, &req); !ok || err != nil ||
		!slices.Contains([]string{"active", "normal", "cautious"}, req.Level) ||
		req.CooldownSeconds < 0 || req.CooldownSeconds > 86400 || req.DailyLimit < 0 || req.DailyLimit > 10000 {
		resp.ToErrorResponse(errors.New("自由回复设置参数错误"))
		return
	}
	if err := service.NewGlobalSettingsService(c).SaveFreeReplySettings(
		req.Enabled, req.Level, req.CooldownSeconds, req.DailyLimit,
	); err != nil {
		resp.ToErrorResponse(err)
		return
	}
	resp.ToResponse(nil)
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
	if req.ChatAIEnabled != nil && *req.ChatAIEnabled {
		providerConfigured := false
		if req.AIProviderID != nil && *req.AIProviderID > 0 {
			provider, providerErr := service.NewAIProviderService(c).GetEnabledByID(req.AIProviderID)
			if providerErr != nil {
				resp.ToErrorResponse(providerErr)
				return
			}
			providerConfigured = provider != nil
		}
		if (!providerConfigured && (req.ChatAPIKey == "" || req.ChatBaseURL == "" || req.ChatModel == "" || req.ImageRecognitionModel == "")) || req.ChatPrompt == "" {
			resp.ToErrorResponse(errors.New("参数错误"))
			return
		}
	}
	if req.ImageAIEnabled != nil && *req.ImageAIEnabled {
		if req.ImageAISettings == nil {
			resp.ToErrorResponse(errors.New("参数错误"))
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
	err := service.NewGlobalSettingsService(c).SaveGlobalSettings(&req)
	if err != nil {
		resp.ToErrorResponse(err)
		return
	}
	resp.ToResponse(nil)
}
