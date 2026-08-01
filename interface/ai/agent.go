package ai

import (
	"context"

	"github.com/openai/openai-go/v3"

	"github.com/xiaoguiwucan/openchat-wx/pkg/mcp"
	"github.com/xiaoguiwucan/openchat-wx/pkg/robotctx"
	"github.com/xiaoguiwucan/openchat-wx/pkg/skills"
)

type AgentService interface {
	Name() string
	Initialize() error
	Shutdown(ctx context.Context) error
	GetMCPManager() *mcp.MCPManager
	GetSkillsManager() *skills.SkillsManager
	GetAllTools(robotCtx *robotctx.RobotContext) ([]openai.ChatCompletionToolUnionParam, error)
	ChatWithTools(
		robotCtx *robotctx.RobotContext,
		client *openai.Client,
		req openai.ChatCompletionNewParams,
	) (openai.ChatCompletionMessage, error)
}
