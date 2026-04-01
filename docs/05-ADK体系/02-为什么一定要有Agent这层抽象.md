# AI 大模型落地系列｜Eino ADK体系篇：为什么一定要有 Agent 这层抽象

> GitHub 主文：[当前文章](./02-为什么一定要有Agent这层抽象.md)
> CSDN 跳转：[AI 大模型落地系列｜Eino ADK体系篇：为什么一定要有 Agent 这层抽象](https://zhumo.blog.csdn.net/article/details/159690023)
> 官方文档：[Eino ADK：Agent 抽象](https://www.cloudwego.io/zh/docs/eino/core_modules/eino_adk/agent_interface/)
>
> 最新版以 GitHub 仓库为准，CSDN 作为分发入口，官方文档作为权威参考。

**一句话摘要**：从接口、输入输出、事件流和自定义实现理解 Agent 抽象为什么不是 Prompt 包装器。
**适合谁看**：准备真正理解 Agent 协议，而不是只会调用现成 Agent 的 Go 工程师。
**前置知识**：什么是 Eino ADK、ChatModelAgent、Runner 基础、Go 接口与泛型基础
**对应 Demo**：[官方 Agent 接口文档与本地自定义 Agent 实战](https://www.cloudwego.io/zh/docs/eino/core_modules/eino_adk/agent_interface/)

**面试可讲点**
- 能解释 Name、Description、Run、AgentEvent、AsyncIterator 这些抽象为什么缺一不可。
- 能说明 Agent 输入为什么是 Messages 而不是单个字符串。

---
本篇只讲一点
> 为什么 ADK 一定要单独定义 `Agent` 这层抽象？

很多人真正没看懂的，不是 `Name`、`Description`、`Run`，而是这套协议到底统一了什么。

本文只做四件事：

1. 讲清 `Agent` 有什么用，为什么它不是 Prompt 包装器
2. 讲透 `AgentInput / AgentRunOption / AsyncIterator / AgentEvent`
3. 给一个 **零外部依赖** 的自定义 Agent demo
4. 帮你把后面 `Workflow / Runner / Interrupt` 的地基先打好

## 1. 为什么 `Agent` 抽象是必要的

如果没有 `Agent` 这一层，AI 应用很容易长成一堆分散的模型调用：

- 这里直接调 `ChatModel`
- 那里自己拼 `Messages`
- Tool 结果自己处理
- 多 Agent 协作时，每一层都重新定义输入输出
- 中断、恢复、链路追踪、状态注入散在业务代码里

但只要系统开始复杂一点，问题就来了：

- 谁是这次执行单元的身份标识
- 别的 Agent 怎么知道它能做什么
- 调用方拿到的是最终字符串，还是过程事件
- 某个请求级参数该影响谁
- 这次是输出了消息，还是触发了跳转、中断、退出

所以 `Agent` 抽象真正解决的，不是“怎么调模型”。

它解决的是：

> 怎么把一次智能体执行，统一成一个可运行、可组合、可治理的对象。

把这件事画开，就是下面这张最小协议图：

![外链图片转存失败,源站可能有防盗链机制,建议将图片保存下来直接上传](https://i-blog.csdnimg.cn/direct/20d3895cf5ca403089a69d4ecfb35a4a.png)


你只要先记住一个判断就够了：

> `Agent` 不单独存在，而是和 `AgentInput`、`AgentRunOption`、`AsyncIterator`、`AgentEvent` 一起构成运行协议。

## 2. `Agent` 接口：为什么这三个方法都不能少

官方定义很短：

```go
type Agent interface {
    Name(ctx context.Context) string
    Description(ctx context.Context) string
    Run(ctx context.Context, input *AgentInput, opts ...AgentRunOption) *AsyncIterator[*AgentEvent]
}
```

### `Name`

`Name` 不只是“取个名字”。

它至少承担三件事：

- Agent 的身份标识
- 执行链路里的节点名
- `DesignateAgent(...)` 这类定向 option 的匹配目标

### `Description`

`Description` 也不只是注释。

它更像对外公开的职责声明：

- 给人看，知道这个 Agent 会什么
- 给别的 Agent 看，判断该不该把任务转给它

### `Run`

`Run` 才是核心。

```go
Run(ctx context.Context, input *AgentInput, opts ...AgentRunOption) *AsyncIterator[*AgentEvent]
```

这一个签名，直接把四件事统一了：

1. 一次 Agent 执行必须带 `context.Context`
2. 输入统一走 `AgentInput`
3. 请求级调参统一走 `AgentRunOption`
4. 输出统一走事件流 `AsyncIterator[*AgentEvent]`

所以 `Run` 不是普通函数。

它是在规定：

> ADK 里的一次 Agent 执行，应该以什么协议被启动、被调整、被消费。

## 3. `AgentInput`：为什么输入是 `Messages`，不是一个字符串

官方定义：

```go
type AgentInput struct {
    Messages        []Message
    EnableStreaming bool
}

type Message = *schema.Message
```

很多人第一次看到这里，会下意识理解成“用户问题 + 一个流式开关”。

这个理解太轻了。

### `Messages` 是任务上下文，不是单条 prompt

`Messages` 里可以放的不只是用户这一句。

它可以承载：

- 当前问题
- 对话历史
- 上游 Agent 结果
- 背景知识
- 样例数据
- 系统约束

也就是说，`Messages` 的意义不是“聊天格式”。

它真正的价值是：

> 把一次任务所需的上下文统一收紧。

如果输入只是一条 `string prompt`，那每个 Agent 都得自己决定历史怎么塞、系统约束怎么塞、Tool 结果怎么塞，输入协议就会发散。

### `EnableStreaming` 是建议，不是强制

这是一个特别容易踩的点。

很多人会误以为：

- `EnableStreaming=true` 就一定流式
- `EnableStreaming=false` 就一定非流式

但官方文档强调得很清楚，它只是一个 **建议**。

它只会影响那些“同时支持流和非流”的组件，比如 `ChatModel`。
如果某个组件天然只支持一种输出方式，比如很多 Tool，它不会因为这个字段就突然变成流式。

看图最直观：
![在这里插入图片描述](https://i-blog.csdnimg.cn/direct/1cfc04d21a224ed69c2fcaa8e967b329.png)


这句最好直接背下来：

> `EnableStreaming` 控制的是偏好，不是强制转换器。

实际输出到底是不是流，请看后面的 `MessageVariant.IsStreaming`。

## 4. `AgentRunOption` 和 `AgentWithOptions`：看起来像一回事，其实不是

这两个概念容易混。

最简单的分法就一张表：

| 能力 | 作用时机 | 你可以先怎么理解 |
| --- | --- | --- |
| `AgentRunOption` | 请求期 | 这一次运行怎么调 |
| `AgentWithOptions` | 运行前 | 这个 Agent 先被怎么包装 |

### `AgentRunOption`

它是传给 `Run()` 的：

```go
Run(ctx context.Context, input *AgentInput, opts ...AgentRunOption)
```

官方内置给了两个很典型的通用 option：

- `WithSessionValues`：设置跨 Agent 读写数据
- `WithSkipTransferMessages`：某些 Transfer 消息不进入 History

除此之外，ADK 还给了两个很实用的扩展点：

```go
adk.WrapImplSpecificOptFn(...)
adk.GetImplSpecificOptions(...)
```

这套设计的价值很直接：

> 每个 Agent 都可以扩展出自己的请求级参数，而不用把所有行为都塞进一套全局 option。

比如后面 demo 里的：

```go
WithAudience("newbie")
WithAudience("interview")
```

它就能证明 `AgentRunOption` 真的是“这次运行怎么调”，而不是静态配置。

`DesignateAgent(...)` 则是更偏多 Agent 场景的能力：

```go
opt := adk.WithSessionValues(map[string]any{}).DesignateAgent("agent_1", "agent_2")
```

它的真正作用就是：在多 Agent 系统里，只让指定名字的 Agent 看见这个 option。

### `AgentWithOptions`

它是这样用的：

```go
func AgentWithOptions(ctx context.Context, agent Agent, opts ...AgentOption) Agent
```

官方当前内置支持的两个点是：

- `WithDisallowTransferToParent`
- `WithHistoryRewriter`

它们都不属于“这一次运行怎么调”。

它们属于：

> 在真正执行前，先把 Agent 包一层通用行为。

所以别把这两个层级混掉。

## 5. `AsyncIterator`：为什么 Agent 不直接返回字符串

官方定义：

```go
type AsyncIterator[T any] struct {
    ...
}

func (ai *AsyncIterator[T]) Next() (T, bool)
```

ADK 这里的一个关键设计是：

> Agent 不是“输入一个值，输出一个值”的普通函数。

一次 Agent 执行，除了最终文本，还可能产生：

- 中间输出
- Tool 消息
- 跳转行为
- 中断行为
- 错误

如果只返回 `string`，这些信息根本没地方放。

所以 ADK 选择的是：

> 不直接给终值，而是给一串按顺序消费的事件。

### `Next()` 为什么重要

`Next()` 是阻塞式的。

也就是每次调用时，只会等两种结果：

- 等到一个新的 `AgentEvent`
- 或者等到迭代器关闭，返回 `ok=false`

这意味着调用方的消费逻辑会非常稳定：

```go
for {
    event, ok := iter.Next()
    if !ok {
        break
    }
    // handle event
}
```

### `NewAsyncIteratorPair` + goroutine 为什么是常见写法

官方给了这套基础设施：

```go
iter, gen := adk.NewAsyncIteratorPair[*adk.AgentEvent]()
```

- `iter` 给调用方消费
- `gen` 给 Agent 内部发事件

自定义 Agent 常见实现会开 goroutine，不是为了炫技，而是因为：

> `Run()` 的目标不是等所有事做完再返回，而是先把事件出口交出去，然后内部异步地产生事件。

如果不这么做，你会把“事件流协议”重新写回“阻塞函数返回值”。

## 6. `AgentEvent` / `AgentOutput` / `AgentAction`：一次执行到底吐出了什么

官方定义：

```go
type AgentEvent struct {
    AgentName string
    RunPath   []RunStep
    Output    *AgentOutput
    Action    *AgentAction
    Err       error
}
```

这部分只要抓住“事件里到底装了哪几类信息”就够了。

### `AgentName` 和 `RunPath`

- `AgentName`：是谁发出的当前事件
- `RunPath`：这个事件是沿着哪条调用链走到这里的

在单 Agent 场景里你可能感受不强。
但一到多 Agent 场景，这两个字段就是链路上下文。

### `AgentOutput`

官方定义：

```go
type AgentOutput struct {
    MessageOutput *MessageVariant
    CustomizedOutput any
}
```

这说明 ADK 默认把“消息输出”当成第一公民，同时也允许你挂自定义输出。

而 `MessageVariant` 的价值是把流式和非流式统一起来：

```go
type MessageVariant struct {
    IsStreaming bool
    Message       Message
    MessageStream MessageStream
    Role          schema.RoleType
    ToolName      string
}
```

最重要的不是字段多，而是这几个判断位很实用：

- `IsStreaming`：当前到底是不是流
- `Role`：当前是 Assistant 还是 Tool
- `ToolName`：如果是 Tool，工具名是什么

### `AgentAction`

很多人看 `AgentEvent` 时，只盯着 `Output`。

但 ADK 还专门留了一条“行为输出通道”：

```go
type AgentAction struct {
    Exit bool
    Interrupted     *InterruptInfo
    TransferToAgent *TransferToAgentAction
    BreakLoop       *BreakLoopAction
    CustomizedAction any
}
```

它的意义很直接：

> Agent 不只会“说什么”，还会“决定接下来怎么跑”。

官方当前内置几类 Action：

- `NewExitAction()`：立刻退出
- `NewTransferToAgentAction(name)`：跳到目标 Agent
- `Interrupted`：通知 Runner 当前中断
- `BreakLoop`：让 LoopAgent 结束循环

你可以先把它们理解成下面这种最小意图：

```go
gen.Send(&adk.AgentEvent{
    Action: adk.NewExitAction(),
})

gen.Send(&adk.AgentEvent{
    Action: adk.NewTransferToAgentAction("planner_agent"),
})
```

### `Err`

消费事件时，`Err` 绝对不能跳过：

```go
if event.Err != nil {
    // handle error
}
```

否则很容易出现一种假象：

看起来“好像有输出”，但实际执行已经坏了。

### `SetLanguage`

`Agent 抽象` 页最后补的 `SetLanguage`，你只要记住 4 句话：

1. 它是全局设置
2. 最好在程序初始化时设置
3. 它只影响 ADK 内置 prompt
4. 不要在运行时来回切

因为一旦同一会话里出现混合语言提示词，问题会很隐蔽。

## 7. 自定义 Agent 实战：从零实现一个 `ConceptTutorAgent`

这段代码的目标不是做知识推理，而是跑通 Agent 协议。

先看执行链：

![外链图片转存失败,源站可能有防盗链机制,建议将图片保存下来直接上传](https://i-blog.csdnimg.cn/direct/9032a2673b25491a99e6b509f4723540.png)

它想证明 4 件事：

- 自定义 Agent 本质上就是实现 `Agent` 接口
- `Run()` 返回的是事件流，不是字符串
- `AgentRunOption` 可以做请求级调参
- 不接模型 API，也能把 Agent 协议本身跑通

### 完整代码

把下面代码保存成 `main.go`：

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

// audienceOptions 是当前自定义 Agent 的实现级运行参数。
// 这类参数不进入通用 Agent 接口，而是通过 impl-specific option 透传。
type audienceOptions struct {
	audience string
}

// WithAudience 为当前 Agent 注入“面向谁讲解”的运行选项。
// 调用方可在不修改 Agent 接口的前提下，按次覆盖执行行为。
func WithAudience(audience string) adk.AgentRunOption {
	return adk.WrapImplSpecificOptFn(func(o *audienceOptions) {
		o.audience = audience
	})
}

// ConceptTutorAgent 是一个最小可运行的自定义 Agent。
// 它不依赖大模型，而是演示：如何实现 Agent 接口、消费 AgentInput、产出 AgentEvent。
type ConceptTutorAgent struct{}

// Name 返回 Agent 的稳定标识，用于日志、协作和运行时识别。
func (a *ConceptTutorAgent) Name(ctx context.Context) string {
	return "ConceptTutorAgent"
}

// Description 返回 Agent 的能力描述，供人类或其他 Agent 判断是否适合处理某类任务。
func (a *ConceptTutorAgent) Description(ctx context.Context) string {
	return "负责把一个技术概念讲成新手能听懂的三段话"
}

// Run 是 Agent 的执行入口。
// 它从输入消息中提取任务内容，读取实现级运行参数，并通过事件流返回结果。
func (a *ConceptTutorAgent) Run(ctx context.Context, input *adk.AgentInput, opts ...adk.AgentRunOption) *adk.AsyncIterator[*adk.AgentEvent] {
	iter, gen := adk.NewAsyncIteratorPair[*adk.AgentEvent]()

	// Agent 的输出协议是事件流，因此这里异步生成事件并通过 iterator 暴露给调用方。
	go func() {
		defer gen.Close()

		// 优先响应上游取消或超时，避免 goroutine 泄漏。
		if err := ctx.Err(); err != nil {
			gen.Send(&adk.AgentEvent{Err: err})
			return
		}

		// 基础入参校验：没有消息就无法构造任务上下文。
		if input == nil || len(input.Messages) == 0 {
			gen.Send(&adk.AgentEvent{Err: fmt.Errorf("agent input messages is empty")})
			return
		}

		// 读取当前 Agent 自己定义的运行选项；未传时使用默认值。
		cfg := adk.GetImplSpecificOptions(&audienceOptions{audience: "newbie"}, opts...)

		// 约定使用最后一条 user message 作为本次要讲解的概念。
		concept := lastUserMessage(input.Messages)
		if strings.TrimSpace(concept) == "" {
			gen.Send(&adk.AgentEvent{Err: fmt.Errorf("last user message is empty")})
			return
		}

		reply := buildReply(concept, cfg.audience, input.EnableStreaming)

		// 将最终文本包装成标准 assistant 消息事件返回。
		gen.Send(adk.EventFromMessage(
			schema.AssistantMessage(reply, nil),
			nil,
			schema.Assistant,
			"",
		))
	}()

	return iter
}

// lastUserMessage 从消息列表中逆序查找最后一条用户消息。
// 这是一种常见约定：最新的 user 输入通常代表当前任务指令。
func lastUserMessage(messages []adk.Message) string {
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		if msg != nil && msg.Role == schema.User {
			return msg.Content
		}
	}
	return ""
}

// buildReply 根据概念、受众和流式标记构造演示用回复。
// 这里故意不接入真实模型，目的是突出 Agent 输入/输出协议本身。
func buildReply(concept, audience string, enableStreaming bool) string {
	prefix := "面向新手"
	if audience == "interview" {
		prefix = "面向面试复盘"
	}

	streamingHint := "这次我没有实现流式输出，所以会一次性返回完整结果。"
	if !enableStreaming {
		streamingHint = "这次按非流式方式返回完整结果。"
	}

	return fmt.Sprintf(
		"%s\n\n一句话定义：这里把“%s”当成当前要讲解的概念。\n为什么重要：这个 demo 不是在做真实知识推理，而是在演示 Agent 如何围绕输入、事件和 option 组织一次执行。\n常见坑：别把 Messages 理解成单条 prompt，它其实承载的是任务上下文。\n补充：%s",
		prefix,
		concept,
		streamingHint,
	)
}

func main() {
	ctx := context.Background()

	// 默认讲解概念；支持命令行覆盖，便于本地快速测试不同输入。
	concept := "Agent 抽象"
	if len(os.Args) > 1 {
		concept = strings.Join(os.Args[1:], " ")
	}

	agent := &ConceptTutorAgent{}

	// AgentInput 承载本次任务上下文，而不只是单条 prompt。
	// 这里同时放入 system message 和 user message，模拟一次最小对话输入。
	input := &adk.AgentInput{
		Messages: []adk.Message{
			schema.SystemMessage("你是一个负责解释技术概念的教学 Agent。"),
			schema.UserMessage(concept),
		},
		EnableStreaming: true,
	}

	fmt.Printf("agent=%s\n", agent.Name(ctx))
	fmt.Printf("description=%s\n\n", agent.Description(ctx))

	// 直接运行自定义 Agent，并通过实现级 option 注入受众信息。
	iter := agent.Run(ctx, input, WithAudience("newbie"))
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		// 事件级错误需要显式处理；这也是事件流协议的一部分。
		if event.Err != nil {
			log.Fatalf("agent failed: %v", event.Err)
		}
		if event.Output == nil || event.Output.MessageOutput == nil {
			continue
		}

		mv := event.Output.MessageOutput
		if mv.Message == nil {
			continue
		}

		fmt.Printf("assistant>\n%s\n", mv.Message.Content)
	}
}
```

### 运行

```bash
go mod init concept-tutor-demo
go get github.com/cloudwego/eino@latest
go run . -- "AsyncIterator"
```

你会看到类似输出：

```text
agent=ConceptTutorAgent
description=负责把一个技术概念讲成新手能听懂的三段话

assistant>
面向新手

一句话定义：这里把“AsyncIterator”当成当前要讲解的概念。
为什么重要：这个 demo 不是在做真实知识推理，而是在演示 Agent 如何围绕输入、事件和 option 组织一次执行。
常见坑：别把 Messages 理解成单条 prompt，它其实承载的是任务上下文。
补充：这次我没有实现流式输出，所以会一次性返回完整结果。
```

### 这段代码对应了哪些抽象

1. `Name()`：给 Agent 身份
2. `Description()`：给 Agent 职责描述
3. `Run()`：按统一协议执行
4. `AgentInput.Messages`：承载任务上下文
5. `WithAudience(...)`：演示请求级 option
6. `NewAsyncIteratorPair()`：建立生产者和消费者
7. `EventFromMessage(...)`：把输出装进 `AgentEvent`
8. `iter.Next()`：调用方按事件流消费

### 进阶补充：流式长什么样

这次 demo 故意没实现流式，就是为了说明：

> `EnableStreaming=true` 不意味着你这个 Agent 必须流式输出。

如果你只想看“流式 `MessageVariant` 怎么发”，一个最小片段是：

```go
stream := schema.StreamReaderFromArray([]adk.Message{
	schema.AssistantMessage("第一段。", nil),
	schema.AssistantMessage("第二段。", nil),
})

gen.Send(adk.EventFromMessage(nil, stream, schema.Assistant, ""))
```

此时：

- `IsStreaming = true`
- `Message = nil`
- `MessageStream != nil`

## 8. 躲坑 + 下一步学习路线

### 最容易踩的 7 个坑

1. 把 `Agent` 当成 Prompt 包装器。
2. 把 `Messages` 当成“用户这一句话”。
3. 把 `EnableStreaming` 当成强制命令。
4. 忘记 `gen.Close()`，导致迭代不结束。
5. 只读输出，不处理 `event.Err`。
6. 把 `AgentRunOption` 和 `AgentWithOptions` 混用。
7. 在运行时随意切 `SetLanguage`。

### 看完这篇，下一步怎么学

建议按这 3 步往后走：

1. 回看 [ADK 首卡总览](./AI%20大模型落地系列｜Eino%20ADK%20篇：为什么很多人看完%20Quickstart，还是搭不出真正的%20Multi-Agent.md)
2. 再看 [ChatModelAgent、Runner、AgentEvent（Console 多轮）](../入门必学/AI大模型落地系列：一文读懂%20ChatModelAgent、Runner、AgentEvent（Console%20多轮）.md)，把“现成 Agent 怎么跑起来”补上
3. 然后进入 `Agent 协作`、`Workflow Agents`、`Agent Runner 与扩展`

这篇想让你建立的认知只有一句：

> `Agent` 不是一段配置，而是一套统一输入、统一事件流、统一行为协议的运行对象。

## 参考资料

- [Eino ADK: Agent 抽象](https://www.cloudwego.io/zh/docs/eino/core_modules/eino_adk/agent_interface/)
- [Eino ADK: 概述](https://www.cloudwego.io/zh/docs/eino/core_modules/eino_adk/agent_preview/)
- [Eino ADK: Quickstart](https://www.cloudwego.io/zh/docs/eino/core_modules/eino_adk/agent_quickstart/)
- [Eino ADK: Agent 协作](https://www.cloudwego.io/zh/docs/eino/core_modules/eino_adk/agent_collaboration/)
- [Eino ADK: Agent Runner 与扩展](https://www.cloudwego.io/zh/docs/eino/core_modules/eino_adk/agent_extension/)

---

## 发布说明

- GitHub 主文：[AI 大模型落地系列｜Eino ADK体系篇：为什么一定要有 Agent 这层抽象](./02-为什么一定要有Agent这层抽象.md)
- CSDN 跳转：[AI 大模型落地系列｜Eino ADK体系篇：为什么一定要有 Agent 这层抽象](https://zhumo.blog.csdn.net/article/details/159690023)
- 官方文档：[Eino ADK：Agent 抽象](https://www.cloudwego.io/zh/docs/eino/core_modules/eino_adk/agent_interface/)
- 最新版以 GitHub 仓库为准。

