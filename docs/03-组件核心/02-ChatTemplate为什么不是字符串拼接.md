# AI 大模型落地系列｜Eino 组件核心篇：ChatTemplate 为什么不是字符串拼接

> GitHub 主文：[当前文章](./02-ChatTemplate为什么不是字符串拼接.md)
> CSDN 跳转：[AI 大模型落地系列｜Eino 组件核心篇：ChatTemplate 为什么不是字符串拼接](https://zhumo.blog.csdn.net/article/details/159500932)
> 官方文档：[ChatTemplate 使用说明](https://www.cloudwego.io/zh/docs/eino/core_modules/components/chat_template_guide/)
>
> 最新版以 GitHub 仓库为准，CSDN 作为分发入口，官方文档作为权威参考。

**一句话摘要**：把 Prompt 组织方式从字符串拼装提升成消息模板和上下文协议。
**适合谁看**：已经写过 Prompt，希望把 ChatTemplate 看成正式组件的读者。
**前置知识**：ChatModel 与 Message、消息角色基础
**对应 Demo**：[examples/chain-graph（包含 ChatTemplate 节点）](../../examples/chain-graph/README.md)

**面试可讲点**
- 能解释 ChatTemplate 和字符串拼接的根本区别在于结构化上下文。
- 能说明模板语法、MessagesPlaceholder 和下游编排的关系。

---
为什么很多人已经会写 prompt(<small>提示词</small>) 了，到了 Eino 里，却还是经常把 `ChatTemplate` 用偏？

因为太多人一看到 template，就条件反射地把它理解成“字符串替换器”。
把 `{role}` 换进去，把 `{task}` 换进去，再把 history 手动拼成一大段文本，看起来也能跑。
可问题恰恰就在这儿：你如果只是这么用，等于把 Eino 这层最关键的上下文组织能力，直接降级成了字符串拼接。

这篇文章就想回答两个问题：

> `ChatTemplate` 到底解决了什么？
> 它为什么不是一个“高级一点的字符串模板”而已？

## 1. `ChatTemplate` 是什么，不是什么

先把结论摆出来：

`ChatTemplate` 不是字符串拼接工具。
它是把变量、角色消息、历史对话组织成 `[]*schema.Message` 的组件。

这句话看起来只差几个字，实际差得很远。

如果你把它当成字符串模板，脑子里的链路通常是这样的：

`变量 -> 替换文本 -> 拼 prompt -> 丢给模型`

而 Eino 真正想让你建立的链路，是这样的：

`变量 / 前驱节点输出 -> ChatTemplate -> []*schema.Message -> ChatModel`

也就是说，`ChatTemplate` 干的不是“把几段字拼起来”，而是“把上下文整理成模型能消费的消息协议”。

这层价值主要体现在三件事上。

**第一**，它让 prompt 变成结构化消息，而不是一坨长字符串。

**第二**，它让多轮 history 的注入有统一入口，不用你手搓字符串去拼上下文。

**第三**，它能直接进入 `Chain`、`Graph`、`Callback` 这些编排和可观测链路里，说明它从一开始就不是一个工具函数，而是一个组件。

所以如果你问，`ChatTemplate` 在 Eino 里到底值不值得单独学？

我的回答很直接：

> 值得。因为它解决的不是“模板替换”，而是“Prompt 怎样以消息协议的方式进入 Eino”。  

## 2. 接口虽短，但起的作用却不小

官方给出的核心接口其实非常短：
```go
type ChatTemplate interface { 
	Format(ctx context.Context, vs map[string]any, opts ...Option) ([]*schema.Message, error) 
}
```

很多人第一次看到这行代码，会觉得不就是个格式化函数吗？
真要这么理解，还是看浅了。

这里最重要的，其实是 `Format` 的三个输入和一个输出。

`ctx` 不只是普通上下文。
它即负责传递请求级信息，同时它也承载 `Callback Manager`。这意味着模板格式化这件事，并不是一个完全封闭的小动作，它是能被观测、能被接入回调链路的。

`vs` 虽是变量映射，但却不是“只能塞字符串”的变量映射。你既可以传
```txt
"role": "专业助手"
```
这种普通文本，也可以传
 ```txt
 "history_key": []*schema.Message{...}
 ```
 这种消息列表。
 换句话说，它接收的不是纯文本变量，而是上下文数据。

`opts` 也很有意思。
官方没有给 `ChatTemplate` 设计一个“大而全的公共参数表”，而是把它作为具体实现的扩展点来留。这个意思其实很明确：Prompt 组件需要统一协议，但不想被统一成一个笨重的大接口。

最后是输出。

`Format` 返回的不是一段 prompt 文本，而是标准消息数组 `[]*schema.Message`。
这一步就是 `ChatTemplate` 和字符串拼接最本质的分水岭。

你自己手拼字符串，最终交给模型的是一段文本。
你用 `ChatTemplate`，最终交给模型的是一组角色明确、结构清晰的消息。

## 3. 官方提供了哪些构建方式

 - **prompt.FromMessages()**：用于把多个 `message` 变成一个 `chat template`。

 - **schema.Message{}**：`schema.Message` 是实现了 `Format` 接口的结构体，因此可直接构建 `schema.Message{}` 作为 template。

 - **schema.SystemMessage()**：此方法是构建 `role` 为 `system` 的 `message` 快捷方法。

 - **schema.AssistantMessage()**：此方法是构建 `role` 为 `assistant` 的 `message` 快捷方法。

 -  **schema.UserMessage()**：此方法是构建 `role` 为 `user` 的 `message` 快捷方法。

 - **schema.ToolMessage()**：此方法是构建 `role` 为 `tool` 的 `message` 快捷方法。

 - **schema.MessagesPlaceholder()**：可用于把一个 `[]*schema.Message` 插入到 `message` 列表中，常用于插入历史对话。

## 4. 一个最小例子，看懂它怎么工作
> 先看一遍，留个整体印象，后面再拆开说。

```go
import (
    "github.com/cloudwego/eino/components/prompt"
    "github.com/cloudwego/eino/schema"
)

// 创建模板
template := prompt.FromMessages(schema.FString,
    schema.SystemMessage("你是一个{role}。"),
    schema.MessagesPlaceholder("history_key", false),
    &schema.Message{
        Role:    schema.User,
        Content: "请帮我{task}。",
    },
)

// 准备变量
variables := map[string]any{
    "role": "专业的助手",
    "task": "写一首诗",
    "history_key": []*schema.Message{
        {Role: schema.User, Content: "告诉我油画是什么?"},
        {Role: schema.Assistant, Content: "油画是xxx"},
    },
}

// 格式化模板
messages, err := template.Format(context.Background(), variables)
```

这段代码真正值得你记住的，不是语法，而是它把几个关键动作放到了一起。

`system` 提示可以参数化，不需要写死。

history 可以整体注入，而且注入进去的仍然是 `[]*schema.Message`，不是你手工拼出来的一大段文本。

当前这一轮的 user 问题也可以模板化，跟 system 和 history 统一走一条格式化链路。

最后 `template.Format(...)` 产出的不是字符串，而是 `messages`。这些 `messages` 才是后面交给 `ChatModel` 的标准输入。

如果继续往下看，真正值得盯住的主要是下面三个点。

## 5. 三个最容易看浅的点

### 5.1 `schema.Message` 是模板单元，不是字符串壳

很多人学到 `prompt.FromMessages(...)` 时，会下意识把它理解成“多个 prompt 片段拼起来”。
这个理解只对了一半。

它确实是在组合内容，但组合的不是普通字符串，而是消息模板。

比如：

- `schema.SystemMessage(...)`
- `schema.UserMessage(...)`
- 甚至一个完整的 `schema.Message{}`

这些东西放进 `prompt.FromMessages(...)` 以后，组成的是一组待格式化的消息模板，不是一篇待替换的大作文。

字符串拼接关心的是“句子怎么连起来”。
而 `ChatTemplate` 关心的是“system 说什么，user 说什么，history 该插在哪，最后怎样变成标准消息协议”。

这两个层级，本来就不是一回事。

### 5.2 `MessagesPlaceholder` 才是很多人真正该盯住的点

如果说 `ChatTemplate` 里有一个最容易被低估、但对真实业务最重要的能力，那大概率就是 `schema.MessagesPlaceholder(...)`。

为什么？

因为多轮对话里最常见的问题，从来不是“怎么替换 `{name}`”，而是“怎么把历史上下文塞进去，而且别塞乱了”。

很多人会这样干：

把历史对话先手动拼成一大段字符串，再把它塞进某个 user prompt 里。

这种写法当然能跑，但它本质上还是字符串拼接。
你原本可以传一个 `[]*schema.Message`，结果你自己把它打平成了纯文本。
看起来省事，实际上是主动绕开了消息协议。

`schema.MessagesPlaceholder("history_key", false)` 的价值就在这儿。
它让你可以把 `history_key` 对应的 `[]*schema.Message` 直接插进消息列表里。

也就是说，这条链路应该这么理解：

`history -> MessagesPlaceholder -> []*schema.Message`

它的重点不是“占位符”三个字，而是“history 仍然以消息数组的形态进入模板”。

这个思路一旦立住，你后面做多轮、做记忆、做 Agent 上下文拼装，脑子都会顺很多。

### 5.3 三种模板语法怎么选，别一上来就上复杂度

官方内置了三种模板化方式：

- `schema.FString`
- `schema.GoTemplate`
- `schema.Jinja2`

它们不是“谁更高级”，而是适用场景不同。

`schema.FString` 最直观，用 `{variable}` 做替换，适合大多数基础场景。
如果你的需求只是把角色、任务、问题这类变量填进去，它通常就够了。

`schema.GoTemplate` 适合需要条件判断、循环拼接这类逻辑的场景。
一旦你的模板里已经出现“有值就展示，没有就省略”“遍历一组数据生成内容”这种诉求，Go 模板会更顺手。

`schema.Jinja2` 更像是给有模板引擎经验的人准备的。（<small>python风格</small>）
如果你平时就熟悉 Jinja 风格，那它上手会更自然。

我的建议很简单：

别把模板引擎选型搞成技术表演。
简单替换就用 `schema.FString`，真有条件逻辑再上 `schema.GoTemplate`，已经习惯 Jinja 再选 `schema.Jinja2`。

你要解决的是消息组织问题，不是比赛谁的模板更花。


## 6. 为什么它能进入 `Chain / Graph / Callback`
只看单独调用，你很容易以为 `ChatTemplate` 不过是个前置小工具。
可一旦站到编排视角，它的定位就完全变了。

在 `Chain` 里，`ChatTemplate` 是一个很标准的上下文准备节点。
它的任务不是回答问题，而是把输入变量整理成后续模型能吃的消息列表。

在 `Graph` 里，这个味道更明显。
它可以消费前驱节点经过 `compose.WithOutputKey(...)` 包装后的 `map[string]any` 输出，然后继续把这些数据组织成消息。

短示意可以看成这样：

```go
// 创建一个 Chain：输入是 map[string]any，输出是 []*schema.Message
// 也就是说，这条链路接收一组变量，最终产出标准消息列表，供后续 ChatModel 使用。
chain := compose.NewChain[map[string]any, []*schema.Message]()

// 把前面定义好的 ChatTemplate 挂到 Chain 上。
// 作用：把输入变量格式化成消息数组。
chain.AppendChatTemplate(template)


// 创建一个 Graph：输入是 string，输出是 []*schema.Message
// 这里的意思是：Graph 接收一段原始字符串，经过节点处理后，最终产出消息列表。
graph := compose.NewGraph[string, []*schema.Message]()

// 添加一个 Lambda 节点，节点名叫 rewrite_query
graph.AddLambdaNode(
    "rewrite_query",

    // 这个 Lambda 的作用是把原始输入改写成一个更完整的用户问题
    // 例如输入："123"
    // 输出："请帮我总结这段需求：123"
    compose.InvokableLambda(func(ctx context.Context, input string) (string, error) {
        return "请帮我总结这段需求：" + input, nil
    }),

    // 把这个节点的输出包装成 map[string]any 里的一个字段，key 叫 query
    // 这样后面的 ChatTemplate 就可以用 {query} 来取这个值
    compose.WithOutputKey("query"),
)

// 添加一个 ChatTemplate 节点，节点名叫 prompt_node
graph.AddChatTemplateNode("prompt_node", prompt.FromMessages(schema.FString,

    // system 消息：给模型设定角色
    // 这里的 {role} 需要在运行时从变量里传入
    schema.SystemMessage("你是一个{role}。"),

    // user 消息：使用上一个节点产出的 query
    // 因为 rewrite_query 节点通过 WithOutputKey("query") 输出了 query，
    // 所以这里可以直接写 {query}
    schema.UserMessage("{query}"),
))
```

翻成人话就是：

前面的节点先产出数据。
如果它通过 `compose.WithOutputKey("query")` 把结果包成 `map[string]any`，那后面的 `ChatTemplate` 节点就可以直接用这个 key 去取值，再把它组织成标准消息。

这时你会发现，`ChatTemplate` 真正扮演的角色，其实是“消息协议装配器”。
它站在模型前面，把上游零散的数据，整理成模型真正能消费的输入。

也正因为如此，它才能自然接进 `Chain` 和 `Graph`，而不是只能当一个局部 helper 用完即弃。说到底，它不是零散的字符串 helper，而是一个可以被编排系统识别的节点。

### 为什么连 `Callback` 也会进来

很多人看到 Prompt 组件的回调支持，会有一个误判：

“模板格式化也要回调？是不是有点小题大做了？”

如果你只是把 `ChatTemplate` 当字符串替换器，你确实会这么想。
但如果你已经接受了它是一个正式组件，这件事就很合理了。

官方给了 `prompt.CallbackInput` 和 `prompt.CallbackOutput`，这意味着你在模板格式化前后，是可以被回调系统观察到的。

你能看到：

- 输入的变量是什么
- 当前模板集合是什么
- 格式化产出的消息结果是什么

而在生命周期上，对应的就是 `OnStart`、`OnEnd`、`OnError` 这几个钩子。

这层能力的意义，不只是“记个日志”。
而是在告诉你：Prompt 组件也属于 Eino 的运行链路，它不是一个藏在角落里的文本处理函数。


## 7. 总结

如果你问我，本篇 `ChatTemplate` 真正想让人学会什么，我会把答案压成三句话：

1、`ChatTemplate` 解决的是消息组织，不是字符串替换。

2、`MessagesPlaceholder` 是多轮上下文接入的关键，因为它让 history 以 `[]*schema.Message` 的形态进入模板，而不是被你手工压成文本。

3、`Chain`、`Graph`、`Callback` 这些能力同时出现，说明 `ChatTemplate` 从一开始就是组件层能力，不是 prompt 拼接小工具。

所以别再把它当“模板语法说明书”看了。
你一旦把这层看懂，后面再去学 `ToolsNode&Tool`，或者继续往 `Retriever / RAG` 的上下文拼装走，很多设计都会顺理成章。

## 参考资料

- CloudWeGo Eino [ChatTemplate 使用说明](https://www.cloudwego.io/zh/docs/eino/core_modules/components/chat_template_guide/)

---

## 发布说明

- GitHub 主文：[AI 大模型落地系列｜Eino 组件核心篇：ChatTemplate 为什么不是字符串拼接](./02-ChatTemplate为什么不是字符串拼接.md)
- CSDN 跳转：[AI 大模型落地系列｜Eino 组件核心篇：ChatTemplate 为什么不是字符串拼接](https://zhumo.blog.csdn.net/article/details/159500932)
- 官方文档：[ChatTemplate 使用说明](https://www.cloudwego.io/zh/docs/eino/core_modules/components/chat_template_guide/)
- 最新版以 GitHub 仓库为准。

