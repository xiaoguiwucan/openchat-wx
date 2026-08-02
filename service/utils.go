package service

import (
	"context"
	"fmt"
	"log"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/xiaoguiwucan/openchat-wx/utils"
)

func newOpenAIClient(apiKey, baseURL string) openai.Client {
	return openai.NewClient(
		option.WithAPIKey(apiKey),
		option.WithBaseURL(utils.NormalizeAIBaseURL(baseURL)),
		option.WithHeader("User-Agent", "openchat-wx/1.1"),
	)
}

func streamChatCompletionMessage(ctx context.Context, client *openai.Client, req openai.ChatCompletionNewParams) (openai.ChatCompletionMessage, error) {
	stream := client.Chat.Completions.NewStreaming(ctx, req)
	acc := openai.ChatCompletionAccumulator{}
	for stream.Next() {
		acc.AddChunk(stream.Current())
	}
	if err := stream.Err(); err != nil {
		log.Printf("[AICompat] 流式请求失败，降级为非流式请求: %v", err)
		return nonStreamingChatCompletionMessage(ctx, client, req)
	}
	if len(acc.Choices) == 0 {
		return openai.ChatCompletionMessage{}, fmt.Errorf("empty response")
	}
	return acc.Choices[0].Message, nil
}

func nonStreamingChatCompletionMessage(ctx context.Context, client *openai.Client, req openai.ChatCompletionNewParams) (openai.ChatCompletionMessage, error) {
	completion, err := client.Chat.Completions.New(ctx, req)
	if err != nil {
		return openai.ChatCompletionMessage{}, err
	}
	if len(completion.Choices) == 0 {
		return openai.ChatCompletionMessage{}, fmt.Errorf("empty response")
	}
	return completion.Choices[0].Message, nil
}
