# AI 大模型落地系列｜Eino ADK体系篇：你对 ChatModelAgent 有了解吗？

> GitHub 主文：[当前文章](./03-你对ChatModelAgent有了解吗？.md)
> CSDN 跳转：[AI 大模型落地系列｜Eino ADK体系篇：你对 ChatModelAgent 有了解吗？](https://zhumo.blog.csdn.net/article/details/159696365)
> 官方文档：[Eino ADK：ChatModelAgent](https://www.cloudwego.io/zh/docs/eino/core_modules/eino_adk/agent_implementation/chat_model/)
>
> 最新版以 GitHub 仓库为准，CSDN 作为分发入口，官方文档作为权威参考。

**一句话摘要**：从 ReAct、Transfer、AgentAsTool、Middleware / Handler 这些扩展点深入理解 ChatModelAgent。
**适合谁看**：已经知道 ChatModelAgent 名字，但还没真正掌握其工程边界的读者。
**前置知识**：什么是 Eino ADK、为什么一定要有 Agent 这层抽象、Tool 调用闭环
**对应 Demo**：[官方 ChatModelAgent 文档与实战示例](https://www.cloudwego.io/zh/docs/eino/core_modules/eino_adk/agent_implementation/chat_model/)

**面试可讲点**
- 能解释 ChatModelAgent 为什么是 Agent 能力的常用承载体，而不是万能 Agent。
- 能区分普通 Tool、Transfer、AgentAsTool、Handler/Middleware 这些扩展位点。

---
> **ChatModelAgent 是一个以 LLM 为决策核心、默认采用 ReAct 式循环来推进任务的 Agent。**

很多人第一次看 `ChatModelAgent`，可能会下意识的认为：

> 不就是 `ChatModel + Instruction + Tools` 吗？

这句话不算错，但不全。



`ChatModelAgent` 在 ADK 里，承担的是“默认思考型 Agent”的角色。它不是单纯帮你调一次模型，而是把模型决策、工具调用、协作跳转、事件输出和扩展钩子，统一规范到一个可运行的 Agent 骨架里。

本篇不在重复 `Runner / Console 多轮` 这种入门动作，而是从以下 6 个更关键的问题入手：

1. `ChatModelAgent` 在 ADK 里到底是什么
2. 它内部为什么是一个 `ReAct` 循环，而不是一次模型调用
3. `ReturnDirectly / Exit / MaxIterations / OutputKey` 这些字段到底解决什么问题
4. `Tool`、`Transfer`、`AgentAsTool` 到底怎么选
5. `Middleware / Handler` 为什么才是工程化分水岭
6. 一个更贴后端场景的 Demo，应该怎么搭

## 1. 为什么很多人会把 `ChatModelAgent` 想简单

很多人一上来就把注意力放在这几个字段上：

- `Instruction`
- `Model`
- `Tools`

然后得出一个很自然的结论：

> 这就是一个“会调模型、也会调工具”的配置对象。


但他的重点其实是另一层：

> `ChatModelAgent` 是 ADK 里最核心、最常用的预构建 Agent 之一，它把“思考 + 决策 + 调工具 + 协作 + 输出事件”规范成了一个统一实现。

也就是说，它不是 `ChatModel` 的语法糖。

它解决的是：当一个 Agent 需要靠 LLM 自己判断下一步该答、该调工具、该转给别人、还是该退出时，系统应该怎么组织这段运行过程。

这也是为什么你会发现：

- 它有 `ReAct` 循环
- 它有 `Transfer`
- 它可以把别的 Agent 当 Tool
- 它有专门的 `Handler`
- 它还要把整个过程输出成 `AgentEvent`

如果只是“模型外面包一层”，根本没必要长出这一整套能力。

## 2. `ChatModelAgent` 在 ADK 里到底是什么

官方定义很直接：

> `ChatModelAgent` 是 Eino ADK 中的一个核心预构建 Agent，它封装了与大语言模型交互、并支持使用工具来完成任务的复杂逻辑。

这句话里最重要的词，不是“模型”，而是“复杂逻辑”。

你可以把 ADK 里的几类 Agent 先粗分一下：

| 类型 | 主要职责 | 决策方式 |
| --- | --- | --- |
| `ChatModelAgent` | 负责思考、推理、工具调用、动态决策 | 由 LLM 决定 |
| `Workflow Agents` | 负责顺序、循环、并行等固定流程 | 由预设流程决定 |
| `Supervisor / Plan-Execute` | 负责多 Agent 协作范式封装 | 仍以内置 ChatModelAgent 为核心 |
| `Custom Agent` | 负责高度定制的执行协议 | 由你自己实现 |

所以 `ChatModelAgent` 的位置，其实非常像默认的“脑子”。

当你的 Agent 需要：

- 根据上下文自行判断下一步动作
- 在回答和工具之间切换
- 在多个 Agent 之间转交任务
- 在运行过程中插入工程逻辑

那它通常就会成为第一个候选。

把这套关系放到运行时视角里，看起来会更清楚：

![外链图片转存失败,源站可能有防盗链机制,建议将图片保存下来直接上传](https://i-blog.csdnimg.cn/direct/03658b1d41bc4e4ab34c45c8b651123e.png)


这张图里最该记住的是两点：

1. `ChatModelAgent` 不等于“模型输出一段话”
2. 它真正对外暴露的是一整段可运行的决策过程

## 3. 其内部本质是一个 `ReAct` 循环

 `ChatModelAgent` 的核心执行模式其实很清楚：它内部走的是 `ReAct`。

其内部是一个循环：

1. 调模型，让模型先做判断
2. 如果模型直接给答案，那就结束
3. 如果模型发起 Tool Call，就执行工具
4. 把工具结果回灌给模型
5. 再让模型决定下一步
6. 直到模型不再需要工具，或者 Agent 被强制结束

这套循环里，用以下四个词能直接对应上：

- `Reason`：模型思考
- `Action`：模型决定调用什么
- `Act`：系统真的去执行动作
- `Observation`：把动作结果喂回去

所以 `ChatModelAgent` 的关键，不在于“它能调工具”。

而在于：

> 它允许模型把一次复杂任务拆成多轮判断，而不是一口气把答案硬生成出来。

这也是它和我们直接手写一段 `ChatModel.Generate(...)` 的根本区别。

### 没有 Tool 时会怎样

可以这么说：

> 如果没有配置工具，`ChatModelAgent` 会退化为一次普通的 ChatModel 调用。

这意味着：

- 不是所有 `ChatModelAgent` 都一定会循环
- 只有当你给了工具、协作能力，或者模型真的产生 Tool Call，它才会进入完整的 `ReAct` 运行形态

### 为什么还需要 `MaxIterations`

`ReAct` 的好处是灵活，风险是兜不住时会一直绕。

所以 `MaxIterations` 本质上是一个保险丝。

默认值是 `20`。超过这个次数还没结束，Agent 会直接报错退出。

这在真实业务里非常有必要。否则你很容易遇到两种问题：

- 模型在几个工具之间来回试探，始终拖沓着
- Prompt 写得含糊，模型不知道该答还是该继续调工具

很多线上“为什么 Agent 一直在调用工具”的问题，本质上都不是框架 bug，而是没有把循环上限和结束策略设计清楚。

## 4. 哪几组配置真正决定了行为
### `Name / Description`

这两个字段经常被初学者轻视。

但实际上它们比你想象的重要。

- `Name` 是 Agent 的身份标识
- `Description` 决定别的 Agent 会不会把任务转给它

尤其在 `Transfer` 场景里，`Description` 不是装饰品，而是模型判断“谁更适合接手这件事”的依据。

### `Instruction / Model`

这两个字段是最直观的：

- `Instruction`：Agent 的系统约束
- `Model`：底层使用哪个 `ChatModel`

但有一点别搞混：

`Instruction` 决定行为风格，`Model` 决定能力底座。

### `ToolsConfig`

这组配置是 `ChatModelAgent` 和普通模型调用真正拉开差距的地方。

其中有两个很关键的扩展字段起到了作用：

- `ReturnDirectly`
- `EmitInternalEvents`

#### `ReturnDirectly`

这个字段的意思是：

> 某些工具一旦被调用成功，就不要再把结果送回模型二次润色了，直接把结果带着返回。

这个能力特别适合两类场景：

- 工具结果本身就是最终答案
- 工具结果本身就是“交接单”“审批单”“跳转结果”，再回模型反而会把结果弄脏

比如这篇后面 demo 里的 `handoff_to_human`，就很适合 `ReturnDirectly`。

#### `EmitInternalEvents`

这个配置只在 `AgentAsTool` 场景里有意义。

默认情况下，当你把一个 Agent 包成 Tool 后，外层只会拿到最终的 ToolResult，看不到内层 Agent 的事件流。

而 `EmitInternalEvents=true` 时，内层 Agent 产生的事件会继续往外透出，调用方就能实时看到里面到底在干什么。

这个能力特别适合：

- 你把一个复杂 Agent 当 Tool 用
- 但又希望前端或调用方还能看到它的实时输出

### `OutputKey`

这个字段很实用：

> 把 Agent 最后一条输出消息，以某个 key 写进 `SessionValues`

如果你的后续 Agent、Workflow、或者外层业务逻辑还要继续消费这次结果，它比你手动到处传字符串干净得多。

### `Exit`

你可以把他当作一个特殊 Tool。

模型调用这个 Tool 并成功执行后，`ChatModelAgent` 会直接退出，效果和 `ReturnDirectly` 很像，但语义更明确：

- `ReturnDirectly` 更像“某个工具调用后直接收口”
- `Exit` 更像“模型自己宣布：到这里结束，把这个最终结果拿出去”

### `ModelRetryConfig`

这是一个典型的工程字段。

它解决的不是“让回答更聪明”，而是“模型调用失败时，系统要不要重试，以及怎么重试”。



> 如果流式响应过程中发生错误，但策略允许重试，调用方读 stream 时会收到 `WillRetryError`。

所以在真实系统里做流式输出时，不能只管 happy path。否则一旦流中途断掉，你都不知道是彻底失败了，还是下一轮马上会补回来。

## 5. `Tool`、`Transfer`、`AgentAsTool` 到底怎么选

这一段是最值得展开讲的地方。

很多人第一次看这三种能力时，会觉得它们都像“把事情交给别人做”。但它们不是一回事。

![在这里插入图片描述](https://i-blog.csdnimg.cn/direct/07897c1fe0bb47faa210ba031c66e9ea.png)


### 普通 `Tool`

适合那种边界特别清晰、输入输出很稳定的能力，比如：

- 查错误码
- 查 runbook
- 算时间
- 调外部 HTTP 接口

它更像函数调用。

### `Transfer`

`Transfer` 的意思不是“调用另一个能力”，而是：

> 当前 Agent 判断，另一个 Agent 更适合接手这件事，于是把任务控制权转过去。

官方页对应的实现机制是：

- 给 `ChatModelAgent` 配置子 Agent
- 框架自动生成一个 `Transfer Tool`
- 模型根据各个 Agent 的 `Description` 决定要不要跳转
- Runner 收到 Transfer Event 后，切到目标 Agent 继续执行

最小示意像这样：

```go
// 创建一个上层 Agent，作为请求分发器使用。
// 它本身由聊天模型驱动，职责是根据用户问题决定该交给谁处理。
supervisor, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
	Name:        "dispatcher", // Agent 名称：运行时用于标识当前 Agent
	Description: "负责分发用户请求", // 描述：帮助上层协作逻辑理解它的职责
	Model:       cm,           // 底层使用的聊天模型
})

// 创建一个子 Agent，专门处理数据库相关问题。
dbExpert, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
	Name:        "db_expert",   // 子 Agent 名称
	Description: "擅长数据库故障排查", // 描述其擅长领域，便于被正确选择
	Model:       cm,            // 同样使用聊天模型驱动
})

// 给 supervisor 挂载可协作的子 Agent。
// 这样 supervisor 在处理请求时，就可以把数据库类问题分发给 dbExpert。
dispatcher, _ := adk.SetSubAgents(ctx, supervisor, []adk.Agent{dbExpert})
```

如果一个问题本来就该交给另一个 Agent 独立负责，那应该优先考虑 `Transfer`，而不是让当前 Agent 硬撑到底。

### `AgentAsTool`

它的语义又不同：

> 我不是把任务彻底交出去，我只是把另一个 Agent 当成一个“高级工具”来用。

什么时候适合这么做？

当被调用的 Agent：

- 不需要完整运行上下文
- 只要一个明确请求参数就能独立完成工作
- 更像一个“复杂工具”而不是一个“新的控制者”

我这里从官方源码 `NewAgentTool(...)` 截取片段举例：

```go
reporterTool := adk.NewAgentTool(ctx, reporterAgent)

agent, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
    Name:        "ops_assistant",
    Description: "负责处理线上故障",
    Model:       cm,
    ToolsConfig: adk.ToolsConfig{
        ToolsNodeConfig: compose.ToolsNodeConfig{
            Tools: []tool.BaseTool{reporterTool},
        },
        EmitInternalEvents: true,
    },
})
```

一句话记忆这三者：

- `Tool`：调用一个函数
- `Transfer`：把控制权交给另一个 Agent
- `AgentAsTool`：把另一个 Agent 当函数来调

## 6. `Middleware / Handler` 才是工程化分水岭

如果说 Tool 解决的是“Agent 能干什么”，那 `Handler` 解决的就是“Agent 在真实系统里怎么管”。

官方页给出的扩展点一共有几层：

- `BeforeAgent`
- `BeforeModelRewriteState`
- `AfterModelRewriteState`
- `WrapModel`
- `WrapInvokableToolCall / WrapStreamableToolCall`

把它们放到一张执行图里，会比只看接口名字更容易懂：

![在这里插入图片描述](https://i-blog.csdnimg.cn/direct/38f9a9fb09e640afb2522ae453ef9553.png)


### `BeforeAgent`

这是最适合做“运行前改配置”的地方。

它能改的不是消息历史，而是本次运行的：

- `Instruction`
- `Tools`
- `ReturnDirectly`

所以它很适合做这些事：

- 动态追加系统约束
- 按租户或环境动态加工具
- 把某个工具临时标记为 `ReturnDirectly`

### `BeforeModelRewriteState / AfterModelRewriteState`

这两个钩子盯的是 `Messages`。

适合做：

- 历史裁剪
- 敏感信息脱敏
- 在模型调用前后检查消息状态

如果你只是想管“发给模型的消息长什么样”，优先看这组。

### `WrapModel`

这个钩子适合拦截模型调用本身。

典型用途是：

- 统一日志
- 指标采集
- 审计
- 对模型输入输出做包装

它的价值在于：你不用改业务代码，就能把“模型调用前后”的工程逻辑拦下来。

### `WrapInvokableToolCall / WrapStreamableToolCall`

这两个钩子盯的是工具层。

特别适合：

- 打工具调用日志
- 统计耗时
- 做参数审计
- 对工具结果二次包装

### 为什么新代码更推荐 `Handlers`

官方和本地源码都已经把这个方向说得很明确了：

- 老的 `AgentMiddleware` 是 struct 风格，适合简单静态扩展
- 新的 `ChatModelAgentMiddleware` 是 interface 风格，更适合动态行为和上下文改写

如果你是现在开始写新的 `ChatModelAgent` 扩展，优先用 `Handlers` 更稳。

## 7. 实战：用 `ChatModelAgent` 搭一个故障分诊助手
目的：
> 做一个“故障分诊助手”，能查 runbook、在高风险场景下直接升级给人工，并通过 handler 统一加上运行约束与工具日志。

本例子只演示三件事：

1. `ChatModelAgent + Tool`
2. `ReturnDirectly`
3. `Handler`

### 先装依赖

```bash
go get github.com/cloudwego/eino@latest
go get github.com/cloudwego/eino-ext/components/model/qwen@latest
```

环境变量至少准备两个：

```powershell
$env:DASHSCOPE_API_KEY="你的百炼 API Key"
$env:QWEN_MODEL="qwen-plus"
```



### 完整代码

这段代码的目标不是做一个真正的运维平台，而是把 `ChatModelAgent` 这一页最重要的几个点跑通。

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/cloudwego/eino-ext/components/model/qwen"
)

// RunbookInput 是查询故障预案工具的输入。
type RunbookInput struct {
	Service   string `json:"service" jsonschema:"description=服务名,enum=user,enum=order,enum=payment,enum=search"`
	ErrorCode string `json:"error_code" jsonschema:"description=错误码，例如 DB_TIMEOUT、AUTH_EXPIRED、NO_STOCK"`
}

// RunbookOutput 是故障预案工具的输出。
type RunbookOutput struct {
	Level      string `json:"level"`
	Suggestion string `json:"suggestion"`
	Owner      string `json:"owner"`
}

// HandoffInput 是转人工工具的输入。
type HandoffInput struct {
	Reason string `json:"reason" jsonschema:"description=需要人工接手的原因"`
}

// HandoffOutput 是转人工工具的输出。
type HandoffOutput struct {
	Ticket string `json:"ticket"`
	Action string `json:"action"`
}

// OpsGuardHandler 是一个自定义 middleware，
// 用来在 Agent 运行前补充约束，并在工具调用时统一打日志。
type OpsGuardHandler struct {
	*adk.BaseChatModelAgentMiddleware
}

// NewOpsGuardHandler 创建自定义 handler。
func NewOpsGuardHandler() *OpsGuardHandler {
	return &OpsGuardHandler{
		BaseChatModelAgentMiddleware: &adk.BaseChatModelAgentMiddleware{},
	}
}

// BeforeAgent 在整次 Agent 运行开始前执行。
// 这里给本次运行动态追加额外指令。
func (h *OpsGuardHandler) BeforeAgent(
	ctx context.Context,
	runCtx *adk.ChatModelAgentContext,
) (context.Context, *adk.ChatModelAgentContext, error) {
	// 拷贝一份运行上下文，避免直接改原对象。
	nRunCtx := *runCtx

	// 动态补充本次运行约束：
	// 1. 始终中文回复
	// 2. 先给结论再给依据
	// 3. 信息不足或风险高时优先转人工
	nRunCtx.Instruction += "\n\n始终使用中文回复。先给结论，再给依据。若缺少关键信息或风险较高，优先调用 handoff_to_human。"

	return ctx, &nRunCtx, nil
}

// WrapInvokableToolCall 包装普通工具调用。
// 这里主要用于统一记录工具名和入参日志。
func (h *OpsGuardHandler) WrapInvokableToolCall(
	ctx context.Context,
	endpoint adk.InvokableToolCallEndpoint,
	tCtx *adk.ToolContext,
) (adk.InvokableToolCallEndpoint, error) {
	return func(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
		log.Printf("[tool] name=%s args=%s", tCtx.Name, argumentsInJSON)
		return endpoint(ctx, argumentsInJSON, opts...)
	}, nil
}

// newRunbookTool 创建“查询故障预案”工具。
// 模型可根据 service + error_code 获取预定义排查建议。
func newRunbookTool() tool.BaseTool {
	t, err := utils.InferTool("search_runbook", "根据服务名和错误码查询故障处置建议", func(ctx context.Context, input *RunbookInput) (*RunbookOutput, error) {
		switch {
		// payment 服务数据库超时时，返回对应预案
		case input.Service == "payment" && input.ErrorCode == "DB_TIMEOUT":
			return &RunbookOutput{
				Level:      "high",
				Suggestion: "先确认只读实例是否可用，再检查连接池是否打满，必要时切换到降级路径。",
				Owner:      "payment-oncall",
			}, nil

		// user 服务鉴权过期时，返回对应预案
		case input.Service == "user" && input.ErrorCode == "AUTH_EXPIRED":
			return &RunbookOutput{
				Level:      "medium",
				Suggestion: "先排查 token 过期时间配置，再确认网关和鉴权服务的时钟是否一致。",
				Owner:      "user-oncall",
			}, nil

		// 未命中预案时，返回兜底结果，引导补充信息
		default:
			return &RunbookOutput{
				Level:      "unknown",
				Suggestion: "没有命中预案，请补充 service、error_code 和最近一次发布时间。",
				Owner:      "triage-bot",
			}, nil
		}
	})
	if err != nil {
		log.Fatalf("new runbook tool failed: %v", err)
	}
	return t
}

// newHandoffTool 创建“转人工”工具。
// 当问题风险较高或信息不足时，用它生成交接单。
func newHandoffTool() tool.BaseTool {
	t, err := utils.InferTool("handoff_to_human", "当风险较高或信息不足时，生成交接给人工处理的说明", func(ctx context.Context, input *HandoffInput) (*HandoffOutput, error) {
		return &HandoffOutput{
			Ticket: "INC-2026-031",
			Action: "已生成交接单，请值班同学继续处理。原因：" + input.Reason,
		}, nil
	})
	if err != nil {
		log.Fatalf("new handoff tool failed: %v", err)
	}
	return t
}

// newModel 创建底层聊天模型。
func newModel(ctx context.Context) *qwen.ChatModel {
	cm, err := qwen.NewChatModel(ctx, &qwen.ChatModelConfig{
		BaseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1",
		APIKey:  mustEnv("DASHSCOPE_API_KEY"),
		Model:   envOrDefault("QWEN_MODEL", "qwen-plus"),
	})
	if err != nil {
		log.Fatalf("new qwen model failed: %v", err)
	}
	return cm
}

// newTriageAgent 创建一个故障分诊 Agent。
// 它可以：
// 1. 调用 runbook 工具查询预案
// 2. 调用 handoff 工具转人工
// 3. 在 handler 中做运行前约束和工具日志
func newTriageAgent(ctx context.Context) adk.Agent {
	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "ops_triage_agent", // Agent 名称
		Description: "负责排查后端线上故障，能查询 runbook，并在高风险时升级给人工处理",
		Instruction: "你是后端故障分诊助手。优先使用工具获取事实，再给结论。",
		Model:       newModel(ctx),

		// 配置可用工具。
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{
					newRunbookTool(),
					newHandoffTool(),
				},
			},

			// handoff_to_human 一旦被调用，工具结果可直接作为输出返回。
			ReturnDirectly: map[string]bool{
				"handoff_to_human": true,
			},
		},

		MaxIterations: 8,          // 最多允许 8 轮 Agent 内部迭代
		OutputKey:     "triage_result", // 本次运行结果的输出键名

		// 注册自定义 middleware。
		Handlers: []adk.ChatModelAgentMiddleware{
			NewOpsGuardHandler(),
		},
	})
	if err != nil {
		log.Fatalf("new triage agent failed: %v", err)
	}
	return agent
}

func main() {
	ctx := context.Background()

	// 默认查询内容，可通过命令行参数覆盖。
	query := "payment 服务出现 DB_TIMEOUT，连接池已满，请给我排查建议。"
	if len(os.Args) > 1 {
		query = strings.Join(os.Args[1:], " ")
	}

	// 创建 Runner，负责驱动 Agent 执行。
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           newTriageAgent(ctx),
		EnableStreaming: false, // 这里关闭流式输出
	})

	// 发起查询，拿到事件流迭代器。
	iter := runner.Query(ctx, query)

	// 逐个消费 Agent 事件并打印结果。
	if err := printEvents(iter); err != nil {
		log.Fatal(err)
	}
}

// printEvents 用于遍历 Agent 运行事件。
// 如果是工具输出，打印工具名；否则按 assistant 输出打印。
func printEvents(iter *adk.AsyncIterator[*adk.AgentEvent]) error {
	for {
		event, ok := iter.Next()
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
		msg, err := mv.GetMessage()
		if err != nil {
			return err
		}

		switch mv.Role {
		case schema.Tool:
			fmt.Printf("\n[tool:%s]\n%s\n", mv.ToolName, msg.Content)
		default:
			fmt.Printf("\n[assistant]\n%s\n", msg.Content)
		}
	}
}

// mustEnv 读取必填环境变量；缺失则直接退出。
func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("environment variable %s is required", key)
	}
	return v
}

// envOrDefault 读取环境变量；没有则返回默认值。
func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
```

### 这个 Demo 到底对应了什么

1. `search_runbook` 是普通 Tool，模型先查事实，再组织答案
2. `handoff_to_human` 被配置成 `ReturnDirectly`，一旦调用就直接退出
3. `OpsGuardHandler` 通过 `BeforeAgent` 和 `WrapInvokableToolCall` 把运行约束和工具日志插进来了

如果你本地跑的时候传一个“高风险但信息不足”的问题，比如：

```bash
go run . "payment 服务持续报错，但我只有一句日志：DB_TIMEOUT，请直接给我下一步动作。"
```

常见表现会是两种：

- 模型先调 `search_runbook`，再组织答案返回
- 模型判断信息不足或风险过高，直接调 `handoff_to_human`，然后因为 `ReturnDirectly` 立即结束

这正是 `ChatModelAgent` 和普通模型调用的差别：它不是只会说话，而是会决定下一步怎么干。



## 8. 总结

本篇最想帮你建立的，不是某个 API 记忆点，而是一个判断：

> `ChatModelAgent` 不是“模型调用升级版”，而是 ADK 里默认的思考型 Agent 实现。

它真正解决的是：

- 让模型在回答、调工具、转交任务之间做动态决策
- 让这些动作按照 `ReAct` 方式循环运行
- 让运行过程以 `AgentEvent` 形式输出
- 让你能通过 `Handler` 把日志、审计、裁剪、动态工具这些工程能力插进去

---

## 发布说明

- GitHub 主文：[AI 大模型落地系列｜Eino ADK体系篇：你对 ChatModelAgent 有了解吗？](./03-你对ChatModelAgent有了解吗？.md)
- CSDN 跳转：[AI 大模型落地系列｜Eino ADK体系篇：你对 ChatModelAgent 有了解吗？](https://zhumo.blog.csdn.net/article/details/159696365)
- 官方文档：[Eino ADK：ChatModelAgent](https://www.cloudwego.io/zh/docs/eino/core_modules/eino_adk/agent_implementation/chat_model/)
- 最新版以 GitHub 仓库为准。

