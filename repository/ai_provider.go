package repository

import (
	"context"
	"time"

	"github.com/xiaoguiwucan/openchat-wx/model"
	"gorm.io/gorm"
)

type AIProvider struct {
	Ctx context.Context
	DB  *gorm.DB
}

func NewAIProviderRepo(ctx context.Context, db *gorm.DB) *AIProvider {
	return &AIProvider{Ctx: ctx, DB: db}
}

func (r *AIProvider) List() ([]*model.AIProvider, error) {
	var providers []*model.AIProvider
	err := r.DB.WithContext(r.Ctx).Order("name ASC, id ASC").Find(&providers).Error
	return providers, err
}

func (r *AIProvider) GetByID(id int64) (*model.AIProvider, error) {
	var provider model.AIProvider
	err := r.DB.WithContext(r.Ctx).First(&provider, id).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &provider, err
}

func (r *AIProvider) GetByName(name string) (*model.AIProvider, error) {
	var provider model.AIProvider
	err := r.DB.WithContext(r.Ctx).Where("name = ?", name).First(&provider).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &provider, err
}

func (r *AIProvider) Create(provider *model.AIProvider) error {
	return r.DB.WithContext(r.Ctx).Create(provider).Error
}

func (r *AIProvider) Update(provider *model.AIProvider) error {
	return r.DB.WithContext(r.Ctx).Model(&model.AIProvider{}).Where("id = ?", provider.ID).Updates(map[string]any{
		"name": provider.Name, "base_url": provider.BaseURL, "api_key": provider.APIKey,
		"chat_model": provider.ChatModel, "image_recognition_model": provider.ImageRecognitionModel,
		"image_generation_model": provider.ImageGenerationModel, "summary_model": provider.SummaryModel,
		"image_size": provider.ImageSize, "image_quality": provider.ImageQuality,
		"available_models": provider.AvailableModels, "models_refreshed_at": provider.ModelsRefreshedAt,
		"enabled": provider.Enabled,
	}).Error
}

func (r *AIProvider) UpdateModelCache(id int64, models []byte, refreshedAt time.Time) error {
	return r.DB.WithContext(r.Ctx).Model(&model.AIProvider{}).Where("id = ?", id).Updates(map[string]any{
		"available_models": models, "models_refreshed_at": refreshedAt,
	}).Error
}

func (r *AIProvider) Delete(id int64) error {
	return r.DB.WithContext(r.Ctx).Delete(&model.AIProvider{}, id).Error
}
