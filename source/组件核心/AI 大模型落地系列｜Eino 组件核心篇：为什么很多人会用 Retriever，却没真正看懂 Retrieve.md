声明：本文数据源于官方文档与官方实现，重点参考 [Retriever 使用说明](https://www.cloudwego.io/zh/docs/eino/core_modules/components/retriever_guide/)、[components/retriever/interface.go](https://github.com/cloudwego/eino/blob/main/components/retriever/interface.go)、[components/retriever/option.go](https://github.com/cloudwego/eino/blob/main/components/retriever/option.go)，以及 `eino-ext/components/retriever/volc_vikingdb/examples/builtin_embedding` 示例。
@[TOC](为什么很多人会用 Retriever，却没真正看懂 Retrieve)

很多人第一次看到 `Retriever`，第一反应都很直接：

不就是调一下向量库或者搜索引擎的 `search`，把最像的几条文档捞出来吗？

代码看起来也确实像这么回事。

可只要你继续往工程里走，问题马上就来了：

- query 到底在哪里做 embedding？
- 多知识库、多子索引怎么切？
- `TopK` 和相似度阈值该放配置里，还是放运行时？
- 过滤条件到底写在 SDK 调用里，还是写在组件 option 里？
- 一次检索到底怎么进 `Chain`、`Graph`、`Callback` 这条正式运行时链路？

如果这些事都散在业务代码里，检索当然也能跑，但通常跑不久就会乱。

所以这篇文章想讲清楚的，不是“怎么搜一次”，而是：

> `Retriever` 为什么会被 Eino 单独抽成一个组件？

上一篇《为什么很多人会用 Indexer，却没真正看懂 Store》讲的是“文档怎么写进去”，这一篇刚好接着讲“query 怎么把文档查出来”。

先把结论摆出来：

> `Retriever` 是 Eino 在读侧给出的统一检索协议，不是某家向量库 SDK 的语法糖。


## 1. Retriever 真正解决的，不只是“搜一下”

先别急着看接口。

如果一上来就盯着 `Retrieve(ctx, query, opts...)` 这一个方法，很容易把它看成“检索调用的统一壳子”。
这个理解不算全错，但还是太浅。

`Retriever` 真正收口的，其实是读侧这几件事：

**第一，把 query 变成标准检索入口。**

上层只需要给出查询字符串，至于后面是关键词检索、向量检索、混合检索，还是带过滤条件的召回，都由组件自己去接具体实现。

**第二，把结果统一成 `[]*schema.Document`。**

不管底层是 VikingDB、Milvus、ES，还是 OpenSearch，最后交给上层的都不是某家 SDK 的 hit 结构，而是标准文档协议。

**第三，把检索正式纳入运行时链路。**

它不是一个你在业务代码里顺手调一下的帮助函数，而是能进 `Chain`、能进 `Graph`、能挂 `Callback` 的正式组件。

你放到 RAG 里看，这层价值会更清楚。

一条典型链路里：

- `Embedding` 负责把文本变成向量
- `Indexer` 负责把文档写成可检索对象
- `Retriever` 负责把 query 变成召回动作
- `ChatModel` 负责基于召回结果生成答案

至于 `Rerank`，它通常在 `Retriever` 之后，对候选结果再做一轮重排；这不是 `Retriever` 本体要解决的事。

所以别把它理解成“搜索函数封装”。

更准确一点说：

> `Retriever` 解决的是“查询如何以统一协议进入检索系统，并把结果以统一协议返回出来”。  


## 2. 接口只有一个方法，但 Retrieve 这个动作一点都不简单

官方给出的核心接口其实非常短：

```go
type Retriever interface {
    Retrieve(ctx context.Context, query string, opts ...Option) ([]*schema.Document, error)
}
```

如果只看长度，这接口甚至比 `Indexer` 还简单。

可真正要看的，不是它有几个方法，而是它在收什么边界。

先看 `retriever.Retriever`。

这说明 Eino 在组件层明确区分了“写入协议”和“读取协议”。
你前面已经有了 `Indexer` 去负责 `Store`，这里再单独给 `Retrieve` 一层抽象，意思已经很明确了：

> 写进去怎么做，和查出来怎么做，是两条边界。

再看 `Retrieve(ctx, query, opts...) ([]*schema.Document, error)`。

这个签名里最重要的有 4 个点。

**1. `ctx` 不只是取消信号。**

在 Eino 里，它同时承担请求级信息和 callback manager 的传递。
也就是说，检索这一步从一开始就被当成正式运行时行为，而不是藏在工具函数里的黑盒调用。

**2. 输入是 `query string`，不是某家后端的专属请求结构。**

这一步把上层调用姿势压得很统一。
至于 query 后面要不要向量化、怎么向量化、要不要混合检索，是组件内部的事。

**3. 返回的是 `[]*schema.Document`，不是原始 hit。**

这点很关键。
如果返回的是后端自己的结果结构，那这层抽象就基本失效了。
现在统一返回 `Document`，说明它抽象的不是“某种数据库搜索请求”，而是“统一读侧输出协议”。

**4. `opts ...Option` 把运行时可变能力单独挂了出来。**

这意味着检索行为不是完全写死在初始化配置里的。
索引、子索引、`TopK`、阈值、embedding、过滤 DSL，都可以在调用时覆写。

再看 `schema.Document`：

```go
type Document struct {
    ID       string
    Content  string
    MetaData map[string]any
}
```

很多人看到这里，会把注意力放在 `Content` 上。
但到了检索场景，真正不能轻视的，反而常常是 `MetaData`。

因为检索结果除了正文，往往还会带上这些信息：

- 分数
- 来源
- 业务标签
- 命中的索引或分区
- 后端返回的其他上下文字段

这些信息不一定都在 `Content` 里，却很可能会被后续节点继续用到。

所以 `Retrieve` 虽然只有一个动作名，但里面可能同时发生：

- query 预处理
- 向量生成
- 后端检索
- 结果解析
- metadata 注入
- callback 生命周期触发

这已经明显不是一句“搜一下”能说完的事了。


## 3. 公共 Option 收口的，不只是几个小参数

官方给 `Retriever` 的公共 option 长这样：

```go
type Options struct {
    Index          *string
    SubIndex       *string
    TopK           *int
    ScoreThreshold *float64
    Embedding      embedding.Embedder
    DSLInfo        map[string]any
}
```

字段不算多，但每一个都对应着读侧真正会变的行为。

### 3.1 `Index`

`Index` 是检索器使用的索引。

别把它只理解成“某个数据库里的索引名”。
在不同实现里，它的含义可能不一样，但统一点在于：

> 它决定你这次检索到底要落到哪一个可检索空间里。

这在多知识库、多业务库、多环境隔离里很常见。

### 3.2 `SubIndex`

`SubIndex` 是子索引。

它更像逻辑上的进一步分流。
比如同一套物理存储下，你可能还会按租户、业务线、数据域、时间分区去做更细颗粒度的检索路由。

这就是为什么 Eino 不把它粗暴合并进 `Index`。

它们虽然都在描述“查哪里”，但层级不一样。

### 3.3 `TopK`

`TopK` 是返回文档数量上限。

这看起来像一个很普通的参数，但它其实会直接影响：

- 召回范围
- 下游模型上下文长度
- 延迟
- 成本

所以它不该永远被写死在初始化配置里。
很多真实业务里，FAQ 检索、知识库问答、长文档辅助分析，它们需要的 `TopK` 根本不是一个数。

### 3.4 `ScoreThreshold`

`ScoreThreshold` 是分数阈值。

这里有一个特别容易被用浅的点：

> 它是过滤条件，不是排序开关。

也就是说，它的意义不是“把低分文档往后排”，而是“低于阈值的文档直接不要”。

所以如果你的召回结果“明明命中了，但又没返回”，除了看 `TopK`，还得看这里是不是把结果过滤掉了。

### 3.5 `Embedding`

`Embedding` 是给 query 做向量化的组件。

这个字段很关键，因为它直接说明：

`Retriever` 虽然吃的是自然语言 query，但它可以在内部把 query 变成向量，再去做相似度检索。

同时也正因为有这个字段，官方源码里还特别强调了一层约束：

> 检索时使用的 embedder，应该和索引写入时使用的模型保持一致。

否则很容易出现一种很典型的线上问题：

文档都在，索引也建好了，可召回效果就是不对。

问题往往不是数据库坏了，而是写入时和查询时压根不在一个向量空间里。

### 3.6 `DSLInfo`

`DSLInfo` 是检索 DSL 信息。

官方文档里提到它在 Viking 类型检索器里会用到，但你更应该记住的是它的设计信号：

> 公共 option 收口的是共性能力，但不会强行抹平所有后端差异。

像过滤表达式、查询 DSL 这种东西，不同后端差异很大。
如果硬要塞成一个统一大接口，最后只会把抽象做笨。

所以 Eino 的做法很克制：

- 共性参数统一收口
- 后端特有能力允许保留

这比“什么都统一”更像工程设计。

### 3.7 不止公共 option，具体实现还能继续扩展

官方还提供了实现级 option 的包装方式。

这意味着你在自定义 `Retriever` 时，既要支持 `GetCommonOptions(...)`，也可以保留自己那套实现专属参数，而不用为了兼容框架把所有细节都塞进公共层。

说白了就是：

> 公共 option 负责定义“所有 Retriever 都该听得懂的话”，实现级 option 负责保留“这一家后端自己的方言”。  


## 4. 它在 RAG 读链路里，到底站在哪

很多人理解 RAG，脑子里只有一句话：

“把文档切块，做 embedding，丢向量库，查询的时候再搜出来。”

这句话当然不算错，但一旦落到 Eino 组件边界上，还是得拆得更清楚一点。

先看最短版本：

```text
Loader / Parser -> Indexer -> Retriever -> ChatModel
```

如果把过程展开一点，大致是这样：

```text
原始资料
  -> Loader / Parser
  -> []*schema.Document
  -> 切块 / 清洗
  -> Indexer.Store
  -> 可检索后端
  -> Retriever.Retrieve(query)
  -> []*schema.Document
  -> ChatModel
```

这里真正重要的，不是流程图本身，而是边界。

写入侧你关心的是：

- 文档如何标准化
- 向量什么时候生成
- 元数据怎么落库
- 写进哪个索引或分区

读取侧你关心的是：

- query 怎么解释
- 要查哪个索引或子索引
- 召回多少条
- 分数阈值怎么设
- 过滤条件怎么下发
- 返回什么 metadata 给下游

这也就是为什么上一篇讲 `Indexer` 时，我一直在强调“写入侧协议统一”。

这一篇换到 `Retriever`，重点就必须换成另一句：

> `Retriever` 解决的是“查询如何进入可检索系统”，不是“文档如何写进去”。  

如果这两层边界不拆开，最常见的结果就是：

- 写入逻辑和检索逻辑缠在一起
- 业务代码里到处散落后端 SDK 细节
- 一旦你要换存储后端、换索引策略、加 callback，改动面会非常大

所以 `Retriever` 站在 RAG 链路里的位置，并不是“后面随便补一层的 search helper”。
它就是读侧入口。


## 5. 用 VikingDB 看一遍最小检索闭环

如果只讲抽象，还是容易飘。
所以最好的办法，还是拿一个官方示例最完整的实现过一遍。

这里用 `Volc VikingDB Retriever`。

```go
package main

import (
    "context"
    "log"

    "github.com/cloudwego/eino-ext/components/retriever/volc_vikingdb"
)

func ptr[T any](v T) *T { return &v }

func main() {
    ctx := context.Background()

    cfg := &volc_vikingdb.RetrieverConfig{
        Host:              "api-vikingdb.volces.com",
        Region:            "cn-beijing",
        AK:                "your-ak",
        SK:                "your-sk",
        Scheme:            "https",
        ConnectionTimeout: 0,
        Collection:        "eino_test",
        Index:             "test_index_1",
        EmbeddingConfig: volc_vikingdb.EmbeddingConfig{
            UseBuiltin:  true,
            ModelName:   "bge-m3",
            UseSparse:   true,
            DenseWeight: 0.4,
        },
        Partition:      "",
        TopK:           ptr(10),
        ScoreThreshold: ptr(0.1),
        FilterDSL:      nil,
    }

    r, err := volc_vikingdb.NewRetriever(ctx, cfg)
    if err != nil {
        log.Fatal(err)
    }

    docs, err := r.Retrieve(ctx, "怎么申请退款")
    if err != nil {
        log.Fatal(err)
    }

    for _, doc := range docs {
        log.Printf("id=%s metadata=%v content=%s", doc.ID, doc.MetaData, doc.Content)
    }
}
```

这段代码看起来不复杂，但里面几个字段都很值得注意。

### `Collection`

`Collection` 对应的是这批文档所在的数据集。
你可以把它理解成“更大一级的检索容器”。

### `Index`

`Index` 对应检索时真正使用的索引。
这一步很像上一篇里 `Indexer` 的镜像面：

- `Indexer` 决定内容怎么写进去
- `Retriever` 决定查的时候落到哪个索引上

### `Partition`

`Partition` 对应索引中的子索引划分字段。
如果你的知识库不是一锅端，而是按租户、业务、区域、版本再做细分，那这层就很有用了。

### `FilterDSL`

`FilterDSL` 对应标量过滤字段。

这点很工程。
因为很多场景你不只是“找最像的内容”，还要先满足一层业务过滤，比如：

- 只看某个知识库
- 只看某个状态的数据
- 只看某个时间范围

如果没有 DSL 这层，后端明明支持过滤，你在组件层就很难把这个能力干净地挂出来。

### `EmbeddingConfig`

这块是 VikingDB 示例最有代表性的地方。

它说明 query 不一定非得由你先手工转成向量再传进去。
像这里 `UseBuiltin: true`，就是让检索器直接使用 VikingDB 的内置 embedding 配置去完成向量化。

这也是为什么我前面一直在说：

> `Retriever` 不是“搜结果”的那一下，而是“query 进入检索系统的整段过程”。  

因为 query 在真正发起搜索前，可能已经先经历了向量化和过滤条件拼装。

### `TopK` 和 `ScoreThreshold`

这两个参数一个控制“最多拿多少”，一个控制“低于多少不要”。
别把它们混成一回事。

如果你后面想在单次调用时临时覆盖，也完全可以通过公共 option 去改，而不用把默认值写死在初始化配置里。

再补一句：

Milvus、Elasticsearch、OpenSearch 这些实现，初始化参数和搜索模式都不一样，但最后都会收口到同一条调用协议上：

```go
docs, err := retriever.Retrieve(ctx, query, opts...)
```

这就说明 `Retriever` 抽象的不是某一家后端，而是读侧检索动作本身。


## 6. 为什么它能直接进 Chain、Graph 和 Callback

如果 `Retriever` 只是一个普通 SDK 包装层，它其实没必要出现在编排系统里。

但官方文档明确给出了这两种挂法：

```go
chain := compose.NewChain[string, []*schema.Document]()
chain.AppendRetriever(retriever)

graph := compose.NewGraph[string, []*schema.Document]()
graph.AddRetrieverNode("retriever_node", retriever)
```

这已经说明一件很重要的事：

> `Retriever` 是正式运行时节点，不是藏在代码角落里的工具函数。  

再看 callback。

官方示例里，`Retriever` 这层可以直接挂 `retriever.CallbackInput` 和 `retriever.CallbackOutput`：

```go
handler := &callbacksHelper.RetrieverCallbackHandler{
    OnStart: func(ctx context.Context, info *callbacks.RunInfo, input *retriever.CallbackInput) context.Context {
        log.Printf("query=%s topK=%d", input.Query, input.TopK)
        return ctx
    },
    OnEnd: func(ctx context.Context, info *callbacks.RunInfo, output *retriever.CallbackOutput) context.Context {
        log.Printf("docs=%d", len(output.Docs))
        return ctx
    },
}

helper := callbacksHelper.NewHandlerHelper().
    Retriever(handler).
    Handler()

chain := compose.NewChain[string, []*schema.Document]()
chain.AppendRetriever(retriever)

runner, _ := chain.Compile(ctx)
docs, _ := runner.Invoke(ctx, "怎么申请退款", compose.WithCallbacks(helper))

_ = docs
```

这段代码最值得盯住的，不是日志打印，而是它暴露出来的事实：

- 你能在 `OnStart` 里看到 query 和运行时参数
- 你能在 `OnEnd` 里拿到检索结果
- 检索过程本身可以进入统一追踪和观测链路

这对排障非常重要。

因为 RAG 项目里最难受的问题之一就是：

“答案不对，到底是模型幻觉，还是前面的召回就错了？”

如果 `Retriever` 没进入 callback 链路，这个问题会很难查。
你最后只能在业务层加一堆散乱日志，既不整洁，也不稳定。


## 7. 自己实现一个 Retriever 时，哪些细节不能省

如果你要自己接一个新的检索后端，官方文档其实已经把骨架给得很清楚了。

真正要守住的顺序，大致就是下面这条：

- `retriever.GetCommonOptions`
- `callbacks.ManagerFromContext`
- `OnStart`
- `doRetrieve`
- `OnError`
- `OnEnd`

可以先看一个收过边界的骨架：

```go
type MyRetriever struct {
    index    string
    topK     int
    embedder embedding.Embedder
}

func (r *MyRetriever) Retrieve(
    ctx context.Context,
    query string,
    opts ...retriever.Option,
) ([]*schema.Document, error) {
    commonOpts := retriever.GetCommonOptions(&retriever.Options{
        Index:     &r.index,
        TopK:      &r.topK,
        Embedding: r.embedder,
    }, opts...)

    cm := callbacks.ManagerFromContext(ctx)
    runInfo := &callbacks.RunInfo{}

    ctx = cm.OnStart(ctx, runInfo, &retriever.CallbackInput{
        Query:          query,
        TopK:           *commonOpts.TopK,
        ScoreThreshold: commonOpts.ScoreThreshold,
        Extra: map[string]any{
            "index":     commonOpts.Index,
            "sub_index": commonOpts.SubIndex,
            "dsl":       commonOpts.DSLInfo,
        },
    })

    docs, err := r.doRetrieve(ctx, query, commonOpts)
    if err != nil {
        ctx = cm.OnError(ctx, runInfo, err)
        return nil, err
    }

    ctx = cm.OnEnd(ctx, runInfo, &retriever.CallbackOutput{
        Docs: docs,
    })
    return docs, nil
}

func (r *MyRetriever) doRetrieve(
    ctx context.Context,
    query string,
    opts *retriever.Options,
) ([]*schema.Document, error) {
    var queryVector []float64

    if opts.Embedding != nil {
        vectors, err := opts.Embedding.EmbedStrings(ctx, []string{query})
        if err != nil {
            return nil, err
        }
        queryVector = vectors[0]
    }

    _ = queryVector

    docs := []*schema.Document{
        {
            ID:      "doc_1",
            Content: "退款申请一般需要先提交订单号和支付凭证。",
            MetaData: map[string]any{
                "score":   0.92,
                "source":  "faq/refund.md",
                "backend": "my_store",
            },
        },
    }

    return docs, nil
}
```

这段骨架里，有两点特别不能省。

**第一，`Embedding` 只在需要时调用。**

不是所有检索后端都要求你在组件里自己生成 query 向量。
有的后端支持内置 embedding，有的实现则会直接走关键词或混合检索。

所以这里正确的姿势不是“无脑先 embed 一下”，而是：

> 调用前先看 `opts.Embedding`，有就用，没有就按实现自己的检索模式走。  

**第二，要把后续节点可能会用到的 metadata 补齐。**

很多人自己实现 `Retriever` 时，只想着把正文查出来。
这当然能跑，但后面一接真实业务就会发现不够用。

至少下面这些信息，通常值得带出去：

- 召回分数
- 来源标识
- 后端文档 ID
- 命中的索引或分区
- 你实现里特有的上下文信息

因为后续节点不一定只看 `Content`。
它可能要做来源展示、结果解释、问题排查，甚至还要继续做 rerank 或引用标注。

如果 metadata 在这里丢了，后面再补就会很别扭。


## 8. 5 个最容易把 Retriever 用浅的坑

### 8.1 把 `Retriever` 当成 SDK 薄封装

这是最常见的误区。

一旦你这么理解，代码里就会到处散落后端专属请求结构、过滤逻辑和日志逻辑。
最后不是 Eino 在帮你统一边界，而是你自己把边界重新打碎了。

### 8.2 不看 `MetaData`，后面就追不动来源和分数

只拿正文，不看 metadata，短 demo 没什么感觉。

可一到线上，你很快就会遇到这些问题：

- 这段答案是从哪篇文档来的？
- 这条结果分数到底高不高？
- 它命中了哪个索引或分区？

这些都离不开 metadata。

### 8.3 `TopK` 和阈值写死

很多项目最开始为了省事，直接把 `TopK=5`、`threshold=0.3` 固定死。

问题是不同场景需要的召回范围并不一样。
而且阈值本身还是过滤条件，不是排序条件。
一旦写死，后面要调优效果就会非常别扭。

### 8.4 查询 embedding 和底库向量配置不匹配

这是检索效果异常里非常高频的一类问题。

写入时用的是一种模型，查询时换了另一种模型，或者维度根本对不上，最后最直观的表现就是：

“库里明明有内容，可就是召不准。”

别一上来就怀疑数据脏了，先看 query embedding 和底库配置是不是同一套。

### 8.5 不接 callback，召回问题很难排

RAG 项目里，很多问题不是“功能坏了”，而是“效果不稳定”。

这类问题如果没有 callback，你很难快速判断：

- 这次 query 进来时到底用了什么参数
- 检索结果到底返回了几条
- 是前面没召回到，还是后面模型没用好

所以 callback 不是锦上添花，它在检索层经常就是排障入口。


## 9. 总结

如果用一句话总结这篇 `Retriever 使用说明`，我会这样说：

> `Retrieve` 的本质不是“调一次 search”，而是“让 query 以统一协议进入读侧检索系统”。  

再压缩成几句，就是：

- `Retriever` 解决的是读侧协议统一，不是某家后端 SDK 的简单包一层
- `Retrieve` 里可能同时发生向量化、过滤、召回、结果解析和 callback 触发
- `Indexer` 管“怎么写进去”，`Retriever` 管“怎么查出来”，两层边界不能混

你把这层看懂，后面无论是继续往 `Rerank`、完整 RAG、还是更复杂的编排链路走，脑子都会顺很多。


## 参考资料

- CloudWeGo Eino [Retriever 使用说明](https://www.cloudwego.io/zh/docs/eino/core_modules/components/retriever_guide/)
- CloudWeGo Eino [components/retriever/interface.go](https://github.com/cloudwego/eino/blob/main/components/retriever/interface.go)
- CloudWeGo Eino [components/retriever/option.go](https://github.com/cloudwego/eino/blob/main/components/retriever/option.go)
- CloudWeGo Eino Ext `components/retriever/volc_vikingdb/examples/builtin_embedding`
