# AI大模型落地系列：一文读懂 ChatModelAgent、Runner、AgentEvent（Console 多轮）

> GitHub 主文：[当前文章](./02-ChatModelAgent、Runner、AgentEvent（Console多轮）.md)
> CSDN 跳转：[AI大模型落地系列：一文读懂 ChatModelAgent、Runner、AgentEvent（Console 多轮）](https://zhumo.blog.csdn.net/article/details/159468400)
> 官方文档：[Eino 第二章：ChatModelAgent、Runner、AgentEvent](https://www.cloudwego.io/zh/docs/eino/quick_start/chapter_02_chatmodelagent_runner_agentevent/)
>
> 最新版以 GitHub 仓库为准，CSDN 作为分发入口，官方文档作为权威参考。

**一句话摘要**：把单轮聊天扩展成 Agent、Runner、事件流，建立 Console 多轮的最小执行闭环。
**适合谁看**：已经跑通 ChatModel，希望继续理解 Agent 运行时的 Go 开发者。
**前置知识**：ChatModel 与 Message、消息角色与流式输出
**对应 Demo**：[官方示例 ch02（本仓后续补充同主题 demo）](https://github.com/cloudwego/eino-examples/blob/main/quickstart/chatwitheino/cmd/ch02/main.go)

**面试可讲点**
- 能讲清楚 ChatModelAgent、Runner、AgentEvent 三者的职责分工。
- 能说明为什么事件流比直接返回字符串更适合后续扩展工具调用和可观测性。

---
很多人第一次把多轮对话跑通，代码都长这样：

```go
history = append(history, schema.UserMessage(line))
events := runner.Run(ctx, history)
history = append(history, schema.AssistantMessage(content, nil))
```

程序确实能聊起来。
但我先泼盆冷水：

> 这还不算你真正理解了 Agent。

你只是把历史消息带进了模型，还没有真正理解 Eino 为什么要在 `ChatModel` 之上再抽出 `ChatModelAgent`、`Runner` 和 `AgentEvent`。

这不是咬文嚼字。
这是两个完全不同的认知层次。

- 前者是在“调模型”
- 后者是在“理解一个可运行的 Agent 抽象到底怎么工作”

如果你前面已经看过上一篇，这一章的位置就会更容易看清。

第一章讲的是：怎么和模型说话。
这一章讲的是：怎么把模型能力放进一套可运行的 Agent 骨架里。

也就是说，学习路径会从 `ChatModel` 再往前走一层，切到 `ChatModelAgent / Runner / AgentEvent` 这套运行时视角。

`Memory / Session`、`Tool`、`Callback / Trace` 这些能力，我会在后续章节继续展开。
所以本篇不会展开持久化记忆、Tool 编排和可观测性细节。
而是盯着以下几个核心主线：

- `ChatModelAgent` 是什么
- `Runner` 为什么要存在
- `AgentEvent` 为什么不是多此一举
- 一个最小 Console 多轮程序到底是怎么跑通的

## 1. 为什么“能多轮”不等于你真的理解了 Agent？

很多人第一次做多轮对话，思路都差不多：

1. 定义一个 `history []*schema.Message`
2. 每次用户输入都 append 进去
3. 把 `history` 扔给模型
4. 再把 assistant 的回复 append 回去

从效果上看，这当然已经是多轮。
模型确实能记住上一轮说了什么。

但如果你把这件事直接等同于“我已经写出了 Agent”，那就有点过早下结论了。

因为这里面至少混了两个不同层级的问题：

**第一层，是上下文累积。**

也就是：上一轮说过的话，这一轮还能不能带上。

**第二层，是执行抽象。**

也就是：一次 Agent 执行，到底怎么被启动、组织、输出、流式消费、以及后续扩展的。

前者更像“把消息继续传给模型”。
后者才是“一个智能体运行时是怎么被定义出来的”。

如果只停留在第一层，你写出来的往往只是“带历史消息的模型调用”。
它离真正的 Agent 运行时，还有一层抽象距离。

这个视角一旦建立起来，后面你再看 `Tool`、`Interrupt`、`CheckPoint`、`Supervisor`，脑子里就不会是一团散的。
## 2. 何为`ChatModelAgent`？
大家可以先思考一个问题：
明明`ChatModel`，已经有了对话能力。可是为何 Eino 却依旧不满足于 `ChatModel`，还要再抽一层 `ChatModelAgent`？

这里我先把边界说清。

- `ChatModel` 是组件。
- 而`ChatModelAgent` 是 Agent。

这两个词只差了一个后缀，但职责并不在一个层面。

### 2.1 `ChatModel` 解决的是“模型调用边界”

前面那篇 `ChatModel` 文章里，已经讲述过了它的核心价值：

- 统一不同模型厂商的调用接口
- 把“和模型说话”抽象成稳定能力
- 为后续编排和测试留出边界

它的关注点很明确：

> 输入一组消息，返回模型输出。

这已经很重要了。
但它仍然只是“能力组件”，还不是完整的应用运行抽象。

### 2.2 `ChatModelAgent` 解决的是“把模型能力提升成可运行的 Agent”

官方在 ADK 里定义的 `Agent` 接口，核心长这样：

```go
type Agent interface {
    Name(ctx context.Context) string
    Description(ctx context.Context) string
    Run(ctx context.Context, input *AgentInput, options ...AgentRunOption) *AsyncIterator[*AgentEvent]
}
```

这里最值得注意的不是 `Name()` 或 `Description()`。
而是 `Run()`。

因为从这里开始，事情已经不是“模型返回一段文本”了。
而是：

> Agent 执行后，返回一个 `AsyncIterator[*AgentEvent]` 形式的事件流。

这说明什么？

说明 `Agent` 关注的已经不是单次模型请求本身，而是一次完整执行过程的输出形态。
因为 `Agent` 这层抽象已经规定：一个 Agent 必须以 `Run() -> AsyncIterator[*AgentEvent]` 的方式对外工作，所以 ChatModelAgent 的任务，其实就是把底层 ChatModel 的调用结果，适配成这套 `Agent 协议`。
所以此时再回头看 `ChatModelAgent`，它就很好理解了：

```go
agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
    Name:        "Ch02ConsoleAgent",
    Description: "A minimal ChatModelAgent for console multi-turn chat.",
    Instruction: "你是一个简洁、专业的 Eino 学习助手。",
    Model:       cm,
})
```

`adk.ChatModelAgentConfig` 在本篇要关注的字段只有四个：

- `Name`：这个 Agent 叫什么
- `Description`：这个 Agent 用来干什么
- `Instruction`：系统级行为约束
- `Model`：它底层使用哪个 `ChatModel`

你会发现，它并没有突然变出什么魔法能力。
它底下还是模型。

但它做了一件非常关键的事：

> 它把“单纯的模型能力”包装进了“统一的 Agent 执行协议”里。

那这层协议化有什么价值？

我认为至少有三点。

**第一，统一上层抽象。**

以后无论你用的是 `ChatModelAgent`、`WorkflowAgent` 还是别的 Agent，实现层可以不同，但对运行时来说，大家都按 `Run() -> AsyncIterator[*AgentEvent]` 这套协议来。

**第二，给扩展留位置。**

今天这个 Agent 只有模型。
明天它可以长出 Tool、Middleware、Interrupt、CheckPoint。
如果没有统一的 Agent 抽象，后面这些能力只能不断往 `ChatModel` 身上硬塞。

**第三，让“AI 应用”真正变成一个能跑的对象。**

`ChatModel` 更像数据库驱动，负责连接和执行。
`ChatModelAgent` 更像服务层抽象，虽然底层还是那个能力，但现在它已经能被 Runner 统一驱动了。

这里也顺手澄清第一个误区：

> `ChatModelAgent` 不是“另一个模型客户端”，它是“基于模型实现的 Agent”。

## 3. `Runner` 为什么不是多余的一层？

很多人第一次看到 `Runner`，心里都会冒出一个问题：
```go
type Agent interface {
    Name(ctx context.Context) string
    Description(ctx context.Context) string
    Run(ctx context.Context, input *AgentInput, options ...AgentRunOption) *AsyncIterator[*AgentEvent]
}
```

“既然官方给的 `Agent` 已经有 `Run()` 了，那为什么还要再包一个 Runner？”

这很正常。
从表面看，好像只是又多包了一层对象。

但如果你从运行时角度去看，`Runner` 并不是装饰品。
它是 Agent 的统一执行入口。

### 3.1 `Runner` 解决的是“谁来驱动 Agent 执行”

官方示例的典型写法是：

```go
runner := adk.NewRunner(ctx, adk.RunnerConfig{
    Agent:           agent,
    EnableStreaming: true,
})
```

`adk.RunnerConfig` 在本篇里要盯住两个字段：

- `Agent`：这次 Runner 负责执行哪个 Agent
- `EnableStreaming`：是否按流式方式消费输出

`Runner` 的价值，不是替你做业务判断。
它的价值是把 Agent 的执行过程，统一收口到一个稳定入口。

你可以把它理解成：

- Agent 定义“这个东西能怎么跑”
- Runner 负责“这次具体怎么驱动它跑”

### 3.2 为什么不能只盯着 `agent.Run()`

如果只从“能不能跑”这个角度，很多事情当然也能绕开。

但工程上真正麻烦的从来不是“这一行代码能不能执行”，而是：

- 执行入口是否统一
- 流式输出怎么消费
- 后面接中断恢复时往哪挂
- 后面扩展 checkpoint、callback、query helper 时边界放哪

```go
type Runner struct {
    a Agent  // 要执行的 Agent
    enableStreaming bool // 是否是流式的
    store CheckPointStore  // 用于中断恢复的状态存储
}
```
而 Runner 却可以提供。
所以 `Runner` 的意义，就是把这些运行时能力集中在一起，而不是散落到业务代码里。

你现在可能只是在写一个最小 Demo。
看起来它只是“让 Agent 跑起来”。

但在更完整的 ADK 体系里，`Runner` 代表的是一种运行时收口点。

### 3.3 多轮对话中，为何要用 `runner.Run(ctx, history)`

之所以提到这个，是因为官方文档里展示了  `runner.Query(ctx, "你好")` 这种便捷方式。

但本篇却故意不用它。

因为多轮对话的实现，最关键的不是“临时对话一句”，而是看清楚：

> 而是每一轮执行，调用方到底传给 Agent 的是什么输入。

而多轮对话里最核心的输入，就是整段 `history`。

所以这里必须显式写：

```go
events := runner.Run(ctx, history)
```

这一句比 `Query()` 更重要。
因为它直接把“多轮靠谁维持”这件事暴露出来了。

从而也能顺手澄清第二个误区：

> `Runner` 负责执行 Agent，但它不负责替你保存历史上下文。
对上下文的持久化与会话管理，后续会在 `Memory / Session` 一章里单独展开。
## 4. 为什么 Agent 不直接返回字符串，而是返回 `AgentEvent` 事件流？

如果你以前主要写的是普通接口服务，第一次看到这种返回值会有点别扭：

```go
Run(...) *AsyncIterator[*AgentEvent]
```

为什么不直接 `return string`？
为什么不直接 `return *schema.Message`？

因为 Agent 的执行过程，本来就不是一个适合被压扁成“最终字符串”的东西。

### 4.1 `AgentEvent` 代表的是“执行过程中的一个事件单元”

官方文档给出的关键字段，大致可以精简成这样：

```go
type AgentEvent struct {
    Output *AgentOutput
    Action *AgentAction
    Err    error
}
```

本篇只需要关注三个点：

- `event.Output`：这次事件有没有产出消息
- `event.Action`：这次有没有控制动作，比如中断、转移、退出
- `event.Err`：这次执行有没有在事件层面报错

这说明一件事：

> Agent 输出的不是一坨最终结果，而是一连串可消费、可观察、可扩展的事件。

### 4.2 为什么必须是事件流

原因并不玄学。
就是因为 Agent 的执行天然是过程性的。

最简单的情况里，模型可能是逐 token 流式返回。
复杂一点的情况里，中间还会穿插：

- Tool 调用
- Tool 结果回灌
- 状态切换
- 中断与恢复

如果你要求它“一次性给我最终字符串”，那你等于把中间所有过程都抹掉了。

这会直接损失掉三类能力：

- 流式体验
- 可观测性
- 更复杂的控制动作表达

所以 `AgentEvent` 不是多此一举。
它是在为后面的复杂执行形态预留表达空间。

### 4.3 `AsyncIterator[*AgentEvent]` 怎么消费

最小消费模式通常就是这样：

```go
for {
    event, ok := events.Next()
    if !ok {
        break
    }
    if event.Err != nil {
        return event.Err
    }
}
```

这里有两个非常关键的点。

**第一，`Next()` 是逐个拿事件。**

它不是“马上返回最终结果”，而是不断把过程中的事件交给你。

**第二，迭代器是一次性的。**

每次 `runner.Run()` 都会生成一个新的 `*adk.AsyncIterator[*adk.AgentEvent]`。
你把这次迭代器消费完，就结束了，不能指望再 rewind 一次重新读。

这一点非常像流。
不是数组。

### 4.4 `event.Err` 和 `Recv()` 错误不是一回事

这里提前埋一个很多人会踩的坑。

如果 `event.Output.MessageOutput` 是流式输出，那你后面通常还会继续读：

```go
frame, err := mv.MessageStream.Recv()
```

那么错误其实有两层：

- 第一层是 `event.Err`
- 第二层是你继续 `Recv()` 流的时候发生的错误

这两个不要混成一件事。

也就是说：

> 你不能只判断 `event.Err == nil` 就以为这轮流式消费一定没问题。

等会看完整 Demo 时，你会看到这两个地方都会显式处理。

## 5. 实战：一个精简的多轮对话程序

上面讲了半天抽象，如果你不把它真的跑起来，很容易只停在概念层。

接下来的例子中你将会看清：保留 Console 多轮 + Agent 执行抽象


### 5.1 先准备依赖和环境变量

```bash
go mod init eino-ch02-demo
go get github.com/cloudwego/eino@latest
go get github.com/cloudwego/eino-ext/components/model/qwen@latest

export DASHSCOPE_API_KEY="你的百炼 API Key"
export QWEN_MODEL="qwen3.5-flash"
```

如果你在 Windows PowerShell 下，可以改成：

```powershell
$env:DASHSCOPE_API_KEY="你的百炼 API Key"
$env:QWEN_MODEL="qwen3.5-flash"
```

### 5.2  多轮对话的小程序

```go
package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
	"github.com/cloudwego/eino-ext/components/model/qwen"
)

func main() {
	ctx := context.Background()

	// 1. 初始化 Qwen ChatModel。
	cm, err := qwen.NewChatModel(ctx, &qwen.ChatModelConfig{
		BaseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1",
		APIKey:  mustEnv("DASHSCOPE_API_KEY"),
		Model:   envOrDefault("QWEN_MODEL", "qwen3.5-flash"),
	})
	if err != nil {
		log.Fatalf("new qwen chat model failed: %v", err)
	}

	// 2. 基于 ChatModel 构建一个最小 ChatModelAgent。
	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "Ch02ConsoleAgent",
		Description: "A minimal ChatModelAgent for console multi-turn chat.",
		Instruction: "你是一个简洁、专业的 Eino 学习助手。",
		Model:       cm,
	})
	if err != nil {
		log.Fatalf("new chat model agent failed: %v", err)
	}

	// 3. 用 Runner 驱动 Agent 执行，并开启流式输出。
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming: true,
	})

	// 4. 用内存里的 history 维护多轮上下文。
	// 注意：这只是进程内多轮，不是持久化记忆。
	history := make([]*schema.Message, 0, 16)

	fmt.Println("Enter your message (empty line to exit):")
	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("you> ")
		if !scanner.Scan() {
			break
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			break
		}

		// 4.1 记录用户输入
		history = append(history, schema.UserMessage(line))

		// 4.2 把完整 history 交给 Runner 执行 Agent
		content, err := collectAssistantFromEvents(runner.Run(ctx, history))
		if err != nil {
			log.Fatalf("run agent failed: %v", err)
		}

		// 4.3 把 assistant 回复也写回 history，进入下一轮
		history = append(history, schema.AssistantMessage(content, nil))
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
}

func collectAssistantFromEvents(events *adk.AsyncIterator[*adk.AgentEvent]) (string, error) {
	var sb strings.Builder

	for {
		event, ok := events.Next()
		if !ok {
			break
		}

		// 第一层错误：AgentEvent 层面的执行错误
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

		if mv.IsStreaming {
			// 自动关闭底层流，避免资源泄漏。
			mv.MessageStream.SetAutomaticClose()

			for {
				frame, err := mv.MessageStream.Recv()
				if errors.Is(err, io.EOF) {
					break
				}
				if err != nil {
					// 第二层错误：流式读取阶段的错误
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
```

### 5.3 运行效果

执行：

```bash
go run .
```

然后你可以像这样连续提问：

```text
you> 你好，解释一下 Eino 里的 Agent 是什么？
assistant> ...
you> 再用一句话总结一下
assistant> ...
```

注意，这里有一个非常重要但特别容易忽略的事实：

> 这个 Demo 的多轮能力，来自调用方维护 `history`，不是 Runner 在背后偷偷帮你做了记忆。

也就是说，程序一退出，这段对话就没了。

所以它是多轮。
但还不是持久化记忆。

这个边界必须分清。

## 6. 这份代码到底在做什么？

现在我们把上面的完整代码拆开，只看最关键的几步。

### 6.1 第一步：先初始化 `ChatModel`

```go
cm, err := qwen.NewChatModel(ctx, &qwen.ChatModelConfig{
    BaseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1",
    APIKey:  mustEnv("DASHSCOPE_API_KEY"),
    Model:   envOrDefault("QWEN_MODEL", "qwen3.5-flash"),
})
```

`qwen.NewChatModel` 做的事很简单：

- 创建千问模型客户端
- 让它以 Eino 的 `ChatModel` 接口形态暴露出来

到这一步为止，你还只是有了一个组件。
它能调用模型，但还没有变成 Agent。

### 6.2 第二步：把 `ChatModel` 提升成 `ChatModelAgent`

```go
agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
    Name:        "Ch02ConsoleAgent",
    Description: "A minimal ChatModelAgent for console multi-turn chat.",
    Instruction: "你是一个简洁、专业的 Eino 学习助手。",
    Model:       cm,
})
```

这一层最重要的不是字段本身，而是角色变化。

在这之前，你拿到的是一个“模型组件”。
在这之后，你拿到的是一个“可被 Runner 执行的 Agent”。

这里的 `Instruction` 可以理解成系统级约束。
它不是用户输入。
它是在定义这个 Agent 的行为风格。


### 6.3 第三步：用 `Runner` 统一驱动 Agent

```go
runner := adk.NewRunner(ctx, adk.RunnerConfig{
    Agent:           agent,
    EnableStreaming: true,
})
```

这一步之后，执行入口就统一了。

后面不管你这个 Agent 是最简单的 `ChatModelAgent`，还是以后更复杂的 Workflow / Supervisor，本质上都能被 Runner 这一层驱动。

这个意义，在最小 Demo 里可能不够显眼。
但一旦系统扩起来，统一执行入口会非常重要。

### 6.4 第四步：多轮对话其实是调用方维护 `history`

最关键的一段代码就是这里：

```go
history = append(history, schema.UserMessage(line))
content, err := collectAssistantFromEvents(runner.Run(ctx, history))
history = append(history, schema.AssistantMessage(content, nil))
```

这三行里，藏着官方第二章最核心的事实：

**第一，用户输入通过 `schema.UserMessage` 变成消息对象。**

这不是普通字符串。
它是有角色的消息。

**第二，`runner.Run(ctx, history)` 传入的是整段历史消息。**

这意味着：

> 这次执行能否“记住上文”，取决于你有没有把历史消息一起传进去。

**第三，assistant 回复必须显式 append 回 `history`。**

如果你只记录用户输入，不记录模型回复，那下一轮上下文就是残缺的。

所以多轮对话最本质的机制并不神秘。
就是：

- 用户消息进 history
- 把完整 history 交给 Agent 跑一轮
- 把 assistant 回复再写回 history

没有 tools 的情况下，这里还有一个必须明确写出的技术事实：

> 一次 `runner.Run()`，本质上只完成一轮模型调用。

多轮不是一次 `Run()` 自动自己循环出来的。
而是调用方在外层 for 循环里一轮轮驱动出来的。

### 6.5 第五步：`collectAssistantFromEvents` 才是理解 `AgentEvent` 的关键

很多人看代码时，会把注意力放在 `NewChatModelAgent()` 或 `NewRunner()` 上。
但真正把事件流消费逻辑讲清楚的，是这个函数。

```go
func collectAssistantFromEvents(events *adk.AsyncIterator[*adk.AgentEvent]) (string, error)
```

这个签名本身就已经说明了两件事：

- 输入不是字符串，而是 `*adk.AsyncIterator[*adk.AgentEvent]`
- 输出才是我们最终想要落盘或回灌的 assistant 文本

它内部主要做了四件事。

**第一，循环调用 `events.Next()`。**

这表示你在逐个消费 Agent 事件，而不是一次性拿最终答案。

**第二，先判断 `event.Err`。**

这处理的是 AgentEvent 这一层已经暴露出来的错误。

**第三，拿到 `event.Output.MessageOutput`。**

这说明当前事件里真的带了消息输出。

**第四，区分流式和非流式。**

- 如果是流式，就继续从 `MessageStream.Recv()` 一帧一帧读
- 如果不是流式，就直接读取完整消息

这就是为什么前面我一直强调：

> Agent 的输出不是“一个字符串”，而是“需要被消费的一段事件流”。



## 7. 需要避开的三个坑
说到这里，相信大家对概念理解已经差不多了。
但真正上手时，最容易混掉的还是下面这三件事。
### 7.1 多轮不等于记忆

本篇代码里有 `history`<small>（存储上下文记忆的切片数组）</small>，所以程序在当前进程里当然能连续聊天。

但这不等于你已经做了 Memory。

只要进程退出：

- `history` 就没了
- 会话 ID 也没了
- 下次无法恢复上一次对话

所以多轮和记忆不是一个词。

- 多轮关注“这一轮能不能带上上一轮上下文”
- 记忆关注“这段上下文能不能脱离当前进程独立存在”

这也是为什么下一章要单独讲 `Memory / Session`。

### 7.2 `Runner` 不替你保存上下文

很多人看到 `Runner`，会下意识把“执行”和“记忆”混在一起。

但 `Runner` 负责的是执行流程，不是状态托管。

它不会替你：

- 自动保存历史
- 自动恢复会话
- 自动管理 session id

在本篇这个 Demo 里，谁维护上下文？

答案非常朴素：

> 就是你自己的 `history []*schema.Message`。

顺带再补一句很容易漏掉的：

> `runner.Run()` 返回的 `*adk.AsyncIterator[*adk.AgentEvent]` 是一次性的，消费完就结束，不能拿来重复读取。

### 7.3 `event.Err` 和流读取错误不是一回事

这是最容易在排障时把人带沟里的点。

很多人只写：

```go
if event.Err != nil {
    return event.Err
}
```

然后就觉得错误处理完整了。

其实并没有。

如果 `event.Output.MessageOutput` 是流式输出，那么真正的错误还可能发生在：

```go
frame, err := mv.MessageStream.Recv()
```

也就是说：

- `event.Err` 处理的是事件层错误
- `Recv()` 返回的 `err` 处理的是流消费阶段错误

这两个都得看。

如果你只查一个地方，很多“明明前面没报错，为什么最后还是失败”的问题就解释不通。

### 7.4 为啥要引入 `AgentEvent`

因为一旦进入 Agent 视角，“回复内容”就不再是唯一输出了。

未来还可能表达：
- 工具调用过程
- 中断信号
- 状态迁移
- 恢复点

## 8. 本章小结
如果只看功能效果，这一章做的事情很简单。

无非就是：

- 用户输入一句话
- 模型回复一句话
- 再带着上下文继续聊下去

但如果只看到这层，你就会低估本章节真正的价值。

因为本章的目的是让你从“`会调模型`”到“`会理解 Agent 运行时`”。

我真正想带给大家的是下面这套认知：

- `ChatModel` 是组件，负责模型调用能力
- `ChatModelAgent` 是 Agent，把模型能力提升成统一执行抽象
- `Runner` 是执行入口，负责驱动 Agent 跑起来
- `AgentEvent` 是输出单元，让执行过程能被按事件流消费
- 多轮对话靠调用方维护 `history`，不是 Runner 自动记忆

如果你把这些边界吃透了，后面再看：

- `Memory / Session`
- `Tool`
- `Callback / Trace`
- `Interrupt / Resume`
- `WorkflowAgent`

你会发现很多概念一下就落地了。

因为你已经不再只是把 Eino 当成“一个能调模型的 Go 包”。
你开始真正从运行时角度去理解它。

如果要用一句话收尾：

> 本篇博客不是在教你“再写一个聊天 Demo”，而是在教你：怎么把“会调模型”这件事，升级成“会理解 Agent 怎么运行”。

所以我个人认为，`ChatModelAgent / Runner / AgentEvent` 是 Eino 学习路径里非常关键的一站。

它不是终点。
但它决定了你后面看 ADK 时，是在背 API，还是在真正理解 Agent 的执行骨架。

在结尾处，补一张本篇博客实战项目的运行视角图：
```txt
调用方 -> Runner -> Agent -> AgentEvent 流 -> 调用方消费
                  ↑
               history 只是输入的一部分
```

---

## 发布说明

- GitHub 主文：[AI大模型落地系列：一文读懂 ChatModelAgent、Runner、AgentEvent（Console 多轮）](./02-ChatModelAgent、Runner、AgentEvent（Console多轮）.md)
- CSDN 跳转：[AI大模型落地系列：一文读懂 ChatModelAgent、Runner、AgentEvent（Console 多轮）](https://zhumo.blog.csdn.net/article/details/159468400)
- 官方文档：[Eino 第二章：ChatModelAgent、Runner、AgentEvent](https://www.cloudwego.io/zh/docs/eino/quick_start/chapter_02_chatmodelagent_runner_agentevent/)
- 最新版以 GitHub 仓库为准。

