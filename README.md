# Go Eino 中文学习手册

面向中文开发者的 CloudWeGo Eino 学习仓，提供系统教程、可运行 Go 示例，以及适合面试与技术分享的表达框架。

`19 篇正文 + 1 篇总纲 + 5 个可运行 demo`

关键词：`CloudWeGo Eino`、`Go`、`AI Agent`、`RAG`、`Workflow`、`ADK`

## 一句话定位

如果你想系统学 Eino、快速跑通关键能力，并顺手沉淀一套可以讲给面试官或团队同事听的表达框架，这个仓库就是为你准备的。

## 你会在这里得到什么

- 一条从 `ChatModel -> Tool -> Chain / Graph -> ADK` 逐步推进的系统学习主线。
- 一组可以直接运行的 Go 示例，用来验证理解、录屏展示和排查概念误区。
- 一套适合面试、技术分享和项目复盘的四层表达框架。

## 适合谁看

- 想系统学习 CloudWeGo Eino，而不是零散看 API 的 Go 开发者。
- 已经能调模型，但想把 Tool、RAG、编排和 Agent 边界讲清楚的工程师。
- 准备做 AI Agent 相关面试展示、内部分享或知识沉淀的人。

## 快速开始

想先验证环境并感受最小模型调用，可以直接从 [`examples/chatmodel-message`](examples/chatmodel-message/README.md) 开始：

```bash
cd examples
go mod tidy
export DASHSCOPE_API_KEY="your_api_key"
export QWEN_MODEL="qwen3.5-flash"
go run ./chatmodel-message -- "用一句话解释 Eino 的 Component 设计"
```

PowerShell:

```powershell
cd examples
go mod tidy
$env:DASHSCOPE_API_KEY="your_api_key"
$env:QWEN_MODEL="qwen3.5-flash"
go run ./chatmodel-message -- "用一句话解释 Eino 的 Component 设计"
```

如果你暂时没有 API Key，可以先跑 [`examples/chain-graph`](examples/chain-graph/README.md)，它不依赖外部模型。更多运行方式见 [`examples/README.md`](examples/README.md)。

## 为什么这个仓库值得看

- 它不是零散笔记，而是按 `前置基础 -> 入门必学 -> 组件核心 -> 编排进阶 -> ADK 体系` 五段主线组织。
- `examples/` 独立放在单独 Go module 下，便于先跑通示例，再回到正文建立概念。
- GitHub 版是主版本，每篇文章顶部固定给出 `GitHub 主文 / CSDN 跳转 / 官方文档` 三链块，便于横向对照。
- 内容强调工程边界和职责分层，不只罗列 API，也解释这些抽象为什么存在。
- [`面试速览`](docs/面试速览.md) 把仓库内容整理成四层表达法，适合准备面试或做技术分享。
- 原始稿件保留在 [`source/README.md`](source/README.md)，成品文档由 [`tools/rebuild-docs.ps1`](tools/rebuild-docs.ps1) 生成，结构更易维护。
- `docs/` 使用中文目录和中文文件名，`examples/` 保留英文目录名，兼顾阅读体验和命令行稳定性。

## 推荐阅读路径

1. 先读 [`Go Eino 学习总纲`](docs/00-学习总纲.md)，建立完整目录感。
2. 再读 [`学习 AI 前需具备的基础知识`](docs/01-前置基础/01-学习AI前需具备的基础知识.md)，把 Agent、Tool、Function Calling、MCP、RAG 放进同一张图。
3. 入门主线先跑通 5 篇：`ChatModel`、`Runner`、`Memory`、`Tool`、`Callback`。
4. 然后补组件核心，重点吃透 `ChatModel`、`ChatTemplate`、`ToolsNode`、`Retriever`。
5. 最后进入 `Chain / Graph / Workflow / ADK`，把执行链路和 Agent 抽象讲顺。

## 精选内容

- [`一文读懂 Eino 的 ChatModel 和 Message`](docs/02-入门必学/01-ChatModel和Message.md)：最适合作为第一次跑通 Eino 的起点。
- [`一文读懂 Eino 的 Memory 与 Session`](docs/02-入门必学/03-Memory与Session（持久化对话）.md)：把“多轮对话”从概念讲成可恢复会话。
- [`一文读懂 Eino 的 Tool 和文件系统访问`](docs/02-入门必学/04-Tool和文件系统访问.md)：最容易体现 Agent 真正开始“动手”的转折点。
- [`一文读懂 Callback、Trace 和生产级可观测性`](docs/02-入门必学/05-Callback、Trace和生产级可观测性.md)：把黑盒 Agent 变成可观测系统。
- [`一文讲透编排（Chain 与 Graph）`](docs/04-编排进阶/01-一文讲透编排（Chain与Graph）.md)：最适合作为进阶和面试的分水岭主题。

## 可运行 Demo

- [`chatmodel-message`](examples/chatmodel-message/README.md)：推荐首跑，先验证模型调用、`Message` 和流式输出。
- [`memory-session`](examples/memory-session/README.md)：看清多轮消息如何持久化，以及会话如何恢复。
- [`tool-filesystem`](examples/tool-filesystem/README.md)：看 Agent 何时真正开始调用外部工具并操作文件系统。
- [`callback-trace`](examples/callback-trace/README.md)：看清模型调用、工具执行和本地回调日志的对应关系。
- [`chain-graph`](examples/chain-graph/README.md)：零 API Key 体验最小 Graph 编排。

## 面试展示怎么用

- [`面试速览`](docs/面试速览.md)：按 `模型调用 -> 组件协议 -> 编排运行时 -> Agent 抽象` 四层组织表达。
- 文章不是只堆 API，而是尽量把“为什么这个边界存在”“为什么要这样分层”讲明白。
- 示例不追求炫技，优先保证“能跑、能讲、能被追问时接住”。

## 系列总目录

### 总纲

先用总纲建立学习地图，知道整套内容为什么这样分层、应该按什么顺序推进。

- [`Go Eino 学习总纲`](docs/00-学习总纲.md)

### 前置基础

先把 Agent、Tool、Function Calling、MCP、RAG 放回同一张工程认知图里，避免后面只背名词。

- [`学习 AI 前需具备的基础知识`](docs/01-前置基础/01-学习AI前需具备的基础知识.md)

### 入门必学

这一段先跑通最关键的执行闭环：模型调用、Agent 运行、多轮会话、工具访问和可观测性。

- [`一文读懂 Eino 的 ChatModel 和 Message`](docs/02-入门必学/01-ChatModel和Message.md)
- [`一文读懂 ChatModelAgent、Runner、AgentEvent（Console 多轮）`](docs/02-入门必学/02-ChatModelAgent、Runner、AgentEvent（Console多轮）.md)
- [`一文读懂 Eino 的 Memory 与 Session（持久化对话）`](docs/02-入门必学/03-Memory与Session（持久化对话）.md)
- [`一文读懂 Eino 的 Tool 和文件系统访问`](docs/02-入门必学/04-Tool和文件系统访问.md)
- [`一文读懂 Callback、Trace 和生产级可观测性`](docs/02-入门必学/05-Callback、Trace和生产级可观测性.md)

### 组件核心

这一段重点理解组件边界和协议层，不再把 `ChatModel`、`Tool`、`Retriever` 当成孤立功能点。

- [`为什么很多人把 ChatModel 想简单了`](docs/03-组件核心/01-为什么很多人把ChatModel想简单了.md)
- [`ChatTemplate 为什么不是字符串拼接`](docs/03-组件核心/02-ChatTemplate为什么不是字符串拼接.md)
- [`Embedding 到底解决了什么`](docs/03-组件核心/03-Embedding到底解决了什么.md)
- [`为什么很多人会写 Tool，却没真正看懂 ToolsNode`](docs/03-组件核心/04-为什么很多人会写Tool，却没真正看懂ToolsNode.md)
- [`为什么很多人会用 Document Loader，却没真正看懂 Parser`](docs/03-组件核心/05-为什么很多人会用DocumentLoader，却没真正看懂Parser.md)
- [`为什么很多人会用 Indexer，却没真正看懂 Store`](docs/03-组件核心/06-为什么很多人会用Indexer，却没真正看懂Store.md)
- [`为什么很多人会用 Retriever，却没真正看懂 Retrieve`](docs/03-组件核心/07-为什么很多人会用Retriever，却没真正看懂Retrieve.md)

### 编排进阶

当组件能力看懂之后，再进入更复杂的执行关系、控制流和人工接管问题。

- [`一文讲透编排（Chain 与 Graph）`](docs/04-编排进阶/01-一文讲透编排（Chain与Graph）.md)
- [`既然有了 Chain、Graph，为何还需要 Workflow`](docs/04-编排进阶/02-既然有了Chain、Graph，为何还需要Workflow.md)
- [`从自动执行到人工接管，如何避免Agent一把梭`](docs/04-编排进阶/03-从自动执行到人工接管，如何避免Agent一把梭.md)

### ADK 体系

这部分放在后面，是因为它讨论的是更高一层的 Agent 抽象、协作方式和扩展能力。

- [`什么是 Eino ADK？`](docs/05-ADK体系/01-什么是EinoADK？.md)
- [`为什么一定要有 Agent 这层抽象`](docs/05-ADK体系/02-为什么一定要有Agent这层抽象.md)
- [`你对 ChatModelAgent 有了解吗？`](docs/05-ADK体系/03-你对ChatModelAgent有了解吗？.md)

## 仓库结构

```text
.
|-- README.md
|-- docs/      # GitHub 正式发布版文档，中文目录、中文文件名
|-- examples/  # 可运行示例，保留英文目录名
|-- source/    # 原始稿件
`-- tools/     # 文档生成脚本和文章映射清单
```

