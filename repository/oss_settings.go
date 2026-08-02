package repository

import (
	"context"
	"github.com/xiaoguiwucan/openchat-wx/model"

	"gorm.io/gorm"
)

type OSSSettings struct {
	Ctx context.Context
	DB  *gorm.DB
}

func NewOSSSettingsRepo(ctx context.Context, db *gorm.DB) *OSSSettings {
	return &OSSSettings{
		Ctx: ctx,
		DB:  db,
	}
}

func (respo *OSSSettings) GetOSSSettings() (*model.OSSSettings, error) {
	var ossSettings model.OSSSettings
	result := respo.DB.WithContext(respo.Ctx).Limit(1).Find(&ossSettings)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, nil
	}
	return &ossSettings, nil
}

func (respo *OSSSettings) Create(data *model.OSSSettings) error {
	return respo.DB.WithContext(respo.Ctx).Create(data).Error
}

func (respo *OSSSettings) Update(data *model.OSSSettings) error {
	return respo.DB.WithContext(respo.Ctx).Updates(data).Error
}
