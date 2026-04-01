# memory-session

这个示例把多轮消息保存到 JSONL，让你看到会话是怎么跨进程恢复的。

## 这个示例解决什么问题

- 让 `ChatModelAgent` 从“单次回答”变成“可恢复会话”。
- 看清会话 ID、消息落盘和上下文恢复是怎么串起来的。

## 运行前提

以下命令默认在 `examples/` 目录执行，并且已经完成 `go mod tidy`。

```bash
export DASHSCOPE_API_KEY="your_api_key"
export QWEN_MODEL="qwen3.5-flash"
export SESSION_DIR="./data/sessions"
```

PowerShell:

```powershell
$env:DASHSCOPE_API_KEY="your_api_key"
$env:QWEN_MODEL="qwen3.5-flash"
$env:SESSION_DIR="./data/sessions"
```

`SESSION_DIR` 可选，默认就是 `./data/sessions`。

## 运行命令

首次创建新会话：

```bash
go run ./memory-session
```

恢复已有会话：

```bash
go run ./memory-session --session "<session-id>"
```

## 你会看到什么

- 首次运行会打印新建的 `session ID`。
- 终端进入交互模式，输入内容后会流式返回回答。
- 空行退出后，会话会写入 `SESSION_DIR`。
- 使用同一个 `session ID` 重开时，可以恢复此前上下文。

## 关联文章

- [`一文读懂 Eino 的 Memory 与 Session（持久化对话）`](../../docs/02-入门必学/03-Memory与Session（持久化对话）.md)
