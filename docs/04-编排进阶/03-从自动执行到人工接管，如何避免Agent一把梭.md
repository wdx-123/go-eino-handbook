# AI 大模型落地系列｜Eino 编排篇：从自动执行到人工接管，如何避免Agent一把梭

> GitHub 主文：[当前文章](./03-从自动执行到人工接管，如何避免Agent一把梭.md)
> CSDN 跳转：[AI 大模型落地系列｜Eino 编排篇：从自动执行到人工接管，如何避免Agent一把梭](https://zhumo.blog.csdn.net/article/details/159642323)
> 官方文档：[Interrupt & CheckPoint 使用手册](https://www.cloudwego.io/zh/docs/eino/core_modules/chain_and_graph_orchestration/checkpoint_interrupt/)
>
> 最新版以 GitHub 仓库为准，CSDN 作为分发入口，官方文档作为权威参考。

**一句话摘要**：把 Interrupt、Resume、CheckPoint 和人工审批串成一套真正可治理的 Agent 执行策略。
**适合谁看**：准备在真实业务里接入敏感工具、审批流或人工接管能力的工程师。
**前置知识**：Tool 调用闭环、Chain / Graph 基础、Agent Runner 基础
**对应 Demo**：[官方 ch07 示例（本仓后续补充审批流 demo）](https://github.com/cloudwego/eino-examples/blob/main/quickstart/chatwitheino/cmd/ch07/main.go)

**面试可讲点**
- 能解释 Interrupt/Resume 解决的是可治理执行，而不是简单暂停。
- 能把 CheckPoint、审批策略、版本边界和失败恢复讲成一套生产方案。

---
很多人前面学到 `Agent + Tool` 时，第一反应都很兴奋：

终于不用自己手动串动作了，模型会判断、会选工具、会自动把活干掉。

但只要你把 `Agent` 真正接到 `execute`、`write_file`、`send_mail` 这类 `Tool` 上，问题立刻就变了。

这时你最该关心的已经不是“它能不能调起来”，而是“它准备做危险操作时，谁来审批，现场怎么保存，用户确认后又怎么恢复执行”。

也正因为这样，`Interrupt / Resume` 在工程里不是一个可有可无的交互点，而是一套人工接管机制。

如果说前几章解决的是“Agent 怎么跑起来”，那本章解决的就是另一个问题：

> 当 Agent 已经有能力调用真实 Tool 时，系统怎样在关键动作前停下来，等人确认，再从原地继续往下走。
> 以及，Eino 又是如何把“中断、审批、恢复、持久化”这一整套链路收进同一个运行时里的。

## 1. 为什么敏感 Tool 不能默认全自动

前几章里的 Tool 调用，很多人都会默认理解成一句话：

模型判断需要什么，就直接去调用什么。

这个思路放在 demo 里当然成立。

可一旦 Tool 不再只是“查天气”“读文档”，而是开始触碰真实环境，自动执行的风险会陡增：

- `execute` 可能执行 shell 命令
- `write_file` 可能覆盖配置
- `send_mail` 可能把错误内容发给真实用户
- 某些数据库 Tool 可能直接修改生产数据

很多人一开始会觉得，这不就是给 Tool 加个确认框吗？

实际上，你要解决的不只是“要不要弹个确认框”，而是下面四件事得一起成立：

- Tool 在危险动作前必须真的停下来，而不是只在 UI 上做个提示
- 中断时要把这次调用的上下文保存住，不能确认完以后参数丢了
- 拒绝和批准都要有确定结果，不能让 Runner 卡在半路
- 进程重启、会话切换甚至跨机器恢复时，仍然要知道上次停在什么地方

这就是 `Interrupt / Resume` 存在的背景。

它关心的是人机协作时的执行控制权。

你可以把它理解成：

- 自动执行像自动驾驶
- `Interrupt` 像人工接管

系统当然还是能自己跑。
但到了高风险动作，方向盘必须重新回到人手里。

## 2. 哪些 Tool 应该审批，哪些可以白名单放行

把风险说清以后，工程上第一个要落地的判断，不是先选 API，而是先划边界。

因为没有任何一个团队，会真的愿意给所有 Tool 都弹审批。

那样系统会慢得不可用。

### 2.1 必须审批的，是那些会对外部世界产生真实副作用的 Tool

这类 Tool 最典型：

- 执行命令
- 写文件或删文件
- 改数据库数据
- 调用外部系统发消息、发邮件、下工单
- 修改云资源、网络策略、系统配置

这些动作一旦执行，就不再只是“推理结果”，而是“现实世界里的变化”。

这类能力默认不该全自动。

### 2.2 可以白名单放行的，是强约束下的只读或低风险 Tool

比如：

- 只读查询
- 本地纯计算
- 固定范围内的格式化转换
- 已经被沙箱限制死权限的安全工具

即便如此，我也不建议直接放飞。

更稳妥的做法通常是：

- 白名单按 Tool 类型放
- 黑名单按参数特征拦
- 动态规则按操作范围升级审批

举个很实际的例子：

- `execute("ls")` 和 `execute("rm -rf")` 不该一视同仁
- `write_file` 写临时目录和写核心配置目录，也不该走同一条策略

这时候，`approvalMiddleware` 的价值就会体现出来。

因为它天然适合做这类“按 Tool 名 + 按参数内容”的集中治理。

## 3. Interrupt/Resume 到底在解决什么问题

把边界划清之后，再看机制本身，就更容易理解它为什么一定要成对出现。

`Interrupt / Resume` 解决的，不是交互花活，而是 Tool 的两阶段执行。

如果只看名字，很多人会把 `Interrupt / Resume` 想成“暂停一下，再继续”。

这话不算错，但只是表皮。

在审批流里，它更准确的意思其实是：

**把一次 Tool 调用拆成两次进入。**

第一次进入，不真的执行业务动作，只负责两件事：

- 把当前输入和必要状态保存下来
- 发出一个中断信号，让 Runner 停住并把审批信息交还给外部

第二次进入，也就是用户批准后的 `Resume`，才会去读回之前保存的状态，再决定真正执行还是拒绝返回。

流程图：**第一次调用 -> 中断 -> 审批 -> 恢复 -> 执行**

```go
func myTool(ctx context.Context, args string) (string, error) {
    // 看当前是不是“上次中断后恢复执行”
    // storedArgs 是中断时保存下来的参数
    wasInterrupted, _, storedArgs := tool.GetInterruptState[string](ctx)

    // 如果是第一次执行，就先发起审批并中断，不继续往下执行
    if !wasInterrupted {
        return "", tool.StatefulInterrupt(ctx, approvalInfo(args), args)
    }

    // 如果已经是恢复执行，就读取恢复时附带的审批结果
    isTarget, hasData, result := tool.GetResumeContext[*ApprovalResult](ctx)

    // 只有审批结果属于当前中断点、并且审批通过，才真正执行危险操作
    if isTarget && hasData && result.Approved {
        return doDangerousThing(storedArgs)
    }

    // 审批没通过，或者恢复数据不对，就拒绝执行
    return "operation rejected", nil
}
```

这个代码案例，是为了展示其背后的职责切分：

- `tool.StatefulInterrupt` 负责“抛出中断，同时把本地状态保存”
- `tool.GetInterruptState[T]` 负责“恢复后拿回上次保存的状态，并判断这是不是第二次进入”
- `tool.GetResumeContext[T]` 负责“读取这次 Resume 是否就是冲着当前中断点来的，以及用户到底给了什么恢复数据”

一旦你把这三件事看懂了，后面无论是 Tool 审批、参数补全、用户二次确认，思路都一样。

## 4. 官方 `execute` 示例：一次审批流是怎么跑完的

官方放到 GitHub 上的案例是 `cmd/ch07/main.go` [\[代码\]](https://github.com/cloudwego/eino-examples/blob/main/quickstart/chatwitheino/cmd/ch07/main.go)。

它演示的不是复杂 Agent，而是一个非常有代表性的最小闭环：

- 用户输入一句自然语言
- Agent 判断需要调用 `execute`
- 中间件拦截这次 Tool 调用
- Runner 收到 `Interrupt` 后暂停
- 用户输入 `y/n`
- 系统恢复执行，继续跑完这轮

控制台输出大概是这样：

```text
you> 请执行命令 echo hello

⚠️  Approval Required ⚠️
Tool: execute
Arguments: {"command":"echo hello"}

Approve this action? (y/n): y
[tool result] hello

hello
```

很多人第一次看这个示例时，会把关注点放在“怎么把 `y/n` 读出来”。

但更值得盯住的，是这条执行链到底断在哪里、又是从哪里接回去的。

```text
┌────────────────────────────────────────────┐
│ 用户输入：请执行命令 echo hello             │
└────────────────────────────────────────────┘
                     ↓
         ┌──────────────────────────┐
         │ Agent 决定调用 execute    │
         └──────────────────────────┘
                     ↓
         ┌──────────────────────────┐
         │ approvalMiddleware 拦截   │
         └──────────────────────────┘
                     ↓
         ┌──────────────────────────┐
         │ 抛出 Interrupt            │
         │ 保存 CheckPoint           │
         └──────────────────────────┘
                     ↓
         ┌──────────────────────────┐
         │ Runner 结束当前执行       │
         │ 把审批信息返回调用侧       │
         └──────────────────────────┘
                     ↓
         ┌──────────────────────────┐
         │ 用户确认或拒绝            │
         └──────────────────────────┘
                     ↓
         ┌──────────────────────────┐
         │ Resume 后重新进入 Tool    │
         └──────────────────────────┘
                     ↓
         ┌──────────────────────────┐
         │ 真正执行 execute 或拒绝   │
         └──────────────────────────┘
```

也就是说，`Interrupt` 不是在 Tool 旁边挂一个提示层。
它是真的把**这次调用变成了一个可恢复的执行断点**。

因为只有这样，审批才不是“前端交互”，而是“运行时协议”。

### 4.1 谁在主导这次中断

很多人会直觉地认为，暂停和恢复一定是 Runner 主导的。

实际上，在 Eino 这套机制里，先举手说“我这里要停一下”的，通常是**节点自己**，或者像 Tool middleware 这种包在节点外面的拦截层。

下面这段代码，就是官方示例里最关键的那一层：

> 第一次进来先中断等审批，第二次恢复进来再看审批结果，批准才调用真正的 `execute`。

```go
func (m *approvalMiddleware) WrapInvokableToolCall(
    _ context.Context,
    endpoint adk.InvokableToolCallEndpoint,
    tCtx *adk.ToolContext,
) (adk.InvokableToolCallEndpoint, error) {
    // 只拦截 execute 这个工具；
    // 其他工具不做审批，直接放行
    if tCtx.Name != "execute" {
        return endpoint, nil
    }

    // 返回一个“包了一层审批逻辑”的新 tool 调用函数
    return func(ctx context.Context, args string, opts ...tool.Option) (string, error) {
        // 看这次调用是不是“中断后恢复执行”
        // storedArgs 是上次中断时保存下来的原始参数
        wasInterrupted, _, storedArgs := tool.GetInterruptState[string](ctx)

        // 第一次执行时，不真正调用 execute，
        // 而是先抛出中断，请求外部审批，并把 args 保存起来
        if !wasInterrupted {
            return "", tool.StatefulInterrupt(ctx, &commontool.ApprovalInfo{
                ToolName:        tCtx.Name,
                ArgumentsInJSON: args,
            }, args)
        }

        // 恢复执行后，读取外部传回来的审批结果
        isTarget, hasData, data := tool.GetResumeContext[*commontool.ApprovalResult](ctx)

        // 只有当前恢复数据确实属于这个中断点，
        // 并且拿到了审批结果、且审批通过，才真正执行原始 execute
        if isTarget && hasData && data.Approved {
            return endpoint(ctx, storedArgs, opts...)
        }

        // 审批没通过，就不执行工具，直接返回拒绝结果
        return fmt.Sprintf("tool '%s' disapproved", tCtx.Name), nil
    }, nil
}
```

这段代码把审批流里最关键的职责分工都摆出来了。

### 4.2 `tool.StatefulInterrupt`

它做的不是“报个错”，而是“挂起并存档”。

很多人第一次看会觉得，中断不就是返回一个特殊错误吗？

从调用面上看，确实像。
但它真正做的事情比普通错误大得多：

- 向上层发出“这里要中断”的明确信号
- 挂上展示给用户看的 `info`
- 顺手把 `state` 一起持久化，供下一次 `Resume` 取回

也正因为这样，`StatefulInterrupt` 很适合放那些“当前输入就是恢复时关键证据”的场景。

审批流就是典型例子。

你第一次拦下来的 `args`，就是第二次真正要执行的 `storedArgs`。

### 4.3 `tool.GetInterruptState[T]`

`tool.GetInterruptState[T]` 解决的是“我现在到底是第一次进，还是恢复后第二次进”。

如果没有这个 API，开发者就得自己做一套状态管理：

- 是不是中断过
- 中断时保存了什么
- 从哪儿取回来

这套事如果全让业务自己管，最后很容易变成一堆零散布尔值和自定义上下文。

### 4.4 `tool.GetResumeContext[T]`

`tool.GetResumeContext[T]` 解决的是“这次恢复是不是找我，以及用户到底给了什么”。

这点也很关键。

真实系统里，不一定每次恢复都只对应一个唯一中断点。
尤其到了嵌套图、并行中断、多 Tool 协作时，“恢复谁”本身就已经是个问题。

所以 `GetResumeContext[T]` 给了三层判断：

- 这次 `Resume` 的目标是不是当前中断点
- 恢复时有没有带数据
- 数据是不是当前 Tool 预期的类型

它的作用，就是把恢复这件事从“继续跑”收紧成“有目标地恢复某个断点”。

## 5. 审批为什么适合放在 `middleware`，而不是写死在 Tool 里

看到这里，很自然会冒出一个问题：

我把审批直接写进 `execute` Tool 里不就行了？为什么还要放到中间件中。

因为审批从来都不是某个单点业务逻辑。
它更像一层横切治理规则。

你真正想控制的通常是：

- 哪些 Tool 需要审批
- 哪些参数命中高风险时才审批
- 同步和流式调用是不是一套策略
- 后面新增 Tool 时，规则能不能统一复用

这也是为什么官方示例把它放到了 middleware，而不是塞进 Tool 本体里。

中间件的好处很直接：

- Tool 本身仍只负责“真正做事”
- 审批规则集中在一处配置
- 新增或调整策略时，不需要改每个 Tool 的业务实现

如果用中间件做治理时，有一点非常值得注意：

> 在 Agent 里做治理，**不能**只盯“业务成功路径”，还得把**同步、流式、恢复路径**一并考虑到。
> 否则你只对同步做了拦截，选择流式路径时，拦截就可能失效。

只有把这些路径一起拦住，才是生产级治理。

## 6. CheckPoint 为什么不是“顺手存个参数”

你可以把 `CheckPoint` 理解成断点存档。

没有 `CheckPoint`，就没有真正意义上的 `Resume`。

很多人会把 `CheckPointStore` 误解成一个“把审批参数存一下”的小缓存。

这个理解太窄了。

`CheckPoint` 在 Eino 里承担的是**运行现场持久化**。

在 ADK 的 Runner 视角里，至少有两件事必须同时成立：

- `RunnerConfig` 里配置了 `CheckPointStore`
- 执行时传入了 `adk.WithCheckPointID(checkPointID)`

只有这样，Runner 才知道：

- 中断发生时要把现场保存到哪里
- 之后恢复时该从哪个 key 把状态拉回来

官方给的典型代码配置如下：

```go
runner := adk.NewRunner(ctx, adk.RunnerConfig{
    Agent:           agent,
    EnableStreaming: true,
    CheckPointStore: adkstore.NewInMemoryStore(),
})

checkPointID := sessionID
events := runner.Run(ctx, history, adk.WithCheckPointID(checkPointID))
```

它其实做了一件很重要的事：

把“这次 Agent 运行”从一次临时调用，变成了一次**可恢复的会话执行**。

你会发现，`CheckPointStore` 不是普通缓存。为什么呢？

缓存的思路通常是：

- 丢了可以重算
- 命中了算赚到
- 不要求严格对应某次运行现场

但 `CheckPoint` 不是。

它保存的是这次运行“停在什么地方、手里拿着什么输入、下一步应该接到哪里”的现场信息。

按官方 `Agent Runner and Extension` 文档的说法，Runner 捕获到 `Interrupted Action` 后，如果同时配置了 `CheckPointStore` 和 `CheckPointID`，会把**原始输入、会话历史以及 InterruptInfo 等运行状态**持久化下来，后续再通过恢复接口继续执行。

所以你最好把 `CheckPointStore` 理解成：

**恢复协议的一部分，而不是缓存层的一个可选优化。**

到这里，其实审批流在 Agent 层的闭环已经完整了：Tool 为什么不能直接执行、中断是谁发起的、状态怎么保存、为什么恢复时还能接着往下跑。

接下来再往下看一层：这套能力在框架分层里到底属于哪里。

## 7. 从审批示例回到底层机制：Agent 只是复用了编排层的中断恢复能力

如果只看到前面这个审批示例，很多人会自然以为：这是 ADK 在 Agent 层单独做出来的一套审批机制。

但官方 `Interrupt & CheckPoint` 手册往下再看一层就会发现，Agent 里的 `Interrupt/Resume` 只是上层用法，底下复用的是更通用的编排中断恢复能力。

```go
func Resume(ctx context.Context, interruptIDs ...string) context.Context
func ResumeWithData(ctx context.Context, interruptID string, data any) context.Context
func BatchResumeWithData(ctx context.Context, resumeData map[string]any) context.Context
```

这些 API 解决的，就是更底层、更通用的问题：

- 是恢复所有中断点，还是只恢复某一个
- 恢复时要不要带自定义数据
- 并行中断时要不要批量恢复

而 Tool 审批流，本质上只是在这个框架上定义了自己的 `ApprovalInfo` 和 `ApprovalResult`。

你现在再回头看 `tool.GetResumeContext[T]` 就会更容易理解了：

它不是“顺便取一下用户输入”。
它是在一个更通用的恢复机制之上，帮当前 Tool 判断：

- 这次 `Resume` 是不是发给我的
- 发给我的数据是不是我能消费的那份数据

这个分层关系一旦看懂，后面再学 `Graph Tool`、`Workflow Agent`、嵌套图里的中断恢复，脑子会清楚很多。

## 8. 理解完主线后，再补几个生产里很容易踩到的边界

上面几节，已经足够支撑你理解“审批流为什么能成立”。

但如果你准备把这套能力接进真实业务，就不能只停在 Quick Start 那一层了。

下面这些点，不是主线机制本身，而是生产环境里非常容易被忽略的边界条件。

### 8.1 静态 Interrupt：不是所有暂停都要在节点内部自己抛

手册里专门给了静态断点能力。

也就是说，你可以在 `Compile` 图的时候，通过 `compose.WithInterruptBeforeNodes(...)` 和 `compose.WithInterruptAfterNodes(...)` 这类选项，声明某些节点执行前或执行后必须暂停。

这类能力适合什么场景？

- 某个节点前必须等人工确认
- 某个步骤后必须做外部审计
- 某些链路希望在固定位置留下可恢复断点

它和 Tool 内部动态 `Interrupt` 的区别在于：

- 静态 Interrupt 是编排层声明式控制
- 动态 Interrupt 是节点运行时自己决定要不要停

两者不是互斥关系。
一个偏治理，一个偏业务时机。

### 8.2 动态 Interrupt：`v0.7.0+` 之后，重点不是“重跑”，而是“带状态地中断”

官方手册明确写了：

- `v0.7.0` 之前，动态中断更像“节点返回特殊错误后 rerun”
- `v0.7.0` 及之后，新增了 `Interrupt`、`StatefulInterrupt`、`CompositeInterrupt`

这个变化很关键。

它意味着新语义下的中断不再只是“等会再跑一遍”。
而是：

- 可以保留局部状态
- 可以透出内部中断信号
- 可以支持并行中断与更精细的恢复目标

所以如果你现在新接这块能力，最好直接按 `v0.7.0+` 之后的语义去理解，不要再把它想成旧式 rerun。

### 8.3 流式传输和 CheckPoint 放在一起时，别忘了拼接规则

这一点特别容易被忽略。

普通 `Invoke` 场景里，保存 checkpoint 还算直接。
但流式场景下，运行中的输出是分块到来的。

这时如果你希望在流中断点也能恢复，就必须告诉框架：

**多个 chunk 最终怎么拼成一个可持久化的整体。**

手册里给了专门的注册方法 `RegisterStreamChunkConcatFunc[T any](fn func([]T) (T, error))`。

默认情况下，Eino 已经给 `string`、`*schema.Message` 这些内置常见类型准备了 concat 逻辑。
但如果你自己定义了流 chunk 结构，不补这层，checkpoint 这件事就不完整。

### 8.4 嵌套图里的 Interrupt/Resume，不只是“子图也能停一下”

很多系统的复杂度，最终都不在单图里，而在嵌套图里。

比如：

- 大图里挂一个子 Workflow
- 某个 Lambda 节点里再调一个独立 Graph
- Agent 里包着 FlowAgent，再包着 Tool 节点

这时中断恢复最难的地方已经不是“停不停”，而是：

- 中断到底发生在第几层
- 状态该保存到哪一层
- `Resume` 到底要对准哪个中断点

官方手册和 `v0.7.*` 版本说明都在强调这一点。

所以你如果打算把审批流往复杂 Agent 里扩，最好从一开始就把“断点地址”和“恢复目标”当正式设计问题来看，而不是等出事了再补。

### 8.5 外部主动 Interrupt：它不是冷门能力，优雅退出时很实用

这也是很工程化的一条能力。

有时候中断不是节点自己想停，而是系统外部要求它先停下来。

典型场景就是：

- 实例要优雅退出
- 运维要求先挂起长链路
- 某条执行流需要临时冻结等待外部资源

手册里提供了 `WithGraphInterrupt` 这套机制，让你在 Graph 外部主动触发 interrupt。

这类能力虽然不在 Quick Start 主线里，但它提醒了一个很重要的事实：

`Interrupt / Resume` 不只是“审批专用功能”。
它更像运行时的可暂停、可恢复协议。

审批只是它最容易理解、也最贴近业务价值的一种用法。

## 9. 真接进业务前，版本边界一定要单独确认

如果第 8 节讲的是能力边界，这一节讲的就是落地时最现实的一层：版本兼容。

只要 checkpoint 真要落盘、真要恢复，版本边界就不能轻描淡写。

### 9.1 `v0.3.26` 的问题，不是小 bug，而是 checkpoint 序列化 break

截至 `2026-03-30`，官方 `Interrupt & CheckPoint 使用手册` 顶部仍然明确提示：

`v0.3.26` 因为代码编写错误，导致 CheckPoint 的序列化内容产生 break。

官方建议也很直接：

- 新接入 checkpoint 的业务，使用 `v0.3.26` 之后的版本
- 更稳妥的做法是直接使用最新版本
- 老业务如果版本低于 `v0.3.26`，可以先走官方兼容分支，等老数据淘汰后再回主干

这个提醒的分量很重。

因为它告诉你，checkpoint 不是无状态功能。
一旦落盘，它就天然和版本兼容性绑在一起。

### 9.2 `v0.7.0` 的变化，是架构重构，不是小范围 API 调整

官方发布记录写得很明确：

- `v0.7.0` 是 Interrupt-Resume 的架构级重构版本
- 发布日期是 `2025-11-20`
- 新增 `GetInterruptState[T]`、`GetResumeContext` 这类类型安全恢复 API
- 支持隐式 `Resume All` 和显式 `Targeted Resume`
- Agent Tool 中断也改成了更标准的状态获取与组合中断处理方式

所以如果你现在写一篇面向新读者的文章，最稳妥的做法不是兼容讲两套，而是：

**主线全部按 `v0.7.0+` 之后的语义写，旧版只作为版本坑提醒。**

这样读者不容易把“历史做法”和“当前推荐做法”混到一起。

## 10. 最容易把审批流写坏的 6 个坑

写到这里，机制本身已经不难了。

真正容易漏掉的，往往是工程上那些看起来不起眼、但一漏就出事的细节。

### 10.1 只拦一个同步入口，忘了流式入口也会走 Tool

这是 Quick Start 已经用代码提醒过你的坑。

如果你系统里 `execute` 可能走流式，而你只实现了 `WrapInvokableToolCall`，那审批就是有缺口的。

### 10.2 有 `Interrupt`，却没有稳定的 `CheckPointID`

很多人本地 demo 能跑，就以为恢复链路已经成立了。

但如果 `CheckPointID` 是临时拼的、每轮都变，或者 `Resume` 时根本拿不到原来的 id，那恢复协议等于没闭环。

### 10.3 把 `CheckPointStore` 当普通缓存，随手改图结构和 CallOption

官方手册专门提醒过：

恢复只能复原输入和运行时节点数据，前提是**Graph 编排完全相同**，并且 `CallOption` 也要完整保持一致。

所以如果你一边保存 checkpoint，一边随手改编排图、改序列化方式、改恢复所需的 option，恢复失败一点都不奇怪。

### 10.4 Tool 审批写进每个 Tool 里，最后策略散落一地

这种写法最开始会显得“实现最快”。

但只要 Tool 数量上来，你就会发现审批策略根本没法统一治理。

中间件存在的意义，就是把这种横切规则从业务动作里剥出来。

### 10.5 自定义类型一上 checkpoint，却没想过序列化注册

手册已经写得很清楚：

- 简单类型或 Eino 内置类型一般不用额外处理
- 自定义结构体进入 checkpoint 时，最好提前注册稳定名字

这本质上是可恢复系统常见的序列化边界，不是 Eino 特有问题。

### 10.6 拒绝分支只想着“返回一句话”，没把治理结果当正式输出

审批被拒绝，不等于这轮执行“什么都没发生”。

至少从系统层面看，这仍然是一次完整决策：

- 谁申请了什么动作
- 谁拒绝了
- 为什么拒绝
- 这轮 Agent 最终该怎么收口

如果你后面要接审计、风控、工单系统，这些信息都不是可有可无的。

## 11. 总结

很多人一开始学 `Interrupt / Resume`，会把它看成 Agent 的一个附属能力。

但真走到生产环境，你会发现它的重要性一点都不比 `Tool Calling` 低。

因为 Agent 越有行动能力，你就越不能把所有控制权都交给它。

`Interrupt` 解决的是“该停时能不能真的停住”。
`Resume` 解决的是“确认以后能不能从原地接着跑”。
`CheckPoint` 解决的是“停住以后，现场能不能被可靠地保存和恢复”。

而 `approvalMiddleware` 则把这套东西从单个 Tool 的临时写法，提升成了整条 Agent 链路的治理策略。

所以如果你现在问我：

> 第七章最值得带走的，到底是哪句话？

我的答案会是：

**Agent 一旦能调真实 Tool，审批就不再是前端交互，而是运行时协议；而 Interrupt / Resume + CheckPoint，就是这套协议在 Eino 里的落地方式。**

## 参考资料

1. [第七章：Interrupt/Resume（中断与恢复）](https://www.cloudwego.io/zh/docs/eino/quick_start/chapter_07_interrupt_resume/)
2. [Interrupt & CheckPoint 使用手册](https://www.cloudwego.io/zh/docs/eino/core_modules/chain_and_graph_orchestration/checkpoint_interrupt/)
3. [Eino ADK: Agent Runner and Extension](https://www.cloudwego.io/zh/docs/eino/core_modules/eino_adk/agent_extension/)
4. [v0.7.*-interrupt resume refactor](https://www.cloudwego.io/zh/docs/eino/release_notes_and_migration/eino_v0.7._-interrupt_resume_refactor/)
5. [eino-examples/quickstart/chatwitheino](https://github.com/cloudwego/eino-examples/tree/main/quickstart/chatwitheino)

---

## 发布说明

- GitHub 主文：[AI 大模型落地系列｜Eino 编排篇：从自动执行到人工接管，如何避免Agent一把梭](./03-从自动执行到人工接管，如何避免Agent一把梭.md)
- CSDN 跳转：[AI 大模型落地系列｜Eino 编排篇：从自动执行到人工接管，如何避免Agent一把梭](https://zhumo.blog.csdn.net/article/details/159642323)
- 官方文档：[Interrupt & CheckPoint 使用手册](https://www.cloudwego.io/zh/docs/eino/core_modules/chain_and_graph_orchestration/checkpoint_interrupt/)
- 最新版以 GitHub 仓库为准。

