package service

import (
	"context"
	"github.com/xiaoguiwucan/openchat-wx/vars"
)

type OfficialAccountService struct {
	ctx context.Context
}

func NewOfficialAccountService(ctx context.Context) *OfficialAccountService {
	return &OfficialAccountService{
		ctx: ctx,
	}
}

func (s *OfficialAccountService) GetAppMsgExt(url string) (string, error) {
	return vars.RobotRuntime.GetAppMsgExt(url)
}
