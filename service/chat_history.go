package service

import (
	"context"
	"github.com/xiaoguiwucan/openchat-wx/dto"
	"github.com/xiaoguiwucan/openchat-wx/model"
	"github.com/xiaoguiwucan/openchat-wx/pkg/appx"
	"github.com/xiaoguiwucan/openchat-wx/repository"
	"github.com/xiaoguiwucan/openchat-wx/vars"
)

type ChatHistoryService struct {
	ctx     context.Context
	msgRepo *repository.Message
}

func NewChatHistoryService(ctx context.Context) *ChatHistoryService {
	return &ChatHistoryService{
		ctx:     ctx,
		msgRepo: repository.NewMessageRepo(ctx, vars.DB),
	}
}

func (s *ChatHistoryService) GetChatHistory(req dto.ChatHistoryRequest, pager appx.Pager) ([]*model.Message, int64, error) {
	return s.msgRepo.GetByContactID(req, pager)
}
