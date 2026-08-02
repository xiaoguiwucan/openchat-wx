package controller

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/xiaoguiwucan/openchat-wx/pkg/appx"
	"github.com/xiaoguiwucan/openchat-wx/service"
)

type AIProvider struct{}

func NewAIProviderController() *AIProvider {
	return &AIProvider{}
}

func (ct *AIProvider) List(c *gin.Context) {
	resp := appx.NewResponse(c)
	providers, err := service.NewAIProviderService(c).List(c.Query("scope"), c.Query("target_id"))
	if err != nil {
		resp.ToErrorResponse(err)
		return
	}
	resp.ToResponse(providers)
}

func (ct *AIProvider) Create(c *gin.Context) {
	var req service.AIProviderInput
	resp := appx.NewResponse(c)
	if ok, err := appx.BindAndValid(c, &req); !ok || err != nil {
		resp.ToErrorResponse(errors.New("参数错误"))
		return
	}
	provider, err := service.NewAIProviderService(c).Create(req)
	if err != nil {
		resp.ToErrorResponse(err)
		return
	}
	resp.ToResponse(provider)
}

func (ct *AIProvider) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		appx.NewResponse(c).ToErrorResponse(errors.New("渠道 ID 无效"))
		return
	}
	var req service.AIProviderInput
	resp := appx.NewResponse(c)
	if ok, bindErr := appx.BindAndValid(c, &req); !ok || bindErr != nil {
		resp.ToErrorResponse(errors.New("参数错误"))
		return
	}
	provider, err := service.NewAIProviderService(c).Update(id, req)
	if err != nil {
		resp.ToErrorResponse(err)
		return
	}
	resp.ToResponse(provider)
}

func (ct *AIProvider) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	resp := appx.NewResponse(c)
	if err != nil || id <= 0 {
		resp.ToErrorResponse(errors.New("渠道 ID 无效"))
		return
	}
	if err := service.NewAIProviderService(c).Delete(id); err != nil {
		resp.ToErrorResponse(err)
		return
	}
	resp.ToResponse(nil)
}

func (ct *AIProvider) Select(c *gin.Context) {
	var req service.AIProviderSelection
	resp := appx.NewResponse(c)
	if ok, err := appx.BindAndValid(c, &req); !ok || err != nil {
		resp.ToErrorResponse(errors.New("参数错误"))
		return
	}
	if err := service.NewAIProviderService(c).Select(req); err != nil {
		resp.ToErrorResponse(err)
		return
	}
	resp.ToResponse(nil)
}

func (ct *AIProvider) Test(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	resp := appx.NewResponse(c)
	if err != nil || id <= 0 {
		resp.ToErrorResponse(errors.New("渠道 ID 无效"))
		return
	}
	result, err := service.NewAIProviderService(c).Test(id)
	if err != nil {
		resp.ToErrorResponse(err)
		return
	}
	resp.ToResponse(result)
}

func (ct *AIProvider) UI(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.Data(http.StatusOK, "text/html; charset=utf-8", aiProviderHTML)
}
