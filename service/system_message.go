package service

import (
	"context"
	"github.com/xiaoguiwucan/openchat-wx/model"
	"github.com/xiaoguiwucan/openchat-wx/repository"
	"github.com/xiaoguiwucan/openchat-wx/vars"
)

type SystemMessageService struct {
	ctx        context.Context
	sysmsgRepo *repository.SystemMessage
}

func NewSystemMessageService(ctx context.Context) *SystemMessageService {
	return &SystemMessageService{
		ctx:        ctx,
		sysmsgRepo: repository.NewSystemMessageRepo(ctx, vars.DB),
	}
}

func (s *SystemMessageService) GetRecentMonthMessages() ([]*model.SystemMessage, error) {
	return s.sysmsgRepo.GetRecentMonthMessages()
}

func (s *SystemMessageService) MarkAsReadBatch(ids []int64) error {
	return s.sysmsgRepo.MarkAsReadBatch(ids)
}
