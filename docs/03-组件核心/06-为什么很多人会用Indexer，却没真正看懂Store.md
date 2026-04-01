# AI 大模型落地系列｜Eino 组件核心篇：为什么很多人会用 Indexer，却没真正看懂 Store

> GitHub 主文：[当前文章](./06-为什么很多人会用Indexer，却没真正看懂Store.md)
> CSDN 跳转：[AI 大模型落地系列｜Eino 组件核心篇：为什么很多人会用 Indexer，却没真正看懂 Store](https://zhumo.blog.csdn.net/article/details/159537753)
> 官方文档：[Indexer 使用说明](https://www.cloudwego.io/zh/docs/eino/core_modules/components/indexer_guide/)
>
> 最新版以 GitHub 仓库为准，CSDN 作为分发入口，官方文档作为权威参考。

**一句话摘要**：从 RAG 入库链路看清 Indexer 和 Store 的职责，避免把它当成向量库插入函数。
**适合谁看**：准备做知识库写链路、需要把组件分层讲清楚的 Go 开发者。
**前置知识**：Document Loader 与 Parser、Embedding 基础
**对应 Demo**：[官方 Indexer 示例（本仓后续补充 Milvus demo）](https://www.cloudwego.io/zh/docs/eino/core_modules/components/indexer_guide/)

**面试可讲点**
- 能解释 Indexer 不只是写向量库，而是统一文档写入协议的组件层。
- 能说明 Store、Embedding、SubIndexes 这些概念为什么被拆到不同位置。

---
很多人做知识库时，前面那几步通常都知道怎么搞：

- 文档切块
- 文本做 embedding
- 向量库建 collection

看起来好像已经差不多了。

可真到落库这一步，问题马上就来了：

- 文档正文怎么写进去？
- 向量是谁来生成、什么时候生成？
- 元数据和来源信息放哪里？
- 多个知识库、多个业务空间，怎么分区写？
- 写进去以后，怎么保证后面真的能被检索到？

这时候你就会发现，`Indexer` 这层不是可有可无。

说白了：

> `Indexer` 本质上就是“把文档写成以后能被检索的样子”的组件。

它不是只管塞一段文本进去。
它更像是把文档、向量、元数据、子索引这些东西，一起整理好，再送进可检索后端。

这也是为什么它在知识库入库、语义搜索底库构建、多知识库分区写入这些场景里特别关键。

但很多人第一次看到 `Indexer`，还是会下意识地把它理解成：

“哦，这不就是调一下 Milvus、VikingDB 或 ES 的写入接口吗？”

问题也恰恰出在这儿。

因为如果事情真这么简单，那我们明明已经有：

- `Embedding`
- 向量库 SDK
- 搜索引擎写入接口

为什么 Eino 还要单独设计一个 `Indexer`？

这篇文章，想讲清楚的就是这件事。

如果你前面刚看过我上一篇 [Document Loader](https://blog.csdn.net/2302_80067378/article/details/159514239?spm=1001.2014.3001.5501) 的文章，那篇讲的是文档怎么进入 `[]*schema.Document` 这套统一协议。
这一篇刚好接上下一站：

> 当文档已经变成标准协议后，它到底怎么被写进“可检索系统”？

## 1. 它的用处，它又被误会在哪？
先别急着看接口。

如果一上来，我就讲 API，这样虽可以记住函数名，但不利于大家理解它能解决什么问题。
`Indexer` 更适合先从“它是拿来干嘛的”讲起。

### 它能干嘛

放到最常见的知识库链路里，`Indexer` 主要做的是这些事：

**1. 把文档写进可检索后端**

不是只写正文。
而是把 `schema.Document` 这套统一协议，整理成后端真正能存、以后也真正能查的样子。

**2. 把向量和元数据一起落进去**

很多场景不是只有文本内容。
你还得把向量、来源、chunk 编号、业务标签这些信息一起写进去，不然后面检索和追踪都会很难受。

**3. 处理逻辑子索引或知识库分区**

你可能不是只有一个知识库。
多租户、多业务空间、多资料域，这些场景都要求写入时就把路由和隔离想清楚。

### 它最适合哪些场景

最常见的就是这几类：

- 知识库入库
- 语义搜索底库构建
- 多知识库 / 多租户分区写入

### 它不能直接干嘛

这块也必须提前讲清楚。否则你会误会它的功能。

`Indexer` 很重要，但它不是万能层。

- 它不负责切块。文档怎么拆 chunk，不是它的职责。
- 它不负责生成最终回答。那是 LLM 干的事。
- 它不负责读侧召回。怎么把内容查出来，是 `Retriever` 的边界。

所以更准确地说：

`Embedding` 是把文本变成向量。
`Indexer` 是把内容写成可检索对象。
`Retriever` 是再把这些东西查出来。

这三个环节如果混在一起，后面链路虽然也能跑，就会造成边界模糊，耦合严重的场景。

## 2. `Indexer` 不是“向量库 insert 封装”

这里先把结论摆出来：

`Indexer` 不是一个“帮你调 Milvus / VikingDB / ES SDK”的小工具。
它是 Eino 在写入侧给出的统一组件协议。

这层协议真正收口的是三件事：

- 文档输入统一成 `[]*schema.Document`
- 写入行为统一成 `Store(ctx, docs, opts...)`
- 写入结果统一成 `[]string` 形式的 `ids`

这就意味着，它关心的是“写入侧边界”，不是某个后端产品自己的调用细节。

所以你看 `Indexer` 时，最好先把几个常见误会排掉：

- 它不负责切块。文档怎么切成 chunk，是 `Loader / Parser` 后面的预处理问题，不是 `Indexer` 本体。
- 它不负责读侧召回。相似度搜索、过滤、排序、返回 topK，那是 `Retriever` 的边界。
- 它不等于 `Embedding`。向量可以在写入时生成，但生成向量这件事本身，仍然是另一层能力。

也正因为这三件事被拆开了，Eino 才能同时挂住 Milvus、VikingDB、ES、OpenSearch 这些看起来很不像一家人的后端实现。

如果它只是“向量库 insert 封装”，那 ES / OpenSearch 这两类实现就会显得很别扭。
可官方偏偏把它们也归在 `Indexer` 下面，这恰恰说明：

> `Indexer` 抽象的不是“向量库”，而是“可检索后端的写入入口”。（一套可插拔的接口）

## 3. `Indexer` 在 RAG 入库链路里，到底站在哪

很多人理解 RAG 时，脑子里只有一句话：

“文档切块，做 embedding，丢进向量库。”

这当然没错，但如果你在 Eino 里写组件，你最好把链路拆得再清楚一点：

```text
Source
  -> Loader / Parser
  -> []*schema.Document
  -> （切块 / 清洗）
  -> Embedding / Field Mapping
  -> Indexer.Store
  -> Retriever
  -> ChatModel
```

这里最关键的是中间那段。

`Loader / Parser` 负责把不同来源的内容，收口成标准 `Document`。
`Indexer` 负责把这些 `Document` 写进后端，让它以后能被查出来。
而 `Retriever` 则负责真正把它们读出来。

也就是说：

- `Loader / Parser` 管“东西从哪来、怎么解释”
- `Indexer` 管“怎么写进去”
- `Retriever` 管“怎么查出来”

很多人之所以会把 `Indexer` 理解歪，就是因为把“写进去”和“以后怎么查”混成了一件事。

可在工程里，这两件事差得很远。

写入时你关心的是：

- 文档 ID 怎么处理
- 向量何时生成
- 元数据写到哪些字段
- 逻辑分区、子索引、批量写入怎么做

召回时你关心的是：

- query 该怎么向量化
- topK 怎么取
- filter 怎么写
- score 怎么解释

如果这两层边界不拆开，最后很容易变成“能跑，但组件职责已经糊了”。

## 4. 接口只有一个方法，但 `Store` 这个词一点都不简单

官方核心接口其实非常短：

```go
type Indexer interface {
    Store(ctx context.Context, docs []*schema.Document, opts ...Option) (ids []string, err error)
}
```

很多人第一次看到这个接口，会觉得信息量不大。
可它真正想收口的边界，恰恰都藏在这几个参数里。

先看 `ctx`。

在 Eino 里，`ctx` 从来都不只是取消信号。
官方文档已经明确写了，它还承担 `Callback Manager` 的传递。
这意味着 `Store` 不是一个藏在角落里的工具函数，而是一段可以被编排、被观察、被追踪的正式运行时行为。

再看 `docs []*schema.Document`。

这很关键。

`Indexer` 吃进去的不是某家向量库自己的 row，也不是某个搜索引擎专属的字段结构，而是统一文档协议。
这件事的价值在上一篇 `Document Loader` 里其实已经埋下了：

> 文档一旦被标准化成 `schema.Document`，后面的写入端就终于可以和“来源差异”解耦。

最后看返回值 `ids []string`。

这块很多人会想当然地把它理解成“就是把 `doc.ID` 原样回给你”。
但实际上，`ids` 更准确的意思是：

> 后端最终确认写入成功的文档标识。

它可能是：

- 直接沿用你传进来的 `Document.ID`
- 后端生成的新 ID
- 一次批量 upsert 之后真正生效的主键集合

所以 `Store` 这个词，千万别按数据库里那种“我插一行，你回一个自增主键”的直觉去理解。

在 Eino 语境里，一次 `Store` 里可能同时发生：

- 文档字段映射
- 向量生成
- 批量写入
- 子索引分流
- 回调触发
- 错误上抛

这已经明显不是一句“insert 一下”能说清的事了。

## 5. 公共 Option 真正控制的，不是“几个小参数”

官方给 `Indexer` 的公共 option 很克制：

```go
type Options struct {
    SubIndexes []string      // 子索引/子分区：这批文档要写到哪些逻辑分组里
    Embedding  embedding.Embedder // 向量模型：写入前用它把文本转成向量
}

func WithSubIndexes(subIndexes []string) Option // 设置子索引/分区
func WithEmbedding(emb embedding.Embedder) Option // 设置本次写入使用的向量生成器
```

字段不多，但信息量不小。

### 4.1 `SubIndexes` 更像逻辑分区，不只是一个字符串数组

很多人第一次看 `SubIndexes`，会把它当成“顺手多传几个名字”。
可如果你把它放回知识库场景里，就会发现它更像逻辑分区入口。

比如同一套物理后端里，你可能会按下面这些维度做隔离：

- 不同知识库
- 不同租户
- 不同业务空间
- 不同文档域

这时 `SubIndexes` 的作用，就不是“多一个参数”这么简单了。
它更接近：

> 在同一个 `Indexer` 抽象之下，把文档路由到不同的逻辑子索引或子分区。

所以我更愿意把它理解成写入侧的 namespace / partition 入口，而不是一个普通切片字段。

### 4.2 `Embedding` 是写入时临时挂接的能力，不是 `Indexer` 本体

`WithEmbedding` 更值得多看两眼。

它说明什么？

说明 Eino 允许你在 `Store` 这一跳里，临时指定“这批文档怎么向量化”。
也就是说，向量生成可以是：

- `Indexer` 初始化时配置好的默认能力
- 本次调用临时覆盖进去的 embedder

这就把“写入协议”和“向量模型选择”拆开了。

而且还有一个容易被忽略的点。

VikingDB 示例里，官方给的是后端内建 embedding 配置：

```go
EmbeddingConfig: volc_vikingdb.EmbeddingConfig{
    UseBuiltin: true,
    ModelName:  "bge-m3",
    UseSparse:  true,
},
```

这恰恰说明：

`Indexer` 可以挂接 embedding，但它本身不等于 embedding。
有些实现会走外部 embedder，有些实现会直接利用后端内建能力。

> 如果想要了解更多，可以打开看一下，官方源码的行为细节。

## 6. 用一个 Milvus 最小例子，把 `Store` 的写入链路看顺
<small>（Milvus 是一个向量数据库，主要用于存储向量及其关联元数据，并支持相似度检索。）</small>

如果只讲概念，还是容易飘。
不如直接看一个最典型的组合：

- 外部 `Embedding`
- Milvus 负责向量存储
- `Indexer.Store` 统一完成写入

```go
package main

import (
    "context"
    "log"

    "github.com/cloudwego/eino/components/embedding"
    "github.com/cloudwego/eino/schema"
    "github.com/cloudwego/eino-ext/components/indexer/milvus2"
    "github.com/milvus-io/milvus/client/v2/milvusclient"
)

func main() {
    ctx := context.Background()

    // 这里假设 emb 已经提前初始化完成，比如 千问 / OpenAI / Ark 等 embedding 组件
    var emb embedding.Embedder

    idx, err := milvus2.NewIndexer(ctx, &milvus2.IndexerConfig{
        ClientConfig: &milvusclient.ClientConfig{
            Address:  addr,
            Username: username,
            Password: password,
        },
        Collection:   "kb_chunks",
        Dimension:    1024, // 必须和 embedding 模型输出维度一致
        MetricType:   milvus2.COSINE,
        IndexBuilder: milvus2.NewHNSWIndexBuilder().WithM(16).WithEfConstruction(200),
        Embedding:    emb,
    })
    if err != nil {
        log.Fatal(err)
    }

    docs := []*schema.Document{
        {
            ID:      "chunk_001",
            Content: "RAG 的第一步不是问模型，而是先把文档变成可检索对象。",
            MetaData: map[string]any{
                "source":   "rag_intro.md",
                "chunk_no": 1,
            },
        },
    }

    ids, err := idx.Store(ctx, docs)
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("stored ids=%v", ids)
}
```

这段代码真正值得看的，不是 Milvus 的参数怎么填，而是职责分工：

- 业务层交给它的仍然是 `schema.Document`
- 向量生成能力通过 `Embedding` 挂进去
- `Store` 统一把内容、向量、元数据写到后端
- 返回的 `ids` 才是这次写入最终确认下来的结果

也就是说，业务层并没有直接面对“Milvus 的行结构”。
它只是在说：

> 我有一批标准文档，请把它们写成以后能被检索的样子。

这才是 `Indexer` 抽象真正值钱的地方。

## 7. 为什么说 `Indexer` 不只服务向量数据库

如果你只看 Milvus 或 VikingDB，很容易觉得 `Indexer` 就是“向量库接口”。
可官方把 ES / OpenSearch 也放在 `Indexer` 下面，这个信号其实非常强。

来看 ES7 这种写法：

```go
indexer, _ := es7.NewIndexer(ctx, &es7.IndexerConfig{
    Client: client,
    Index:  "kb_chunks", // 写入到 ES 的哪个索引

    // 把统一的 Document 转成 ES 里的字段结构
    DocumentToFields: func(ctx context.Context, doc *schema.Document) (map[string]es7.FieldValue, error) {
        return map[string]es7.FieldValue{
            "content": {
                Value:    doc.Content,      // 文档正文
                EmbedKey: "content_vector", // 对 content 做向量化，结果写到 content_vector 字段
            },
            "source": {
                Value: doc.MetaData["source"], // 普通元数据字段，不做向量化
            },
        }, nil
    },

    Embedding: emb, // 向量模型：把指定字段文本转成向量
})
```

这段代码很能说明问题。

这里的 `Indexer` 已经不是“往向量列里塞一个浮点数组”那么简单了，而是在做两件事：

- 把 `Document` 映射成搜索引擎的字段结构
- 决定哪些字段要向量化，哪些字段按普通字段存储

这说明 `Indexer` 抽象的是“检索后端的写入协议”，不是“某一家向量数据库的专属写法”。

换句话说：

Milvus / VikingDB 让你更容易看见 `vector`。
ES / OpenSearch 则提醒你别把 `Indexer` 只看成 `vector`。

它真正落的，是 `Document -> backend indexable representation` 这层转换。

## 8. 放进 Chain / Graph 里，你会发现 Indexer 也是正式组件

很多人平时把 `Indexer` 单独调用一下，就觉得这层已经懂了。
其实不够。

只有当你把它放进编排里，才会更清楚它在 Eino 里的定位。

```go
// 在 Chain 中使用
chain := compose.NewChain[[]*schema.Document, []string]()
chain.AppendIndexer(indexer)

// 在 Graph 中使用
graph := compose.NewGraph[[]*schema.Document, []string]()
graph.AddIndexerNode("indexer_node", indexer)
```

这段代码表达的不是“语法还能这么写”。
它真正表达的是：

`Indexer` 从一开始就不是一个 helper。
它和 `ChatModel`、`Tool`、`Retriever` 一样，是能直接进入编排图的正式节点。

这带来的工程价值非常实际：

- 你可以把文档加载、清洗、索引串成一条稳定流水线
- 你可以通过 `compose.WithCallbacks` 统一观察整个入库过程
- 你可以在更复杂的 Graph 里，把不同写入策略拆成不同节点

一旦你从“帮我写一下数据”切换到“它是编排节点”的视角，`Indexer` 的位置就完全不一样了。

## 9. `Callback` 和自定义实现

到了生产环境中，很多问题不是“能不能写进去”，而是：

- 哪批文档写失败了
- 哪一步失败的，是 embedding 还是 backend 写入
- 返回的 `ids` 和输入文档是否一一对应
- 某次写入到底落到了哪个子索引

这时候，`Callback` 的价值就出来了。

官方给的回调输入输出很精妙：

```go
type CallbackInput struct {
    Docs  []*schema.Document
    Extra map[string]any
}

type CallbackOutput struct {
    IDs   []string
    Extra map[string]any
}
```

字段不多，但刚好卡在写入侧最该观察的地方：

- 进来的是什么文档
- 出去的是哪些 ID

如果你自己实现一个 `Indexer`，真正该守住的顺序也很明确：

1. 先收公共 option（框架统一认的）
2. 再收实现级 option （你自己需要的）
3. 从 `ctx` 里拿 callback manager
4. OnStart
5. 执行真实写入
6. OnError / OnEnd

一个更稳的骨架，可以像下面这样写：

```go
// MyIndexerOptions 是当前这个自定义 Indexer 的“实现级 option”。
// 也就是：只有 MyIndexer 自己认识和使用的参数。
type MyIndexerOptions struct {
    BatchSize  int // 批量写入时，每批处理多少条
    MaxRetries int // 写入失败时，最多重试几次
}

// WithBatchSize 用来生成一个实现级 option。
// 调用方可以在 Store(..., opts...) 时传入它，覆盖默认批大小。
func WithBatchSize(size int) indexer.Option {
    return indexer.WrapImplSpecificOptFn(func(o *MyIndexerOptions) {
        o.BatchSize = size
    })
}

// Store 是 Indexer 对外暴露的统一写入入口。
// 它做的不是“直接 insert”，而是：
// 1. 收集通用 option
// 2. 收集当前实现自己的 option
// 3. 触发回调开始事件
// 4. 执行真实写入
// 5. 根据结果触发结束或错误回调
func (i *MyIndexer) Store(ctx context.Context, docs []*schema.Document, opts ...indexer.Option) ([]string, error) {
    // 解析“公共 option”：
    // 比如 SubIndexes、Embedding 这类所有 Indexer 都能理解的参数。
    commonOpts := indexer.GetCommonOptions(nil, opts...)

    // 解析“实现级 option”：
    // 先给一个默认值，再用调用方传进来的 opts 覆盖。
    implOpts := indexer.GetImplSpecificOptions(&MyIndexerOptions{
        BatchSize: i.batchSize,
    }, opts...)

    // 从 ctx 中拿到 callback manager。
    // 它负责记录这次 Store 的开始、结束和错误。
    cm := callbacks.ManagerFromContext(ctx)
    runInfo := &callbacks.RunInfo{}

    // 通知回调系统：这次写入开始了。
    // 这里把输入文档和一些额外上下文信息带进去，便于日志、追踪和调试。
    ctx = cm.OnStart(ctx, runInfo, &indexer.CallbackInput{
        Docs: docs,
        Extra: map[string]any{
            "sub_indexes": commonOpts.SubIndexes,
            "batch_size":  implOpts.BatchSize,
        },
    })

    // 执行真正的写入逻辑。
    // 这里会进入 doStore，完成 embedding、字段映射、批量写入等动作。
    ids, err := i.doStore(ctx, docs, commonOpts, implOpts)
    if err != nil {
        // 如果写入失败，通知回调系统发生了错误。
        cm.OnError(ctx, runInfo, err)
        return nil, err
    }

    // 如果写入成功，通知回调系统结束，并把最终写入成功的 IDs 带出去。
    cm.OnEnd(ctx, runInfo, &indexer.CallbackOutput{
        IDs: ids,
    })
    return ids, nil
}

// doStore 是真正执行写入细节的地方。
// Store 负责“流程控制”，doStore 负责“实际干活”。
func (i *MyIndexer) doStore(
    ctx context.Context,
    docs []*schema.Document,
    commonOpts *indexer.Options,
    implOpts *MyIndexerOptions,
) ([]string, error) {
    // 如果本次写入指定了 Embedding，就先把文档内容转成向量。
    // 这样后续写入后端时，就能把文本和向量一起存进去。
    if commonOpts.Embedding != nil {
        // 先提取所有文档的正文内容，准备批量做 embedding。
        texts := make([]string, len(docs))
        for j, doc := range docs {
            texts[j] = doc.Content
        }

        // 调用 embedding 模型，把文本批量转成向量。
        vectors, err := commonOpts.Embedding.EmbedStrings(ctx, texts)
        if err != nil {
            return nil, err
        }

        // 把生成出来的向量挂回到每个 Document 上。
        for j, doc := range docs {
            doc.WithVector(vectors[j])
        }
    }

    // implOpts 里一般会继续参与下面的写入逻辑，
    // 比如按 BatchSize 分批写、按 MaxRetries 做重试等。
    _ = implOpts

    // 这里继续做：
    // - 批量写入
    // - 字段映射
    // - 分区/子索引路由
    // - 调用具体后端 SDK
    //
    // 最后返回后端确认写入成功的文档 ID 列表。
    return []string{"stored_doc_1"}, nil
}
```

这段骨架最重要的，不是细节实现，而是以下这几点：

- 公共 option 和实现级 option 分开处理
- callback 生命周期完整触发
- embedding 是“写入前可挂接能力”
- 真正的 backend 写入逻辑被收敛在 `doStore`

这才是一个能进工程的 `Store` 形状。


## 10. 总结

用一句话总结：

> `Store` 的本质不是一次 insert，而是“文档协议进入检索系统的统一写入入口”。

其中有3点需要重视：

- `Indexer` 解决的是写入侧协议统一，不是某家后端 SDK 的薄封装
- `Store` 里可能同时发生字段映射、向量生成、分区路由、批量写入和回调触发
- ES / OpenSearch 的实现已经足够说明，`Indexer` 抽象的不是“向量库”，而是“可检索后端”


## 参考资料

- CloudWeGo Eino [Indexer 使用说明](https://www.cloudwego.io/zh/docs/eino/core_modules/components/indexer_guide/)
- CloudWeGo Eino [components/indexer/interface.go](https://github.com/cloudwego/eino/blob/main/components/indexer/interface.go)
- CloudWeGo Eino [components/indexer/option.go](https://github.com/cloudwego/eino/blob/main/components/indexer/option.go)

---

## 发布说明

- GitHub 主文：[AI 大模型落地系列｜Eino 组件核心篇：为什么很多人会用 Indexer，却没真正看懂 Store](./06-为什么很多人会用Indexer，却没真正看懂Store.md)
- CSDN 跳转：[AI 大模型落地系列｜Eino 组件核心篇：为什么很多人会用 Indexer，却没真正看懂 Store](https://zhumo.blog.csdn.net/article/details/159537753)
- 官方文档：[Indexer 使用说明](https://www.cloudwego.io/zh/docs/eino/core_modules/components/indexer_guide/)
- 最新版以 GitHub 仓库为准。

