package plugins

import "testing"

func TestStickerRequestPatterns(t *testing.T) {
	tests := []struct {
		content string
		query   string
	}{
		{content: "来个表情包", query: ""},
		{content: "发个开心表情包", query: "开心"},
		{content: "整张梗图", query: ""},
	}

	for _, tt := range tests {
		match := stickerRequestPattern.FindStringSubmatch(tt.content)
		if len(match) != 2 || match[1] != tt.query {
			t.Fatalf("content %q produced %#v, want query %q", tt.content, match, tt.query)
		}
	}
	if !stickerBattlePattern.MatchString("斗图") {
		t.Fatal("斗图 should match the sticker battle trigger")
	}
}
