# AI 大模型落地系列｜Eino 组件核心篇：Embedding 到底解决了什么

> GitHub 主文：[当前文章](./03-Embedding到底解决了什么.md)
> CSDN 跳转：[AI 大模型落地系列｜Eino 组件核心篇：Embedding 到底解决了什么](https://zhumo.blog.csdn.net/article/details/159079089)
> 官方文档：[Embedding 使用说明](https://www.cloudwego.io/zh/docs/eino/core_modules/components/embedding_guide/)
>
> 最新版以 GitHub 仓库为准，CSDN 作为分发入口，官方文档作为权威参考。

**一句话摘要**：把 Embedding 从一个能调的接口讲成语义检索、RAG 入库和相似度计算的基础设施。
**适合谁看**：准备进入 RAG 或语义检索体系的 Go 开发者。
**前置知识**：前置基础篇中的 RAG 概念、向量与相似度的基本理解
**对应 Demo**：[官方 Embedding 示例（本仓后续补充独立 demo）](https://www.cloudwego.io/zh/docs/eino/core_modules/components/embedding_guide/)

**面试可讲点**
- 能说明 Embedding 解决的是把文本映射成可计算语义空间的问题。
- 能把 Embedding 放进切块、入库、检索、生成的完整 RAG 链路里。

---
说到 embedding 组件，本质上就是把**文本变成一串数字向量**，让**程序**能“按语义理解文本”，而不只是按字符串匹配。

你可以把它理解成：

* 原始文本：`"今天天气不错"`
* 转成向量后：`[0.12, -0.87, 0.44, ...]`

这串向量人是看不懂的，因为他是拿个程序看的。
机器可以拿它来算“两个文本像不像”。

## 有啥用？！
### 他能干嘛？
<small>平时大家会用到的地方</small>

最常见就是这几类：

**1. 文本相似度计算**
比如：

* “怎么退款”
* “我要申请退钱”

虽然字不一样，但意思接近。
Embedding 后，这两句话的向量距离会比较近，所以系统知道它们语义相似。
<small>这个我在之前的博客<ai基础知识>中提到过</small>

**2. 语义搜索**
这也是最常见的用途。
比如你有很多文档、知识库、FAQ，用户问：

* “怎么修改收货地址”

系统不是只搜关键词“修改”“地址”，而是把这个问题也做成向量，然后去找**语义最接近**的文档片段。
这样即使文档里写的是“变更配送地址”，也能搜出来。

**3. RAG / 知识库问答**
这类项目里 embedding 基本是核心组件之一了。流程通常是：

* 先把知识库里的文本切块
* 然后为每个文本块生成 embedding
* 存到向量库里
* 用户提问时，也生成一个 embedding
* 去向量库里找最相关的内容
* 再把找到的内容喂给大模型回答

也就是说，它是“**先找资料**”这一步的关键。

**4. 文本聚类 / 分类 / 去重**
<small>这个是生活中其他方面的应用，非AI</small>
比如你有很多评论、工单、反馈，可以用 embedding 做：

* 相似工单归类
* 重复问题合并
* 用户反馈主题聚类

---

### 它不能直接干嘛？

它**不是直接拿来生成回答的**。
它更像一个“文本编码器”或者“语义检索工具”。

也就是：

* **LLM**：负责生成、总结、对话
* **Embedding**：负责把文本映射到语义空间，方便检索、匹配、聚类

---

### 总结：

这个组件的核心用途就一句话：

> **把文字转换成可计算的语义特征，方便程序判断哪些文本意思接近。**

---

## 浅用之法
接下来，我先说下基础语法。
```go
EmbedStrings(ctx, texts []string, opts ...Option) ([][]float64, error)
```

意思就是：

* 输入：多段文本
* 输出：每段文本对应的一个向量

例如：

```go
texts := []string{
    "hello",
    "how are you",
}
vectors, err := embedder.EmbedStrings(ctx, texts)
```

返回的 `vectors` 就是两段文本的向量表示。
后面你可以拿这些向量去做：

* 相似度比较
* 存入向量数据库
* 召回相关知识片段
* 聚类分析

---

## 食用之法
它的使用可以分成两层来看：

1. **最直接的用法：给几段文本生成向量**<small>也就是我在浅用之法中提到的</small>
2. **真正落地的用法：配合检索 / 向量库 / RAG 一起用**

我直接教你 “你上手怎么写”。

---

### 一、最基本用法：直接调用 `EmbedStrings`

本质核心就这几步：

#### 1. 创建 embedder

```go
import "github.com/cloudwego/eino-ext/components/embedding/openai"
// 这个导入的包，是兼容openai的。
// 如果你要用豆包，可以专门调用embedding/ark 这个包。

embedder, err := openai.NewEmbedder(ctx, &openai.EmbeddingConfig{
    APIKey:     accessKey,
    Model:      "text-embedding-3-large",
    Dimensions: &defaultDim,
    Timeout:    0,
})
if err != nil {
    panic(err)
}
```

这里的作用是初始化一个“文本转向量”的对象。

几个关键参数：

* `APIKey`：调用模型服务的密钥
* `Model`：选哪个 embedding 模型
* `Dimensions`：向量维度
* `Timeout`：超时时间

---

#### 2. 调用 `EmbedStrings`

```go
texts := []string{
    "hello",
    "how are you",
}

vectors, err := embedder.EmbedStrings(ctx, texts)
if err != nil {
    panic(err)
}
```

这一步做完后：

* `texts[0]` 对应 `vectors[0]`
* `texts[1]` 对应 `vectors[1]`

也就是说，输入几段文本，输出几组向量。

---

#### 3. 向量拿来干嘛

生成出来的 `vectors` 一般不会直接打印给用户看，而是继续做下面这些事：

* 存到向量数据库
* 跟别的向量算相似度
* 做召回
* 做聚类
* 做去重

---

### 二、完整demo

你可以把它理解成一个普通组件，哪里需要文本转向量，哪里调用。

例如：

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/cloudwego/eino-ext/components/embedding/openai"
)

func main() {
    ctx := context.Background()
    defaultDim := 3072
    accessKey := "your-api-key"

    embedder, err := openai.NewEmbedder(ctx, &openai.EmbeddingConfig{
        APIKey:     accessKey,
        Model:      "text-embedding-3-large",
        Dimensions: &defaultDim,
        Timeout:    0,
    })
    if err != nil {
        log.Fatal(err)
    }

    texts := []string{
        "退款怎么申请",
        "如何进行退钱操作",
        "今天天气不错",
    }

    vectors, err := embedder.EmbedStrings(ctx, texts)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println("文本数量:", len(vectors))
    fmt.Println("第一条文本向量维度:", len(vectors[0]))
}
```

这就是最标准的“用”了！

---

### 三、带 Option 怎么用

公共 option 其实也挺有用的，比如 `WithModel`。

这表示你在调用时，可以临时覆盖模型参数。

```go
vectors, err := embedder.EmbedStrings(ctx, texts,
    embedding.WithModel("text-embedding-3-small"),
)
```

大致意思就是：

* `embedder` 初始化时有一个默认模型
* 这次调用时，临时改成另一个模型

这个适合：

* 平时默认用大模型
* 某些场景为了省钱/提速，改用小模型

<small>但是我在此，不得点明一下，虽然向量在不同模型之前还是有一定的兼容，但是尽量不切换，就不要切换，影响效果</small>

---

### 四、在编排中怎么用

如果你不是手动一行一行写，而是用 Eino 的 `Chain` 或 `Graph`，就可以把 embedding 当成节点塞进去。

#### 在 Chain 中使用
<small>初次接触chain的话，你可以将其当成一条流水线</small>
```go
chain := compose.NewChain[[]string, [][]float64]()
chain.AppendEmbedding(embedder)
```

意思是：

* 输入：`[]string`
* 输出：`[][]float64`

也就是整条链专门做“文本 -> 向量”。

---

#### 在 Graph 中使用

```go
graph := compose.NewGraph[[]string, [][]float64]()
graph.AddEmbeddingNode("embedding_node", embedder)
```

意思是把 embedding 作为图里的一个节点，后面可以接别的节点一起跑。

---

### 五、带 Callback 怎么用

这个一般用于：

* 记录日志
* 统计 token
* 监控调用过程
* 调试输入输出

<small>Callback有点像 给整个链路，外挂了一层“生命周期 中间件 / 钩子机制"</small>

通常是：定义 handler，然后通过 `compose.WithCallbacks` 传进去。

例如：

```go
handler := &callbacksHelper.EmbeddingCallbackHandler{
    OnStart: func(ctx context.Context, runInfo *callbacks.RunInfo, input *embedding.CallbackInput) context.Context {
        log.Printf("开始 embedding，文本数: %d, 内容: %v\n", len(input.Texts), input.Texts)
        return ctx
    },
    OnEnd: func(ctx context.Context, runInfo *callbacks.RunInfo, output *embedding.CallbackOutput) context.Context {
        log.Printf("embedding 完成，生成向量数: %d\n", len(output.Embeddings))
        return ctx
    },
}
```

然后运行时：

```go
callbackHandler := callbacksHelper.NewHandlerHelper().Embedding(handler).Handler()

runnable, _ := chain.Compile(ctx)
vectors, err := runnable.Invoke(ctx, []string{"hello", "how are you"},
    compose.WithCallbacks(callbackHandler),
)
```

这样你就能看到：

* 输入了什么
* 什么时候开始
* 什么时候结束
* 输出了多少向量
* token 消耗多少

---

### 六、真实场景

真正业务里，embedding 很少是“调一下就结束”，
我拿知识库问答，给大家描绘一下整体流程。

#### 场景：做知识库问答

#### 第一步：把知识库切块

比如一篇文档切成很多段：

```go
chunks := []string{
    "退款申请需要在订单完成后7天内提交",
    "修改收货地址请在发货前联系人工客服",
    "发票可在订单详情页申请",
}
```

#### 第二步：给每个 chunk 生成向量

```go
chunkVectors, err := embedder.EmbedStrings(ctx, chunks)
```

#### 第三步：存起来

通常会存到向量数据库里，同时保存原文：

* 文本内容
* 对应向量
* 文档来源
* chunk id

#### 第四步：用户提问时，也生成向量

```go
query := []string{"订单下完以后地址还能改吗"}
queryVector, err := embedder.EmbedStrings(ctx, query)
```

#### 第五步：拿 query 的向量去检索最相近的 chunk

找出最相似的几段知识。

#### 第六步：把召回结果交给大模型回答

这才变成完整的 RAG。

---

### 七、语法总结

#### 最小步骤

1. 初始化 embedder
2. 调 `EmbedStrings`
3. 拿到 `[][]float64`

#### 常见增强

4. 用 `Option` 临时覆盖参数
5. 用 `Callback` 打日志和监控
6. 放进 `Chain` / `Graph` 编排

---

### 八、模板总结

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/cloudwego/eino/components/embedding"
    embeddingOpenAI "github.com/cloudwego/eino-ext/components/embedding/openai"
)

func main() {
    ctx := context.Background()
    defaultDim := 3072 // 通常是定死的
    accessKey := "your-api-key"

    embedder, err := embeddingOpenAI.NewEmbedder(ctx, &embeddingOpenAI.EmbeddingConfig{
        APIKey:     accessKey,
        Model:      "text-embedding-3-large",
        Dimensions: &defaultDim,
        Timeout:    0,
    })
    if err != nil {
        log.Fatal(err)
    }

    texts := []string{
        "退款怎么申请",
        "如何退钱",
        "修改收货地址的方法",
    }

    vectors, err := embedder.EmbedStrings(
        ctx,
        texts,
        embedding.WithModel("text-embedding-3-small"),
    )
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("生成了 %d 个向量\n", len(vectors))
    fmt.Printf("每个向量维度: %d\n", len(vectors[0]))
}
```

---


### 九、尾声

大家可以把它记成：

> **Embedding 的“用法”就是：先把文本喂进去生成向量，再把这个向量用于检索、匹配、聚类等后续处理。**

相信大家看到这里，应该也明白了：
**“会调用 embedding”** 和 **“会用 embedding 做业务”** 是两回事。

前者很简单，就是：

* NewEmbedder
* EmbedStrings

后者才是完整链路，比如：

* 文本切块
* 向量生成
* 向量存储
* 相似检索
* 大模型回答


---
1、OpenAI开发者([向量嵌入](https://developers.openai.com/api/docs/guides/embeddings/))
2、官方文档 CloudWeGo([Embedding 使用说明](https://www.cloudwego.io/zh/docs/eino/core_modules/components/embedding_guide/#option-%E5%92%8C-callback-%E4%BD%BF%E7%94%A8))

---

## 发布说明

- GitHub 主文：[AI 大模型落地系列｜Eino 组件核心篇：Embedding 到底解决了什么](./03-Embedding到底解决了什么.md)
- CSDN 跳转：[AI 大模型落地系列｜Eino 组件核心篇：Embedding 到底解决了什么](https://zhumo.blog.csdn.net/article/details/159079089)
- 官方文档：[Embedding 使用说明](https://www.cloudwego.io/zh/docs/eino/core_modules/components/embedding_guide/)
- 最新版以 GitHub 仓库为准。

