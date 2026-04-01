声明：本文数据源于官方文档与官方实现，重点参考 [ChatModel 使用说明](https://www.cloudwego.io/zh/docs/eino/core_modules/components/chat_model_guide/)。
@[TOC](很多人学 Eino，都把 ChatModel 想简单了：)

为什么很多人已经会调用大模型了，到了 Eino 里却依旧容易把 `ChatModel` 用浅？

因为太多人把它当成一个“聊天接口”。
发一组消息，回一段文本，事情好像就结束了。

可如果 `ChatModel` 只值这么点钱，Eino 根本没必要单独给它设计-（接口、Option、Callback、Tool 绑定和流式输出）等...

你以为官方在讲用法，实际上它在交代边界。

如果你前面刚看过 入门篇的`ChatModel和Message`，那篇博客更多是为了教你把对话先跑起来。
而本篇换到了另一个角度切入：

> `ChatModel` 在 Eino 里，到底解决了什么问题？

## 1. 为什么 `ChatModel` 不是“普通聊天接口”

如果你直接调厂商 SDK，拿到的是“某家的模型能力”。
如果你接的是 `ChatModel`，拿到的才是“Eino 能认的模型组件”。

这两者的差别，不在于能不能聊天，而在于边界有没有被收口。

**第一，它统一了模型接入方式。**

今天你接 OpenAI，明天接千问，后天可能接公司内部网关。
业务层最怕的不是换模型，而是换模型就换一套调用姿势。

`ChatModel` 把这件事压成了统一接口：给它一组消息，拿回一条消息，或者拿回一个流。
上层逻辑不用直接面对每家供应商那些参数细节。

**第二，它给编排层留了稳定支点。**

后面的 `Chain`、`Graph`、`Agent`、`Runner` 为什么能往上长？
不是因为这些名词更高级，而是因为底下先有一个统一的组件协议。

没有这层协议，所谓编排，很容易写成一堆和供应商 SDK 紧耦合的胶水代码。

**第三，它从一开始就给扩展留了位置。**

如果你把本篇读完，在回过头来，就会发现：
它还拓展的有：

- `Stream`
- `WithTools`
- 多模态字段
- `Callback`
- 自定义实现

这已经说得很清楚了：官方压根没把 `ChatModel` 当成一个“打印回答”的玩具层。
它是模型能力进入 Eino 体系的总入口。

一句话说透：

> `ChatModel` 解决的不是“我能不能调模型”，而是“模型能力该怎样以组件的方式进入 Eino”。

## 2. 看过接口，你才能知道官方真正想让你理解的

官方给出的核心接口很短，但信息量不小：

```go
type BaseChatModel interface {
    Generate(ctx context.Context, input []*schema.Message, opts ...Option) (*schema.Message, error)
    Stream(ctx context.Context, input []*schema.Message, opts ...Option) (*schema.StreamReader[*schema.Message], error)
}

type ToolCallingChatModel interface {
    BaseChatModel
    WithTools(tools []*schema.ToolInfo) (ToolCallingChatModel, error)
}
```

这段代码里，最值得盯住的是三个动作。

**1. `Generate`**

一次性拿完整回复。
适合摘要、改写、离线任务、后台处理这类“等结果出来再继续”的场景。

**2. `Stream`**

按流式往外吐内容。
这不是锦上添花，而是产品化中，与用户交互的常态。

控制台逐字打印、前端打字机效果、边生成边观察 ToolCall，靠的都是这条链路。

**3. `WithTools`**

这个非常关键。
它说明 `ChatModel` 在 Eino 里从来就不只是“聊聊天”。
它还可以被绑定工具，让模型从“只会生成文本”进入“可以做 tool calling（<small>工具调用</small>）”的状态。
但同时，你会发现官方没有把 ChatModel 定义成一个巨大的万能接口。
相反，它先给你了一个最小基座：

- 完整输出
- 流式输出
- 工具绑定

这就是很典型的组件设计思路。
先守住稳定的地基，再把扩展能力挂在边上。

为了更直观点，你可以先把最小调用记成这样：

```go
// 传给模型的多条消息
messages := []*schema.Message{
    schema.SystemMessage("你是一个简洁、专业的 Go 助手。"),
    schema.UserMessage("用一句话解释 Eino 的 ChatModel。"),
}

// 直接生成
reply, err := cm.Generate(ctx, messages)
if err != nil {
    return err
}
fmt.Println(reply.Content)

// 流式生成
stream, err := cm.Stream(ctx, messages)
if err != nil {
    return err
}
defer stream.Close()

for {
    chunk, err := stream.Recv()
    if errors.Is(err, io.EOF) {
        break
    }
    if err != nil {
        return err
    }
    fmt.Print(chunk.Content)
}
```

这段代码不复杂，但它把 `Generate` 和 `Stream` 的边界已经说清了：

- `Generate` 更像一次性拿结果
- `Stream` 更像面向真实交互过程

## 3. `Message` 为什么不是字符串，而是“对话协议”

很多人第一次接触 `schema.Message`，会觉得它不过是 prompt 的壳。

这就看浅了。

在 Eino 里，你操作的不是“一个大字符串”，而是一组有角色、有结构、有上下文语义的消息。

可以先看一个精简后的结构：

```go
type Message struct {
    // 表示当前消息的角色类型，比如 system、user、assistant、tool
    Role schema.RoleType

    // 表示消息的纯文本内容
    Content string

    // 表示用户输入的多段内容，可包含文本、图片等多模态输入片段
    UserInputMultiContent []schema.MessageInputPart

    // 表示模型生成的多段输出，可包含文本、工具结果等多模态输出片段
    AssistantGenMultiContent []schema.MessageOutputPart

    // 表示当前消息中包含的工具调用信息
    ToolCalls []schema.ToolCall

    // 表示这条消息附带的响应元信息，比如 token 使用情况、模型信息等
    ResponseMeta *schema.ResponseMeta
}
```

这里最重要的不是字段多，而是字段背后的含义。

**`Role` 说明这条消息是谁说的。**

常见角色有四个：

- `system`
- `user`
- `assistant`
- `tool`

一旦角色明确了，整个上下文组织方式就变了。
你不再是“把几段文字拼起来”，而是在维护一套对话协议。

**`Content` 只是最基础的文本承载。**

如果你只看到这个字段，很容易误以为 `Message` 还是老式 prompt。
但后面的字段已经告诉你，官方想解决的问题远不止纯文本。

**`UserInputMultiContent` 和 `AssistantGenMultiContent` 说明它天生考虑了多模态。**

文本、图片、音频、视频、文件，不是后补功能，而是消息层就留出了位置。

**`ToolCalls` 说明工具调用结果不是外挂。**

以后你做 tool calling，多轮链路里 assistant 发起工具调用、tool 返回结果，最终都要回到 `Message` 这套协议里。

**`ResponseMeta` 说明模型输出不只是正文。**

结束原因、token 统计这类信息，后面做可观测、排障、成本分析都要靠它。

所以真正该记住的不是“`Message` 有哪些字段”，而是这句话：

> `schema.Message` 不是 prompt 文本，而是 Eino 里组织上下文的基本单元。

这一层一旦想明白，后面的多轮、Tool、多模态、Callback，你都会看得顺很多。

## 4. `Option` 不是参数补丁，而是模型调用的统一调度入口

很多人对 `Option` 的第一反应是：哦，就是几个可选参数。

这么理解不算错，但还是浅了半层。

因为 Eino 把模型参数统一塞进 `Option`，不是为了写法好看，而是为了让上层代码用统一方式调度不同模型能力。

官方这页文档提到的公共 Option 里，最常用的是这些：

* `WithTemperature` // 设置采样温度，控制输出随机性
* `WithMaxTokens` // 设置本次生成的最大输出 token 数
* `WithModel` // 指定调用的模型名称
* `WithTopP` // 设置 TopP 采样范围，控制候选词筛选
* `WithStop` // 设置停止词，命中后终止生成
* `WithTools` // 设置当前可供模型调用的工具列表
* `WithToolChoice` // 设置模型的工具调用策略或指定调用某个工具


你可以像这样传：

```go
reply, err := cm.Generate(ctx, messages,
    model.WithTemperature(0.7),
    model.WithMaxTokens(1024),
    model.WithModel("qwen-plus"),
    model.WithTopP(0.9),
    model.WithStop([]string{"Observation:"}),
)
```

工具相关的配置也能走统一入口：

```go
reply, err := cm.Generate(ctx, messages,
    model.WithTools(tools),
    model.WithToolChoice(schema.ToolChoiceRequired, "query_weather"),
)
```

这里有个很容易混淆的点，顺手说清。

处于不同位置的同名 `WithTools`：

- `ToolCallingChatModel.WithTools(tools)`：把工具绑定到一个新 model 实例上
- `model.WithTools(tools)`：把工具作为本次调用的 Option 传进去

一个偏“模型实例能力绑定”。
一个偏“单次调用配置”。

这就是组件体系里很常见的设计手法。
同样叫 `WithTools`，但因为层次不一样，所以职责也不同。

本段落的意义，在于彰显 `Option` 的价值，并不是给你多几个旋钮而已。
它真正解决的是：

- 不同模型调用参数的统一入口
- 工具和模型选择的统一配置方式
- 上层业务不必和厂商私有参数直接耦合

如果你以后做的是平台化 AI 能力接入，这一层会特别有价值。

## 5. 为什么官方单独强调 `Callback`

很多人第一次看 `Callback`，会本能地把它当成“日志钩子”。

这也不算错，但仍然低估了它。

`Callback` 真正解决的是：你怎么观察一次模型调用到底发生了什么。

官方给的输入输出结构也很直接：

```go
type CallbackInput struct {
    Messages    []*schema.Message
    Model       string
    Temperature *float32
    MaxTokens   *int
    Extra       map[string]any
}

type CallbackOutput struct {
    Message    *schema.Message
    TokenUsage *schema.TokenUsage
    Extra      map[string]any
}
```

这意味着你至少能在三个时点做事：

- 调用前看输入消息和模型配置
- 调用后看输出消息和 token 使用
- 流式调用时观察中间输出，尤其是 ToolCall 片段

最小示意可以写成这样：

```go
handler := &callbacksHelper.ModelCallbackHandler{
    OnStart: func(ctx context.Context, info *callbacks.RunInfo, input *model.CallbackInput) context.Context {
        fmt.Printf("start model=%s messages=%d\n", input.Model, len(input.Messages))
        return ctx
    },
    OnEnd: func(ctx context.Context, info *callbacks.RunInfo, output *model.CallbackOutput) context.Context {
        fmt.Printf("done tokens=%+v\n", output.TokenUsage)
        return ctx
    },
}
```

如果你已经开始做 Agent，这个东西就更重要了。

因为一旦链路里出现：

- 多轮消息
- 工具调用
- 流式输出
- 多次模型往返

没有 `Callback`，你很快就会重新掉进黑盒里。
而有了 `Callback`，后面接 `Trace`、接可观测平台、看 token 成本、查工具调用问题，才有抓手。


## 6. 你可以自己实现 `ChatModel`
我觉得这页最容易被忽略、但却最有味道的部分，不是 `Generate`，也不是 `Message`。

而是最后那段“自行实现参考”。

**官网原话：**
![在这里插入图片描述](https://i-blog.csdnimg.cn/direct/4ea977f6952644c491731f7d458a95d9.png)




这意味着 Eino 不只是给你几个现成适配器。
它还在定义一条规范：

> 如果你要接第三方模型、公司内网网关、私有推理服务，你应该按什么协议把它接进 Eino。

这件事对做企业内部平台的人尤其重要。

可以看成这种感觉：

```go id="d6kyoz"
Eino的消息  -->  你包装一下  -->  你公司的模型接口
你公司的返回 -->  你再包装一下 -->  Eino的消息
```

比如你公司已经有这样一个接口：

```go id="xsxg2w"
func CallCompanyLLM(prompt string) (string, error) {
    return "这是公司模型返回的结果", nil
}
```

那你自己实现 `ChatModel`，本质上就是再包一层：

```go id="q0m71m"
func (m *MyChatModel) Generate(ctx context.Context, messages []*schema.Message) (*schema.Message, error) {
    prompt := messages[len(messages)-1].Content

    text, err := CallCompanyLLM(prompt)
    if err != nil {
        return nil, err
    }

    return &schema.Message{
        Role:    schema.Assistant,
        Content: text,
    }, nil
}
```

意思就是：

* 从 Eino 收到消息
* 拿出内容
* 调你自己的模型接口
* 把结果塞回 `schema.Message`
* 这样 Eino 就能继续用了

所以“自己实现 `ChatModel`”翻成人话就是：

**“把你自己的模型调用方式，包装成 Eino 规定的样子。”**

这时再看前面开头的那条规范：

- **第三方模型**：比如别家的大模型服务
- **公司内网网关**：比如你们公司统一封装过的模型接口
- **私有推理服务**：比如你们自己部署的模型服务

这时候，如果你只会直接调 SDK，后面的 Agent、Tool、Callback、Graph 往往都很难复用。
但如果你按 `ChatModel` 协议接一层，整个系统就顺了。

官方给出的实现重点，可以压成一个 checklist：

- 兼容公共 Option
- 正确触发 `OnStart / OnEnd / OnError`
- 流式输出结束后及时关闭 writer
- `WithTools` 返回新实例，而不是偷偷改当前实例
- 让自定义模型也能被 `Chain`、`Graph`、`Agent` 直接消费

这才是组件ChatModel最硬的一层价值。
所以本篇不仅在教你如何用框架，也在教你怎么把自己的模型能力接成框架的一部分。
## 7. 总结

如果你问我，这篇 `ChatModel 使用说明` 到底在讲什么，我会给一个很直接的回答：

> 它讲的不是“怎么调一次模型”，而是“模型能力在 Eino 里该怎样被标准化、结构化、可扩展地接入”。

再压缩成三句话，就是：

- `Message` 是协议，不是字符串
- `Stream` 是正经产品形态，不是锦上添花
- `WithTools + Callback` 说明 `ChatModel` 从一开始就不是玩具层

所以别急着一上来就冲 `Agent`。
很多人第一次学 Eino，最容易忽略的，恰恰是下面这层最关键的地基。

你把 `ChatModel` 看浅了，后面学到的很多东西都会像“会用”，但不一定“真懂”。
而这层一旦吃透，`Tool`、`Trace`、`Runner`、`Agent` 这些能力，都会开始变得顺理成章。

## 参考资料

- Eino [ChatModel 使用说明](https://www.cloudwego.io/zh/docs/eino/core_modules/components/chat_model_guide/)
- AI大模型落地系列的入门必备教程中的[ChatModel与Message](https://blog.csdn.net/2302_80067378/article/details/159393888?spm=1001.2014.3001.5501)
