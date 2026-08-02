package plugins

import (
	"bytes"
	"log"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/xiaoguiwucan/openchat-wx/interface/plugin"
	"github.com/xiaoguiwucan/openchat-wx/service"
)

const imageGenerationCommandPattern = `(?:请|麻烦|可以|能不能)?(?:帮我|给我|替我|来)?(?:生成(?:一张|一个|个)?(?:图片|图像|图)?|做(?:一张|一个|个)?(?:图片|图)?|画(?:一张|一下|一个|个|一幅|幅)?|绘图)`

var (
	imageGenerationPattern          = regexp.MustCompile(`^(?:@[^\s\x{2005},，:：]+[\s\x{2005},，:：]*)?` + imageGenerationCommandPattern + `[：:\s\x{2005},，]*(.+)$`)
	addressedImageGenerationPattern = regexp.MustCompile(`^([\p{Han}A-Za-z0-9_-]{2,16}?)[\s\x{2005},，:：]*` + imageGenerationCommandPattern + `[：:\s\x{2005},，]*(.+)$`)
)

func extractImageGenerationPrompt(content, triggerWord string) (string, bool) {
	content = strings.TrimSpace(content)
	if triggerWord = strings.TrimSpace(triggerWord); triggerWord != "" {
		content = strings.TrimSpace(strings.TrimPrefix(content, triggerWord))
	}
	content = strings.TrimSpace(strings.TrimLeft(content, "，,：:\u2005"))

	match := imageGenerationPattern.FindStringSubmatch(content)
	if len(match) != 2 {
		addressedMatch := addressedImageGenerationPattern.FindStringSubmatch(content)
		if len(addressedMatch) != 3 || !isLikelyBotAddress(addressedMatch[1]) {
			return "", false
		}
		match = []string{addressedMatch[0], addressedMatch[2]}
	}

	prompt := strings.TrimSpace(match[1])
	if prompt == "" || strings.HasPrefix(prompt, "了") || strings.HasPrefix(prompt, "过") {
		return "", false
	}
	for _, suffix := range []string{"的图片", "的图像", "的图"} {
		trimmed := strings.TrimSpace(strings.TrimSuffix(prompt, suffix))
		if trimmed != prompt && trimmed != "" {
			prompt = trimmed
			break
		}
	}
	return prompt, true
}

func isLikelyBotAddress(value string) bool {
	value = strings.TrimSpace(value)
	if utf8.RuneCountInString(value) < 2 || utf8.RuneCountInString(value) > 16 {
		return false
	}
	return !containsAny(value, []string{"我", "你", "他", "她", "它", "昨天", "今天", "刚才", "已经", "正在", "曾经", "之前", "之后"})
}

func containsAny(value string, terms []string) bool {
	for _, term := range terms {
		if strings.Contains(value, term) {
			return true
		}
	}
	return false
}

func tryGenerateImage(ctx *plugin.MessageContext) bool {
	if ctx == nil || ctx.Message == nil {
		return false
	}
	prompt, matched := extractImageGenerationPrompt(ctx.MessageContent, ctx.Settings.GetAITriggerWord())
	if !matched {
		return false
	}
	log.Printf("[ImageGeneration] matched msg_id=%d from=%s prompt=%q", ctx.Message.MsgId, ctx.Message.FromWxID, prompt)

	if !ctx.Settings.IsAIDrawingEnabled() {
		_ = ctx.MessageService.SendTextMessage(ctx.Message.FromWxID, "绘图能力尚未启用，请先在机器人 AI 设置中启用并配置绘图模型。")
		return true
	}
	_ = ctx.MessageService.SendTextMessage(ctx.Message.FromWxID, "正在生成图片，请稍候...")

	result, err := service.NewAIImageGenerationService().Generate(ctx.Context, ctx.Settings.GetAIConfig(), prompt)
	if err != nil {
		log.Printf("[ImageGeneration] failed: %v", err)
		_ = ctx.MessageService.SendTextMessage(ctx.Message.FromWxID, imageGenerationFailureMessage(err))
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
	} else {
		log.Printf("[ImageGeneration] sent msg_id=%d to=%s", ctx.Message.MsgId, ctx.Message.FromWxID)
	}
	return true
}

func imageGenerationFailureMessage(err error) string {
	if err == nil {
		return "图片生成失败，请检查绘图模型和渠道状态。"
	}
	message := strings.ToLower(err.Error())
	switch {
	case strings.Contains(message, "unexpected eof"), strings.Contains(message, "connection reset"), strings.Contains(message, "broken pipe"):
		return "图片生成失败：上游在生成过程中断开连接，机器人已自动重试，请稍后再试。"
	case strings.Contains(message, "timeout"), strings.Contains(message, "deadline exceeded"):
		return "图片生成失败：生图渠道响应超时，机器人已自动重试，请稍后再试。"
	default:
		return "图片生成失败，请检查绘图模型、Base URL 和中转站状态。"
	}
}
