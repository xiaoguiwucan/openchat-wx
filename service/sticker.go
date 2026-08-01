package service

import (
	"context"
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/xiaoguiwucan/openchat-wx/pkg/robot"
	"github.com/xiaoguiwucan/openchat-wx/repository"
	"github.com/xiaoguiwucan/openchat-wx/vars"
)

type Sticker struct {
	MD5         string
	TotalLen    int32
	Description string
}

type StickerService struct {
	repo *repository.Message
}

func NewStickerService(ctx context.Context) *StickerService {
	return &StickerService{repo: repository.NewMessageRepo(ctx, vars.DB)}
}

func (s *StickerService) Pick(fromWxID, query string, seed int64) (*Sticker, error) {
	messages, err := s.repo.GetRecentEmoticons(fromWxID, 200)
	if err != nil {
		return nil, err
	}
	all := make([]Sticker, 0, len(messages))
	matched := make([]Sticker, 0, len(messages))
	query = strings.ToLower(strings.TrimSpace(query))
	for _, message := range messages {
		sticker, ok := parseSticker(message.Content)
		if !ok {
			continue
		}
		all = append(all, sticker)
		if query != "" && strings.Contains(strings.ToLower(sticker.Description), query) {
			matched = append(matched, sticker)
		}
	}
	if len(matched) == 0 {
		matched = all
	}
	if len(matched) == 0 {
		return nil, fmt.Errorf("当前会话还没有可复用的表情包")
	}
	if seed < 0 {
		seed = -seed
	}
	selected := matched[int(seed%int64(len(matched)))]
	return &selected, nil
}

func parseSticker(content string) (Sticker, bool) {
	var message robot.XMLEmojiMessage
	if err := xml.Unmarshal([]byte(content), &message); err != nil {
		return Sticker{}, false
	}
	if message.Emoji.MD5 == "" || message.Emoji.Len <= 0 {
		return Sticker{}, false
	}
	description := strings.TrimSpace(strings.Join([]string{message.Emoji.Desc, message.Emoji.AttachedText}, " "))
	return Sticker{
		MD5:         message.Emoji.MD5,
		TotalLen:    int32(message.Emoji.Len),
		Description: description,
	}, true
}
