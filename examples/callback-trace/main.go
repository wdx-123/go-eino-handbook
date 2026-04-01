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
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/deep"
	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components/model"
	toolcb "github.com/cloudwego/eino/components/tool"
	localbk "github.com/cloudwego/eino-ext/adk/backend/local"
	"github.com/cloudwego/eino-ext/components/model/qwen"
	"github.com/cloudwego/eino/schema"
	ucb "github.com/cloudwego/eino/utils/callbacks"
)

type modelStartKey struct{}
type toolStartKey struct{}

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

	callbacks.AppendGlobalHandlers(buildLocalTraceHandler())

	agent, err := deep.New(ctx, &deep.Config{
		Name:        "CallbackTraceAgent",
		Description: "A minimal Eino agent with callback tracing.",
		ChatModel:   cm,
		Instruction: fmt.Sprintf(`你是一个专业的 Eino 助手。
当你需要访问文件系统时，必须调用工具，并且必须使用绝对路径。
项目根目录是：%s。`, projectRoot),
		Backend:        backend,
		StreamingShell: backend,
		MaxIteration:   20,
	})
	if err != nil {
		log.Fatalf("new deep agent failed: %v", err)
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming: true,
	})

	query := "请先列出当前项目根目录下的 Markdown 文件，再告诉我哪一个最适合继续学习 Eino Callback。"
	if len(os.Args) > 1 {
		query = strings.Join(os.Args[1:], " ")
	}

	answer, err := collectAssistantOutput(runner.Run(ctx, []*schema.Message{schema.UserMessage(query)}))
	if err != nil {
		log.Fatalf("runner query failed: %v", err)
	}

	fmt.Printf("\nassistant> %s\n", answer)
}

func buildLocalTraceHandler() callbacks.Handler {
	modelHandler := &ucb.ModelCallbackHandler{
		OnStart: func(ctx context.Context, info *callbacks.RunInfo, input *model.CallbackInput) context.Context {
			name, typ, component := describeRunInfo(info)
			log.Printf("[model:start] component=%s name=%s type=%s messages=%d tools=%d",
				component, name, typ, len(input.Messages), len(input.Tools))
			return context.WithValue(ctx, modelStartKey{}, time.Now())
		},
		OnEnd: func(ctx context.Context, info *callbacks.RunInfo, output *model.CallbackOutput) context.Context {
			name, typ, component := describeRunInfo(info)
			totalTokens := 0
			if output.TokenUsage != nil {
				totalTokens = output.TokenUsage.TotalTokens
			}

			replyPreview := ""
			if output.Message != nil {
				replyPreview = truncate(output.Message.Content, 80)
			}

			log.Printf("[model:end] component=%s name=%s type=%s duration=%s total_tokens=%d reply=%q",
				component, name, typ, elapsed(ctx, modelStartKey{}), totalTokens, replyPreview)
			return ctx
		},
		OnError: func(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
			name, typ, component := describeRunInfo(info)
			log.Printf("[model:error] component=%s name=%s type=%s err=%v",
				component, name, typ, err)
			return ctx
		},
	}

	toolHandler := &ucb.ToolCallbackHandler{
		OnStart: func(ctx context.Context, info *callbacks.RunInfo, input *toolcb.CallbackInput) context.Context {
			name, _, component := describeRunInfo(info)
			log.Printf("[tool:start] component=%s name=%s args=%s",
				component, name, truncate(input.ArgumentsInJSON, 120))
			return context.WithValue(ctx, toolStartKey{}, time.Now())
		},
		OnEnd: func(ctx context.Context, info *callbacks.RunInfo, output *toolcb.CallbackOutput) context.Context {
			name, _, component := describeRunInfo(info)
			log.Printf("[tool:end] component=%s name=%s duration=%s response=%q",
				component, name, elapsed(ctx, toolStartKey{}), truncate(toolResponsePreview(output), 120))
			return ctx
		},
		OnError: func(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
			name, _, component := describeRunInfo(info)
			log.Printf("[tool:error] component=%s name=%s err=%v",
				component, name, err)
			return ctx
		},
	}

	return ucb.NewHandlerHelper().
		ChatModel(modelHandler).
		Tool(toolHandler).
		Handler()
}

func collectAssistantOutput(events *adk.AsyncIterator[*adk.AgentEvent]) (string, error) {
	var sb strings.Builder

	for {
		event, ok := events.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			return "", event.Err
		}
		if event.Output == nil || event.Output.MessageOutput == nil {
			continue
		}

		mv := event.Output.MessageOutput
		if mv.Role != schema.Assistant && mv.Role != "" {
			continue
		}

		if mv.IsStreaming && mv.MessageStream != nil {
			mv.MessageStream.SetAutomaticClose()
			for {
				frame, err := mv.MessageStream.Recv()
				if errors.Is(err, io.EOF) {
					break
				}
				if err != nil {
					return "", err
				}
				if frame != nil && frame.Content != "" {
					fmt.Print(frame.Content)
					sb.WriteString(frame.Content)
				}
			}
			fmt.Println()
			continue
		}

		if mv.Message != nil {
			fmt.Println(mv.Message.Content)
			sb.WriteString(mv.Message.Content)
		}
	}

	return sb.String(), nil
}

func describeRunInfo(info *callbacks.RunInfo) (string, string, string) {
	if info == nil {
		return "unknown", "unknown", "unknown"
	}

	name := info.Name
	if name == "" {
		name = "unknown"
	}

	typ := info.Type
	if typ == "" {
		typ = "unknown"
	}

	component := info.Component
	componentName := fmt.Sprint(component)
	if componentName == "" {
		componentName = "unknown"
	}

	return name, typ, componentName
}

func elapsed(ctx context.Context, key any) time.Duration {
	v := ctx.Value(key)
	start, ok := v.(time.Time)
	if !ok {
		return 0
	}
	return time.Since(start).Round(time.Millisecond)
}

func truncate(s string, n int) string {
	if len([]rune(s)) <= n {
		return s
	}
	return string([]rune(s)[:n]) + "..."
}

func toolResponsePreview(output *toolcb.CallbackOutput) string {
	if output == nil {
		return ""
	}
	if output.Response != "" {
		return output.Response
	}
	return ""
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
