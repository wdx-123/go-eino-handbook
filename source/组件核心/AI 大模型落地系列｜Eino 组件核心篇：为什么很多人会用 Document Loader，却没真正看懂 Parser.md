声明：本文数据源于官方文档与官方实现，重点参考 [Document Loader 使用说明](https://www.cloudwego.io/zh/docs/eino/core_modules/components/document_loader_guide/)、[ToolsNode&Tool 使用说明](https://www.cloudwego.io/zh/docs/eino/core_modules/components/tools_node_guide/) 以及 `eino/components/document`、`eino/components/document/parser` 相关源码。
@[TOC](为什么很多人会用 Document Loader，却没真正看懂 Parser)

很多人第一次看到 `Document Loader`，第一反应都很直接：

不就是“读文件”或者“抓网页”吗？

本地文件读出来，网页内容拉下来，能拿到一段文本，事情似乎就结束了。

可如果你真把它只理解成一个“读取器”，后面一旦进入知识库入库、文档追踪、多格式解析、链路编排，你很快就会发现这个理解太浅了。

因为在 Eino 里，`Document Loader` 真正要解决的，不只是“把内容读出来”，而是：

> 把不同来源的原始内容，统一收口成标准的 `[]*schema.Document`。

而在这条链路里，最容易被忽视的，其实不是 `Load` 本身，而是 Loader 背后的 `Parser`。

你可以把这篇文章先记成一句话：

> `Loader` 管来源接入，`Parser` 管内容解释；前者解决“东西从哪来”，后者解决“这些内容该怎么进文档协议”。

如果这两层边界没拆开，很多人后面做 RAG 时，文档链路虽然也能跑，但通常会写得很糙。

## 1. `Document Loader` 到底解决什么，不只是“把文件读出来”

先说结论：

`Document Loader` 不是简单的 I/O 封装，它是文档进入系统前的“来源收口层”。

这层价值主要有三件事。

**第一，它统一了来源。**

你的文档可能来自本地文件、网络 URL、S3，甚至以后还可能接企业内部对象存储。
如果每一种来源都让上层逻辑直接自己读、自己转、自己拼元数据，后面的链路很快就会变得很散。

`Loader` 做的，就是把“来源差异”先压平。

**第二，它统一了输出协议。**

不管前面读到的是 Markdown、HTML、PDF，还是普通文本，出去的时候都得变成 `[]*schema.Document`。
一旦这个协议立住了，后面的 `Chain`、`Graph`、切分、索引、检索，才有稳定输入。

**第三，它把文档接入正式纳入运行时链路。**

这也是很多人容易忽略的点。
在 Eino 里，`Loader` 的 `ctx` 不只是拿来取消请求，它还承担 Callback Manager 的传递。
这就意味着，文档加载不是一段藏在角落里的工具函数，而是可以被观察、被编排、被扩展的正式组件。

放到 RAG 里看，它是“数据进入系统的第一站”，但它还不是检索、不是索引、也不是切分策略本身。

它解决的是入口统一，不是后续所有问题。

## 2. 看懂 `Loader` 接口后，才知道官方真正想收口什么

官方给出的核心接口其实非常短：

```go
type Loader interface {
    Load(ctx context.Context, src Source, opts ...LoaderOption) ([]*schema.Document, error)
}

type Source struct {
    URI string
}
```

很多人第一次看到这段代码，会觉得信息量不大。
可实际上，官方想收口的边界已经放得很清楚了。

先看 `Load`。

它返回的不是 `string`，也不是 `[]byte`，而是 `[]*schema.Document`。
这一步非常关键。

它说明 Loader 的目标从来不是“把内容读出来就算完”，而是“把内容整理成系统认可的文档协议再交出去”。

再看 `src Source`。

`Source` 现在只有一个 `URI` 字段，设计得很克制。
这个做法的好处是，它把“来源描述”压成了一个统一入口：

- 本地文件路径可以是 URI
- 网络 URL 可以是 URI
- 存储系统对象地址也可以是 URI

这其实是在提醒你：Loader 关注的是“统一来源标识”，不是给每种来源单独造一套接口。

最后看 `opts ...LoaderOption`。

官方没有给 Loader 设计一套很重的公共参数表，而是把公共层保持极简，把可变部分留给各个具体实现。

这代表的不是“设计不完整”，恰恰相反，它说明官方很清楚这层该怎么收：

- 公共协议统一
- 具体实现差异下放
- 运行时扩展通过 Option 接进去

所以这段接口真正表达的是：

> Loader 要统一的是调用姿势和输出协议，不是把所有来源都塞进一个笨重的大接口里。

## 3. `Source` 和 `schema.Document` 为什么是这条链路的关键协议

如果说 `Load` 是入口方法，那 `Source` 和 `schema.Document` 才是整条文档链路真正的协议地基。

`Source` 看起来简单，但 `URI` 的意义其实比“文件路径”大得多。

它不只告诉 Loader 去哪里取内容，也会影响后面的解析策略。
尤其当你接 `ExtParser` 这类“基于扩展名选择解析器”的实现时，`URI` 不只是来源地址，它还是格式判断线索。

再看 `schema.Document`：

```go
type Document struct {
    ID       string
    Content  string
    MetaData map[string]any
}
```

这三个字段里，很多人最容易低估的是 `MetaData`。

可在工程里，`MetaData` 根本不是附赠字段，它几乎就是后续链路的挂载点。

它至少承载这些信息：

- 文档来源
- 原始 URI
- 文件扩展名
- 页码、分段、子索引
- 向量、分数、排序相关信息
- 其他业务自定义字段

你现在如果把 `MetaData` 看轻，后面通常会在三个地方吃亏：

- **来源追踪**：查到一段内容，却不知道它从哪来的
- **排序和召回**：拿到了文档，却缺少分数、层级、子索引等附加信息
- **链路排障**：内容不对，却没法判断是 Loader 问题、Parser 问题，还是后处理问题

所以别把 `Document` 理解成“内容字符串 + 一个 map”。

在 Eino 里，它更像是文档在系统里的统一载体。
`Content` 是正文，`MetaData` 是上下文，二者缺一不可。

## 4. 为什么 `Parser` 不是配角，而是 Loader 内部真正的内容解释层

很多人学到 Loader 这一层时，会把注意力都放在“怎么读 URL”“怎么读文件”上。

可只要你继续往下一看，就会发现真正决定文档质量的，往往不是“读到了没有”，而是“读到以后怎么解释”。

这就是 `Parser` 的职责。

官方接口同样很短：

```go
type Parser interface {
    Parse(ctx context.Context, reader io.Reader, opts ...Option) ([]*schema.Document, error)
}
```

它做的事情也很清楚：

> 从一个 `io.Reader` 里解析原始内容，并产出标准文档。

这层和 Loader 的边界一定要拆开看：

- `Loader` 解决“从哪里拿内容”
- `Parser` 解决“拿到内容后按什么规则解释”

这个区别看着像概念问题，实际上很工程。

因为同样是一份原始数据：

- 当它是 `.txt` 时，你可能直接按文本处理
- 当它是 `.html` 时，你通常要提正文、去标签
- 当它是 `.pdf` 时，你可能要按页或按布局抽取内容

这些差异，不该压在 Loader 里写成一个越来越大的 `switch`，而应该下沉到 Parser 层。

官方给 Parser 的两个公共 Option 也很有意思：

- `WithURI`
- `WithExtraMeta`

这两个能力其实已经把 Parser 的工程定位说透了。

`WithURI` 说明解析器不只是吃字节流，它还会利用来源信息决定解析行为。
`ExtParser` 能按扩展名挑解析器，靠的就是这个。

`WithExtraMeta` 则说明解析不是只管正文，元数据也应该在这一层被合理补齐并合并进文档。

说白了，很多人以为 Loader 是主角、Parser 是配件。
但真到了多格式和工程落地场景里，你会发现：

> Loader 决定入口通不通，Parser 决定进来的内容是不是“可用的文档”。

## 5. 一条完整链路在 Eino 里到底怎么走

如果把文档接入链路压成一条直线，它大致是这样：

```text
URI
  -> Loader 获取原始内容
  -> Parser 依据格式解析
  -> 构造 []*schema.Document
  -> 进入 Chain / Graph
  -> 再进入后续切分、索引、检索链路
```

这里最关键的一点是：

`Loader` 的输出不是一个局部变量，而是后续编排系统的正式输入。

所以你才能在 `Chain` 里直接接它：

```go
chain := compose.NewChain[document.Source, []*schema.Document]()
chain.AppendLoader(loader)
```

也能在 `Graph` 里把它当节点挂进去：

```go
graph := compose.NewGraph[document.Source, []*schema.Document]()
graph.AddLoaderNode("loader_node", loader)
```

这已经说明，官方设计 `Loader` 时，压根没把它当成一个“顺手写的帮助函数”，它从一开始就是可编排组件。

你如果把这层看明白，再回头看 RAG，就会顺很多。

很多人把知识库理解成“拿一堆文件，切一切，存向量库”。
这当然没错，但真正第一步其实是：

> 让不同来源的文档，以统一协议、带着必要元数据、可被观察地进入系统。

这一步就是 Loader 和 Parser 共同完成的。

## 6. 一个最小例子，把 `FileLoader`、`ExtParser` 和元数据串起来

如果只讲概念，还是容易飘。
所以可以看一个最小组合：

```go
textParser := parser.TextParser{}
htmlParser, _ := html.NewParser(ctx, &html.Config{Selector: gptr.Of("body")})
pdfParser, _ := pdf.NewPDFParser(ctx, &pdf.Config{})

extParser, _ := parser.NewExtParser(ctx, &parser.ExtParserConfig{
    Parsers: map[string]parser.Parser{
        ".html": htmlParser,
        ".pdf":  pdfParser,
    },
    FallbackParser: textParser,
})

loader, _ := file.NewFileLoader(ctx, &file.FileLoaderConfig{
    UseNameAsID: true,
    Parser:      extParser,
})

docs, _ := loader.Load(ctx, document.Source{
    URI: "./testdata/test.html",
})

fmt.Println(docs[0].ID)
fmt.Println(docs[0].Content)
fmt.Printf("%#v\n", docs[0].MetaData)
```

这段代码里，最该看的不是 API 语法，而是职责分工：

- `FileLoader` 负责把本地文件变成可读内容
- `ExtParser` 负责按扩展名把内容交给合适的解析器
- 最终统一产出 `schema.Document`

也就是说，真正让 `.html`、`.pdf`、普通文本走出不同解析路径的，不是 `FileLoader`，而是 `ExtParser` 背后的 Parser 选择机制。

这里还有一个很容易被忽视的点：

`MetaData` 不是在最后“随手补一点信息”，而是从解析阶段就应该被认真传递和保存。

比如来源 URI、扩展名、页面信息，这些字段现在看着不起眼，可一旦你后面要做来源回溯、切分定位、召回解释，它们都会变得非常值钱。

## 7. `Option` 和 `Callback` 为什么不是装饰品

很多人一看到 Option，会下意识把它理解成“几个可选参数”；一看到 Callback，又觉得“加载文档还需要回调吗”。
这么理解不能说错，但都太轻了。

先说 Option。

Loader 公共层没有很重的通用 Option，具体实现可以通过 `WrapLoaderImplSpecificOptFn` 扩展自己的运行时参数。
Parser 这边则分成两层：

- 公共 Option：`WithURI`、`WithExtraMeta`
- 实现特定 Option：通过 `WrapImplSpecificOptFn` 扩展

这意味着 Option 在这里真正扮演的是“运行时扩展入口”，不是参数补丁。

再说 Callback。

Loader 的回调输入输出是官方明确给出来的：

- `LoaderCallbackInput`
- `LoaderCallbackOutput`

这件事的意义很直接：

你可以观察文档什么时候开始加载、加载了哪个来源、最后产出了多少个文档、失败发生在哪一步。

一旦链路里同时有本地文件、网页、S3，多种 Parser 并存，没有观测你会很快掉进黑盒。

所以 Callback 的价值，不是“打印两行日志”，而是把文档加载这一步正式接进可观测链路。

## 8. 自己实现 Loader / Parser 时，真正该守住哪些边界

如果你要自己写一个 Loader，最容易犯的错，就是把“来源获取”“内容解析”“元数据组装”“回调处理”全部揉进一个大函数里。
代码当然也能跑，但只要格式一多、来源一多、链路一长，维护成本就会立刻上来。

更稳的做法，应该像下面这样收：

```go
func (l *CustomLoader) Load(
    ctx context.Context,
    src document.Source,
    opts ...document.LoaderOption,
) ([]*schema.Document, error) {
    loaderOpts := document.GetLoaderImplSpecificOptions(&loaderOptions{
        Timeout: l.timeout,
    }, opts...)

    reader, err := l.open(ctx, src, loaderOpts)
    if err != nil {
        return nil, err
    }
    defer reader.Close()

    ctx = callbacks.OnStart(ctx, &document.LoaderCallbackInput{
        Source: src,
    })

    docs, err := l.parser.Parse(ctx, reader,
        parser.WithURI(src.URI),
        parser.WithExtraMeta(map[string]any{
            "source": src.URI,
        }),
    )
    if err != nil {
        callbacks.OnError(ctx, err)
        return nil, err
    }

    callbacks.OnEnd(ctx, &document.LoaderCallbackOutput{
        Source: src,
        Docs:   docs,
    })
    return docs, nil
}
```

这段骨架里，真正该守住的是四条边界：

**1. Loader 负责来源接入，不负责格式解释。**

打开文件、请求网页、拉取对象存储，这些属于 Loader。
至于 HTML 怎么提正文、PDF 怎么抽文本，这些应该交给 Parser。

**2. Parser 负责内容解释，不负责到处拿数据。**

它吃的是 `io.Reader`，不是 URL，也不是文件系统路径。
这样它才可复用，也更容易做单测。

**3. URI 和 MetaData 要沿着链路往下传。**

如果你自己写 Loader，却忘了把 `src.URI` 和额外元数据传给 Parser，很多扩展能力就会直接失效。
最典型的就是 `ExtParser` 选不对解析器，或者解析后的文档丢了来源信息。

**4. 回调和错误不要被吞。**

加载失败时要返回有意义的错误。
能进回调链路的地方，也别省。
真正到了线上，排障时你会感谢自己没把这一步写成黑盒。

## 9. 总结

如果把今天这篇压成一句话，那就是：

> `Document Loader` 解决的是来源收口，`Parser` 解决的是内容解释；前者让文档能进系统，后者决定进来的到底是不是“可用文档”。 

再压缩成三句话，就是：

- `Loader` 不是简单读取器，而是文档进入 Eino 的统一入口
- `Parser` 不是配角，它决定原始内容怎样被解释成标准文档
- `MetaData`、`Option`、`Callback` 说明这条链路从一开始就是工程组件，不是一次性 demo 代码

所以别把 `Document Loader` 只当成“读取 PDF、读取网页”的小功能。
你一旦把这层看懂，后面再去接 `Indexer`、`Retriever`，或者继续往更完整的 RAG 流程走，很多设计都会顺理成章。

## 参考资料

- CloudWeGo Eino [Document Loader 使用说明](https://www.cloudwego.io/zh/docs/eino/core_modules/components/document_loader_guide/)
- CloudWeGo Eino [ToolsNode&Tool 使用说明](https://www.cloudwego.io/zh/docs/eino/core_modules/components/tools_node_guide/)
- CloudWeGo Eino [components/document/interface.go](https://github.com/cloudwego/eino/blob/main/components/document/interface.go)
- CloudWeGo Eino [components/document/parser/interface.go](https://github.com/cloudwego/eino/blob/main/components/document/parser/interface.go)
