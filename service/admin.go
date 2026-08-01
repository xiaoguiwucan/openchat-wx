package service

import (
	"context"
	"github.com/xiaoguiwucan/openchat-wx/model"
	"github.com/xiaoguiwucan/openchat-wx/repository"
	"github.com/xiaoguiwucan/openchat-wx/vars"
)

type AdminService struct {
	ctx            context.Context
	robotAdminRepo *repository.RobotAdmin
}

func NewAdminService(ctx context.Context) *AdminService {
	return &AdminService{
		ctx:            ctx,
		robotAdminRepo: repository.NewRobotAdminRepo(ctx, vars.AdminDB),
	}
}

func (s *AdminService) GetRobotByID(robotID int64) (*model.RobotAdmin, error) {
	return s.robotAdminRepo.GetByRobotID(robotID)
}
