package startup

import (
	"context"

	"github.com/xiaoguiwucan/openchat-wx/service"
	"github.com/xiaoguiwucan/openchat-wx/vars"
)

func InitAgent() error {
	ctx := context.Background()
	vars.Agent = service.NewAgentService(ctx, vars.DB, vars.KnowledgeService)
	err := vars.Agent.Initialize()
	if err != nil {
		return err
	}
	return nil
}
