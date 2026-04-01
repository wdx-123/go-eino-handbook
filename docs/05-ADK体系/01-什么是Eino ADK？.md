# AI 大模型落地系列｜Eino ADK体系篇：什么是 Eino ADK？

> GitHub 主文：[当前文章](./01-什么是Eino ADK？.md)
> CSDN 跳转：[AI 大模型落地系列｜Eino ADK体系篇：什么是 Eino ADK？](https://zhumo.blog.csdn.net/article/details/159656025)
> 官方文档：[Eino ADK 概述](https://www.cloudwego.io/zh/docs/eino/core_modules/eino_adk/agent_preview/)
>
> 最新版以 GitHub 仓库为准，CSDN 作为分发入口，官方文档作为权威参考。

**一句话摘要**：从概念总览切入，把 ADK 放回 Agent 开发套件而不是单个接口的层级来理解。
**适合谁看**：已经接触过基础组件，准备系统进入 Agent 开发体系的读者。
**前置知识**：Chain / Graph 基础、ChatModelAgent、Runner 基础
**对应 Demo**：[官方 ADK 示例（本仓后续补充同主题 demo）](https://github.com/cloudwego/eino-examples/tree/main/adk)

**面试可讲点**
- 能说明 ADK 不是某个 Agent 实现，而是一整套 Agent 开发与扩展能力集合。
- 能把 ADK 中的 Agent 抽象、协作、Runner、扩展点串成体系。

---
> **ADK是“构建单 Agent 和多 Agent 系统的一整套框架”**

如果之前你看过 Eino ADK 官方文档，就会有一种很真实的感觉：

名词我都见过。

`ChatModelAgent`、`Workflow Agents`、`Supervisor`、`Plan-Execute`、`Agent Runner`，这些词单独看都不陌生。

可如果现在关掉浏览器，自己讲一遍：

- `Eino ADK` 到底是什么
- `Agent` 为什么是它的核心抽象
- 几类 Agent 到底是什么关系
- 多 Agent 协作到底在协作什么
- 我第一次真正上手，应该从哪里开始

估计很多人还是很蒙的。


所以本篇文章，将会围绕以下 6点 讲清楚：

1. `Eino ADK` 是什么
2. `Agent` 是什么
3. `ChatModel Agent / Workflow Agents / Custom Agent / Built-in Multi-Agent` 分别是什么
4. 多 Agent 协作到底在协作什么
5. 第一个最小可运行入口应该怎么搭
6. 看完这篇以后，下一步该读什么


## 1. 为啥初学者会觉得 ADK 很高深

 Eino ADK 天然就横跨了几层不同的问题：

- 有一层在回答：什么叫 Agent
- 有一层在回答：多个 Agent 怎么协作
- 有一层在回答：哪些模式是开箱即用的
- 还有一层在回答：Runner 怎么把 Agent 真的跑起来

如果你第一次接触时，直接把这些内容混在一起看，脑子里就很容易形成一种误解：

“`ChatModelAgent`、`SequentialAgent`、`Supervisor`、`Plan-Execute` 不都是 Agent 吗？那我是不是直接挑一个最厉害的就行了？”

问题就在这里。

它们都和 Agent 有关。
但它们不是同一层的概念。

有的是**实现一种 Agent**。
有的是**组合多个 Agent**。
有的是**把多个基础能力封装成成熟范式**。
还有的是**负责运行 Agent 的执行器**。

所以本篇文章，将会带你建立对整个体系的基础认知，并带给你一个基础小demo。

这样你后面再看 `Workflow Agents`、`Supervisor`、`Plan-Execute`，就不会觉得自己手足无措。

## 2. 什么是 Eino ADK

先给一句最短的定义。

> `Eino ADK` 是 Eino 提供的 Go 语言 Agent / Multi-Agent 开发框架。

它参考了 Google-ADK 的设计，但不是简单照搬概念。
它真正想解决的是：

> 当你开始写 Agent，而且不止一个 Agent 时，如何把“运行、协作、上下文、治理”这些问题统一起来。

如果只说“它是一个 Agent 框架”，其实还不够全面。

更准确一点，你可以把它理解成：

- 它给你一个统一的 Agent 抽象
- 它给你多 Agent 协作时的通用原语
- 它给你几种开箱即用的协作范式
- 它还给你运行时能力，比如 `Runner`、中断恢复、切面能力等

官方在概述页里强调的几个关键词，其实非常关键：

- 上下文传递
- 事件流分发与转换
- 任务控制权转让
- 中断与恢复
- 通用切面

这几个词合在一起看，你就会发现：

`ADK` 不是“给模型外面再包一层壳”。

它更像是一个 **Agent 运行时和协作框架**。

这点后端开发者应该可以更敏感的感受到：

- Eino 的 Components 层更像“零件层”
- ADK 更像“让这些零件长成会运行、会协作、可治理的智能体系统”

所以它适合的就不只是“能聊天的 Agent”。

还包括：

- 对话型智能体
- 非对话型智能体
- 多步骤任务型智能体
- 工作流式智能体
- Multi-Agent 协同系统

官方概述页里给了一个总览图，先看这个图，你会更容易理解 ADK 到底在体系里的哪个位置：

![外链图片转存失败,源站可能有防盗链机制,建议将图片保存下来直接上传](https://i-blog.csdnimg.cn/direct/a391587d4bc14fc18ce2d54c17086f6f.png)


> 图源： 这是我从 CloudWeGo 官方文档扒拉出来的。

先别急着把图里每个词都吃透。

你只要先抓住一个核心结论：

> ADK 的目标，不是让你多学几个模式名。
> 而是让你围绕 Agent 抽象，把单体 Agent、多 Agent 协作和运行时能力串成一套完整开发方法。

## 3. 什么是 Agent，以及它为什么是 ADK 的核心抽象

官方给的定义很朴素且准确：

> Agent 是一个独立的、可执行的智能任务单元。

大家可以把它先想象成一个“有明确身份、有明确职责、能被调起来执行的智能体”。

只要一个场景需要和大语言模型交互，它通常都可以被抽象成 Agent。

例如：

- 一个查询天气的 Agent
- 一个安排会议的 Agent
- 一个回答特定领域知识的 Agent

### 3.1 为什么 ADK 要先定义 `Agent` 接口

因为没有统一抽象，后面的协作、组合、Runner、Interrupt、Callback 都没法成立。

Eino ADK 把 Agent 的基础接口定义成这样：

```go
type Agent interface {
    Name(ctx context.Context) string
    Description(ctx context.Context) string
    Run(ctx context.Context, input *AgentInput, opts ...AgentRunOption) *AsyncIterator[*AgentEvent]
}
```

这 3 个方法里，最值得注意的是下面这层含义：

- `Name`：这个 Agent 叫什么，它的身份标识是什么
- `Description`：这个 Agent 会什么，其他 Agent 怎么判断要不要找它协作
- `Run`：这个 Agent 怎么真正被运行起来

所以 Agent 不是 Prompt 的别名。

它至少同时包含了三件事：

- 身份
- 职责
- 执行入口

### 3.2  三个基础方法：`AgentInput`、`AgentEvent`、`Runner` 

很多人第一次卡住，不是在 `Agent` 三个方法本身。

而是在这几个配套概念：

| 名词 | 你可以先怎么理解 |
| --- | --- |
| `AgentInput` | 这次要交给 Agent 的任务材料。默认最重要的是 `Messages`，也就是消息、上下文、历史、背景数据。 |
| `AgentEvent` | Agent 运行过程中吐出来的事件。不是只返回最终字符串，而是把执行过程和结果按事件流交给调用方。 |
| `Runner` | Agent 的执行器。真正把 Agent 跑起来，并负责很多运行时能力。 |

比如 `AgentInput` 的核心定义，官方抽象页里写得很直接：

```go
type AgentInput struct {
    Messages        []Message
    EnableStreaming bool
}
```

这说明一个很重要的事实：

> Agent 的输入并不只是“一句话”。
> 它更像一份任务上下文。

而 `AgentEvent` 为什么重要？

因为 ADK 不是把 Agent 看成“同步返回一个字符串的函数”。
它把 Agent 看成“会在运行中持续产生事件的对象”。

这也是为什么 `Run()` 的返回值不是 `string`，而是 `AsyncIterator[*AgentEvent]`。

### 3.3  `ChatModelAgent` 为何重要

在 ADK 里，`ChatModelAgent` 是最关键、也最适合作为第一站的 Agent 实现。

原因很简单：

- 它直接封装了和大语言模型的交互逻辑
- 它本身就是一个“会思考、会生成、能调用工具”的 Agent
- 你第一次上手，最容易从它开始建立直觉

可以先把它理解成：

> 用 LLM 做“大脑”的 Agent 实现。

### 3.4 为什么 `Agent Runner` 不该被忽略

很多人会把注意力全放在 Agent 身上，然后忽略 `Runner`。

但官方文档里说得很清楚：

> Runner 是 Eino ADK 中负责执行 Agent 的核心引擎。
> 任何 Agent 都应通过 Runner 来运行。

而且只有通过 `Runner` 跑起来时，你才能真正用到：

- 多 Agent 协作过程中的上下文管理
- 中断与恢复
- 切面机制
- Context 环境预处理

所以第一次学 ADK 时，一定要先把握住一个关系：

> `Agent` 是任务单元。
> `Runner` 是执行器。

二者缺一不可。

## 4. ADK 的四类基础扩展与封装关系，到底该怎么理解

这部分是很多人最容易略过去，但其实最该慢下来看的一段。

因为它在告诉你：
> ADK 围绕 Agent 抽象，至少有四种不同层次的能力块。


| 类别 | 你可以先把它理解成 | 核心 | 典型代表 | 更适合干什么 |
| --- | --- | --- | --- | --- |
| `ChatModel Agent` | 用 LLM 做大脑的 Agent | 推理、生成、工具调用 | `ChatModelAgent` | 单 Agent 推理、ReAct、动态决策 |
| `Workflow Agents` | 预先写好流程的 Agent 组合器 | 顺序 / 循环 / 并发 | `SequentialAgent`、`LoopAgent`、`ParallelAgent` | 结构化编排、固定流程 |
| `Custom Logic` | 你自己实现的 Agent | 自定义代码与定制逻辑 | `type MyAgent struct{}` | 官方预置能力不够时的定制需求 |
| `EinoBuiltInAgent` | 开箱即用的 Multi-Agent 范式封装 | 基于前几类能力做出的成熟模式 | `Supervisor`、`Plan-Execute` | 更复杂的协作与问题求解 |

如果你非要把它再压缩成一句话，我会这么说：

- `ChatModel Agent` 是会“想”的 Agent
- `Workflow Agents` 是会“排流程”的 Agent
- `Custom Agent` 是你自己“写”的 Agent
- `Built-in Agent` 是官方帮你“封装好范式”的 Agent

官方 Quickstart 也给了一个分类关系图：

![外链图片转存失败,源站可能有防盗链机制,建议将图片保存下来直接上传](https://i-blog.csdnimg.cn/direct/fd3861fa306b4add9a6768c17ed130ba.png)


> 图源： 这是我从 CloudWeGo 官方文档扒拉出来的。

### 4.1 为什么这四类东西不能混着看

因为它们虽然都围绕 Agent 展开，但解决的问题根本不同。




比如：

- `ChatModelAgent` 关注的是“一个 Agent 如何自己思考和调用工具”
- `Sequential / Loop / Parallel` 关注的是“多个 Agent 如何按规则协作”
- `Supervisor / Plan-Execute` 关注的是“复杂协作范式怎么开箱即用”

你如果把这些东西都当成“同一层菜单”，就很容易出现两种误解：

- 明明只是要一个会思考的 Agent，却上来就找 `Plan-Execute`
- 明明只是固定三步流程，却非要套 `Supervisor`


提醒：

> 本篇说的 `Workflow Agents`，是 ADK 里的 Agent 级协作抽象。
> 它和 Eino Compose 层 `Chain / Graph / Workflow` 那套字段映射与节点编排，不是同一个层级。
> （一个是多个Agent之间的协作，另一个是节点之间的编排）

## 5. 多 Agent 协作到底在说什么

这也是 ADK 最容易被说得很玄、但其实完全可以讲明白的一部分。

很多人一听“多 Agent 协作”，脑子里就只有一句话：

“几个 Agent 一起干活。”

这当然没错。

但如果只停在这句话，后面你还是不知道 ADK 到底在设计什么。

这其实可以从三个维度来讲这件事：

- 协作方式
- 上下文策略
- 决策自主性

### 5.1 协作方式：
<small>`Transfer` 和 `AgentAsTool`</small>

先看最关键的一层。

ADK 里两个最基础的协作动作，可以先这么理解：

| 协作方式 | 新手能先怎么理解 | 什么时候更像它 |
| --- | --- | --- |
| `Transfer` | 当前 Agent 把任务转交给另一个 Agent，自己退出当前这轮 | 像“任务移交” |
| `ToolCall(AgentAsTool)` | 当前 Agent 把另一个 Agent 当成工具调一下，拿到结果后自己继续 | 像“调用一个子能力” |

这个区别非常重要。

因为很多人第一次上手时，会误把“找别的 Agent 帮忙”和“把任务完全转给别的 Agent”混成一回事。

### 5.2 上下文策略：
<small>为什么多 Agent 不只是“把消息丢过去”</small>

官方文档里强调了两种核心上下文策略：

| 上下文策略 | 新手理解 |
| --- | --- |
| 上游 Agent 全对话 | 当前 Agent 直接看到上游 Agent 的完整历史与事件结果 |
| 全新任务描述 | 不直接传完整历史，而是把上游结果压缩成一份新的任务摘要，再交给下游 Agent |

如果你再把官方协作文档继续往下看，会看到 ADK 还把上下文传递拆成了两个核心机制：


- `History`（<small>更像“运行过程中的对话与事件历史”</small>）
- `SessionValues`（<small>更像“跨 Agent 共享的结构化状态”</small>）

其中 `History` 对新手尤其重要，因为它解释了很多人第一次看多 Agent 时的一个疑问：

> 为什么后一个 Agent 会“知道”前一个 Agent 干了什么？

答案不是魔法。

而是：

> 前面 Agent 产生的 `AgentEvent` 会进入 History，后面的 Agent 构造 `AgentInput` 时可以读到这些历史。

### 5.3 决策自主性：
<small>谁在决定下一个 Agent 是谁</small>

它本质上只是在区分两件事：

| 决策方式 | 新手理解 |
| --- | --- |
| 自主决策 | Agent 自己决定要不要找谁协作 |
| 预设决策 | 开发者提前把执行顺序写死 |

这个维度一旦加进来，你就更容易理解：

- `ChatModelAgent` / `SubAgents` 往往更接近“自主决策”
- `Sequential / Parallel / Loop` 更接近“预设决策”

### 5.4 把组合原语放在一起看，就清楚多了

下面这张表，是我按照官方协作文档的结构，专门给新手重写的一版：

| 组合原语 | 你可以先怎么理解 | 协作方式 | 上下文 | 决策方式 |
| --- | --- | --- | --- | --- |
| `SubAgents` | 父 Agent 带一组子 Agent，自主决定是否移交任务 | `Transfer` | 上游 Agent 全对话 | 自主决策 |
| `Sequential` | 多个 Agent 按顺序一个接一个执行 | `Transfer` | 上游 Agent 全对话 | 预设决策 |
| `Parallel` | 多个 Agent 基于同一输入并发执行 | `Transfer` | 上游 Agent 全对话 | 预设决策 |
| `Loop` | 一组 Agent 按顺序循环执行 | `Transfer` | 上游 Agent 全对话 | 预设决策 |
| `AgentAsTool` | 把一个 Agent 转成 Tool 给别的 Agent 调用 | `ToolCall` | 全新任务描述 | 自主决策 |

写这张表的目的，不只是帮大家记住概念，更是想建立一个简单的判断：

> 你到底是在做“任务移交”，还是在做“能力调用”？
> 你到底要的是“自主路由”，还是“预设流程”？

只要这两个问题能回答清楚，你后面再看 `Workflow Agents`、`Supervisor`、`Plan-Execute`，理解速度会快非常多。

## 6. `ADK Examples` 案例

官方 Quickstart 里给了很多 examples。

很多人第一时间，会把这些例子当成“代码仓库目录”。

其实更有效的看法是：

> 每个 example 都是在帮使用者建立一种 Agent 模式的直觉。

所以之下的案例，将会带你明白：**每个例子你到底该学什么。**

| 示例 | 你该从它身上学到什么 | 第一次学时的建议 |
| --- | --- | --- |
| `intro/workflow/sequential` | 看清楚顺序接力：一个 Agent 的结果如何成为下一个 Agent 的输入背景 | 最先看，最好 |
| `intro/workflow/loop` | 看清楚反思迭代：为什么“写完再批判再改”天然适合 Loop | 第二个看 |
| `intro/workflow/parallel` | 看清楚并行协作：几个独立分析任务如何同时运行 | 第三个看 |
| `multiagent/supervisor` | 看清楚中心调度：一个总控 Agent 如何挑选专家 Agent | 前三个看懂后再看 |
| `multiagent/layered-supervisor` | 看清楚层级协作：为什么复杂任务会出现多层监督者 | 放在 supervisor 后面看 |
| `multiagent/plan-execute-replan` | 看清楚“计划 - 执行 - 重规划”的长任务闭环 | 先别急着实操，先理解结构 |
| `intro/chatmodel`（书籍推荐） | 看清楚中断恢复、Checkpoint、runner.Query / Resume 的配合 | 当你开始关心运行时治理时再看 |

### 6.1 如果你是第一次学，我建议这样看 examples

第一次上手，不要把 examples 全部平铺打开。

更稳的顺序是：

1. 先看 `sequential`
2. 再看 `loop`
3. 再看 `parallel`
4. 然后再看 `supervisor`
5. 最后再去理解 `plan-execute-replan`

这个顺序的本质不是“由简单到复杂”这么空泛。

而是：

- 先建立 `Workflow Agents` 的直觉
- 再看更高级的 Multi-Agent 协作范式

如果这条顺序不安排好，很多人第一次看 `Supervisor` 或 `Plan-Execute`，会直接觉得：

“这不就是又包了一层 Agent 吗？”

但其实它们都是在前面基础之上更高层的封装。

## 7. 最小 runnable 入口：
先跑通你的第一个 `ChatModelAgent + Runner`

已讲完基础体系，终于该回到“第一段代码”了。

注意，这里我故意不直接上 `Sequential`、`Loop` 或 `Supervisor`。是因为：

> 你第一次要跑通的，不是 Multi-Agent。
> 而是 ADK 里最小、最完整的执行骨架。

也就是：

`ChatModel -> ChatModelAgent -> Runner`

![外链图片转存失败,源站可能有防盗链机制,建议将图片保存下来直接上传](https://i-blog.csdnimg.cn/direct/2754a342e3a544498bed7294de4bd71f.png)


### 7.1 安装依赖

```bash
mkdir eino-adk-first-card
cd eino-adk-first-card

go mod init eino-adk-first-card
go get github.com/cloudwego/eino@latest
go get github.com/cloudwego/eino-ext/components/model/qwen@latest
```

### 7.2 配环境变量

如果你在 macOS / Linux：

```bash
export DASHSCOPE_API_KEY="你的百炼 API Key"
export QWEN_MODEL="qwen3.5-flash"
```

如果你在 Windows PowerShell：

```powershell
$env:DASHSCOPE_API_KEY="你的百炼 API Key"
$env:QWEN_MODEL="qwen3.5-flash"
```

### 7.3 第一份完整代码

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
	"github.com/cloudwego/eino-ext/components/model/qwen"
)

func main() {
	ctx := context.Background()

	// 默认问题；也支持从命令行覆盖，方便本地调试与演示。
	query := "请用新手能看懂的话，解释一下什么是 Eino ADK。"
	if len(os.Args) > 1 {
		query = strings.Join(os.Args[1:], " ")
	}

	// 初始化底层大模型客户端。
	// 这里使用阿里百炼兼容接口，通过环境变量读取密钥与模型名，避免硬编码敏感信息。
	cm, err := qwen.NewChatModel(ctx, &qwen.ChatModelConfig{
		BaseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1",
		APIKey:  mustEnv("DASHSCOPE_API_KEY"),
		Model:   envOrDefault("QWEN_MODEL", "qwen3.5-flash"),
	})
	if err != nil {
		log.Fatalf("new qwen chat model failed: %v", err)
	}

	// 将底层模型封装为一个可执行的 Agent。
	// Name / Description 用于标识与协作；Instruction 用于约束该 Agent 的行为风格。
	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "ADKIntroAgent",
		Description: "负责向新手解释 Eino ADK 的基础概念",
		Instruction: "你是一个面向 Go 新手的 Eino ADK 讲解助手。先给一句结论，再给三点解释，控制在 300 字以内。",
		Model:       cm,
	})
	if err != nil {
		log.Fatalf("new chat model agent failed: %v", err)
	}

	// Runner 是 Agent 的统一执行入口。
	// 这里关闭流式输出，改为按事件迭代读取完整结果。
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming: false,
	})

	fmt.Printf("user> %s\n\n", query)

	// 发起一次查询，并消费 Agent 返回的事件流。
	if err := printAssistantOutputs(runner.Query(ctx, query)); err != nil {
		log.Fatalf("run agent failed: %v", err)
	}
}

// printAssistantOutputs 负责从事件流中提取 assistant 消息并打印。
// 这里只关心最终可读的消息内容，忽略中间无效事件或非 assistant 输出。
func printAssistantOutputs(events *adk.AsyncIterator[*adk.AgentEvent]) error {
	for {
		event, ok := events.Next()
		if !ok {
			// 事件流结束，说明本次执行完成。
			return nil
		}

		// 运行过程中的错误会挂在事件上，需要显式向上返回。
		if event.Err != nil {
			return event.Err
		}

		// 非消息类输出或空输出直接跳过。
		if event.Output == nil || event.Output.MessageOutput == nil {
			continue
		}

		mv := event.Output.MessageOutput

		// 只处理 assistant 角色的消息。
		// 某些场景下 Role 可能为空，这里一并兼容。
		if mv.Role != schema.Assistant && mv.Role != "" {
			continue
		}
		if mv.Message == nil {
			continue
		}

		content := strings.TrimSpace(mv.Message.Content)
		if content == "" {
			continue
		}

		fmt.Printf("assistant>\n%s\n", content)
	}
}

// mustEnv 读取必填环境变量；缺失时直接终止进程。
// 适用于 API Key、数据库地址等启动必需配置。
func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("%s is empty", key)
	}
	return v
}

// envOrDefault 读取可选环境变量；若未配置则回退到默认值。
// 适用于模型名、超时、开关等可提供默认行为的配置项。
func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
```

### 7.4 运行

```bash
go run . -- "请解释一下为什么 Eino ADK 不只是几个 Agent 模式名"
```

你大概会看到这样一段输出：

```text
user> 请解释一下为什么 Eino ADK 不只是几个 Agent 模式名

assistant>
...
```

### 7.5 这份最小代码，真正让你建立的是什么

第一次跑通时，你最该看见的不是“模型回复成功了”。

而是下面这条骨架终于成型了：

1. `qwen.NewChatModel`：先有一个可调用的大模型
2. `adk.NewChatModelAgent`：再把模型包成一个可执行的 Agent
3. `adk.NewRunner`：再交给 Runner 驱动执行
4. `runner.Query(...)`：最后发起一次真正的 Agent 运行

也就是说，这段代码不是在教你“怎么问模型一个问题”。

它是在教你：

> ADK 里第一个能跑起来的 Agent，到底是怎么被组装出来的。

## 8. What’s Next：这篇之后，你该怎么继续学 ADK

你可以把这个当成 “ADK 后续学习树”。

![外链图片转存失败,源站可能有防盗链机制,建议将图片保存下来直接上传](https://i-blog.csdnimg.cn/direct/edbe1493e1f64459818a704507111f85.png)


> 图源： 这是我从 CloudWeGo 官方文档扒拉出来的。

### 8.1 如果你是第一次学，我建议按这条顺序往下走

先看懂整体目录：

1. `Quickstart`
2. `概述`
3. `Agent 抽象`
4. `Agent 协作`
5. `ChatModelAgent`
6. `Workflow Agents`
7. `Agent Runner 与扩展`


再往后，如果你要继续深入，再接着看：

8. `Supervisor Agent`
9. `Plan-Execute Agent`
10. `Agent Callback`
11. `Interrupt / Resume / HITL`




## 参考资料

- [Eino ADK: 概述](https://www.cloudwego.io/zh/docs/eino/core_modules/eino_adk/agent_preview/)
- [Eino ADK: Quickstart](https://www.cloudwego.io/zh/docs/eino/core_modules/eino_adk/agent_quickstart/)
- [Eino ADK: Agent 抽象](https://www.cloudwego.io/zh/docs/eino/core_modules/eino_adk/agent_interface/)
- [Eino ADK: Agent 协作](https://www.cloudwego.io/zh/docs/eino/core_modules/eino_adk/agent_collaboration/)
- [Eino ADK: Agent Runner 与扩展](https://www.cloudwego.io/zh/docs/eino/core_modules/eino_adk/agent_extension/)
- [Eino-examples/adk](https://github.com/cloudwego/eino-examples/tree/main/adk)

---

## 发布说明

- GitHub 主文：[AI 大模型落地系列｜Eino ADK体系篇：什么是 Eino ADK？](./01-什么是Eino ADK？.md)
- CSDN 跳转：[AI 大模型落地系列｜Eino ADK体系篇：什么是 Eino ADK？](https://zhumo.blog.csdn.net/article/details/159656025)
- 官方文档：[Eino ADK 概述](https://www.cloudwego.io/zh/docs/eino/core_modules/eino_adk/agent_preview/)
- 最新版以 GitHub 仓库为准。

