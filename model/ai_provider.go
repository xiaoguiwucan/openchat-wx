package model

import (
	"time"

	"gorm.io/datatypes"
)

// AIProvider stores one OpenAI-compatible model channel and its task models.
type AIProvider struct {
	ID                    int64          `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Name                  string         `gorm:"column:name;type:varchar(100);not null;uniqueIndex:idx_ai_provider_name" json:"name"`
	BaseURL               string         `gorm:"column:base_url;type:varchar(255);not null" json:"base_url"`
	APIKey                string         `gorm:"column:api_key;type:varchar(512);not null" json:"-"`
	ChatModel             string         `gorm:"column:chat_model;type:varchar(100);not null" json:"chat_model"`
	ImageRecognitionModel string         `gorm:"column:image_recognition_model;type:varchar(100);default:''" json:"image_recognition_model"`
	ImageGenerationModel  string         `gorm:"column:image_generation_model;type:varchar(100);default:''" json:"image_generation_model"`
	SummaryModel          string         `gorm:"column:summary_model;type:varchar(100);default:''" json:"summary_model"`
	ImageSize             string         `gorm:"column:image_size;type:varchar(30);default:'1024x1024'" json:"image_size"`
	ImageQuality          string         `gorm:"column:image_quality;type:varchar(30);default:''" json:"image_quality"`
	AvailableModels       datatypes.JSON `gorm:"column:available_models;type:json" json:"-"`
	ModelsRefreshedAt     *time.Time     `gorm:"column:models_refreshed_at" json:"models_refreshed_at"`
	Enabled               bool           `gorm:"column:enabled;not null;default:true;index" json:"enabled"`
	CreatedAt             time.Time      `gorm:"column:created_at" json:"created_at"`
	UpdatedAt             time.Time      `gorm:"column:updated_at" json:"updated_at"`
}

func (AIProvider) TableName() string {
	return "ai_providers"
}
