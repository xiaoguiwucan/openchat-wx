package controller

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/xiaoguiwucan/openchat-wx/dto"
	"github.com/xiaoguiwucan/openchat-wx/pkg/appx"
	"github.com/xiaoguiwucan/openchat-wx/pkg/robot"
	"github.com/xiaoguiwucan/openchat-wx/service"
)

type WechatServerCallback struct {
}

func NewWechatServerCallbackController() *WechatServerCallback {
	return &WechatServerCallback{}
}

func (ct *WechatServerCallback) SyncMessageCallback(c *gin.Context) {
	wechatID := c.Param("wechatID")
	var req robot.ClientResponse[robot.SyncMessage]
	resp := appx.NewResponse(c)
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.ToErrorResponse(err)
		return
	}
	log.Printf("Received SyncMessageCallback for wechatID: %s, add_msgs=%d", wechatID, len(req.Data.AddMsgs))
	service.NewLoginService(c).SyncMessageCallback(wechatID, req.Data)

	resp.ToResponse(nil)
}

func (ct *WechatServerCallback) LogoutCallback(c *gin.Context) {
	wechatID := c.Param("wechatID")
	log.Printf("Received LogoutCallback for wechatID: %s", wechatID)
	var req dto.LogoutNotificationRequest
	resp := appx.NewResponse(c)
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("LogoutCallback binding error: %v", err)
		resp.ToErrorResponse(err)
		return
	}
	err := service.NewLoginService(c).LogoutCallback(req)
	if err != nil {
		log.Printf("LogoutCallback failed: %v\n", err)
		resp.ToErrorResponse(err)
		return
	}
	resp.ToResponse(nil)
}
