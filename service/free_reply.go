package service

import (
	"hash/fnv"
	"log"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/xiaoguiwucan/openchat-wx/vars"
)

type freeReplyRuntimeState struct {
	Day       string
	Count     int
	LastReply time.Time
}

var freeReplyRuntime = struct {
	sync.Mutex
	rooms map[string]freeReplyRuntimeState
}{rooms: make(map[string]freeReplyRuntimeState)}

func (s *ChatRoomSettingsService) shouldFreeReply(content string) bool {
	if s.Message == nil || s.Message.SenderWxID == vars.RobotRuntime.WxID || !s.IsAIChatEnabled() {
		return false
	}
	enabled, level, cooldown, dailyLimit := s.freeReplyConfig()
	if !enabled {
		return false
	}

	content = strings.TrimSpace(content)
	if shouldSuppressFreeReply(content) {
		return false
	}
	score, reasons := scoreFreeReply(content)
	threshold := map[string]int{"active": 2, "normal": 4, "cautious": 6}[level]
	if threshold == 0 {
		threshold = 4
	}
	if score < threshold {
		return false
	}

	// Borderline messages are sampled deterministically to prevent a busy group
	// from receiving a reply to every low-confidence prompt.
	if score == threshold && level != "active" && stablePercent(s.Message.FromWxID+content) >= 65 {
		return false
	}

	now := time.Now()
	today := now.Format("2006-01-02")
	freeReplyRuntime.Lock()
	defer freeReplyRuntime.Unlock()
	state := freeReplyRuntime.rooms[s.Message.FromWxID]
	if state.Day != today {
		state = freeReplyRuntimeState{Day: today}
	}
	if dailyLimit > 0 && state.Count >= dailyLimit {
		return false
	}
	if cooldown > 0 && !state.LastReply.IsZero() && now.Sub(state.LastReply) < time.Duration(cooldown)*time.Second {
		return false
	}
	state.Count++
	state.LastReply = now
	freeReplyRuntime.rooms[s.Message.FromWxID] = state
	log.Printf("[FreeReply] accepted room=%s score=%d threshold=%d level=%s reasons=%s", s.Message.FromWxID, score, threshold, level, strings.Join(reasons, ","))
	return true
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
	if level != "active" && level != "normal" && level != "cautious" {
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
	length := utf8.RuneCountInString(content)
	if length < 3 || length > 240 {
		return true
	}
	lower := strings.ToLower(content)
	return strings.HasPrefix(content, "#") || strings.HasPrefix(content, "/") ||
		strings.HasPrefix(content, "<") || strings.Contains(lower, "<?xml") ||
		strings.Contains(lower, "http://") || strings.Contains(lower, "https://")
}

func scoreFreeReply(content string) (int, []string) {
	score := 0
	reasons := make([]string, 0, 4)
	if strings.ContainsAny(content, "?？") {
		score += 3
		reasons = append(reasons, "question_mark")
	}
	if containsAny(content, []string{"怎么", "为什么", "咋", "哪里", "哪儿", "什么", "啥意思", "能不能", "可不可以", "有没有", "谁知道", "求助", "帮忙", "看看"}) {
		score += 2
		reasons = append(reasons, "question_phrase")
	}
	if containsAny(content, []string{"大家", "有人", "各位", "请问", "求推荐", "建议"}) {
		score += 2
		reasons = append(reasons, "group_invitation")
	}
	if containsAny(content, []string{"哈哈", "笑死", "离谱", "确实", "懂了", "绝了"}) {
		score++
		reasons = append(reasons, "banter")
	}
	length := utf8.RuneCountInString(content)
	if length >= 8 && length <= 100 {
		score++
		reasons = append(reasons, "useful_length")
	}
	return score, reasons
}

func containsAny(content string, terms []string) bool {
	for _, term := range terms {
		if strings.Contains(content, term) {
			return true
		}
	}
	return false
}

func stablePercent(value string) int {
	hash := fnv.New32a()
	_, _ = hash.Write([]byte(value))
	return int(hash.Sum32() % 100)
}
