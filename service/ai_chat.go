package service

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/openai/openai-go/v3"

	"github.com/xiaoguiwucan/openchat-wx/interface/settings"
	"github.com/xiaoguiwucan/openchat-wx/pkg/aicompat"
	"github.com/xiaoguiwucan/openchat-wx/pkg/robotctx"
	"github.com/xiaoguiwucan/openchat-wx/repository"
	"github.com/xiaoguiwucan/openchat-wx/vars"
)

type AIChatService struct {
	ctx    context.Context
	config settings.Settings
}

var imageURLInContextRegexp = regexp.MustCompile(`图片地址:\s*(https?://\S+)`)

var chatToolRequestPattern = regexp.MustCompile(`(?i)(搜索|搜一下|查一下|查找|查询|联网|网页|链接|总结|归纳|群聊记录|聊天记录|记忆|回忆|刚才说|之前说|谁说|知识库|文档|文件|pdf|word|excel|ppt|提醒|定时|天气|新闻|热搜|小红书|抖音|视频|图片地址|识图|识别图片|截图|发送给|发给)`)

func NewAIChatService(ctx context.Context, config settings.Settings) *AIChatService {
	return &AIChatService{
		ctx:    ctx,
		config: config,
	}
}

func (s *AIChatService) Chat(robotCtx robotctx.RobotContext, aiMessages []openai.ChatCompletionMessageParamUnion) (openai.ChatCompletionMessage, error) {
	// 获取 AI 配置
	aiConfig := s.config.GetAIConfig()

	// 构建系统提示词
	var basePrompt strings.Builder
	basePrompt.WriteString(aiConfig.Prompt)

	// 注入当前世界时间
	now := time.Now()
	weekdayMap := map[time.Weekday]string{
		time.Sunday:    "星期日",
		time.Monday:    "星期一",
		time.Tuesday:   "星期二",
		time.Wednesday: "星期三",
		time.Thursday:  "星期四",
		time.Friday:    "星期五",
		time.Saturday:  "星期六",
	}
	basePrompt.WriteString("\n\n【当前世界时间】\n")
	fmt.Fprintf(&basePrompt, "%d 年 %d 月 %d 日，%s", now.Year(), int(now.Month()), now.Day(), weekdayMap[now.Weekday()])

	if aiConfig.MaxCompletionTokens > 0 {
		fmt.Fprintf(&basePrompt, "\n\n请注意，每次回答不能超过%d个汉字。", aiConfig.MaxCompletionTokens)
	}

	// 构建系统消息
	var systemMessages []openai.ChatCompletionMessageParamUnion
	// 系统提示词
	systemMessages = append(systemMessages, openai.SystemMessage(basePrompt.String()))
	if strings.Contains(robotCtx.FromWxID, "@chatroom") {
		start := time.Now()
		// 群聊上下文：当前用户元信息 + 最近其他群友消息
		if groupCtx := s.buildGroupChatContext(robotCtx.FromWxID, robotCtx.SenderWxID); groupCtx != "" {
			systemMessages = append(systemMessages, openai.SystemMessage(groupCtx))
		}
		log.Printf("[GroupContext] 构建群聊上下文耗时: %v", time.Since(start))
	}
	if imageDescription := s.describeLatestImage(aiConfig, aiMessages); imageDescription != "" {
		systemMessages = append(systemMessages, openai.SystemMessage("【图片/表情识别结果】\n"+imageDescription))
	}
	// 群友单独的对话记录
	aiMessages = append(systemMessages, aiMessages...)

	client := newOpenAIClient(aiConfig.APIKey, aiConfig.BaseURL)
	req := openai.ChatCompletionNewParams{
		Model:    aiConfig.Model,
		Messages: aiMessages,
	}

	aiStart := time.Now()
	useTools := chatRequestNeedsTools(s.latestChatMessageText(aiMessages))
	var reply openai.ChatCompletionMessage
	var err error
	if useTools {
		reply, err = vars.Agent.ChatWithTools(&robotCtx, &client, req)
	} else {
		reply, err = streamChatCompletionMessage(s.ctx, &client, req)
		if err != nil {
			err = fmt.Errorf("failed to call ai: %w", err)
		}
	}
	log.Printf("[AI] 接口调用耗时: %v mode=%s", time.Since(aiStart), map[bool]string{true: "tools", false: "fast"}[useTools])

	return reply, err
}

func chatRequestNeedsTools(content string) bool {
	return chatToolRequestPattern.MatchString(strings.TrimSpace(content))
}

func (s *AIChatService) describeLatestImage(aiConfig settings.AIConfig, messages []openai.ChatCompletionMessageParamUnion) string {
	if aiConfig.ImageRecognitionModel == "" {
		return ""
	}
	for i := len(messages) - 1; i >= 0; i-- {
		matches := imageURLInContextRegexp.FindStringSubmatch(s.chatMessageParamText(messages[i]))
		if len(matches) != 2 {
			continue
		}
		description, err := aicompat.NewClient().DescribeImage(s.ctx, aicompat.Config{
			BaseURL: firstNonEmpty(aiConfig.ImageRecognitionBaseURL, aiConfig.BaseURL),
			APIKey:  firstNonEmpty(aiConfig.ImageRecognitionAPIKey, aiConfig.APIKey),
			Model:   aiConfig.ImageRecognitionModel,
		}, strings.TrimRight(matches[1], "，,。.!！?？"), "")
		if err != nil {
			log.Printf("[Vision] 图片识别失败，继续文本回复: %v", err)
			return ""
		}
		return description
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func (s *AIChatService) latestChatMessageText(messages []openai.ChatCompletionMessageParamUnion) string {
	for i := len(messages) - 1; i >= 0; i-- {
		text := s.chatMessageParamText(messages[i])
		if strings.TrimSpace(text) != "" {
			return text
		}
	}
	return ""
}

func (s *AIChatService) chatMessageParamText(message openai.ChatCompletionMessageParamUnion) string {
	switch content := message.GetContent().AsAny().(type) {
	case *string:
		return *content
	case *[]openai.ChatCompletionContentPartTextParam:
		var builder strings.Builder
		for _, part := range *content {
			builder.WriteString(part.Text)
		}
		return builder.String()
	case *[]openai.ChatCompletionContentPartUnionParam:
		var builder strings.Builder
		for _, part := range *content {
			if text := part.GetText(); text != nil {
				builder.WriteString(*text)
			}
		}
		return builder.String()
	case *[]openai.ChatCompletionAssistantMessageParamContentArrayOfContentPartUnion:
		var builder strings.Builder
		for _, part := range *content {
			if text := part.GetText(); text != nil {
				builder.WriteString(*text)
			}
		}
		return builder.String()
	default:
		return ""
	}
}

// buildGroupChatContext 构建群聊上下文：当前用户元信息 + 最近其他群友消息
func (s *AIChatService) buildGroupChatContext(chatRoomID, senderWxID string) string {
	var sb strings.Builder

	crmRepo := repository.NewChatRoomMemberRepo(s.ctx, vars.DB)
	member, err := crmRepo.GetChatRoomMember(chatRoomID, senderWxID)
	if err != nil {
		log.Printf("[GroupContext] 获取群成员信息失败: %v", err)
	}
	if member != nil {
		// 性别缺失时通过接口获取并回写数据库
		if member.Sex == nil {
			if detailResp, err := vars.RobotRuntime.GetContactDetail("", []string{senderWxID}); err != nil {
				log.Printf("[GroupContext] 获取联系人详情失败: %v", err)
			} else if len(detailResp.ContactList) > 0 {
				sexVal := detailResp.ContactList[0].Sex
				member.Sex = &sexVal
				if err := crmRepo.UpdateMemberInfo(chatRoomID, senderWxID, map[string]any{"sex": sexVal}); err != nil {
					log.Printf("[GroupContext] 回写性别失败: %v", err)
				}
			}
		}

		sb.WriteString("[当前对话用户信息]\n")
		fmt.Fprintf(&sb, "微信 ID: %s\n", member.WechatID)
		if member.Nickname != "" {
			fmt.Fprintf(&sb, "昵称: %s\n", member.Nickname)
		}
		if member.Remark != "" {
			fmt.Fprintf(&sb, "备注: %s\n", member.Remark)
		}
		if member.Sex != nil {
			gender := "未知"
			switch *member.Sex {
			case 1:
				gender = "男"
			case 2:
				gender = "女"
			}
			fmt.Fprintf(&sb, "性别: %s\n", gender)
		}
		if member.Avatar != "" {
			fmt.Fprintf(&sb, "头像: %s\n", member.Avatar)
		}
	}

	msgRepo := repository.NewMessageRepo(s.ctx, vars.DB)
	excludeWxIDs := make([]string, 0, 2)
	if senderWxID != "" {
		excludeWxIDs = append(excludeWxIDs, senderWxID)
	}
	if robotWxID := vars.RobotRuntime.WxID; robotWxID != "" && robotWxID != senderWxID {
		excludeWxIDs = append(excludeWxIDs, robotWxID)
	}
	recentMsgs, err := msgRepo.GetRecentChatRoomMessages(chatRoomID, excludeWxIDs, 10)
	if err != nil {
		log.Printf("[GroupContext] 获取最近群消息失败: %v", err)
	}
	if len(recentMsgs) > 0 {
		if sb.Len() > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString("[最近群聊消息]\n")
		for _, msg := range recentMsgs {
			nickname := msg.SenderNickname
			if nickname == "" {
				nickname = msg.SenderWxID
			}
			fmt.Fprintf(&sb, "[%s]: %s\n", nickname, msg.Content)
		}
	}

	return sb.String()
}
