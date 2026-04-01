# chatmodel-message

推荐首跑的最小模型调用示例，帮助你先把 `ChatModel`、`Message` 和流式输出跑通。

## 这个示例解决什么问题

- 看清一次最小 Eino 模型调用的输入输出边界。
- 观察 `system` / `user` 消息和流式输出在代码里的基本形态。

## 运行前提

以下命令默认在 `examples/` 目录执行，并且已经完成 `go mod tidy`。

```bash
export DASHSCOPE_API_KEY="your_api_key"
export QWEN_MODEL="qwen3.5-flash"
```

PowerShell:

```powershell
$env:DASHSCOPE_API_KEY="your_api_key"
$env:QWEN_MODEL="qwen3.5-flash"
```

## 运行命令

```bash
go run ./chatmodel-message -- "用一句话解释 Eino 的 Component 设计"
```

## 你会看到什么

- 终端按流式打印模型回答。
- 命令行参数会覆盖默认问题。
- 如果你只想先验证模型接入和输出链路，这是最适合的第一个示例。

## 关联文章

- [`一文读懂 Eino 的 ChatModel 和 Message`](../../docs/02-入门必学/01-ChatModel和Message.md)
