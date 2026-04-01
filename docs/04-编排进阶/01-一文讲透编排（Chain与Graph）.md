# AI 大模型落地系列｜Eino 编排进阶篇：一文讲透编排（Chain 与 Graph）

> GitHub 主文：[当前文章](./01-一文讲透编排（Chain与Graph）.md)
> CSDN 跳转：[AI 大模型落地系列｜Eino 编排进阶篇：一文讲透编排（Chain 与 Graph）](https://zhumo.blog.csdn.net/article/details/159571042)
> 官方文档：[Chain/Graph 编排介绍](https://www.cloudwego.io/zh/docs/eino/core_modules/chain_and_graph_orchestration/chain_graph_introduction/)
>
> 最新版以 GitHub 仓库为准，CSDN 作为分发入口，官方文档作为权威参考。

**一句话摘要**：从执行关系、类型边界、统一运行时三条线看清为什么复杂链路最终都会走到编排。
**适合谁看**：已经理解基础组件，准备进入复杂链路和运行时建模的 Go 开发者。
**前置知识**：ChatTemplate、Tool、Retriever 等核心组件、基础泛型和接口认知
**对应 Demo**：[examples/chain-graph](../../examples/chain-graph/README.md)

**面试可讲点**
- 能解释 Chain 和 Graph 的差别在于心智模型和关系显式度，而不是复杂度高低。
- 能讲清 Compile、Runnable、类型对齐、状态边界这些编排层关键词。

---
很多人第一次看 Eino 的 `Chain / Graph`，第一反应都差不多：

不就是把 `prompt`、`model`、`tool` 接一下吗？

这件事自己写几个函数也能干，为什么还要单独学一套编排？

如果你只是跑个 demo，这个想法没什么问题。
但只要链路一变长，问题马上就会冒出来：

- 上一个节点到底给下一个节点传了什么，靠 `any` 还是靠猜？
- 同一条链路既要支持完整输出，又要支持流式输出，代码是不是得写两套？
- 工具调用、分支执行、状态共享，到底写在业务里，还是写在框架里？
- 多个节点汇聚到一个节点时，数据怎么合并，谁来兜底？

这些问题如果都散在业务代码里，系统也能跑。
但通常跑不久就会开始难改、难查、难扩。

所以这篇文章不打算做一份“API 目录导览”。
我更想讲清楚这件事：

> `Chain / Graph` 解决的不是“怎么把几个组件连起来”，而是“复杂执行链路怎样以稳定、可检查、可扩展的方式跑起来”。

本文会按三条线往下讲：

- 为啥要用
- 工程视角怎么拆
- 代码场景怎么落

`Graph` 是主角。
`Chain` 放到后面讲，因为它本质上是更顺手的入口，不是另一套世界观。

## 1. 为什么很多项目最后都会走到编排

如果你的程序只有一步，比如“把一条用户消息送进模型，然后拿回回答”，那当然不需要太重的编排。

问题在于，大多数真实项目不会永远停在这一步。

你很快就会碰到下面这些事：

- 先做 prompt 组装，再调用模型
- 模型命中了工具调用，还要执行工具，再把结果接回消息链路
- 有的节点要走同步，有的节点要走流式
- 某些运行过程需要保留状态，供后续节点继续判断
- 某些节点前面有多个上游，后面还有多个分支

这时你会发现，麻烦不在于“多写几个函数”。
麻烦在于，这些函数之间其实已经存在明确的运行关系。

它们不是散点逻辑，而是一张执行图。

从工程角度看，`Chain / Graph` 真正想收口的是 4 件事：

**第一，节点之间的输入输出边界。**

上游吐出来的值，下游到底能不能接，不能靠上线以后才发现。
Eino 的思路是：尽量在 `Compile` 阶段就把这件事说明白。

**第二，执行关系。**

谁先跑，谁后跑，谁分支，谁汇聚，谁结束，这些都不该藏在几十行 if/else 和回调里。

**第三，运行时范式。**

同一条编排链，不应该“同步是一套写法，流式又是一套写法”。
Eino 最后编译出来的是统一的 `Runnable`，可以 `Invoke`、可以 `Stream`、也可以 `Transform`。

**第四，工程扩展点。**

状态、回调、工具调用、嵌套图，这些东西都不是 demo 里最抢眼的功能，但它们才决定你这套链路后面能不能活得久。

所以你如果要问：

> 为什么很多后端工程师一开始觉得 `Chain / Graph` 有点“重”，做一阵子又会反过来觉得它有必要？

答案很简单：

因为系统一复杂，你迟早都要面对“编排”这件事。
区别只在于，你是显式地把它交给框架，还是隐式地把它塞进业务代码。

## 2. 先把 Graph 看懂：它编排的不是函数，而是运行关系

很多人第一次看 `Graph`，关注点全在“怎么连节点”。

这只看到了表面。

`Graph` 真正重要的，不是你能不能写出 `AddEdge`。
而是它把“节点”和“节点之间的运行关系”明确成了一张图。

看一个最小闭环：

```go
ctx := context.Background()

g := compose.NewGraph[map[string]any, *schema.Message]()

tpl := prompt.FromMessages(
    schema.FString,
    schema.UserMessage("what's the weather in {location}?"),
)

_ = g.AddChatTemplateNode("prompt", tpl)
_ = g.AddChatModelNode("model", &mockChatModel{})

_ = g.AddEdge(compose.START, "prompt")
_ = g.AddEdge("prompt", "model")
_ = g.AddEdge("model", compose.END)

r, err := g.Compile(ctx)
if err != nil {
    panic(err)
}

out, err := r.Invoke(ctx, map[string]any{"location": "beijing"})
if err != nil {
    panic(err)
}
fmt.Println(out.Content)

stream, err := r.Stream(ctx, map[string]any{"location": "beijing"})
if err != nil {
    panic(err)
}
defer stream.Close()

for {
    chunk, err := stream.Recv()
    if errors.Is(err, io.EOF) {
        break
    }
    if err != nil {
        panic(err)
    }
    fmt.Println(chunk.Content)
}
```

- 这段代码解决了什么工程问题：它把 `prompt -> model -> end` 这条执行链路显式表达了出来，并在 `Compile` 之后收口成一个统一可运行对象。
- 不用编排时，这段逻辑会散到哪里：模板格式化、模型调用、同步输出、流式输出这些逻辑通常会散在 controller、service、helper 甚至 goroutine 里，最后没人说得清“这条链路本来长什么样”。

这段代码最值得盯住的，不是天气问题，也不是 `mockChatModel`。

而是 5 个关键词：

**1. `compose.NewGraph[I, O]`**

图的输入、输出类型在一开始就定下来了。
这和“全程都传 `map[string]any`，最后再断言类型”的思路不一样。

**2. `START / END`**

图不是一堆松散节点。
它有明确入口，也有明确终点。

**3. Node**

`ChatTemplate`、`ChatModel`、`ToolsNode`、`Lambda`、甚至另一个 `Graph`，都可以是节点。
也就是说，`Graph` 编排的不是某一种固定组件，而是“逻辑节点”。

**4. Edge**

边不是装饰。
边定义的是下一个要跑谁。
这意味着“关系”本身成了第一等公民。

**5. `Compile`**

这一步最容易被低估。

很多人会想：我都已经把节点和边加完了，为什么还要多一次编译？

因为 `Compile` 干的不是“形式化地收个尾”。
它是在把你刚才搭出来的图，转成一个真正可运行、可检查的 `Runnable`。

说白一点：

> `Graph` 不是“边写边跑”的胶水脚本，它更像是先把执行拓扑搭清楚，再生成运行体。

这也是为什么 `Compile` 不是多余一步。

它把“图长什么样”和“图怎么跑”切开了。
前者是结构定义，后者是运行时。

## 3. Graph 最值钱的，不是能连线，而是把边界定死

如果只把 `Graph` 理解成“可以把节点画成一张图”，那还是太浅。

它真正值钱的地方，在于把一条复杂链路里最容易失控的边界收住了。

### 3.1 先定类型，再谈连接

官方在“编排的设计理念”里反复强调一个词：`类型对齐`。

这不是文档里的漂亮话。
这其实是在回答一个很现实的问题：

> 上一个节点的输出，凭什么就能当下一个节点的输入？

如果你的方案是“先都塞成 `any` 再说”，那后面每个节点都得自己做类型断言。
如果你的方案是“统一都传 `map[string]any`”，那心智负担也只是换了个地方。

Eino 走的是另一条路：

- 节点尽量保持开发者预期中的具体类型
- 在 `Compile` 阶段检查上下游能不能对齐
- 必要时通过 `WithOutputKey`、`WithInputKey` 做受控转换

这套设计对 Go 工程师很友好。

因为你脑子里想的，不再是“这团 `any` 里面可能装了什么”。
而是“这个节点吐出来的东西，下一个节点有没有资格接”。

这就像搭积木。
尺寸对上了，才能接上。

### 3.2 `WithOutputKey / WithInputKey` 不是小技巧，而是汇聚场景的正道

很多人把 `WithOutputKey`、`WithInputKey` 当成“偶尔拿来修一下类型”的小技巧。

其实不是。

它们真正重要的地方，在于多上游汇聚时，你必须正面回答两个问题：

- 多个上游输出怎么合并？
- 下游到底从哪一个 key 取值？

比如上游输出的是 `string`，但多个节点最终要汇聚到一个 `map[string]any` 节点，这时可以用 `compose.WithOutputKey("query")` 把它包成 map。

反过来，如果上游已经是 `map[string]any`，而下游只想拿其中一个字段，则用 `compose.WithInputKey("query")` 明确取值。

这件事看起来只是类型转换。
本质上是在避免“汇聚以后到底该读哪份数据”变成隐式约定。

### 3.3 外部变量只读，不是洁癖，是并发和流式场景的底线

这是我觉得很多人最容易忽略、但又最工程化的一条原则。

官方明确提到，图里节点之间的数据流转，本质上是变量赋值，不是深拷贝。
所以当输入是 `map`、`slice`、指针这类引用类型时，如果你在节点内部直接修改它，就可能把副作用带到外面。

这在分支、扇出、流式场景里尤其危险。

因为你以为自己只是“顺手改一下”。
实际上你改的可能是整个运行过程共享着的那份值。

所以 Eino 的建议很明确：

> Node、Branch、Handler 内部默认不要修改输入；真要改，先自己 Copy。

这不是框架保守。
这是运行时系统必须守住的底线。

### 3.4 `Runnable` 统一了运行姿势

`Graph` 一旦 `Compile` 完，拿到的是 `Runnable`。

这件事很关键。

因为这说明编排产物最终不是“某个特殊 Graph 对象”。
而是一个统一运行入口。

它至少有三种常用姿势：

- `Invoke`：完整输入，完整输出
- `Stream`：完整输入，流式输出
- `Transform`：流式输入，流式输出

这意味着你不需要为了“换成流式”就重新发明一条执行链。

框架会在运行时帮你补齐缺失的流式范式。
这比业务层自己维护两套流程稳定得多。

## 4. ToolCallAgent 这种场景，为什么更适合挂进 Graph

如果说最能体现 `Graph` 工程价值的场景，我认为不是天气 demo。

而是 `ToolCallAgent`。

因为这个场景刚好包含了三层边界：

- Prompt 怎么组
- 模型怎么做工具决策
- 工具结果怎么重新回到消息链路

看一个裁剪后的主链：

```go
chatTpl := prompt.FromMessages(
    schema.FString,
    schema.SystemMessage("你是一名房产经纪人，结合用户信息推荐房产。"),
    schema.MessagesPlaceholder("message_histories", true),
    schema.UserMessage("{user_query}"),
)

chatModel, _ := openai.NewChatModel(ctx, modelConf)

userInfoTool := utils.NewTool(
    &schema.ToolInfo{
        Name: "user_info",
        Desc: "根据用户姓名和邮箱查询公司、职位、薪酬",
        ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
            "name":  {Type: "string", Desc: "用户姓名"},
            "email": {Type: "string", Desc: "用户邮箱"},
        }),
    },
    func(ctx context.Context, input *userInfoRequest) (*userInfoResponse, error) {
        return &userInfoResponse{
            Name:     input.Name,
            Email:    input.Email,
            Company:  "Bytedance",
            Position: "CEO",
            Salary:   "9999",
        }, nil
    },
)

info, _ := userInfoTool.Info(ctx)
_ = chatModel.BindForcedTools([]*schema.ToolInfo{info})

toolsNode, _ := compose.NewToolNode(ctx, &compose.ToolsNodeConfig{
    Tools: []tool.BaseTool{userInfoTool},
})

g := compose.NewGraph[map[string]any, []*schema.Message]()
_ = g.AddChatTemplateNode("template", chatTpl)
_ = g.AddChatModelNode("chat_model", chatModel)
_ = g.AddToolsNode("tools", toolsNode)

_ = g.AddEdge(compose.START, "template")
_ = g.AddEdge("template", "chat_model")
_ = g.AddEdge("chat_model", "tools")
_ = g.AddEdge("tools", compose.END)

r, _ := g.Compile(ctx)
out, _ := r.Invoke(ctx, map[string]any{
    "message_histories": []*schema.Message{},
    "user_query":        "我叫 zhangsan，邮箱是 zhangsan@bytedance.com，帮我推荐一处房产",
})
```

- 这段代码解决了什么工程问题：它把“提示词准备 -> 模型决策 -> 工具执行 -> 结果回链路”压成了一条明确执行链，而不是让工具调用散在业务流程里。
- 不用编排时，这段逻辑会散到哪里：prompt 组装、模型调用、`ToolCall` 解析、工具分发、工具结果封装、下一轮消息拼接，最后大概率会混在同一个 service 方法里。

这里最重要的一句话是：

> `ChatModel` 负责决定调谁，`Graph` 负责把整条执行链跑通，`ToolsNode` 负责把已经做出的调用真正执行掉。

这个边界一旦看清，很多误解都会消失。

比如：

- `ToolsNode` 不是决策器
- `Tool` 不是流程控制器
- `Graph` 不是为了让代码更“好看”才存在

它存在的原因很现实：

如果你手写这条链，早期也能跑。
但一旦你要加第二个工具、要记录 callback、要改成流式、要嵌套别的节点，代码会迅速变成“谁都能改，谁都不敢动”的样子。

`Graph` 的价值就在这儿。

它不是替你写业务。
它是替你把执行边界立住。

## 5. Graph with state，重点不是“能存数据”，而是“数据放在哪一层”

很多人一看到 `state`，直觉会很兴奋：

那我是不是终于有地方塞各种临时变量了？

如果你这样理解，后面很容易把 `state` 用偏。

`Graph with state` 的重点，不是“图里可以放全局变量”。
而是“这次运行过程中，需要有一份只属于这次运行的上下文”。

看一个精简后的例子：

```go
type runState struct {
    Steps []string
}

g := compose.NewGraph[string, string](
    compose.WithGenLocalState(func(ctx context.Context) *runState {
        return &runState{}
    }),
)

_ = g.AddLambdaNode(
    "prepare",
    compose.InvokableLambda(func(ctx context.Context, in string) (string, error) {
        return strings.ToUpper(in), nil
    }),
    compose.WithStatePreHandler(func(ctx context.Context, in string, state *runState) (string, error) {
        state.Steps = append(state.Steps, "input:"+in)
        return in, nil
    }),
    compose.WithStatePostHandler(func(ctx context.Context, out string, state *runState) (string, error) {
        state.Steps = append(state.Steps, "prepare:"+out)
        return out, nil
    }),
)

_ = g.AddLambdaNode(
    "finish",
    compose.InvokableLambda(func(ctx context.Context, in string) (string, error) {
        var history string
        err := compose.ProcessState[*runState](ctx, func(_ context.Context, state *runState) error {
            history = strings.Join(state.Steps, " -> ")
            state.Steps = append(state.Steps, "finish:"+in)
            return nil
        })
        if err != nil {
            return "", err
        }
        return history + " -> finish:" + in, nil
    }),
)

_ = g.AddEdge(compose.START, "prepare")
_ = g.AddEdge("prepare", "finish")
_ = g.AddEdge("finish", compose.END)
```

- 这段代码解决了什么工程问题：它把“单次运行上下文”显式挂在图上，而不是让节点通过包变量、共享 map 或上下文外的全局对象偷偷交换信息。
- 不用编排时，这段逻辑会散到哪里：某些人会把状态塞到闭包里，有些人会塞进全局 map，还有些人会把它挂到业务 struct 上，最后状态边界和生命周期一起失控。

这段代码里有 3 个点要分开看。

**1. `WithGenLocalState`**

它定义的是：每次运行这张图时，怎么生成一份新的状态。

注意，是“每次运行一份新的”。
不是“整个应用启动以后共用一份”。

**2. `WithStatePreHandler / WithStatePostHandler`**

它们是节点外侧的钩子。

你可以理解成：

- 节点真正执行前，先看一下输入和状态
- 节点真正执行后，再看一下输出和状态

这很适合做运行过程中的记录、补充、调整。

**3. `ProcessState`**

这是节点内部读写状态的入口。

当节点本身需要根据历史状态做判断时，就该走这里，而不是绕出去摸别的共享变量。

所以 `state` 的正确打开方式，不是“我终于有个地方可以乱塞东西”。

而是：

> 这次运行里，哪些上下文确实属于图本身，而且后续节点还要继续用？

如果不满足这个条件，就别放。

比如数据库连接、全局配置、跨请求缓存，这些都不该进这里。
它们不是“单次运行上下文”。

## 6. Chain 为什么是更顺手的入口，而不是另一套框架

官方文档里有一句话我很认同：

> `Chain` 可以视为 `Graph` 的简化封装。

这句话很重要。

因为很多人学到这里，会产生两个相反的误解：

- 要么觉得 `Chain` 太简单，像玩具
- 要么觉得 `Chain` 和 `Graph` 是两套并列框架

这两个理解都不对。

`Chain` 的本质，是把“线性链路”写得更顺。

看一个裁剪过的例子：

```go
parallel := compose.NewParallel().
    AddLambda("role", compose.InvokableLambda(func(ctx context.Context, kvs map[string]any) (string, error) {
        role, _ := kvs["role"].(string)
        if role == "" {
            role = "bird"
        }
        return role, nil
    })).
    AddLambda("input", compose.InvokableLambda(func(ctx context.Context, kvs map[string]any) (string, error) {
        return "你的叫声是怎样的？", nil
    }))

rolePlayer := compose.NewChain[map[string]any, *schema.Message]()
rolePlayer.
    AppendChatTemplate(prompt.FromMessages(
        schema.FString,
        schema.SystemMessage("You are a {role}."),
        schema.UserMessage("{input}"),
    )).
    AppendChatModel(cm)

chain := compose.NewChain[map[string]any, string]()
chain.
    AppendLambda(compose.InvokableLambda(func(ctx context.Context, kvs map[string]any) (map[string]any, error) {
        return kvs, nil
    })).
    AppendBranch(
        compose.NewChainBranch(branchCond).
            AddLambda("b1", b1).
            AddLambda("b2", b2),
    ).
    AppendPassthrough().
    AppendParallel(parallel).
    AppendGraph(rolePlayer).
    AppendLambda(compose.InvokableLambda(func(ctx context.Context, m *schema.Message) (string, error) {
        return m.Content, nil
    }))

r, _ := chain.Compile(ctx)
output, _ := r.Invoke(ctx, map[string]any{})
```

- 这段代码解决了什么工程问题：它把一条以线性推进为主的执行链写得更紧凑，同时保留了分支、并行和图嵌套能力。
- 不用编排时，这段逻辑会散到哪里：每一步都要手工传值、手工判断分支、手工等待并行结果、手工把子流程接回来，最后“主链”本身会淹没在细节里。

这段代码说明了两件事。

**第一，`Chain` 并不弱。**

它不是只有 `AppendChatTemplate`、`AppendChatModel` 这种最简单的串联。
它还能接 `branch`、接 `parallel`、接另一个 `graph`。

**第二，`Chain` 仍然是线性心智模型。**

你写的时候，脑子里想的是“先做 A，再做 B，再做 C”。
这比直接上图更顺手。

所以很多场景下，`Chain` 应该是你的第一选择。
尤其是：

- 处理链路天然线性
- 中间节点之间没有太多复杂汇聚
- 你更想快速表达主路径

但如果你已经明显开始关心：

- 节点关系是不是要显式画出来
- 哪些节点是多上游汇聚
- 哪些节点是复杂分支
- 哪些地方要更强的状态控制

那就别再硬拿 `Chain` 扛所有场景了。

## 7. 什么时候用 Chain，什么时候直接上 Graph

这件事不复杂。
我直接给结论。

### 更适合 `Chain` 的场景

- 主路径基本是线性的
- 你更在乎表达“步骤顺序”
- 分支和并行只是局部点缀，不是主体结构
- 你想快速把一个可运行流程搭起来

### 更适合 `Graph` 的场景

- 你需要显式表达节点关系
- 多上游汇聚、扇出、复杂分支比较多
- 你需要更清楚地控制输入输出边界
- 你希望状态、工具调用、嵌套流程都放在一张正式运行图里

### 一个实用判断法

如果你现在写流程时，脑子里更像是在想：

“下一步接谁？”

那多半该用 `Graph`。

如果你现在写流程时，脑子里更像是在想：

“下一步做什么？”

那多半先用 `Chain`。

`Workflow` 则是下一层话题。
它更强调字段映射、控制流和更细颗粒度的编排控制。
但如果你现在还没把 `Chain / Graph` 看清，先别急着跳过去。

## 8. 5 个最容易把编排用浅的坑

### 8.1 把 `map[string]any` 当万能胶

`map[string]any` 不是不能用。

但如果你从头到尾都靠它传值，最后还是会回到“每个节点都在猜 key、猜类型”的老路上。

它更适合：

- 明确的汇聚场景
- 经由 `WithOutputKey`、`WithInputKey` 做受控转换

而不是变成整条链路的默认协议。

### 8.2 只写 `Invoke`，从来不看 `Stream / Transform`

很多 demo 只写 `Invoke`，这是可以理解的。

但你如果做的是实际产品链路，迟早会遇到流式输出。
更进一步，某些节点本身就要吃流、吐流，这时你就得理解 `Transform`。

如果你从设计阶段就把这件事忽略了，后面通常要补一套平行逻辑。

### 8.3 在节点里直接改外部引用类型

这是最隐蔽的坑之一。

尤其是 `map`、`slice`、指针。
你以为自己只是改了当前节点的输入，实际上可能改的是整个运行过程共享的那份值。

这类 bug 一旦叠上分支、并发、流式，排起来会非常难受。

### 8.4 把 `state` 当“什么都能放”的储物箱

`state` 不是跨请求缓存。
不是全局依赖容器。
也不是你懒得设计边界时的逃生门。

它只该放这次运行过程中确实需要被后续节点继续消费的上下文。

### 8.5 把 `Chain` 当成 `Agent`

`Chain` 可以承接很多 agent 的执行步骤。
但它本身不是 agent 概念本身。

如果你把这两个层级混在一起，后面讨论 tool calling、event、runner、workflow agent 时，脑子会越来越乱。

`Chain / Graph` 解决的是编排。
`Agent` 解决的是更上层的智能体运行抽象。

这两个层级要分开。

## 9. 总结

很多人学 `Chain / Graph` 时，最容易走偏的一点，就是把它当成“更高级一点的流程写法”。

这个理解不算错，但远远不够。

它真正值钱的地方在于：

- 把节点和关系显式化
- 把上下游边界定清楚
- 把同步、流式、状态、工具调用纳入统一运行时
- 把复杂链路从业务胶水里拆出来

所以如果你现在还在犹豫：

> 这套东西到底是不是必须学？

我的看法很直接：

如果你只是写一个一屏能看完的小 demo，未必急着学。
但只要你要做的不是一次性脚本，而是一条会持续演进的 AI 应用链路，那你迟早都要面对编排。

与其把编排偷偷写进业务代码，不如正面把它建模出来。

`Graph` 适合你把复杂关系讲清楚。
`Chain` 适合你把主路径写顺。

这两层一旦看懂，后面的 `Workflow`、`Agent`、`GraphTool`，你会顺很多。

## 参考资料

1. [Chain/Graph 编排介绍](https://www.cloudwego.io/zh/docs/eino/core_modules/chain_and_graph_orchestration/chain_graph_introduction/)
2. [编排的设计理念](https://www.cloudwego.io/zh/docs/eino/core_modules/chain_and_graph_orchestration/orchestration_design_principles/)
3. [eino-examples/compose](https://github.com/cloudwego/eino-examples/tree/main/compose)

---

## 发布说明

- GitHub 主文：[AI 大模型落地系列｜Eino 编排进阶篇：一文讲透编排（Chain 与 Graph）](./01-一文讲透编排（Chain与Graph）.md)
- CSDN 跳转：[AI 大模型落地系列｜Eino 编排进阶篇：一文讲透编排（Chain 与 Graph）](https://zhumo.blog.csdn.net/article/details/159571042)
- 官方文档：[Chain/Graph 编排介绍](https://www.cloudwego.io/zh/docs/eino/core_modules/chain_and_graph_orchestration/chain_graph_introduction/)
- 最新版以 GitHub 仓库为准。

