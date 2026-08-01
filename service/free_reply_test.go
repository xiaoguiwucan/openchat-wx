package service

import (
	"testing"

	"github.com/xiaoguiwucan/openchat-wx/model"
)

func TestScoreFreeReply(t *testing.T) {
	score, _ := scoreFreeReply("请问大家，这个模型为什么连不上？")
	if score < 6 {
		t.Fatalf("expected a strong group question score, got %d", score)
	}
	if !shouldSuppressFreeReply("https://example.com/test?") {
		t.Fatal("URL payload must be suppressed")
	}
	if shouldSuppressFreeReply("大家知道怎么配置吗？") {
		t.Fatal("normal group question should not be suppressed")
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
