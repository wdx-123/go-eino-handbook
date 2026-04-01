package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/cloudwego/eino-ext/components/model/qwen"
	"github.com/cloudwego/eino/schema"
)

func main() {
	ctx := context.Background()

	query := "用一句话解释 Eino 的 Component 设计解决了什么问题？"
	if len(os.Args) > 1 {
		query = strings.Join(os.Args[1:], " ")
	}

	cm, err := qwen.NewChatModel(ctx, &qwen.ChatModelConfig{
		BaseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1",
		APIKey:  mustEnv("DASHSCOPE_API_KEY"),
		Model:   envOrDefault("QWEN_MODEL", "qwen3.5-flash"),
	})
	if err != nil {
		log.Fatalf("new qwen chat model failed: %v", err)
	}

	messages := []*schema.Message{
		schema.SystemMessage("你是一个简洁、专业的 Go AI 框架助手。"),
		schema.UserMessage(query),
	}

	stream, err := cm.Stream(ctx, messages)
	if err != nil {
		log.Fatalf("stream chat failed: %v", err)
	}
	defer stream.Close()

	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("recv stream failed: %v", err)
		}

		fmt.Print(chunk.Content)
	}

	fmt.Println()
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("%s is empty", key)
	}
	return v
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

