# chain-graph

一个不依赖外部模型的最小 Graph 例子，适合零门槛理解节点、边和 `Runnable` 是怎么串起来的。

## 这个示例解决什么问题

- 让你先看懂最小编排骨架，而不是一上来就把注意力放在模型调用上。
- 观察 `Lambda` 节点如何改写输入，再交给 `ChatTemplate` 产出消息列表。

## 运行前提

以下命令默认在 `examples/` 目录执行，并且已经完成 `go mod tidy`。

这个示例不需要 `DASHSCOPE_API_KEY`，也不依赖外部模型。

## 运行命令

```bash
go run ./chain-graph
```

## 你会看到什么

- 终端打印两条消息：一条 `system`，一条 `user`。
- `user` 消息会包含 Lambda 节点改写后的内容。
- 如果你只想先验证编排概念是否理解，这个示例比接模型更轻量。

## 关联文章

- [`一文讲透编排（Chain 与 Graph）`](../../docs/04-编排进阶/01-一文讲透编排（Chain与Graph）.md)
