package plugins

import (
	"log"
	"regexp"
	"strings"

	"github.com/xiaoguiwucan/openchat-wx/interface/plugin"
	"github.com/xiaoguiwucan/openchat-wx/service"
)

var stickerRequestPattern = regexp.MustCompile(`^(?:来|发|整)(?:一个|个|一张|张)?(?:[：:\s]*([^，,。！？!?]{1,16}))?(?:表情包|表情|梗图)[吧呀呗]?[！!。.]?$`)
var stickerBattlePattern = regexp.MustCompile(`^(?:来)?斗图[吧呀呗]?[！!。.]?$`)

func trySendSticker(ctx *plugin.MessageContext) bool {
	if ctx == nil || ctx.Message == nil {
		return false
	}
	content := strings.TrimSpace(ctx.MessageContent)
	content = strings.TrimSpace(strings.TrimPrefix(content, ctx.Settings.GetAITriggerWord()))
	content = strings.TrimSpace(strings.TrimLeft(content, "，,：:"))

	query := ""
	match := stickerRequestPattern.FindStringSubmatch(content)
	if len(match) == 2 {
		query = strings.TrimSpace(match[1])
	} else if !stickerBattlePattern.MatchString(content) {
		return false
	}

	sticker, err := service.NewStickerService(ctx.Context).Pick(ctx.Message.FromWxID, query, ctx.Message.MsgId)
	if err != nil {
		_ = ctx.MessageService.SendTextMessage(ctx.Message.FromWxID, err.Error())
		return true
	}
	if err := ctx.MessageService.SendEmoji(ctx.Message.FromWxID, sticker.MD5, sticker.TotalLen); err != nil {
		log.Printf("[Sticker] send failed: %v", err)
		_ = ctx.MessageService.SendTextMessage(ctx.Message.FromWxID, "表情包发送失败，请稍后重试。")
	}
	return true
}
