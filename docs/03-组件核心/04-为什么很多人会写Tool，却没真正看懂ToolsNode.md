# AI 大模型落地系列｜Eino 组件核心篇：为什么很多人会写 Tool，却没真正看懂 ToolsNode

> GitHub 主文：[当前文章](./04-为什么很多人会写Tool，却没真正看懂ToolsNode.md)
> CSDN 跳转：[AI 大模型落地系列｜Eino 组件核心篇：为什么很多人会写 Tool，却没真正看懂 ToolsNode](https://zhumo.blog.csdn.net/article/details/159511006)
> 官方文档：[ToolsNode & Tool 使用说明](https://www.cloudwego.io/zh/docs/eino/core_modules/components/tools_node_guide/)
>
> 最新版以 GitHub 仓库为准，CSDN 作为分发入口，官方文档作为权威参考。

**一句话摘要**：从 Tool、ToolsNode、ToolCall 到自定义 Tool 的完整链路重新理解工具能力是怎么接进 Eino 的。
**适合谁看**：已经能写简单 Tool，但想把工具调用链讲透的工程师。
**前置知识**：Tool 与文件系统访问、消息角色与函数调用基础
**对应 Demo**：[examples/tool-filesystem（展示工具调用闭环）](../../examples/tool-filesystem/README.md)

**面试可讲点**
- 能解释 Tool 和 ToolsNode 的职责边界，不把两者混为一谈。
- 能说明 ToolCall 是怎么从模型决策一路落到工具执行的。

---
很多人学 Eino 的 `Tool Calling`，第一反应是先把几个 `Tool` 注册上，再让 `Agent` 跑起来。
代码能跑，演示也有。
可一旦你继续追问：到底是谁决定调用哪个 `Tool`？`ToolsNode` 到底做了什么？工具结果又是怎么回到消息链路里的？很多人就开始说不清了。

这不奇怪。
因为很多文章只教你“怎么把工具接上”，很少有人去讲透“执行边界”。
结果就是，很多人会写 `Tool`，但对 `ToolsNode` 的理解还停留在“工具箱”这三个字上。

如果你前面已经看过我那篇入门篇[《AI大模型落地系列：一文读懂 Eino 的 Tool 和文件系统访问》](https://blog.csdn.net/2302_80067378/article/details/159395909?spm=1001.2014.3001.5501)，那篇主要在讲：`Tool` 怎么让 Agent 真正碰到外部世界。
这一篇换个角度。
不讲文件系统接入，不讲 `DeepAgent`，只讲组件层最关键的一件事：

> 在 Eino 里，`Tool` 和 `ToolsNode` 到底分别在解决什么？

## 1. 工具调用（Tool Calling）中，调用链路十分值得重视

很多人对 `Tool` 的第一印象，是“给模型加插件”。
很多人对 `ToolsNode` 的第一印象，是“工具执行器”。

这两个理解都不算错。
但如果只停在这一步，还是太粗了。

因为真正影响你后面写 `Agent`、排查 `Tool` 问题、做多轮编排的，不是“知道有这么两个组件”，而是你能不能把它们的边界拆开。

说白一点：

- `Tool` 解决的是“能力怎样被声明出来，供模型选择”
- `ToolsNode` 解决的是“模型一旦决定调用，系统怎样把调用真正执行掉”

这两层一旦混在一起，后面就很容易出现三种常见误判：

- 以为 `ToolsNode` 会替你决定该调哪个工具
- 以为只要写了 `InvokableRun`，工具调用链路就算理解完了
- 以为 `Tool Calling` 的核心只是“把函数包一层”

这三种理解，都会让你在工程里很快撞墙。

因为一轮真正的工具调用，从来不是“注册一个函数”这么简单。
它至少包括：

- 模型基于 `ToolInfo` 决定要不要调工具
- 模型产出 `ToolCall`
- `ToolsNode` 根据 `ToolCall` 找到对应 `Tool`
- 工具被实际执行
- 执行结果再被封回消息链路，继续交给模型

真正该盯住的，不只是“工具怎么写”，而是“这一整条链路是怎么串起来的”。

## 2. `Tool` 是能力协议，`ToolsNode` 是执行中枢

先把最重要的一句话摆出来：

> `ChatModel` 决定调用谁，`ToolsNode` 负责把调用真正落地。

这个顺序不能反。

`Tool` 不是决策器。
`ToolsNode` 也不是决策器。
真正做“要不要调工具、调哪个工具、传什么参数”这件事的，是前面的 `ChatModel`。

你可以把一轮链路先压成下面这样：

```text
用户问题
   -> ChatModel
   -> assistant message（内含 ToolCalls）
   -> ToolsNode
   -> tool message / ToolResult
   -> ChatModel
   -> 最终回答
```

这里最容易被忽略的一点是：

`ToolsNode` 不负责“思考”。
它只负责“执行”。

也就是说，`ToolsNode` 不会自己判断“天气工具和搜索工具哪个更合适”。
它做的事情更接近后端里的调度层：

- 从输入消息里拿到 `ToolCalls`
- 按名称找到对应的 `Tool`
- 按参数实际执行
- 把结果包装成后续消息

所以如果你问，`Tool` 和 `ToolsNode` 分别像什么？

- `Tool` 更像一份对外能力协议
- `ToolsNode` 更像一次工具调用的执行中枢

一个负责“把能力暴露给模型”，一个负责“把模型已经做出的调用决定执行掉”。

## 3. 一轮 Tool Calling 在 Eino 里到底怎么走

如果只看概念，很多人会觉得已经懂了。
但真要把链路讲清楚，还是得抓住几个关键类型。

先看入口。

`schema.Message` 不只是对话消息，它也是工具调用的载体。
当 `assistant` 这条消息里带上 `schema.Message.ToolCalls` 时，就意味着：模型已经产出了要执行的工具调用列表。

而 `schema.ToolCall` 里最关键的是两块：

- `ID`：标识这一次具体调用
- `Function`：里面有 `schema.FunctionCall.Name` 和 `schema.FunctionCall.Arguments`

翻成人话就是：

- `Name` 说明要调哪个工具
- `Arguments` 是 JSON 字符串，说明这次调用传什么参数

官方源码是这么定义的：
这样一个顺序：**message**->**ToolCall**->**FuntionCall**
```go
// schema/message.go
type Message struct {
    // 对于工具调用消息，这里的 role 应该是 'assistant'
    Role RoleType `json:"role"`

    // 这里的每一个 ToolCall 都由 ChatModel 生成，并交给 ToolsNode 执行
    ToolCalls []ToolCall `json:"tool_calls,omitempty"`

    // 其他字段……
}

// ToolCall 表示消息中的一次工具调用。
// 当 assistant 消息里需要发起工具调用时，会使用它。
type ToolCall struct {
    // Index 用于标识一条消息中存在多个工具调用时的顺序位置。
    // 在流式模式下，它也用于标识某个工具调用的分片，以便后续合并。
    Index *int `json:"index,omitempty"`

    // ID 是这次工具调用的唯一标识，可用于定位某一次具体调用。
    ID string `json:"id"`

    // Type 是工具调用的类型，默认值是 "function"。
    Type string `json:"type"`

    // Function 表示这次要执行的函数调用信息。
    Function FunctionCall `json:"function"`

    // Extra 用来存储这次工具调用的额外信息。
    Extra map[string]any `json:"extra,omitempty"`
}

// FunctionCall 表示消息中的函数调用。
// 它用于 assistant 消息中。
type FunctionCall struct {
    // Name 是要调用的函数名称，可用于标识具体调用哪个函数。
    Name string `json:"name,omitempty"`

    // Arguments 是调用该函数时传入的参数，格式为 JSON 字符串。
    Arguments string `json:"arguments,omitempty"`
}
```

这时 `ToolsNode` 真正依赖的，不是你的业务 prompt，也不是用户原始问题，而是这条带着 `ToolCalls` 的 `assistant message`。

再往下，就是 `compose.ToolsNodeConfig`。
这块配置的重点，虽然它 “字段多”。

最值得盯住的是这几个字段：

- `Tools []tool.BaseTool`：当前可执行的工具列表
- `ExecuteSequentially bool`：多个 `ToolCall` 时，是否按消息里的顺序串行执行
- `UnknownToolsHandler`：模型调了一个未注册工具时怎么处理
- `ToolArgumentsHandler`：工具执行前，是否要对参数做统一修正或预处理
- `ToolCallMiddlewares`：是否要给工具调用挂统一中间件

这里有个点特别容易被理解错：

`ExecuteSequentially` 控制的是**执行时序**，不是**模型决策顺序**。

模型在 `ToolCalls` 里给出的顺序，是它产出调用计划的顺序。
`ToolsNode` 如果开启串行执行，就按这个顺序一个一个跑。
如果不开启，就允许按自己的执行策略处理多个调用。

也就是说，这个配置回答的是：

> 多个调用来了以后，执行层怎么跑？

它回答的不是：

> 模型为什么先调 A 再调 B？

后一个问题，仍然属于 `ChatModel` 的决策范围。

至于 `UnknownToolsHandler` 和 `ToolArgumentsHandler`，它们都很像后端系统里的“兜底钩子”：

- `UnknownToolsHandler` 适合处理模型幻觉出来的工具名，或者做统一降级
- `ToolArgumentsHandler` 适合做参数清洗、默认值补齐、审计或兼容旧参数格式

所以一轮 `Tool Calling` 真正的边界应该这样看：

- 模型负责产出 `ToolCall`
- `ToolsNode` 负责消费 `ToolCall`
- `Tool` 负责提供实际能力

这三层拆开以后，整条链路就顺了。

## 4. 一个 Tool 至少要包含什么

很多人第一次写自定义工具，容易把注意力全放在“函数体怎么写”上。
其实不是。

对 Eino 来说，一个 `Tool` 至少要同时解决两件事：

- 告诉模型“我是谁、能干什么、需要什么参数”
- 告诉运行时“真调到我时，我该怎么执行”

所以最小定义一定绕不开 `tool.BaseTool`。
```go
// 基础工具接口，提供工具信息
type BaseTool interface {
    Info(ctx context.Context) (*schema.ToolInfo, error)
}
```

`tool.BaseTool` 只要求一个 `Info(ctx)` 方法，返回 `*schema.ToolInfo`。
而 `schema.ToolInfo` 本质上就是工具协议：

- `Name`：工具名
- `Desc`：工具描述
- `ParamsOneOf`：参数约束

这一步不是走形式。
模型到底能不能正确构造 `ToolCall`，很大程度上就取决于这份 `ToolInfo` 写得清不清楚。

在可执行接口上，Eino 把 `Tool` 分成两组。

第一组是标准工具：

- `tool.InvokableTool`：同步调用，输入是 JSON 字符串，输出是字符串
- `tool.StreamableTool`：流式调用，输出是字符串流

```go
type InvokableTool interface {
    BaseTool
    InvokableRun(ctx context.Context, argumentsInJSON string, opts ...Option) (string, error)
}

type StreamableTool interface {
    BaseTool
    StreamableRun(ctx context.Context, argumentsInJSON string, opts ...Option) (*schema.StreamReader[string], error)
}
```


第二组是增强型工具：
- `tool.EnhancedInvokableTool`：输入是 `*schema.ToolArgument`，输出是 `*schema.ToolResult`
- `tool.EnhancedStreamableTool`：输入是 `*schema.ToolArgument`，输出是 `*schema.StreamReader[*schema.ToolResult]`

```go
// EnhancedInvokableTool 是支持返回结构化多模态结果的工具接口
// 与返回字符串的 InvokableTool 不同，此接口返回 *schema.ToolResult
// 可以包含文本、图片、音频、视频和文件
type EnhancedInvokableTool interface {
    BaseTool
    InvokableRun(ctx context.Context, toolArgument *schema.ToolArgument, opts ...Option) (*schema.ToolResult, error)
}

// EnhancedStreamableTool 是支持返回结构化多模态结果的流式工具接口
// 提供流式读取器以逐步访问多模态内容
type EnhancedStreamableTool interface {
    BaseTool
    StreamableRun(ctx context.Context, toolArgument *schema.ToolArgument, opts ...Option) (*schema.StreamReader[*schema.ToolResult], error)
}
```

其中官方对，ToolPartType 定义了五种类型：
```go
// ToolPartType 定义工具输出部分的内容类型
type ToolPartType string

const (
    ToolPartTypeText  ToolPartType = "text"   // 文本
    ToolPartTypeImage ToolPartType = "image"  // 图片
    ToolPartTypeAudio ToolPartType = "audio"  // 音频
    ToolPartTypeVideo ToolPartType = "video"  // 视频
    ToolPartTypeFile  ToolPartType = "file"   // 文件
)
```

而恰恰 `schema.ToolResult` 不是一个简单字符串。
它的核心是 `Parts []ToolOutputPart`，也就是你可以返回：

- 文本
- 图片
- 音频
- 视频
- 文件

这也是为什么增强型 `Tool` 不只是“返回值换个结构体”。
它其实是在告诉框架：这个工具的结果，不一定是一段纯文本。


因此：
- 标准工具适合“查一下天气、算个表达式、调个普通接口”这类文本结果场景
- 增强型工具适合“返回图片、音频、视频、文件，或者结构化多模态内容”这类场景




还有一个细节必须记住。

> 当同一个工具同时实现了标准接口和增强型接口时，`ToolsNode` 会优先走增强型接口。

这点如果你没建立起心智，后面排查“为什么没走我预想的那个执行分支”时会很别扭。

## 5. 怎么自己创建一个 Tool

说到“怎么写 Tool”，很多人最容易陷进去的地方，是把精力全花在“选哪个 helper 函数”上。
其实 helper 重要，但不是最重要。

真正更重要的是三件事：

- 参数约束是否和真实输入一致
- 工具职责是否单一
- 返回形态是否和消费场景匹配

从使用顺序上，我更推荐这样理解。

第一优先，`InferTool` 和 `InferEnhancedTool`。

这两个方法最适合日常业务开发。
原因很简单：参数约束可以直接写在输入结构体上，你不需要一边维护函数入参，一边再手动维护一份 `ParamsOneOf`。

第二优先，`NewTool` 和 `NewEnhancedTool`。

这更适合你已经有很明确的 `schema.ToolInfo`，或者你就是想手工控制参数描述。
它的优势是灵活，代价是你自己要保证 `ToolInfo` 和真实入参别跑偏。

第三种，直接实现接口。

这类写法最原始，但也最自由。
如果你要做底层封装、复杂参数处理、或者对执行过程有更强控制欲，这种方式最稳。
代价也最明显：参数解析、错误处理、结构约束，都得你自己收拾。

再往后，就是生态层选择：

- 能直接复用的，优先看 `eino-ext`
- 外部系统已经通过 MCP 暴露能力的，可以直接把 MCP Tool 接进来

但无论你选哪一种创建方式，都别把重点放错。

很多人以为“把函数包成 Tool”是难点。
其实真正的难点通常是：

- 你有没有把 `schema.ToolInfo` 写清楚
- 你给模型的参数约束，和真实执行需要的参数，是否一致
- 你到底该返回普通字符串，还是该返回 `schema.ToolResult`

如果这三件事没想清楚，helper 再顺手，后面也会开始歪。

## 6. 用一个最小例子，把“会注册”和“看懂执行链路”连起来

只说概念还是容易飘。
不如直接看一个最小例子。

这个例子只做一件事：查询温度。

- 先定义一个最小 `weather` 工具
- 再手工构造一条带 `ToolCalls` 的 `assistant message`
- 最后交给 `compose.ToolsNode` 执行

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

// Tool 入参：城市名
type WeatherInput struct {
	City string `json:"city" jsonschema:"required" jsonschema_description:"要查询天气的城市"`
}

// Tool 出参：城市 + 天气结果
type WeatherOutput struct {
	City    string `json:"city"`
	Weather string `json:"weather"`
}

// Tool 的实际执行逻辑
func queryWeather(_ context.Context, input *WeatherInput) (*WeatherOutput, error) {
	return &WeatherOutput{
		City:    input.City,
		Weather: "晴，28度",
	}, nil
}

func main() {
	ctx := context.Background()

	// 1) 把普通 Go 函数包装成 Eino Tool
	weatherTool, err := utils.InferTool("weather", "查询城市天气", queryWeather)
	if err != nil {
		log.Fatal(err)
	}

	// 2) 创建 ToolsNode：它只负责执行 ToolCall，不负责决定调用哪个 Tool
	toolsNode, err := compose.NewToolNode(ctx, &compose.ToolsNodeConfig{
		Tools:               []tool.BaseTool{weatherTool},
		ExecuteSequentially: true, // 多个 ToolCall 时按顺序执行
	})
	if err != nil {
		log.Fatal(err)
	}

	// 3) 模拟 ChatModel 已经产出的 assistant 消息，其中带有 ToolCalls
	input := &schema.Message{
		Role: schema.Assistant,
		ToolCalls: []schema.ToolCall{
			{
				ID:   "call_weather_1", // 本次调用的唯一标识
				Type: "function",
				Function: schema.FunctionCall{
					Name:      "weather",          // 要调用的 Tool 名称
					Arguments: `{"city":"深圳"}`, // 传给 Tool 的 JSON 参数
				},
			},
		},
	}

	// 4) ToolsNode 执行 ToolCall，返回 tool message
	toolMessages, err := toolsNode.Invoke(ctx, input)
	if err != nil {
		log.Fatal(err)
	}

	// 5) 打印执行结果；实际链路里这些结果通常会继续交给 ChatModel
	for _, msg := range toolMessages {
		fmt.Printf("role=%s content=%s\n", msg.Role, msg.Content)
	}
}
```

如果执行顺利，你看到的大意会是这样：

```text
role=tool content={"city":"深圳","weather":"晴，28度"}
```

这一刻最该记住的，不是“天气查出来了”。
而是下面这件事：

> 这条 `role=tool` 的消息，不是你手工拼出来的，而是 `ToolsNode` 根据 `assistant message` 里的 `ToolCalls` 执行后产出的。

这就把很多人原来脑子里断掉的那一截补上了：

- `ChatModel` 先产出 `ToolCalls`
- `ToolsNode` 再逐个执行
- 每次执行结果都回到消息链路里
- 后面模型可以继续基于这些结果生成最终回答

如果你需要的不是纯文本，而是图片、文件这类结果，那就该考虑增强型工具。
比如下面这个片段：

```go
type ImageSearchInput struct {
	Query string `json:"query" jsonschema:"required" jsonschema_description:"搜索关键词"`
}

// 使用增强型 Tool：返回的不是纯字符串，而是结构化的 ToolResult
imageTool, err := utils.InferEnhancedTool(
	"image_search",   // Tool 名称
	"搜索并返回相关图片", // Tool 描述
	func(ctx context.Context, input *ImageSearchInput) (*schema.ToolResult, error) {
		_ = ctx
		_ = input

		imageURL := "https://example.com/cat.png"

		return &schema.ToolResult{
			Parts: []schema.ToolOutputPart{
				// 返回一段文本说明
				{
					Type: schema.ToolPartTypeText,
					Text: "找到 1 张图片",
				},
				// 返回一张图片；这里用 URL 形式表示图片资源
				{
					Type: schema.ToolPartTypeImage,
					Image: &schema.ToolOutputImage{
						MessagePartCommon: schema.MessagePartCommon{
							URL: &imageURL,
						},
					},
				},
			},
		}, nil
	},
)

_ = imageTool
_ = err
```

这段代码真正想表达的，不是“语法还能这么写”。
而是：

- 如果你的结果天然带多模态，别硬塞成字符串
- `schema.ToolResult` 本来就是给这种场景准备的

什么时候该用增强型工具？
一句话判断：

> 结果如果不仅仅是文本，而是要把图片、文件、音视频作为一等输出返回，就别再用普通字符串接口硬撑。

## 7. 真正到了工程里，你还得关心这些

如果你只是写一个 demo，到这里已经够用了。
但只要进入真实工程，下面这几个点很快就会变得比“工具能不能跑”更重要。

先说 `Option`。

很多人把它理解成“可选参数”。
这当然没错，但还是太轻了。
放到工具体系里，`Option` 更像运行时动态调度入口。
比如超时、重试、最大返回条数、质量等级，这些都更适合走 `tool.Option` 机制，而不是硬编码在函数体里。

再说 `Middleware`。

`ToolsNode` 支持给工具调用挂 `ToolCallMiddlewares`。
这件事的价值，不是“高级”，而是它让日志、指标、参数审计、统一包装这些横切逻辑终于有了稳定落点。
特别是标准工具和增强型工具并存时，这层中间件会非常顺手。

然后是 `Callback`。

一旦链路里有多个 `ToolCall`、有流式输出、或者有失败重试，没有观测你会很快掉进黑盒。
而工具级 `Callback` 至少能帮你看到：

- 工具什么时候开始执行
- 参数长什么样
- 最终返回了什么
- 流式输出有没有正常结束

最后是 `compose.GetToolCallID(ctx)`。

这个能力很朴素，但特别好用。
不管是在 tool 函数体里打日志，还是在 callback handler 里串 trace，只要把 `ToolCallID` 打出来，单次调用链路就很容易串上。
## 8. 总结

如果把今天这篇压成一句话，那就是：

> `Tool` 负责把能力声明出来，`ToolsNode` 负责把一次 tool calling 执行到底。

前者解决的是“模型能调用什么”，后者解决的是“模型已经决定调用以后，系统怎样真正去做”。
这两层一旦看懂，后面的 `Agent`、`Callback`、`Trace`、`Workflow`，你会顺很多。

## 参考资料

- CloudWeGo Eino [ToolsNode&Tool 使用说明](https://www.cloudwego.io/zh/docs/eino/core_modules/components/tools_node_guide/)
- CloudWeGo Eino [How to Create a Tool](https://www.cloudwego.io/zh/docs/eino/core_modules/components/tools_node_guide/how_to_create_a_tool/)
- CloudWeGo Eino [第四章：Tool 与文件系统访问](https://www.cloudwego.io/zh/docs/eino/quick_start/chapter_04_tool_and_filesystem/)

---

## 发布说明

- GitHub 主文：[AI 大模型落地系列｜Eino 组件核心篇：为什么很多人会写 Tool，却没真正看懂 ToolsNode](./04-为什么很多人会写Tool，却没真正看懂ToolsNode.md)
- CSDN 跳转：[AI 大模型落地系列｜Eino 组件核心篇：为什么很多人会写 Tool，却没真正看懂 ToolsNode](https://zhumo.blog.csdn.net/article/details/159511006)
- 官方文档：[ToolsNode & Tool 使用说明](https://www.cloudwego.io/zh/docs/eino/core_modules/components/tools_node_guide/)
- 最新版以 GitHub 仓库为准。

