声明：本文数据源于官方文档与官方实现：[[可观测性](https://www.cloudwego.io/zh/docs/eino/quick_start/chapter_06_callback_and_trace/)]
@[TOC](AI 大模型落地系列：为什么你的 Eino Agent 一出问题就像黑盒？一文读懂 Callback、Trace 和生产级可观测性)

如果你前面已经看过了这三篇文章：

- [《AI大模型落地系列：一文读懂 Eino 的 ChatModel 和 Message》](https://blog.csdn.net/2302_80067378/article/details/159393888?spm=1001.2014.3001.5501)
- [《AI大模型落地系列：一文读懂 Eino 的 Tool 和文件系统访问》](https://blog.csdn.net/2302_80067378/article/details/159395909?spm=1001.2014.3001.5501)
- [《AI大模型落地系列：一文读懂 Eino 的 Memory 与 Session（持久化对话）》](https://blog.csdn.net/2302_80067378/article/details/159430416?spm=1001.2014.3001.5501)

那你现在其实已经把 Eino 的三条基础线都摸到了：

- `ChatModel` 让你和模型说上了话
- `Tool` 让 Agent 能碰到外部世界
- `Memory / Session` 让对话状态不再只活在内存里

但真正一进项目，第四个问题往往比前三个更早把人卡住：

> 这次回答为什么慢？
> 到底调了几次模型？
> 是哪个 Tool 卡住了？
> Token 到底花在了哪一段链路上？
> 报错是模型、工具，还是你自己的编排出了问题？

这就是很多 Agent 项目一上强度就开始像“黑盒”的原因。
你只能看到用户输入和最后输出，中间那条调用链几乎是雾里的。

从后端工程角度来看，这不是“调试体验差”这么简单。
它会直接影响你排障、限流、成本分析、稳定性判断，甚至影响你敢不敢把这套东西真的放进生产环境中。

所以本篇博客，我不打算把 `Callback` 当成一个抽象概念来讲。
我想回答一个更有趣的问题：

> 在 Eino 里，到底用什么机制，才能把 Agent 的运行过程从黑盒变的透明可见？

答案就是：`Callback` 负责打观测点，`Trace` 负责把这些观测点组织成一条可读的链路。

## 1. 为什么 Agent 会变成黑盒？

在只写 `ChatModel` Demo 的阶段，事情其实很简单。

你给模型一组 `Message`，它回你一段内容。
出了问题，大不了把请求参数和返回值打印出来。

可一旦你把 `Tool` 接上，这条链路就变了。

模型不再只是“收消息 -> 回答案”，而是会在中间做以下这些动作：

- 先判断要不要调用工具
- 选择具体 Tool
- 组织 Tool 参数
- 等待 Tool 返回结果
- 再把 Tool 结果回灌给模型
- 最后生成给用户的自然语言回答

如果再往前走一步，把[《一文读懂 Eino 的 Tool 和文件系统访问》](https://blog.csdn.net/2302_80067378/article/details/159395909?spm=1001.2014.3001.5501)里的 `DeepAgent` 用起来，再叠上 [《 Eino 的 Memory 与 Session（持久化对话）》](https://blog.csdn.net/2302_80067378/article/details/159430416?spm=1001.2014.3001.5501)里的多轮会话，你的链路会更长：

- 一次用户输入可能触发多次模型调用
- 一次模型调用可能触发多次 Tool
- 不同轮次之间还会夹着历史消息和会话状态

这个时候，只看“最终回答对不对”已经远远不够了。

后端排障真正关心的是另外几件事：

- 这次请求慢，是模型慢，还是 Tool 慢
- 这次没有调到 Tool，是模型判断错了，还是工具注册没生效
- 这次报错出在 `ChatModel`、`Tool`，还是流式消费阶段
- 这次 Token 暴涨，是 prompt 变长了，还是模型在反复推理
- 这次链路和上一次相比，到底多了一步还是少了一步

这些问题，靠“多打几行业务日志”很难真正解决。

因为业务日志通常写在主流程里，而此时需要的，是一种能附着在组件生命周期上的旁路观察能力。
这正是 `Callback` 的位置。

## 2. Callback 到底是什么？
(<small>它不是业务逻辑，而是旁路观察机制</small>)

很多人第一次看到 `Callback`，会下意识把它理解成“拦截器”或者“钩子函数”。

这个理解只对了一半，因为只停在这里，还是太浅了。

在 Eino 里，`Callback` 更准确的定位是：

> 它不是拿来做业务编排的，而是拿来在固定生命周期点上抽取运行信息的。

你可以把它理解成一条“旁路”：

- 主路：`ChatModel`、`Tool`、`Graph`、`Agent` 真正在干活
- 旁路：`Callback` 在关键节点上观察输入、输出、错误和流式数据

这个区别非常重要。

因为从工程职责上看，`Callback` 更适合做以下这些事：

- 打日志
- 采集耗时
- 统计 token
- 上报 trace
- 记录错误链路
- 做调试、审计和指标采集

也就是说，`Callback` 是一种机制，`Trace` 只是它的一种典型用途。

这句话可以单独记住：

> `Callback` 解决“在哪些点能拿到运行信息”，`Trace` 解决“这些信息怎么被串成一条可读的链路”。

所以 CozeLoop、Langfuse、OpenTelemetry 这些东西，本质上都不是 Callback 本身。
它们更像是 Callback 的落地方向。

## 3. Eino 的 5 个触发时机
这个五个触发时期将会很切实的展示，为什么它能做日志、追踪、指标和调试

Eino 把回调点固定在组件生命周期的 5 个时机上。
这也是它为什么适合做可观测性的原因。

| 时机 | 触发点 | 能拿到什么 |
| --- | --- | --- |
| `TimingOnStart` | 组件开始执行前 | 非流式输入 |
| `TimingOnEnd` | 组件成功返回后 | 非流式输出 |
| `TimingOnError` | 组件返回错误时 | `error` |
| `TimingOnStartWithStreamInput` | 组件接收流式输入时 | 流式输入 |
| `TimingOnEndWithStreamOutput` | 组件返回流式输出时 | 流式输出 |

放到最常见的 `ChatModel -> Tool -> ChatModel` 链路里，大概是这样：

```text
用户问题
   |
   v
Runner / Agent
   |
   v
ChatModel.OnStart
   |
   v
模型决定调用 Tool
   |
   v
Tool.OnStart
   |
   v
Tool 执行
   |
   +---- 成功 ----> Tool.OnEnd
   |
   +---- 失败 ----> Tool.OnError
   |
   v
ChatModel 继续推理
   |
   v
ChatModel.OnEnd / OnEndWithStreamOutput
   |
   v
返回给用户
```

这里有三个容易忽略点，但在生产里却又非常关键。

#### 第一，`OnError` 只负责“组件返回错误”
如果你拿到的是一个 `StreamReader`，真正的错误可能发生在消费流的过程中。
这种错误不会自动走到 `OnError`，而是在你读流的时候返回出来。
所以做流式排障时，你不能只盯着 `OnError`。

因为这一点非常绕，所以我举个具体的例子：
##### 1. 非流式场景：
这个很好理解，比如一个 Tool 读文件：

```go
content, err := ReadFile("/tmp/a.txt")
```

如果这里直接报错了：

```cmd
open /tmp/a.txt: no such file or directory
```

那么这属于 **组件执行时就返回 error**。
这种错误，Callback 的 `OnError` 就能收到。

也就是：

* Tool 开始执行 → `OnStart`
* Tool 返回 error → `OnError`

这个没问题。


##### 2. 流式场景：
这个问题，错误可能发生在“读的过程中”

比如模型是流式输出：

```go
stream, err := chatModel.Stream(ctx, input)
```

这一步可能是成功的。也就是说：

* 模型已经成功返回了一个 `StreamReader`
* 所以从“组件调用”角度看，它**没有报错**

但后面你真正开始读流：

```go
for {
    msg, err := stream.Recv()
    ...
}
```

这时才可能发生错误，比如：

* 网络中断
* 上游服务超时
* 流被提前关闭
* 某一帧解析失败

比如读到一半时报：

```text
unexpected EOF
```

这个错误发生在 **消费流的时候**，不是组件一开始返回的时候。
所以它**不一定会自动触发 `OnError`**。

**一个最小例子:**

```go
stream, err := model.Stream(ctx, input)
if err != nil {
    // 这里的 err，通常会对应组件级错误，可能触发 OnError
    return err
}

for {
    chunk, err := stream.Recv()
    if err == io.EOF {
        break
    }
    if err != nil {
        // 这里的 err 是“读流过程中的错误”
        // 不一定自动走到 Callback 的 OnError
        log.Printf("stream recv failed: %v", err)
        return err
    }

    fmt.Print(chunk.Content)
}
```

#### 第二，`RunInfo` 是你判断“现在是谁在执行”的关键元信息。

它通常会带着三类信息：

- `Name`：这次执行的业务名称或节点名
- `Type`：具体实现类型，比如某个模型实现
- `Component`：这是 `ChatModel`、`Tool`，还是别的组件



我这样说，你可能没感觉，我举个例子：没有它，你的全局回调里只能知道：

* “有个东西开始跑了”
* “有个东西结束了”

但你不知道是谁。

那这种日志几乎没法排障。

比如你看到：

```text
[start]
[end]
[start]
[end]
```

没有意义。

但如果加上 `RunInfo`，就变成：

```text
[model:start] component=ChatModel type=Qwen
[tool:start] component=Tool name=glob
[tool:end] component=Tool name=glob duration=18ms
[model:end] component=ChatModel type=Qwen duration=1.3s
```

一下子就能看懂整条链路。




#### 第三，同一个 Handler 可以通过 `context` 传状态。

这意味着你完全可以在 `OnStart` 里记开始时间，在 `OnEnd` 里算耗时。
这比把时钟塞进业务逻辑里干净得多。

## 4. 实战练习：把链路看见

本篇博客，我不再单独造一个“纯 Callback 玩具 Demo”。

那种 Demo 最大的问题是：看起来学会了，实际上你还是感受不到它在真实 Agent 链路里的价值。

所以我直接复用上篇博客中的工具访问语境，做一个最小可运行的版本：

- 模型还是用 Qwen
- Agent 还是用 `DeepAgent`
- 文件系统还是走 `LocalBackend`
- 新增一层本地 Callback 日志
- 可选再接 CozeLoop

这样一跑，你就能同时看到 `ChatModel` 和 `Tool` 两类节点的回调。

### 先准备依赖和环境变量

```bash
go mod init eino-ch06-demo
go get github.com/cloudwego/eino@latest
go get github.com/cloudwego/eino-ext/components/model/qwen@latest
go get github.com/cloudwego/eino-ext/adk/backend/local@latest
go get github.com/cloudwego/eino-ext/callbacks/cozeloop@latest
go get github.com/coze-dev/cozeloop-go@latest

export DASHSCOPE_API_KEY="你的百炼 API Key"
export QWEN_MODEL="qwen3.5-flash"
export PROJECT_ROOT=/path/to/your/project

# 可选：只有想接 CozeLoop 时才需要
export COZELOOP_WORKSPACE_ID="your_workspace_id"
export COZELOOP_API_TOKEN="your_api_token"
```

如果你在 Windows PowerShell 下，写法是：

```powershell
$env:DASHSCOPE_API_KEY="你的百炼 API Key"
$env:QWEN_MODEL="qwen3.5-flash"
$env:PROJECT_ROOT="D:\\your\\project"

# 可选
$env:COZELOOP_WORKSPACE_ID="your_workspace_id"
$env:COZELOOP_API_TOKEN="your_api_token"
```

### 完整demo案例
把下面代码保存成 `main.go`

这里有两个点先说在前面：

- 当前版本里更适合用 `github.com/cloudwego/eino/utils/callbacks` 里的 `HandlerHelper`
- 我们只给 `ChatModel` 和 `Tool` 打观测点，这样最接近真实排障需求

```go
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
	clc "github.com/cloudwego/eino-ext/callbacks/cozeloop"
	"github.com/cloudwego/eino-ext/components/model/qwen"
	"github.com/cloudwego/eino/schema"
	ucb "github.com/cloudwego/eino/utils/callbacks"
	"github.com/coze-dev/cozeloop-go"
)

// 用自定义空 struct 作为 context key，避免和其他包发生 key 冲突。
// 这是一种 Go 里常见且更安全的写法。
type modelStartKey struct{}
type toolStartKey struct{}

func main() {
	ctx := context.Background()

	// PROJECT_ROOT 用于告诉 Agent 当前项目根目录。
	// 如果没配，默认使用当前工作目录。
	projectRoot := envOrDefault("PROJECT_ROOT", ".")
	projectRoot, err := filepath.Abs(projectRoot)
	if err != nil {
		log.Fatalf("resolve project root failed: %v", err)
	}

	// 初始化 Qwen ChatModel。
	// DASHSCOPE_API_KEY 是必填；QWEN_MODEL 可不填，走默认值。
	cm, err := qwen.NewChatModel(ctx, &qwen.ChatModelConfig{
		BaseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1",
		APIKey:  mustEnv("DASHSCOPE_API_KEY"),
		Model:   envOrDefault("QWEN_MODEL", "qwen3.5-flash"),
	})
	if err != nil {
		log.Fatalf("new qwen chat model failed: %v", err)
	}

	// LocalBackend 提供本地文件系统能力，供 DeepAgent 的工具链使用。
	backend, err := localbk.NewBackend(ctx, &localbk.Config{})
	if err != nil {
		log.Fatalf("new local backend failed: %v", err)
	}

	// 注册本地 trace/log Handler。
	// 注意：全局 Handler 应只在进程启动阶段注册一次，不要在每个请求里重复追加。
	callbacks.AppendGlobalHandlers(buildLocalTraceHandler())

	// 可选接入 CozeLoop，把本地 Callback 事件上报为可视化 Trace。
	client, err := setupCozeLoop(ctx)
	if err != nil {
		// CozeLoop 失败不影响主流程运行，只降级为本地日志观测。
		log.Printf("setup cozeloop failed: %v", err)
	}
	if client != nil {
		defer func() {
			// 给异步上报留一点 flush 时间。
			// 真正生产服务里，更建议接入统一的优雅停机流程。
			time.Sleep(5 * time.Second)
			client.Close(ctx)
		}()
	}

	// 构建 DeepAgent。
	// 这里把“访问文件系统时必须用绝对路径”的约束写进 Instruction，
	// 避免模型生成相对路径导致工具执行不稳定。
	agent, err := deep.New(ctx, &deep.Config{
		Name:        "Ch06TraceAgent",
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

	// Runner 负责驱动 Agent 执行。
	// 这里开启流式输出，方便同时演示普通回调与流式消费场景。
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming: true,
	})

	// 默认问题，也支持通过命令行参数覆盖。
	query := "请先列出当前项目根目录下的 Markdown 文件，再告诉我哪一个最适合继续学习 Eino Callback。"
	if len(os.Args) > 1 {
		query = strings.Join(os.Args[1:], " ")
	}

	log.Printf("query=%s", query)

	// 执行查询并消费 AgentEvent，收集最终 assistant 输出。
	answer, err := collectAssistantOutput(runner.Query(ctx, query))
	if err != nil {
		log.Fatalf("runner query failed: %v", err)
	}

	fmt.Printf("\nassistant> %s\n", answer)
}

// buildLocalTraceHandler 构建本地观测 Handler。
// 这里不做业务逻辑，只做日志、耗时、token、响应预览等旁路观测。
// 这是 Callback 在生产中的推荐职责边界。
func buildLocalTraceHandler() callbacks.Handler {
	modelHandler := &ucb.ModelCallbackHandler{
		OnStart: func(ctx context.Context, info *callbacks.RunInfo, input *model.CallbackInput) context.Context {
			name, typ, component := describeRunInfo(info)

			// 记录本轮模型调用的基本输入规模，便于排查：
			// - 消息是否异常膨胀
			// - 工具列表是否如预期注入
			log.Printf("[model:start] component=%s name=%s type=%s messages=%d tools=%d",
				component, name, typ, len(input.Messages), len(input.Tools))

			// 通过 context 在 OnStart -> OnEnd 之间传递开始时间。
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

			// 这里只打预览，不打完整响应，避免日志过大或泄漏过多内容。
			log.Printf("[model:end] component=%s name=%s type=%s duration=%s total_tokens=%d reply=%q",
				component, name, typ, elapsed(ctx, modelStartKey{}), totalTokens, replyPreview)
			return ctx
		},
		OnError: func(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
			name, typ, component := describeRunInfo(info)

			// 注意：这里只能覆盖“组件返回错误”的情况。
			// 如果是流式输出，真正错误也可能发生在后续 Recv() 消费过程中。
			log.Printf("[model:error] component=%s name=%s type=%s err=%v",
				component, name, typ, err)
			return ctx
		},
	}

	toolHandler := &ucb.ToolCallbackHandler{
		OnStart: func(ctx context.Context, info *callbacks.RunInfo, input *toolcb.CallbackInput) context.Context {
			name, _, component := describeRunInfo(info)

			// 记录工具参数预览，有助于排查：
			// - 参数是否组装正确
			// - 模型是否把路径/JSON 拼错
			log.Printf("[tool:start] component=%s name=%s args=%s",
				component, name, truncate(input.ArgumentsInJSON, 120))

			return context.WithValue(ctx, toolStartKey{}, time.Now())
		},
		OnEnd: func(ctx context.Context, info *callbacks.RunInfo, output *toolcb.CallbackOutput) context.Context {
			name, _, component := describeRunInfo(info)

			// 只打印工具结果预览，避免日志过长。
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

	// HandlerHelper 按组件类型注册回调，比手动分发 Component 更清晰。
	// 把“模型回调”和“工具回调”注册进一个统一的 callbacks.Handler 里
	return ucb.NewHandlerHelper().
		ChatModel(modelHandler).
		Tool(toolHandler).
		Handler()
}

// setupCozeLoop 可选初始化 CozeLoop。
// 若未配置环境变量，则直接返回 nil，表示不启用远程 Trace 上报。
func setupCozeLoop(ctx context.Context) (cozeloop.Client, error) {
	apiToken := os.Getenv("COZELOOP_API_TOKEN")
	workspaceID := os.Getenv("COZELOOP_WORKSPACE_ID")
	if apiToken == "" || workspaceID == "" {
		return nil, nil
	}

	client, err := cozeloop.NewClient(
		cozeloop.WithAPIToken(apiToken),
		cozeloop.WithWorkspaceID(workspaceID),
	)
	if err != nil {
		return nil, err
	}

	// 追加 CozeLoop Handler，把本地 Callback 事件同步上报。
	callbacks.AppendGlobalHandlers(clc.NewLoopHandler(client))
	log.Println("cozeloop tracing enabled")

	return client, nil
}

// collectAssistantOutput 统一消费 Agent 事件流，兼容普通输出与流式输出。
// 返回最终拼接后的 assistant 文本。
func collectAssistantOutput(events *adk.AsyncIterator[*adk.AgentEvent]) (string, error) {
	var sb strings.Builder

	for {
		event, ok := events.Next()
		if !ok {
			break
		}

		// 这里处理的是 AgentEvent 层面的错误。
		// 这和 Callback 的 OnError 不是一回事，二者都要关注。
		if event.Err != nil {
			return "", event.Err
		}

		if event.Output == nil || event.Output.MessageOutput == nil {
			continue
		}

		mv := event.Output.MessageOutput

		// 只消费 assistant 输出，跳过其他角色事件。
		if mv.Role != schema.Assistant && mv.Role != "" {
			continue
		}

		if mv.IsStreaming {
			// 自动关闭底层流，避免消费完成后资源泄漏。
			mv.MessageStream.SetAutomaticClose()

			for {
				frame, err := mv.MessageStream.Recv()
				if errors.Is(err, io.EOF) {
					break
				}
				if err != nil {
					// 流式输出的真实错误可能发生在 Recv() 阶段，
					// 而不是模型组件初始化流时。
					return "", err
				}
				if frame != nil && frame.Content != "" {
					fmt.Print(frame.Content)
					sb.WriteString(frame.Content)
				}
			}
			continue
		}

		// 非流式输出直接收集完整消息。
		if mv.Message != nil {
			fmt.Print(mv.Message.Content)
			sb.WriteString(mv.Message.Content)
		}
	}

	return sb.String(), nil
}

// describeRunInfo 对 RunInfo 做容错包装，避免日志里出现大量空值。
// 在生产里，RunInfo 不应被默认假设为一定完整。
func describeRunInfo(info *callbacks.RunInfo) (name, typ, component string) {
	if info == nil {
		return "unknown", "unknown", "unknown"
	}

	name = strings.TrimSpace(info.Name)
	if name == "" {
		name = "unnamed"
	}

	typ = strings.TrimSpace(info.Type)
	if typ == "" {
		typ = "unknown"
	}

	component = fmt.Sprintf("%v", info.Component)
	if component == "" {
		component = "unknown"
	}

	return name, typ, component
}

// elapsed 从 context 中取出开始时间并计算耗时。
// 如果 key 不存在，返回 0，避免因为观测逻辑影响主流程。
func elapsed(ctx context.Context, key any) time.Duration {
	start, ok := ctx.Value(key).(time.Time)
	if !ok {
		return 0
	}
	return time.Since(start).Round(time.Millisecond)
}

// toolResponsePreview 尽量抽取适合打印到日志里的工具结果预览。
// 不保证结果结构固定，因此按“Response -> ToolOutput -> 空串”顺序兜底。
func toolResponsePreview(output *toolcb.CallbackOutput) string {
	if output == nil {
		return ""
	}
	if output.Response != "" {
		return output.Response
	}
	if output.ToolOutput != nil {
		return fmt.Sprintf("%v", output.ToolOutput)
	}
	return ""
}

// truncate 用于限制日志体积，避免长文本直接打满终端或日志系统。
func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "..."
}

// mustEnv 读取必需环境变量；缺失时直接终止进程。
// 适合 API Key 这类“程序无法降级运行”的配置。
func mustEnv(key string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		log.Fatalf("%s is empty", key)
	}
	return v
}

// envOrDefault 读取可选环境变量；未配置则使用默认值。
func envOrDefault(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

```

### 直接运行

```bash
go run . -- "请列出当前项目根目录下的 Markdown 文件，再告诉我哪一篇最值得先看"
```

如果你没有配置 CozeLoop，这段代码依然能跑。
你会先在终端里看到一层本地日志。

输出大概会长这样：

```text
2026/03/24 21:08:41 query=请列出当前项目根目录下的 Markdown 文件，再告诉我哪一篇最值得先看
2026/03/24 21:08:41 [model:start] component=ChatModel name=unnamed type=Qwen messages=2 tools=8
2026/03/24 21:08:42 [tool:start] component=Tool name=glob args={"pattern":"D:\\workspace_go\\personal_assistant\\docs\\csdn\\go eino\\*.md"}
2026/03/24 21:08:42 [tool:end] component=Tool name=glob duration=18ms response="[D:\\workspace_go\\personal_assistant\\docs\\csdn\\go eino\\A学习总纲.md ...]"
2026/03/24 21:08:43 [model:end] component=ChatModel name=unnamed type=Qwen duration=1.324s total_tokens=286 reply="如果你要继续学 Callback，建议先看 A学习总纲，再结合 Tool 和 Memory 两篇..."
assistant> 你可以先看 A学习总纲，再回到 Tool 和 Memory 两篇文章，因为 Callback 的价值只有放到完整链路里才会明显。
```

这段输出最关键的，不是“日志变多了”，而是你终于能回答这些问题了：

- 这次有没有真的调到 Tool
- 调的是哪个 Tool
- Tool 参数长什么样
- Tool 花了多久
- 模型这一轮用了多少 token
- 最终回答是不是建立在工具返回结果之上

从工程视角看，这就是从黑盒走向透明的第一步。

### 总结为三点

**第一，用 `HandlerHelper` 按组件类型拆观测点。**

如果你直接用通用 Handler，很多时候得自己 `switch RunInfo.Component`，再手动把 `CallbackInput` 转成具体类型。
`HandlerHelper` 已经把这层胶水收掉了。

**第二，用 `context` 在 `OnStart -> OnEnd` 之间传状态。**

这里我传的是开始时间，所以 `OnEnd` 能直接算出耗时。
同一个模式也能用来透传 trace id、采样标记，或者一些只属于当前回调链路的上下文数据。

**第三，日志要面向排障，而不是面向“证明程序跑过”。**

所以我打的不是“开始了”“结束了”这种空日志，而是这些真正有判断价值的信息：

- 组件类型
- 组件名称
- 输入规模
- Tool 参数
- 响应预览
- 耗时
- token 数

这类信息，到了线上你就会感受到真正的价值。

## 5. 再接 CozeLoop，把日志升级成 Trace

如果说上一节解决的是“我能看见每个点”，那这一节解决的就是：

> 我能不能把这些点串成一条真正可追踪的链路？

这就是 `Trace` 的价值。

在 Eino 里，CozeLoop 的接法其实不复杂。
之前我在运行代码中，依旧有了这一段：

```go

// setupCozeLoop 可选初始化 CozeLoop。
// 若未配置环境变量，则直接返回 nil，表示不启用远程 Trace 上报。
func setupCozeLoop(ctx context.Context) (cozeloop.Client, error) {
	apiToken := os.Getenv("COZELOOP_API_TOKEN")
	workspaceID := os.Getenv("COZELOOP_WORKSPACE_ID")
	if apiToken == "" || workspaceID == "" {
		return nil, nil
	}

	client, err := cozeloop.NewClient(
		cozeloop.WithAPIToken(apiToken),
		cozeloop.WithWorkspaceID(workspaceID),
	)
	if err != nil {
		return nil, err
	}

	// 追加 CozeLoop Handler，把本地 Callback 事件同步上报。
	callbacks.AppendGlobalHandlers(clc.NewLoopHandler(client))
	log.Println("cozeloop tracing enabled")

	return client, nil
}
```

注意这里：

- 没有改 Agent 主流程
- 没有改 Tool 逻辑
- 没有改消息结构
- 只是多注册了一个全局 Handler

这也是我前面一直强调“Callback 是旁路机制”的原因。
你的业务代码并不会因为你接了 Trace 而变成另一套写法。

一旦 CozeLoop 配好，你就会从“能看终端日志”升级到“能看完整链路”。
它更适合解决这些问题：

- 哪个节点最慢
- 哪一轮模型 token 最高
- 哪个 Tool 的失败率高
- 某次异常链路到底是怎么走出来的

还有一个很现实的工程细节要补一句：
`AppendGlobalHandlers` 应该只在服务启动期注册一次，它不是给你在请求中间动态追加用的。


## 6. 六大细节


1. `RunInfo` 可能为 `nil`。  
   顶层调用或者某些独立组件场景下，元信息不一定完整，所以 Handler 里先做 nil-check 是基本动作。

2. 不要修改 Callback 的输入输出对象。  
   这些对象会被下游节点和其他 Handler 共享，直接改内容很容易引入竞态和脏数据。

3. 流式输出要关注 `OnEndWithStreamOutput`。  
   如果你用的是流式模型或流式工具，真正的错误和观测点可能发生在读流阶段，而不是普通的 `OnEnd` / `OnError`。

4. `StreamReader` 必须关闭，否则可能泄漏 goroutine。  
   只要你注册了流式回调，拿到的就是一份私有流副本。你读完后不关闭，这条链路就可能一直挂着。

5. 同一 Handler 可以通过 `context` 传状态，不同 Handler 不要依赖执行顺序。  
   `OnStart -> OnEnd` 之间传耗时、trace id 没问题，但你不能假设“先执行 A Handler，再执行 B Handler”。

6. Callback 是旁路，不要把业务逻辑塞进去。  
   它适合做日志、指标、追踪和调试，不适合做权限、补偿、持久化写入这类主业务动作。

你会发现，这 6 条看起来都不复杂。
但一旦踩中，后果通常都不是“日志少打了一行”，而是定位混乱、内存泄漏，甚至把回调链本身写成新的不稳定源。

## 7. 一分钟复盘

看完本篇文章，你应该有了以下的想法：

- `Callback` 不是让 Agent 更聪明的能力，而是让你更看得见它的能力
- `Callback` 是机制，`Trace` 是它的典型落地方式
- Eino 把观测点固定在 5 个生命周期时机上，所以它很适合做日志、指标和追踪
- 本地日志适合第一层排障，CozeLoop 适合把点串成链路
- 真正上线前，最该小心的是流式回调、共享输入输出、Handler 顺序和资源释放



## 参考资料

- Eino 第六章：[Callback 与 Trace（可观测性）](https://www.cloudwego.io/zh/docs/eino/quick_start/chapter_06_callback_and_trace/)  
  
- Eino [Callback 用户手册](https://www.cloudwego.io/zh/docs/eino/core_modules/chain_and_graph_orchestration/callback_manual/)  

  
