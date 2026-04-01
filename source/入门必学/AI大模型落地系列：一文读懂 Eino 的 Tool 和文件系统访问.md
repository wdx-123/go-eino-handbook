声明：本文数据源于官方文档与官方实现[[Eino]](https://www.cloudwego.io/zh/docs/eino/quick_start/chapter_04_tool_and_filesystem/)。
@[TOC](如何让你的 Eino Agent 长出双手？一文读懂 Tool 和文件系统访问)

上一篇，我们把 Eino 的 `ChatModel` 和 `Message` 跑通了。
但很多人到这一步，会误以为自己已经摸到了 Agent 开发的门槛。
其实没有。
因为会对话，不等于会执行。
一个只能生成文本的 Agent，在工程上还远远谈不上“能干活”。
真正的分水岭，往往是 `Tool`。

上一篇解决模型调用边界，这一篇解决执行能力边界。放在 Eino 里，这个执行能力最直接的落点，就是给 `Agent` 接上 `Tool`、接上文件系统、接上 `DeepAgent`。如果还停留在“输入一段 prompt，输出一段文本”，那你写出来的东西更像一个高级聊天框，而不是一个真正能落地的 Agent。

## 1. 为什么跑通 ChatModel 以后，你的 Agent 还是只会聊天

很多 Go 后端工程师第一次接 Eino，最容易产生一个错觉：

“我已经能把模型调通了，也能拿到回复了，那我是不是已经在做 Agent 了？”

这话只对了一半。

`ChatModel` 解决的是“怎么和模型说话”，`Message` 解决的是“上下文怎么表达”。但这两个边界打通之后，你得到的，本质上还是一个**只能生成文本**的能力。

它能回答问题。
它能续写内容。
它甚至能看起来像是在“思考”。

但它依然：

- 读不了文件
- 查不了目录
- 访问不了外部资源
- 执行不了真实动作

很多所谓的 Agent 项目，本质上只是把 ChatModel 外面再包了一层壳。

这就像什么？

像你写了一个返回 JSON 的接口，但接口后面没连数据库、没连缓存、没连业务系统。它当然“能响应”，但你很难说它真的“有业务能力”。

所以继上一篇文章之后，`ChatModel` 真正该补上的，不是更花哨的编排，而是让模型先有能力碰到外部世界。而这个入口，就是 `Tool`。

## 2. Tool 到底是什么

很多人一看到 `Tool` 这个词，会下意识把它理解成“插件”。

这个理解不算错，但还不够准。

在 Eino 里，`Tool` 更像一层统一的**外部能力声明**。模型不需要知道你的文件读取逻辑怎么写、shell 怎么执行、数据库怎么连，它只需要知道：

- 这个工具叫什么
- 它是干什么的
- 它收什么参数
- 它调用后会返回什么结果

从职责上看，可以简单分成三层：

- `BaseTool`：提供工具元信息，让模型知道“这里有个工具可用”
- `InvokableTool`：一次性执行工具，输入通常是 JSON 参数，输出是字符串结果
- `StreamableTool`：流式执行工具，适合 shell 这类会持续返回内容的场景
```go
// BaseTool 提供工具的元信息,ChatModel 使用这些信息决定是否以及如何调用工具
type BaseTool interface {
    Info(ctx context.Context) (*schema.ToolInfo, error)
}

// InvokableTool 是可以被 ToolsNode 执行的工具
type InvokableTool interface {
    BaseTool
    // InvokableRun 执行工具,参数是 JSON 编码的字符串,返回字符串结果
    InvokableRun(ctx context.Context, argumentsInJSON string, opts ...Option) (string, error)
}

// StreamableTool 是 InvokableTool 的流式变体
type StreamableTool interface {
    BaseTool
    // StreamableRun 流式执行工具,返回 StreamReader
    StreamableRun(ctx context.Context, argumentsInJSON string, opts ...Option) (*schema.StreamReader[string], error)
}

```

> 对模型而言，Tool 不是一段代码，而是一份可以被选择调用的说明书（协议）。

这也是为什么 Tool 会成为 Agent 和普通聊天程序之间的分水岭。模型一旦具备 Tool Calling，它就不再只能“说”，而是可以“先调工具，再组织答案”。

## 3. 如何为Agent装上操作文件系统能力

如果你是做 ChatWithDoc、代码问答、项目助手这类场景，最可信的资料是什么？

不是二手教程。
不是群聊截图。
也不是别人写的“速通笔记”。

最可信的，其实是项目自己的源码、注释和示例。

这也是为什么这将成为Agent的一次飞跃性进步。因为一旦 Agent 能读目录、读文件、grep 搜索、按 glob 查找，它就第一次具备了“自己去找依据”的能力。

如果 Agent 连文件都读不了，它通常还没从“聊天程序”跨进“执行程序”。

这里会出现两个容易混的概念。

**第一，`Backend`。**

它是文件系统操作的抽象层，负责定义“列目录、读文件、搜索、写入、编辑”这些能力。

**第二，`LocalBackend`。**

它是 `Backend` 的本地实现，直接访问你机器上的文件系统。你可以把它理解成：

> Eino 没有把“读文件”硬编码在 Agent 里，而是先抽象成 Backend，再给出一个本地版实现。
```go
import localbk "github.com/cloudwego/eino-ext/adk/backend/local"

backend, err := localbk.NewBackend(ctx, &localbk.Config{})
```

之所以这样设计。是因为今天你读的是本地目录，明天就可能换成别的存储后端。抽象先顶上，能力才有复用空间。

另外，`LocalBackend` 还有一个特别值得注意的点：**文件系统工具最好使用绝对路径。** 

## 4. 啥是DeepAgent
咱们先不谈其他，你先看看这些import导入的包。
```go
import (
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/deep"
	"github.com/cloudwego/eino/schema"
)
```
#### 先了解何为adk
**adk** 可以理解为 Eino 里专门面向 Agent 的基础开发层。你可以认为他是一套针对底层**封装好的接口**。它把 Agent 运行所需的一套底层抽象、接口、事件流和执行机制先封装好，然后对上层的 Agent 实现和业务代码提供统一能力。
#### 水道渠成的deepAgent
`github.com/cloudwego/eino/adk/prebuilt/deep` 则是建立在 adk 之上的一个 **开箱即用的**预置 Agent 实现，官方叫 DeepAgents。官方文档也明确说了，它是在 ChatModelAgent 基础上实现的一种现成 agent 方案，你不用自己从零拼提示词、工具和上下文管理，就能直接得到一个可运行的 Agent。
**官方表述：**
![在这里插入图片描述](https://i-blog.csdnimg.cn/direct/d7af48dfa4734e2e85e97d1dd851498d.png)

`DeepAgent` 的优势，在于它把文件系统、命令执行和任务能力抬成了一等配置。你不需要从零拼每一个螺丝，直接把 `Backend` 和 `StreamingShell` 传进去，它就能把相关工具接起来。
注：<small>所谓的一等配置，就是能直接在Config中配置的参数</small>

#### ChatModelAgent与DeepAgent区别

| 能力 | ChatModelAgent | DeepAgent |
| --- | --- | --- |
| 多轮对话 | 支持 | 支持 |
| 自定义 Tool | 需要手动逐个注册 | 可以手动注册，也可以接一级配置 |
| 文件系统访问 | 需要自己创建并注册相关 Tool | 配置 `Backend` 后自动接入 |
| 命令执行 | 需要自己额外接入 | 配置 `StreamingShell` 后自动接入 |
| 内置任务管理 | 无 | 默认带 `write_todos` |
| 子 Agent 能力 | 无 | 支持 |

这里最重要的结论其实就一句：

- 纯对话场景，用 `ChatModelAgent`
- 一旦要接文件系统、命令执行、任务规划，就切 `DeepAgent`

官方第四章明确给出了这一组自动注册工具：

- `read_file`
- `write_file`
- `edit_file`
- `glob`
- `grep`
- `execute`

所以，很多 Agent 项目真正的第一步，不是上 Workflow，而是先把 Tool 接进去，先为你的大模型接上双手。

## 5.  跑通一个小 Demo

本demo将会使用：
- `LocalBackend`
- `DeepAgent`
- 千问大模型

你将会使 “Agent 第一次碰到外部世界”。

同样，先准备依赖和环境变量：

```bash
go mod init eino-ch04-demo
go get github.com/cloudwego/eino@latest
go get github.com/cloudwego/eino-ext/components/model/qwen@latest
go get github.com/cloudwego/eino-ext/adk/backend/local@latest

export DASHSCOPE_API_KEY="你的百炼 API Key"
export QWEN_MODEL="qwen3.5-flash"
export PROJECT_ROOT=/path/to/your/project
```

如果你在 Windows PowerShell 下，环境变量改成：

```powershell
$env:DASHSCOPE_API_KEY="你的百炼 API Key"
$env:QWEN_MODEL="qwen3.5-flash"
$env:PROJECT_ROOT="D:\\your\\project"
```

如果不设置 `PROJECT_ROOT`，上面这份代码会默认使用当前工作目录。

然后把下面这份代码保存成 `main.go`：

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

	localbk "github.com/cloudwego/eino-ext/adk/backend/local"
	"github.com/cloudwego/eino-ext/components/model/qwen"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/deep"
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
		Name:           "Ch04ToolAgent",
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

// printEvents 不断消费 Agent 运行产生的事件流，
// 把助手回复、工具调用、工具结果按可读方式打印到终端。
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

		// 实际输出
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

// 拼接成完整string在返回
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
```

直接执行：

```bash
go run . -- "请列出当前目录下的 Go 文件，并读取 main.go 的前 20 行"
```

你会看到控制台里先出现 `tool call`，然后出现 `tool result`，最后才是模型整理后的自然语言回复。

这一步非常关键。因为它说明 Agent 已经不是“凭空回答”，而是在**先找依据，再组织答案**。

## 6. 一次 Tool 调用，在 Eino 里到底怎么走

当用户说“列出当前目录的文件，并读取 main.go”时，Eino 里发生的大致是这件事：

```text
用户提问
  -> 模型判断这不是纯文本回答能解决的问题
  -> 生成 tool call(JSON 参数)
  -> DeepAgent 把调用路由到对应 Tool
  -> Backend/LocalBackend 真正执行文件系统操作
  -> tool result 回到上下文
  -> 模型基于结果生成最终回答
```

这条链一旦跑通，你对 Agent 的理解就会发生变化。

不是“模型突然变聪明了”，而是：

- 模型负责理解问题和决定要不要调工具
- Tool 负责提供能力入口
- Backend 负责把动作真正落到外部世界
- Agent 负责把这一切串起来

这也是为什么 `DeepAgent` 比“单纯会聊天的 ChatModel”更接近工程里的执行型 Agent。

## 7. 一分钟复盘

如果你读完这篇，希望你能收获这些：

- `ChatModel` 解决的是模型调用边界，不是执行能力边界
- `Tool` 是 Agent 第一次真正碰到外部世界的入口
- 文件系统能力之所以重要，是因为源码、注释、示例本身就是最可信的知识源
- 纯对话继续用 `ChatModelAgent`，一旦要接文件系统和命令执行，就该切到 `DeepAgent`


## 参考资料

- Eino 第四章：Tool 与文件系统访问  
  https://www.cloudwego.io/zh/docs/eino/quick_start/chapter_04_tool_and_filesystem/
- Eino DeepAgents 文档  
  https://www.cloudwego.io/zh/docs/eino/core_modules/eino_adk/agent_implementation/deepagents/
- Eino 官方示例 `cmd/ch04/main.go`  
  https://github.com/cloudwego/eino-examples/blob/main/quickstart/chatwitheino/cmd/ch04/main.go
