package service

import (
	"log"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/xiaoguiwucan/openchat-wx/model"
	"github.com/xiaoguiwucan/openchat-wx/repository"
	"github.com/xiaoguiwucan/openchat-wx/vars"
)

type freeReplyRuntimeState struct {
	Day                     string
	Count                   int
	LastReply               time.Time
	RepeaterTextTriggeredAt map[string]time.Time
}

var freeReplyRuntime = struct {
	sync.Mutex
	rooms map[string]freeReplyRuntimeState
}{rooms: make(map[string]freeReplyRuntimeState)}

const (
	freeReplyBotNameScore          = 45
	freeReplyGroupQuestionScore    = 30
	freeReplyUnansweredScore       = 25
	freeReplyCapabilityScore       = 25
	freeReplyMemoryScore           = 20
	freeReplyAIOpinionScore        = 35
	freeReplyStickerScore          = 50
	freeReplyRepeaterScore         = 50
	freeReplyRepeaterCooldown      = 30 * time.Minute
	freeReplyRecentContextDuration = 30 * time.Minute
)

var (
	freeReplyWhitespacePattern       = regexp.MustCompile("\\s+")
	freeReplyQuestionPattern         = regexp.MustCompile("(谁能|谁有|有没有人|大家|帮我|帮忙|求|看看|看下|咋办|怎么办|怎么|如何|为啥|为什么|啥意思|什么意思|哪个|哪位|哪里|哪儿|能不能|可不可以|会不会|吗|嘛|呢|？|\\?)")
	freeReplyOpenGroupPattern        = regexp.MustCompile("(大家|各位|群友|谁能|谁有|有没有人|有人|哪位|诸位|大伙|你们)")
	freeReplyOpenQuestionPattern     = regexp.MustCompile("(吗|嘛|呢|么|咋|怎么|如何|为何|为什么|为啥|谁|哪|多少|几|[?？])")
	freeReplyCapabilityPattern       = regexp.MustCompile("(?i)(总结|归纳|方案|记录|上下文|刚才|讨论|记忆|群聊|文档|报告|代码|识图|图片|截图|视频|解析|表情包|梗图|斗图|文件|txt|pdf|word|excel|ppt|链接|网页|搜索|查一下)")
	freeReplyCapabilityTargetPattern = regexp.MustCompile("(帮我|请|麻烦|能不能|可以帮我|谁能|替我).{0,12}(总结|归纳|查|搜索|识图|识别|解析|生成|画|写|翻译|提醒|定时)|(总结一下|归纳一下|查一下|搜索一下|识别一下|解析一下|翻译一下|提醒我|定时提醒|生成一|画一|写一)")
	freeReplyMemoryPattern           = regexp.MustCompile("(记得|群记忆|聊天记录|刚才说|之前|上面|前面|谁说|谁发|谁讲|群里|群友|这个人|他们|她们)")
	freeReplyAIOpinionPattern        = regexp.MustCompile("(?i)(ai怎么看|问问ai)")
	freeReplyStickerPattern          = regexp.MustCompile("(?i)(表情包|表情|梗图|斗图|gif|动图|来张图|来个图|发张图|发个图|整张图|整一个图)")
	freeReplyBanterPattern           = regexp.MustCompile("(?i)(笑死|绷不住|蚌埠住|破防|离谱|抽象|逆天|乐|哈哈哈|hhh|草|卧槽|吐槽|好家伙|典|急了|整活|活了|不愧是|太对了|这也行|烂活|名场面|赢麻了|上强度|电子榨菜|遥遥领先|尊嘟假嘟|栓q|泰裤辣|绝绝子)")
	freeReplyMediaPattern            = regexp.MustCompile("(?i)<(img|emoji|videomsg|appmsg|voicemsg)\\b")
	freeReplyImageFailurePattern     = regexp.MustCompile("(?i)(图片生成失败|图像生成失败|生图失败|绘图密钥|绘图.*密钥|画不了|生成不了图)")
	freeReplyShortQuestionPattern    = regexp.MustCompile("(吗|嘛|呢|么|咋|怎么|如何|为何|为什么|为啥|谁|哪|多少|几|[?？])")
	freeReplyPunctuationPattern      = regexp.MustCompile("[\\s，。！？!?、,.~～…：:；;（）()【】\\[\\]{}<>《》]+")
	freeReplyNonWordPattern          = regexp.MustCompile("[\\p{P}\\p{S}\\s_]+")
	freeReplyFillerPattern           = regexp.MustCompile("(?i)(哈|啊|呀|哦|噢|嗯|额|呃|呵|hi|hello|ok)+")
	freeReplySensitivePattern        = regexp.MustCompile("(?i)\\bapi\\s*key\\b|\\btoken\\b|\\bsecret\\b|file://|[a-z]:\\\\")
)

func (s *ChatRoomSettingsService) shouldFreeReply(content string) bool {
	if s.Message == nil || s.Message.SenderWxID == vars.RobotRuntime.WxID || !s.IsAIChatEnabled() {
		return false
	}
	enabled, level, cooldown, dailyLimit := s.freeReplyConfig()
	if !enabled {
		return false
	}

	content = normalizeFreeReplyText(content)
	threshold := freeReplyThreshold(level)
	if suppression := freeReplyContentSuppression(content); suppression != "" {
		s.logFreeReplySkipped(suppression, 0, threshold, level)
		return false
	}

	recentMessages := s.recentFreeReplyMessages()
	botNames := s.freeReplyBotNames()
	if isLikelyHumanFollowup(s.Message, content, recentMessages, botNames) {
		s.logFreeReplySkipped("likely_human_followup", 0, threshold, level)
		return false
	}

	score, reasons := scoreFreeReplyForLevel(content, level, botNames)
	repeater := isFreeReplyRepeater(content, s.Message.SenderWxID, recentMessages)
	if repeater {
		score += freeReplyRepeaterScore
		reasons = append(reasons, "repeater_message")
	}
	if score < threshold {
		s.logFreeReplySkipped("below_threshold", score, threshold, level)
		return false
	}

	now := time.Now()
	today := now.Format("2006-01-02")
	freeReplyRuntime.Lock()
	defer freeReplyRuntime.Unlock()

	state := freeReplyRuntime.rooms[s.Message.FromWxID]
	if state.Day != today {
		state = freeReplyRuntimeState{
			Day:                     today,
			RepeaterTextTriggeredAt: make(map[string]time.Time),
		}
	}
	if state.RepeaterTextTriggeredAt == nil {
		state.RepeaterTextTriggeredAt = make(map[string]time.Time)
	}
	if repeater {
		lastReply := state.RepeaterTextTriggeredAt[content]
		if !lastReply.IsZero() && now.Sub(lastReply) < freeReplyRepeaterCooldown {
			s.logFreeReplySkipped("repeater_text_cooldown", score, threshold, level)
			return false
		}
	}
	if dailyLimit > 0 && state.Count >= dailyLimit {
		s.logFreeReplySkipped("daily_limit", score, threshold, level)
		return false
	}
	if cooldown > 0 && !state.LastReply.IsZero() && now.Sub(state.LastReply) < time.Duration(cooldown)*time.Second {
		s.logFreeReplySkipped("min_interval", score, threshold, level)
		return false
	}

	state.Count++
	state.LastReply = now
	if repeater {
		state.RepeaterTextTriggeredAt[content] = now
	}
	for text, triggeredAt := range state.RepeaterTextTriggeredAt {
		if now.Sub(triggeredAt) >= freeReplyRepeaterCooldown {
			delete(state.RepeaterTextTriggeredAt, text)
		}
	}
	freeReplyRuntime.rooms[s.Message.FromWxID] = state
	log.Printf("[FreeReply] accepted room=%s score=%d threshold=%d level=%s reasons=%s",
		s.Message.FromWxID, score, threshold, level, strings.Join(reasons, ","))
	return true
}

func (s *ChatRoomSettingsService) logFreeReplySkipped(reason string, score, threshold int, level string) {
	if s.Message == nil {
		return
	}
	log.Printf("[FreeReply] skipped room=%s msg_id=%d reason=%s score=%d threshold=%d level=%s",
		s.Message.FromWxID, s.Message.MsgId, reason, score, threshold, level)
}

func (s *ChatRoomSettingsService) freeReplyConfig() (bool, string, int, int) {
	enabled := false
	level := "normal"
	cooldown := 60
	dailyLimit := 30
	if s.globalSettings != nil {
		if s.globalSettings.FreeReplyEnabled != nil {
			enabled = *s.globalSettings.FreeReplyEnabled
		}
		if s.globalSettings.FreeReplyLevel != "" {
			level = s.globalSettings.FreeReplyLevel
		}
		if s.globalSettings.FreeReplyCooldownSeconds != nil {
			cooldown = *s.globalSettings.FreeReplyCooldownSeconds
		}
		if s.globalSettings.FreeReplyDailyLimit != nil {
			dailyLimit = *s.globalSettings.FreeReplyDailyLimit
		}
	}
	if s.chatRoomSettings != nil {
		if s.chatRoomSettings.FreeReplyEnabled != nil {
			enabled = *s.chatRoomSettings.FreeReplyEnabled
		}
		if s.chatRoomSettings.FreeReplyLevel != nil && *s.chatRoomSettings.FreeReplyLevel != "" {
			level = *s.chatRoomSettings.FreeReplyLevel
		}
		if s.chatRoomSettings.FreeReplyCooldownSeconds != nil {
			cooldown = *s.chatRoomSettings.FreeReplyCooldownSeconds
		}
		if s.chatRoomSettings.FreeReplyDailyLimit != nil {
			dailyLimit = *s.chatRoomSettings.FreeReplyDailyLimit
		}
	}
	level = strings.ToLower(strings.TrimSpace(level))
	if level != "active" && level != "normal" && level != "cautious" && level != "crazy" {
		level = "normal"
	}
	if cooldown < 0 {
		cooldown = 0
	}
	if dailyLimit < 0 {
		dailyLimit = 0
	}
	return enabled, level, cooldown, dailyLimit
}

func shouldSuppressFreeReply(content string) bool {
	return freeReplyContentSuppression(content) != ""
}

func scoreFreeReply(content string) (int, []string) {
	return scoreFreeReplyForLevel(content, "normal", defaultFreeReplyBotNames())
}

func scoreFreeReplyForLevel(content, level string, botNames []string) (int, []string) {
	content = normalizeFreeReplyText(content)
	lower := strings.ToLower(content)
	score := 0
	reasons := make([]string, 0, 8)

	for _, name := range botNames {
		name = strings.TrimSpace(name)
		if name != "" && strings.Contains(lower, strings.ToLower(name)) {
			score += freeReplyBotNameScore
			reasons = append(reasons, "bot_name_match")
			break
		}
	}
	if freeReplyQuestionPattern.MatchString(content) {
		score += freeReplyGroupQuestionScore
		reasons = append(reasons, "group_question")
	}
	if isExplicitOpenGroupQuestion(content) {
		score += freeReplyUnansweredScore
		reasons = append(reasons, "unanswered_question")
	}
	if freeReplyCapabilityPattern.MatchString(content) {
		score += freeReplyCapabilityScore
		reasons = append(reasons, "bot_capability_match")
	}
	if freeReplyMemoryPattern.MatchString(content) {
		score += freeReplyMemoryScore
		reasons = append(reasons, "memory_or_transcript")
	}
	if freeReplyAIOpinionPattern.MatchString(content) {
		score += freeReplyAIOpinionScore
		reasons = append(reasons, "ai_opinion")
	}
	if freeReplyStickerPattern.MatchString(content) {
		score += freeReplyStickerScore
		reasons = append(reasons, "sticker_request")
	}
	if freeReplyBanterPattern.MatchString(content) {
		score += map[string]int{"cautious": 5, "normal": 10, "active": 18, "crazy": 28}[level]
		reasons = append(reasons, "banter_opportunity")
	}
	return score, reasons
}

func freeReplyThreshold(level string) int {
	if threshold := map[string]int{"cautious": 65, "normal": 50, "active": 35, "crazy": 20}[level]; threshold > 0 {
		return threshold
	}
	return 50
}

func normalizeFreeReplyText(content string) string {
	return strings.TrimSpace(freeReplyWhitespacePattern.ReplaceAllString(content, " "))
}

func freeReplyContentSuppression(content string) string {
	content = normalizeFreeReplyText(content)
	if isMediaFreeReply(content) {
		return "media_payload"
	}
	if isLowInformationFreeReply(content) {
		return "low_information"
	}
	if isBotSilentNoticeFreeReply(content) {
		return "bot_silent_notice"
	}
	if isSensitiveFreeReply(content) {
		return "sensitive_or_dangerous"
	}
	if freeReplyImageFailurePattern.MatchString(content) {
		return "image_generation_failure_discussion"
	}
	return ""
}

func isMediaFreeReply(content string) bool {
	lower := strings.ToLower(normalizeFreeReplyText(content))
	return strings.HasPrefix(lower, "<?xml") ||
		strings.HasPrefix(lower, "<msg") ||
		freeReplyMediaPattern.MatchString(content)
}

func isLowInformationFreeReply(content string) bool {
	compact := strings.Join(strings.Fields(content), "")
	if utf8.RuneCountInString(compact) <= 2 {
		return true
	}
	lower := strings.ToLower(compact)
	switch lower {
	case "哈哈", "呵呵", "嗯嗯", "好的", "ok", "hi", "hello":
		return true
	}
	withoutFillers := freeReplyNonWordPattern.ReplaceAllString(lower, "")
	withoutFillers = freeReplyFillerPattern.ReplaceAllString(withoutFillers, "")
	return withoutFillers == "" && utf8.RuneCountInString(compact) <= 8
}

func isBotSilentNoticeFreeReply(content string) bool {
	trimmed := strings.TrimSpace(content)
	if !strings.HasPrefix(trimmed, "（") && !strings.HasPrefix(trimmed, "(") {
		return false
	}
	return containsAny(strings.ToLower(trimmed),
		[]string{"没@我", "不是在问我", "无需ai回复", "无需回复", "不用插嘴", "不插嘴", "保持安静"})
}

func isSensitiveFreeReply(content string) bool {
	lower := strings.ToLower(content)
	return freeReplySensitivePattern.MatchString(lower) ||
		containsAny(lower, []string{"本机", "桌面", "私钥", "密码", "cookie"})
}

func isExplicitOpenGroupQuestion(content string) bool {
	return freeReplyOpenGroupPattern.MatchString(content) && freeReplyOpenQuestionPattern.MatchString(content)
}

func defaultFreeReplyBotNames() []string {
	return []string{"小风", "LightAgent", "机器人", "AI"}
}

func (s *ChatRoomSettingsService) freeReplyBotNames() []string {
	names := defaultFreeReplyBotNames()
	if s.globalSettings != nil && s.globalSettings.ChatAITrigger != nil {
		names = append(names, strings.TrimSpace(*s.globalSettings.ChatAITrigger))
	}
	if s.chatRoomSettings != nil && s.chatRoomSettings.ChatAITrigger != nil {
		names = append(names, strings.TrimSpace(*s.chatRoomSettings.ChatAITrigger))
	}
	return names
}

func (s *ChatRoomSettingsService) recentFreeReplyMessages() []*model.Message {
	if s.Message == nil {
		return nil
	}
	createdAt := s.Message.CreatedAt
	if createdAt <= 0 {
		createdAt = time.Now().Unix()
	}
	messages, err := repository.NewMessageRepo(s.ctx, vars.DB).GetRecentChatRoomMessagesBefore(
		s.Message,
		12,
		createdAt-int64(freeReplyRecentContextDuration/time.Second),
	)
	if err != nil {
		log.Printf("[FreeReply] recent context unavailable room=%s msg_id=%d error=%v",
			s.Message.FromWxID, s.Message.MsgId, err)
		return nil
	}
	return messages
}

func isLikelyHumanFollowup(current *model.Message, content string, recent []*model.Message, botNames []string) bool {
	if current == nil ||
		len(recent) == 0 ||
		!isShortFreeReplyQuestion(content) ||
		isExplicitOpenGroupQuestion(content) ||
		hasExplicitFreeReplyBotTarget(content, botNames) {
		return false
	}
	previous := recent[len(recent)-1]
	if previous == nil ||
		previous.SenderWxID == "" ||
		previous.SenderWxID == current.SenderWxID ||
		previous.SenderWxID == vars.RobotRuntime.WxID ||
		!isMeaningfulFreeReplyStatement(previous.Content) {
		return false
	}
	currentTime := current.CreatedAt
	if currentTime <= 0 {
		currentTime = time.Now().Unix()
	}
	age := currentTime - previous.CreatedAt
	return age >= 0 && age <= 120
}

func isShortFreeReplyQuestion(content string) bool {
	compact := freeReplyPunctuationPattern.ReplaceAllString(strings.TrimSpace(content), "")
	return compact != "" &&
		utf8.RuneCountInString(compact) <= 12 &&
		freeReplyShortQuestionPattern.MatchString(content)
}

func hasExplicitFreeReplyBotTarget(content string, botNames []string) bool {
	lower := strings.ToLower(content)
	for _, name := range botNames {
		name = strings.TrimSpace(name)
		if name != "" && strings.Contains(lower, strings.ToLower(name)) {
			return true
		}
	}
	return freeReplyCapabilityTargetPattern.MatchString(content)
}

func isMeaningfulFreeReplyStatement(content string) bool {
	if isMediaFreeReply(content) {
		return false
	}
	return utf8.RuneCountInString(freeReplyPunctuationPattern.ReplaceAllString(content, "")) >= 2
}

func isFreeReplyRepeater(content, senderWxID string, recent []*model.Message) bool {
	target := normalizeFreeReplyText(content)
	if target == "" {
		return false
	}
	senders := map[string]struct{}{senderWxID: {}}
	for _, message := range recent {
		if message == nil ||
			normalizeFreeReplyText(message.Content) != target ||
			message.SenderWxID == "" {
			continue
		}
		senders[message.SenderWxID] = struct{}{}
		if len(senders) >= 3 {
			return true
		}
	}
	return false
}

func containsAny(content string, terms []string) bool {
	for _, term := range terms {
		if strings.Contains(content, term) {
			return true
		}
	}
	return false
}
