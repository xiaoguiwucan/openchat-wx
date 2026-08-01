package service

import "testing"

func TestParseSticker(t *testing.T) {
	sticker, ok := parseSticker(`<msg><emoji md5="abc123" len="456" desc="开心" attachedtext="收到" /></msg>`)
	if !ok {
		t.Fatal("expected sticker XML to be parsed")
	}
	if sticker.MD5 != "abc123" || sticker.TotalLen != 456 || sticker.Description != "开心 收到" {
		t.Fatalf("unexpected sticker: %+v", sticker)
	}
}
