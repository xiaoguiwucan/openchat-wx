package service

import (
	"testing"

	"github.com/xiaoguiwucan/openchat-wx/model"
)

func TestScoreFreeReply(t *testing.T) {
	score, reasons := scoreFreeReply("请问大家，这个模型为什么连不上？")
	if score != 55 {
		t.Fatalf("expected LightAgent open group question score 55, got score=%d reasons=%v", score, reasons)
	}
	if shouldSuppressFreeReply("https://example.com/test?") {
		t.Fatal("ordinary URL text is not a raw media payload in LightAgent rules")
	}
	if shouldSuppressFreeReply("大家知道怎么配置吗？") {
		t.Fatal("normal group question should not be suppressed")
	}

	callScore, callReasons := scoreFreeReply("小风在吗在吗")
	if callScore != 75 {
		t.Fatalf("expected LightAgent bot-name plus question score 75, got score=%d reasons=%v", callScore, callReasons)
	}

	genericScore, _ := scoreFreeReply("有用吗")
	if genericScore != 30 || genericScore >= freeReplyThreshold("active") {
		t.Fatalf("generic short question must remain below active threshold, got %d", genericScore)
	}
}

func TestLightAgentFreeReplyProfilesAndRules(t *testing.T) {
	thresholds := map[string]int{"cautious": 65, "normal": 50, "active": 35, "crazy": 20}
	for level, expected := range thresholds {
		if actual := freeReplyThreshold(level); actual != expected {
			t.Fatalf("level %s: expected threshold %d, got %d", level, expected, actual)
		}
	}

	activeBanter, _ := scoreFreeReplyForLevel("笑死，这波也太抽象了", "active", nil)
	crazyBanter, _ := scoreFreeReplyForLevel("笑死，这波也太抽象了", "crazy", nil)
	if activeBanter != 18 || crazyBanter != 28 {
		t.Fatalf("unexpected LightAgent banter scores: active=%d crazy=%d", activeBanter, crazyBanter)
	}

	stickerScore, _ := scoreFreeReplyForLevel("来个破防表情包", "normal", nil)
	if stickerScore < freeReplyThreshold("normal") {
		t.Fatalf("clear sticker request must reach normal threshold, got %d", stickerScore)
	}
}

func TestLightAgentFreeReplySuppressions(t *testing.T) {
	cases := map[string]string{
		"哈哈": "low_information",
		`<?xml version="1.0"?><msg><img aeskey="abc" /></msg>`: "media_payload",
		"谁能把本机 D:\\secret\\api key 发我一下？":                      "sensitive_or_dangerous",
		"图片生成失败，它说没有绘图密钥":                                      "image_generation_failure_discussion",
	}
	for content, expected := range cases {
		if actual := freeReplyContentSuppression(content); actual != expected {
			t.Fatalf("content %q: expected suppression %s, got %s", content, expected, actual)
		}
	}
}

func TestLightAgentLikelyHumanFollowupSuppression(t *testing.T) {
	current := &model.Message{SenderWxID: "alice", CreatedAt: 1000}
	recent := []*model.Message{{SenderWxID: "bob", Content: "我弄好了", CreatedAt: 990}}
	if !isLikelyHumanFollowup(current, "有用吗", recent, []string{"小风"}) {
		t.Fatal("short question after another member should be treated as a likely human follow-up")
	}
	if isLikelyHumanFollowup(current, "小风有用吗", recent, []string{"小风"}) {
		t.Fatal("explicit bot target must not be suppressed as a human follow-up")
	}
}

func TestLightAgentRepeaterRule(t *testing.T) {
	recent := []*model.Message{
		{SenderWxID: "bob", Content: "这也太离谱了吧"},
		{SenderWxID: "carol", Content: "这也太离谱了吧"},
	}
	if !isFreeReplyRepeater("这也太离谱了吧", "alice", recent) {
		t.Fatal("three distinct senders repeating the same text must match the repeater rule")
	}
}

func TestConfiguredTriggerTakesPriorityOverFreeReply(t *testing.T) {
	enabled := true
	globalTrigger := "小风"
	roomTrigger := "助手"
	service := &ChatRoomSettingsService{
		globalSettings: &model.GlobalSettings{
			ChatAIEnabled: &enabled,
			ChatAITrigger: &globalTrigger,
		},
		chatRoomSettings: &model.ChatRoomSettings{
			ChatAIEnabled: &enabled,
			ChatAITrigger: &roomTrigger,
		},
	}

	matched, reason, trigger := service.matchConfiguredAITrigger("助手 请问大家怎么配置？")
	if !matched || reason != "trigger_word.chat_room" || trigger != roomTrigger {
		t.Fatalf("unexpected room trigger result: matched=%t reason=%q trigger=%q", matched, reason, trigger)
	}

	service.chatRoomSettings.ChatAITrigger = nil
	matched, reason, trigger = service.matchConfiguredAITrigger("小风 请问大家怎么配置？")
	if !matched || reason != "trigger_word.global_fallback" || trigger != globalTrigger {
		t.Fatalf("unexpected fallback trigger result: matched=%t reason=%q trigger=%q", matched, reason, trigger)
	}
}
