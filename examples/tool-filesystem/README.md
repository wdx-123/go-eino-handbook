# tool-filesystem

用 `DeepAgent` + `LocalBackend` 让 Agent 真正调用本地文件系统工具，看到一次完整的 `tool call -> tool result -> assistant answer` 闭环。

## 这个示例解决什么问题

- 让 Agent 从“只会回答”变成“能读取本地文件和组织结果”。
- 看清 `Tool`、`Backend` 和 `DeepAgent` 在一次工具调用里各自承担什么职责。

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

这里建议把 `PROJECT_ROOT` 设为 `..`，让工具默认作用到仓库根目录。示例里的 Agent 会被明确要求使用绝对路径访问本地文件系统。

## 运行命令

```bash
go run ./tool-filesystem -- "请列出 examples 目录下的 Go 文件，并读取 examples/callback-trace/main.go 的前 20 行"
```

## 你会看到什么

- 输出中会先出现 `tool call`，再出现 `tool result`。
- 工具返回的内容会被回灌给模型，最后生成总结性回答。
- 这是理解 Eino 工具调用闭环最直接的示例之一。

## 关联文章

- [`一文读懂 Eino 的 Tool 和文件系统访问`](../../docs/02-入门必学/04-Tool和文件系统访问.md)
- [`为什么很多人会写 Tool，却没真正看懂 ToolsNode`](../../docs/03-组件核心/04-为什么很多人会写Tool，却没真正看懂ToolsNode.md)
