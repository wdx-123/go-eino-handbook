# callback-trace

用本地 Callback 日志把模型调用和工具执行过程“看见”，适合理解 Eino 的可观测性入口。

## 这个示例解决什么问题

- 看清 `ChatModel -> Tool -> ChatModel` 这条链路在运行时发生了什么。
- 观察本地日志里是怎么记录开始、结束、耗时、参数和结果预览的。

## 运行前提

以下命令默认在 `examples/` 目录执行，并且已经完成 `go mod tidy`。

```bash
export DASHSCOPE_API_KEY="your_api_key"
export QWEN_MODEL="qwen3.5-flash"
export PROJECT_ROOT=".."
```

PowerShell:

```powershell
$env:DASHSCOPE_API_KEY="your_api_key"
$env:QWEN_MODEL="qwen3.5-flash"
$env:PROJECT_ROOT=".."
```

这里建议把 `PROJECT_ROOT` 设为 `..`，让工具默认围绕仓库根目录工作。示例里的 Agent 同样会被要求使用绝对路径访问本地文件系统。

## 运行命令

```bash
go run ./callback-trace -- "请列出 docs 目录下的 Markdown 文件，再告诉我哪一篇最适合继续学习 Eino Callback。"
```

## 你会看到什么

- 日志中会看到模型开始、结束、耗时、工具调用参数和工具结果预览。
- 终端最后会打印拼接后的 assistant 输出。
- 如果你后续要接 Trace、排查慢调用或定位工具异常，这个示例是很好的起点。

## 关联文章

- [`一文读懂 Callback、Trace 和生产级可观测性`](../../docs/02-入门必学/05-Callback、Trace和生产级可观测性.md)
