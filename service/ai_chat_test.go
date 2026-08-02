package service

import "testing"

func TestChatRequestNeedsTools(t *testing.T) {
	tests := []struct {
		content string
		want    bool
	}{
		{content: "在吗，出来聊两句", want: false},
		{content: "这也太离谱了吧", want: false},
		{content: "帮我搜索一下今天的新闻", want: true},
		{content: "总结一下刚才的聊天记录", want: true},
		{content: "提醒我明天开会", want: true},
	}
	for _, tt := range tests {
		if got := chatRequestNeedsTools(tt.content); got != tt.want {
			t.Fatalf("chatRequestNeedsTools(%q) = %v, want %v", tt.content, got, tt.want)
		}
	}
}
