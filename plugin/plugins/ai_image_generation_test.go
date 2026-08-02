package plugins

import "testing"

func TestExtractImageGenerationPrompt(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		triggerWord string
		want        string
		wantMatch   bool
	}{
		{name: "direct natural command", content: "画个小狗的图", want: "小狗", wantMatch: true},
		{name: "bot nickname without separator", content: "小风画个哈士奇小狗的图", want: "哈士奇小狗", wantMatch: true},
		{name: "wechat mention", content: "@小风\u2005帮我生成一张图片：戴墨镜的柯基", want: "戴墨镜的柯基", wantMatch: true},
		{name: "configured trigger", content: "小风，画一下 海边日落", triggerWord: "小风", want: "海边日落", wantMatch: true},
		{name: "past tense statement", content: "我昨天画了一张小狗的图", wantMatch: false},
		{name: "ordinary conversation", content: "这个小狗图片真好看", wantMatch: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, matched := extractImageGenerationPrompt(tt.content, tt.triggerWord)
			if matched != tt.wantMatch || got != tt.want {
				t.Fatalf("extractImageGenerationPrompt(%q, %q) = (%q, %v), want (%q, %v)", tt.content, tt.triggerWord, got, matched, tt.want, tt.wantMatch)
			}
		})
	}
}
