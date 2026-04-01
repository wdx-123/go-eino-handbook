package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/deep"
	localbk "github.com/cloudwego/eino-ext/adk/backend/local"
	"github.com/cloudwego/eino-ext/components/model/qwen"
	"github.com/cloudwego/eino/schema"
)

func main() {
	ctx := context.Background()

	projectRoot := envOrDefault("PROJECT_ROOT", ".")
	projectRoot, err := filepath.Abs(projectRoot)
	if err != nil {
		log.Fatalf("resolve project root failed: %v", err)
	}

	cm, err := qwen.NewChatModel(ctx, &qwen.ChatModelConfig{
		BaseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1",
		APIKey:  mustEnv("DASHSCOPE_API_KEY"),
		Model:   envOrDefault("QWEN_MODEL", "qwen3.5-flash"),
	})
	if err != nil {
		log.Fatalf("new qwen chat model failed: %v", err)
	}

	backend, err := localbk.NewBackend(ctx, &localbk.Config{})
	if err != nil {
		log.Fatalf("new local backend failed: %v", err)
	}

	instruction := fmt.Sprintf(`你是一个专业的 Eino 助手。
当你调用文件系统工具时，必须使用绝对路径。
项目根目录是：%s
如果用户说“当前目录”，默认指 %s。`, projectRoot, projectRoot)

	agent, err := deep.New(ctx, &deep.Config{
		Name:           "ToolFilesystemAgent",
		Description:    "A minimal Eino agent with filesystem access.",
		ChatModel:      cm,
		Instruction:    instruction,
		Backend:        backend,
		StreamingShell: backend,
		MaxIteration:   20,
	})
	if err != nil {
		log.Fatalf("new deep agent failed: %v", err)
	}

	query := "请列出当前目录下的 Go 文件，并读取 main.go 的前 20 行"
	if len(os.Args) > 1 {
		query = strings.Join(os.Args[1:], " ")
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming: true,
	})

	events := runner.Run(ctx, []*schema.Message{
		schema.UserMessage(query),
	})

	if err := printEvents(events); err != nil {
		log.Fatalf("run agent failed: %v", err)
	}
}

func printEvents(events *adk.AsyncIterator[*adk.AgentEvent]) error {
	for {
		event, ok := events.Next()
		if !ok {
			return nil
		}
		if event.Err != nil {
			return event.Err
		}
		if event.Output == nil || event.Output.MessageOutput == nil {
			continue
		}

		mv := event.Output.MessageOutput
		if mv.Role == schema.Tool {
			content, err := drainMessageVariant(mv)
			if err != nil {
				return err
			}
			fmt.Printf("[tool result]\n%s\n\n", content)
			continue
		}

		if mv.Role != schema.Assistant && mv.Role != "" {
			continue
		}

		if mv.IsStreaming && mv.MessageStream != nil {
			mv.MessageStream.SetAutomaticClose()
			var toolCalls []schema.ToolCall
			for {
				frame, err := mv.MessageStream.Recv()
				if errors.Is(err, io.EOF) {
					break
				}
				if err != nil {
					return err
				}
				if frame == nil {
					continue
				}
				if frame.Content != "" {
					fmt.Print(frame.Content)
				}
				if len(frame.ToolCalls) > 0 {
					toolCalls = append(toolCalls, frame.ToolCalls...)
				}
			}
			fmt.Println()
			for _, tc := range toolCalls {
				fmt.Printf("[tool call] %s(%s)\n", tc.Function.Name, tc.Function.Arguments)
			}
			continue
		}

		if mv.Message != nil {
			fmt.Println(mv.Message.Content)
		}
	}
}

func drainMessageVariant(mv *adk.MessageVariant) (string, error) {
	if mv.Message != nil {
		return mv.Message.Content, nil
	}
	if !mv.IsStreaming || mv.MessageStream == nil {
		return "", nil
	}

	var sb strings.Builder
	for {
		chunk, err := mv.MessageStream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return "", err
		}
		if chunk != nil && chunk.Content != "" {
			sb.WriteString(chunk.Content)
		}
	}
	return sb.String(), nil
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

