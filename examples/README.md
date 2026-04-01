# Examples

这些示例覆盖 Eino 学习与面试中最常见的 5 个场景，统一放在一个 Go module 下，默认从 `examples/` 目录执行。

## 先选哪个

- 想先验证模型接入是否跑通：[`chatmodel-message`](chatmodel-message/README.md)
- 想零门槛理解编排：[`chain-graph`](chain-graph/README.md)
- 想看 Agent 怎么调工具：[`tool-filesystem`](tool-filesystem/README.md)
- 想看多轮会话怎么持久化：[`memory-session`](memory-session/README.md)
- 想看运行时日志和耗时：[`callback-trace`](callback-trace/README.md)

## 运行前提

```bash
cd examples
go mod tidy
```

需要模型的示例请先参考 [`.env.example`](.env.example) 手动设置环境变量。当前有 4 个示例需要 `DASHSCOPE_API_KEY`：`chatmodel-message`、`memory-session`、`tool-filesystem`、`callback-trace`。

Bash:

```bash
export DASHSCOPE_API_KEY="your_api_key"
export QWEN_MODEL="qwen3.5-flash"
```

PowerShell:

```powershell
$env:DASHSCOPE_API_KEY="your_api_key"
$env:QWEN_MODEL="qwen3.5-flash"
```

如果你要运行 `tool-filesystem` 或 `callback-trace`，建议把 `PROJECT_ROOT` 设为 `..`，让工具默认作用到仓库根目录而不是 `examples/` 子目录。

```bash
export PROJECT_ROOT=".."
```

```powershell
$env:PROJECT_ROOT=".."
```

`SESSION_DIR` 只对 `memory-session` 生效，默认值是 `./data/sessions`。

## 快速开始

- 无 API Key：`go run ./chain-graph`
- 有 API Key：`go run ./chatmodel-message -- "用一句话解释 Eino 的 Component 设计"`

## 示例目录

- [`chatmodel-message`](chatmodel-message/README.md)：最小模型调用 + 流式输出，推荐首跑。
- [`memory-session`](memory-session/README.md)：JSONL 会话持久化 + ChatModelAgent 多轮恢复。
- [`tool-filesystem`](tool-filesystem/README.md)：DeepAgent + 本地文件系统工具调用。
- [`callback-trace`](callback-trace/README.md)：模型与工具执行的本地日志观测。
- [`chain-graph`](chain-graph/README.md)：最小 Graph + Lambda + ChatTemplate，不需要 API Key。
