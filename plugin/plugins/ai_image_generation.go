package plugins

import (
	"bytes"
	"log"
	"regexp"
	"strings"

	"github.com/xiaoguiwucan/openchat-wx/interface/plugin"
	"github.com/xiaoguiwucan/openchat-wx/service"
)

var imageGenerationPattern = regexp.MustCompile(`^(?:请|麻烦|可以)?(?:帮我)?(?:生成一张图片|生成图片|做一张图|画一张|画一下|绘图|画)[：:\s,，]*(.+)$`)

func tryGenerateImage(ctx *plugin.MessageContext) bool {
	if ctx == nil || ctx.Message == nil {
		return false
	}
	content := strings.TrimSpace(ctx.MessageContent)
	content = strings.TrimSpace(strings.TrimPrefix(content, ctx.Settings.GetAITriggerWord()))
	content = strings.TrimSpace(strings.TrimLeft(content, "，,：:"))
	match := imageGenerationPattern.FindStringSubmatch(content)
	if len(match) != 2 || strings.TrimSpace(match[1]) == "" {
		return false
	}

	if !ctx.Settings.IsAIDrawingEnabled() {
		_ = ctx.MessageService.SendTextMessage(ctx.Message.FromWxID, "绘图能力尚未启用，请先在机器人 AI 设置中启用并配置绘图模型。")
		return true
	}

	result, err := service.NewAIImageGenerationService().Generate(ctx.Context, ctx.Settings.GetAIConfig(), match[1])
	if err != nil {
		log.Printf("[ImageGeneration] failed: %v", err)
		_ = ctx.MessageService.SendTextMessage(ctx.Message.FromWxID, "图片生成失败，请检查绘图模型、Base URL 和中转站状态。")
		return true
	}

	if result.URL != "" {
		err = ctx.MessageService.SendImageMessageByRemoteURL(ctx.Message.FromWxID, result.URL)
	} else {
		_, err = ctx.MessageService.MsgUploadImg(ctx.Message.FromWxID, bytes.NewReader(result.Data))
	}
	if err != nil {
		log.Printf("[ImageGeneration] send image failed: %v", err)
		_ = ctx.MessageService.SendTextMessage(ctx.Message.FromWxID, "图片已生成，但发送到微信失败，请稍后重试。")
	}
	return true
}
