package repository

import (
	"context"
	"github.com/xiaoguiwucan/openchat-wx/model"

	"gorm.io/gorm"
)

type ChatRoomSettings struct {
	Ctx context.Context
	DB  *gorm.DB
}

func NewChatRoomSettingsRepo(ctx context.Context, db *gorm.DB) *ChatRoomSettings {
	return &ChatRoomSettings{
		Ctx: ctx,
		DB:  db,
	}
}

func (respo *ChatRoomSettings) GetChatRoomSettings(chatRoomID string) (*model.ChatRoomSettings, error) {
	var chatRoomSettings model.ChatRoomSettings
	err := respo.DB.WithContext(respo.Ctx).Where("chat_room_id = ?", chatRoomID).First(&chatRoomSettings).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &chatRoomSettings, nil
}

func (respo *ChatRoomSettings) GetAllEnableGoodMorning() ([]*model.ChatRoomSettings, error) {
	var chatRoomSettings []*model.ChatRoomSettings
	err := respo.DB.WithContext(respo.Ctx).Where("morning_enabled = ?", 1).Find(&chatRoomSettings).Error
	if err != nil {
		return nil, err
	}
	return chatRoomSettings, nil
}

func (respo *ChatRoomSettings) GetAllEnableNews() ([]*model.ChatRoomSettings, error) {
	var chatRoomSettings []*model.ChatRoomSettings
	err := respo.DB.WithContext(respo.Ctx).Where("news_enabled = ?", 1).Find(&chatRoomSettings).Error
	if err != nil {
		return nil, err
	}
	return chatRoomSettings, nil
}

func (respo *ChatRoomSettings) GetAllEnableChatRank() ([]*model.ChatRoomSettings, error) {
	var chatRoomSettings []*model.ChatRoomSettings
	err := respo.DB.WithContext(respo.Ctx).Where("chat_room_ranking_enabled = ?", 1).Find(&chatRoomSettings).Error
	if err != nil {
		return nil, err
	}
	return chatRoomSettings, nil
}

func (respo *ChatRoomSettings) GetAllEnableAISummary() ([]*model.ChatRoomSettings, error) {
	var chatRoomSettings []*model.ChatRoomSettings
	err := respo.DB.WithContext(respo.Ctx).Where("chat_room_summary_enabled = ?", 1).Find(&chatRoomSettings).Error
	if err != nil {
		return nil, err
	}
	return chatRoomSettings, nil
}

func (respo *ChatRoomSettings) DeleteByChatRoomID(chatRoomID string) error {
	return respo.DB.WithContext(respo.Ctx).Where("chat_room_id = ?", chatRoomID).Delete(&model.ChatRoomSettings{}).Error
}

func (respo *ChatRoomSettings) Create(data *model.ChatRoomSettings) error {
	return respo.DB.WithContext(respo.Ctx).Create(data).Error
}

func (respo *ChatRoomSettings) SaveByChatRoomID(data *model.ChatRoomSettings) error {
	return respo.DB.WithContext(respo.Ctx).Transaction(func(tx *gorm.DB) error {
		var existing model.ChatRoomSettings
		err := tx.Select("id").Where("chat_room_id = ?", data.ChatRoomID).First(&existing).Error
		if err == gorm.ErrRecordNotFound {
			data.ID = 0
			return tx.Create(data).Error
		}
		if err != nil {
			return err
		}
		data.ID = existing.ID
		return updateChatRoomSettings(tx, data)
	})
}

func (respo *ChatRoomSettings) Update(data *model.ChatRoomSettings) error {
	return respo.DB.WithContext(respo.Ctx).Transaction(func(tx *gorm.DB) error {
		return updateChatRoomSettings(tx, data)
	})
}

func updateChatRoomSettings(tx *gorm.DB, data *model.ChatRoomSettings) error {
	if err := tx.Where("id = ?", data.ID).Updates(data).Error; err != nil {
		return err
	}
	providerUpdates := map[string]any{}
	for column, providerID := range map[string]*int64{
		"ai_provider_id":                data.AIProviderID,
		"chat_ai_provider_id":           data.ChatAIProviderID,
		"image_recognition_provider_id": data.ImageRecognitionProviderID,
		"image_generation_provider_id":  data.ImageGenerationProviderID,
		"summary_ai_provider_id":        data.SummaryAIProviderID,
	} {
		if providerID != nil {
			providerUpdates[column] = *providerID
		}
	}
	if len(providerUpdates) == 0 {
		return nil
	}
	return tx.Model(&model.ChatRoomSettings{}).Where("id = ?", data.ID).Updates(providerUpdates).Error
}
