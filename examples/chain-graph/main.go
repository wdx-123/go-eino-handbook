package main

import (
	"context"
	"fmt"
	"log"

	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

func main() {
	ctx := context.Background()

	graph := compose.NewGraph[map[string]any, []*schema.Message]()

	if err := graph.AddLambdaNode(
		"rewrite_query",
		compose.InvokableLambda(func(ctx context.Context, input map[string]any) (map[string]any, error) {
			role, _ := input["role"].(string)
			query, _ := input["query"].(string)
			if role == "" {
				role = "Eino 助手"
			}
			return map[string]any{
				"role":  role,
				"query": "请帮我总结这段需求：" + query,
			}, nil
		}),
	); err != nil {
		log.Fatalf("add lambda node failed: %v", err)
	}

	if err := graph.AddChatTemplateNode(
		"prompt_node",
		prompt.FromMessages(
			schema.FString,
			schema.SystemMessage("你是一个{role}。"),
			schema.UserMessage("{query}"),
		),
	); err != nil {
		log.Fatalf("add chat template node failed: %v", err)
	}

	if err := graph.AddEdge(compose.START, "rewrite_query"); err != nil {
		log.Fatalf("add start edge failed: %v", err)
	}
	if err := graph.AddEdge("rewrite_query", "prompt_node"); err != nil {
		log.Fatalf("add internal edge failed: %v", err)
	}
	if err := graph.AddEdge("prompt_node", compose.END); err != nil {
		log.Fatalf("add end edge failed: %v", err)
	}

	runnable, err := graph.Compile(ctx)
	if err != nil {
		log.Fatalf("compile graph failed: %v", err)
	}

	out, err := runnable.Invoke(ctx, map[string]any{
		"role":  "专业的 Go AI 架构助手",
		"query": "Chain 和 Graph 的差别是什么？",
	})
	if err != nil {
		log.Fatalf("invoke graph failed: %v", err)
	}

	for i, msg := range out {
		fmt.Printf("[%d] role=%s content=%s\n", i, msg.Role, msg.Content)
	}
}

