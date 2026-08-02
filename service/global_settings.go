package service

import (
	"context"

	"github.com/xiaoguiwucan/openchat-wx/model"
	"github.com/xiaoguiwucan/openchat-wx/repository"
	"github.com/xiaoguiwucan/openchat-wx/vars"
)

type GlobalSettingsService struct {
	ctx    context.Context
	gsRepo *repository.GlobalSettings
}

func (s *GlobalSettingsService) SaveFreeReplySettings(enabled bool, level string, cooldownSeconds, dailyLimit int) error {
	if err := s.gsRepo.DB.WithContext(s.ctx).Model(&model.GlobalSettings{}).Where("id > 0").Updates(map[string]any{
		"free_reply_enabled":          enabled,
		"free_reply_level":            level,
		"free_reply_cooldown_seconds": cooldownSeconds,
		"free_reply_daily_limit":      dailyLimit,
	}).Error; err != nil {
		return err
	}
	newData, err := s.GetGlobalSettings()
	if err != nil {
		return err
	}
	vars.SettingsObserver.NotifyAll(newData)
	return nil
}

func NewGlobalSettingsService(ctx context.Context) *GlobalSettingsService {
	return &GlobalSettingsService{
		ctx:    ctx,
		gsRepo: repository.NewGlobalSettingsRepo(ctx, vars.DB),
	}
}

func (s *GlobalSettingsService) GetGlobalSettings() (*model.GlobalSettings, error) {
	return s.gsRepo.GetGlobalSettings()
}

func (s *GlobalSettingsService) SaveGlobalSettings(data *model.GlobalSettings) error {
	data.FriendSyncCron = "" // 这个不允许用户修改
	err := s.gsRepo.Update(data)
	if err != nil {
		return err
	}
	// 重新读取最新的完整配置，通知所有观察者
	newData, err := s.GetGlobalSettings()
	if err != nil {
		return err
	}
	vars.SettingsObserver.NotifyAll(newData)
	return nil
}
