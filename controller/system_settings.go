package controller

import (
	"errors"
	"github.com/xiaoguiwucan/openchat-wx/model"
	"github.com/xiaoguiwucan/openchat-wx/pkg/appx"
	"github.com/xiaoguiwucan/openchat-wx/service"

	"github.com/gin-gonic/gin"
)

type SystemSettings struct{}

func NewSystemSettingsController() *SystemSettings {
	return &SystemSettings{}
}

func (s *SystemSettings) GetSystemSettings(c *gin.Context) {
	resp := appx.NewResponse(c)
	data, err := service.NewSystemSettingService(c).GetSystemSettings()
	if err != nil {
		resp.ToErrorResponse(err)
		return
	}
	resp.ToResponse(data)
}

func (s *SystemSettings) SaveSystemSettings(c *gin.Context) {
	var req model.SystemSettings
	resp := appx.NewResponse(c)
	if ok, err := appx.BindAndValid(c, &req); !ok || err != nil {
		resp.ToErrorResponse(errors.New("参数错误"))
		return
	}
	err := service.NewSystemSettingService(c).SaveSystemSettings(&req)
	if err != nil {
		resp.ToErrorResponse(err)
		return
	}
	resp.ToResponse(nil)
}
