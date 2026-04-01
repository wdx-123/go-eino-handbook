声明：本文数据源于官方文档与官方实现[[第三章官方文档]](https://www.cloudwego.io/zh/docs/eino/quick_start/chapter_03_memory_and_session/)。
@[TOC](AI大模型落地系列：为什么你的 Eino Agent 一退出就“失忆”？一文读懂 Memory、Session 和持久化对话)

上一篇，我们把 Eino 的 `Tool` 和文件系统接上了。

现在看起来，Agent 好像已经不只是“会聊天”，而是真的能做点事了。

但只要你把项目一停，问题就会立刻暴露：

- 它不记得你上一轮说了什么
- 它不知道上一次会话停在什么地方
- 它更不可能跨进程恢复上下文

这不是模型突然变笨了。
而是你的对话历史根本没被保存下来。

很多人第一次做多轮对话时，容易把“上下文带着一起发给模型”误以为“已经做了记忆”。
其实大多数时候，你只是把消息临时堆在内存里而已。
程序一退出，这段“记忆”也就跟着一起没了。

所以本篇文章，我先引出一个很有趣的问题：

> 为什么多轮对话一旦进程退出就会“失忆”，以及在 Eino 里，这件事到底该由谁负责？

而接下来，这篇文章我将会分成两块来讲：

- 首先带你做出最小可运行的 Demo，将持久化对话跑通
- 再回头对照官方第三章源码，看它到底是怎么落地 `Memory / Session / Store` 的

## 1. 你以为的多轮对话vs实际上的多轮对话

前面几篇文章，我们已经解决了两个关键边界：

- `ChatModel` 解决了“怎么和模型说话”
- `Tool` 为我们的大模型安装上了接触这个世界的双手

但真实项目里，还有第三个同样关键的问题：

> 这次对话的状态，到底存哪儿？

如果你现在的程序是这样的：

```go
history := []*schema.Message{
    schema.UserMessage("你好"),
}
```

然后每轮把新消息 append 进去，再把整段 `history` 丢给模型。

从“单次运行项目”的角度，这当然能实现多轮有记忆对话。
但从“工程系统”的角度，这仍然是一次**纯内存会话**。

它至少有三个非常现实的问题：

- 进程一退出，对话历史就丢了
- 你没法通过 `session-id` 恢复之前的会话
- 你也没法做会话列表、删除、搜索、导出这些管理能力

说得再直一点：

**多轮对话，不等于持久化会话。**

前者只是“这一轮请求能不能带上上一轮消息”。
后者问的是“这段状态能不能`脱离`当前进程独立存在”。

这个区别，在 demo 阶段不明显，一到真实业务里就立刻会变成刚需。

比如：

- 客服对话要能下次继续
- Copilot 类助手要能恢复上次的问题现场
- 审批流或长任务要能停下来后继续
- 用户会话要能按 ID 管理，而不是只活在某个进程变量里

所以本篇博客的重点，不是是“再教你一种新组件”，而是让你开始正视**会话状态**这件事。

## 2. Memory、Session、Store 到底在解决什么问题

先把最容易混的一点讲清楚。

> `Memory`、`Session`、`Store` 是业务层概念，不是 Eino 框架内置的核心组件。

这一点官方第三章写得很明确。
Eino 负责的是“如何处理消息”，而“消息如何被保存、恢复、管理”这件事，完全是业务层自己决定的。

换句话说：

- Eino 负责把消息交给模型或 Agent 处理
- 业务层负责把消息存起来，并在下一次再取出来

所以，若这两个边界混了，就会造成 `Memory`、`Session`、`Store` 就会越看越乱。

### `Memory` 是什么

如果用后端的语言讲，`Memory` 不是“模型脑子里的一块魔法区域”。

它更像是：

> 一套对话历史的持久化方案。

你可以把它存在：

- 本地文件
- MySQL
- Redis
- 对象存储


### `Session` 是什么

你可以将`Session` 理解成一次完整对话的边界。

他至少能带给你3个锚点：

- 这次会话的 ID 是谁
- 这次会话什么时候创建
- 这次会话目前积累了哪些消息

例如：一个简略版的结构

```go
type Session struct {
    ID        string
    CreatedAt time.Time
    messages  []*schema.Message
}
```

注意这里的重点不在字段多少，而在含义：

`Session` 不是“某一次请求”，而是“同一段对话生命周期里的状态容器”。

### `Store` 是什么

如果说 `Session` 是单个会话，那么 `Store` 解决的就是：

> 这些会话到底存在哪，怎么取回来，怎么创建和管理。

这里做一个极简版本：

```go
type Store struct {
    dir   string
    cache map[string]*Session
}
```

它通常需要提供以下这些接口：

- `GetOrCreate(id)`：有就加载，没有就新建
- `List()`：列出已存在会话
- `Delete(id)`：删除某个会话

所以读完本篇博客最应该建立的认知，不是些结构体名词，而是：

> Eino 负责处理消息，`Memory / Session / Store` 负责让消息可恢复。

我再重复一次，这句话很重要：

> `Memory / Session / Store` 是业务层概念，不是 Eino 框架核心组件。

## 3. 实战 Demo

前面讲了半天，如果你脑子里还是抽象的，那最好的办法就是自己先跑一次。

看这个demo的时候，你需要怀着两个目的：

- 将核心链路看懂
- 再回头看官方源码时，不用在被目录和细节分散注意力

### 先准备依赖和环境变量

```bash
go mod init eino-ch03-demo
go get github.com/cloudwego/eino@latest
go get github.com/cloudwego/eino-ext/components/model/qwen@latest
go get github.com/google/uuid@latest

export DASHSCOPE_API_KEY="你的百炼 API Key"
export QWEN_MODEL="qwen3.5-flash"
export SESSION_DIR="./data/sessions"
```

如果你在 Windows PowerShell 下：

```powershell
$env:DASHSCOPE_API_KEY="你的百炼 API Key"
$env:QWEN_MODEL="qwen3.5-flash"
$env:SESSION_DIR=".\data\sessions"
```

### 把下面代码保存成 `main.go`

```go
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
	"github.com/cloudwego/eino-ext/components/model/qwen"
	"github.com/google/uuid"
)

// Session 表示一个对话会话。
// 会话元信息和消息都会持久化到对应的 jsonl 文件中。
type Session struct {
	ID        string
	CreatedAt time.Time
	filePath  string
	messages  []*schema.Message
}

// Append 将一条消息追加到内存和会话文件中。
func (s *Session) Append(msg *schema.Message) error {
	s.messages = append(s.messages, msg)

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(s.filePath, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = fmt.Fprintf(f, "%s\n", data)
	return err
}

// GetMessages 返回一份消息切片副本，避免外部直接修改内部状态。
func (s *Session) GetMessages() []*schema.Message {
	result := make([]*schema.Message, len(s.messages))
	copy(result, s.messages)
	return result
}

// Title 使用第一条用户消息生成会话标题，便于展示和识别。
func (s *Session) Title() string {
	for _, msg := range s.messages {
		if msg.Role == schema.User && msg.Content != "" {
			title := msg.Content
			if len([]rune(title)) > 40 {
				title = string([]rune(title)[:40]) + "..."
			}
			return title
		}
	}
	return "New Session"
}

// Store 负责管理会话文件和内存缓存。
type Store struct {
	dir   string
	cache map[string]*Session
}

func NewStore(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &Store{
		dir:   dir,
		cache: make(map[string]*Session),
	}, nil
}

// GetOrCreate 优先从缓存获取会话；如果磁盘不存在则创建，存在则加载。
func (s *Store) GetOrCreate(id string) (*Session, error) {
	if sess, ok := s.cache[id]; ok {
		return sess, nil
	}

	filePath := filepath.Join(s.dir, id+".jsonl")
	if _, err := os.Stat(filePath); err != nil {
		if os.IsNotExist(err) {
			sess, createErr := createSession(id, filePath)
			if createErr != nil {
				return nil, createErr
			}
			s.cache[id] = sess
			return sess, nil
		}
		return nil, err
	}

	sess, err := loadSession(filePath)
	if err != nil {
		return nil, err
	}

	s.cache[id] = sess
	return sess, nil
}

// sessionHeader 是 jsonl 文件的第一行，用来保存会话元信息。
type sessionHeader struct {
	Type      string    `json:"type"`
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
}

// createSession 创建一个新的会话文件，并写入头信息。
func createSession(id, filePath string) (*Session, error) {
	header := sessionHeader{
		Type:      "session",
		ID:        id,
		CreatedAt: time.Now().UTC(),
	}

	data, err := json.Marshal(header)
	if err != nil {
		return nil, err
	}

	if err := os.WriteFile(filePath, append(data, '\n'), 0o644); err != nil {
		return nil, err
	}

	return &Session{
		ID:        id,
		CreatedAt: header.CreatedAt,
		filePath:  filePath,
		messages:  make([]*schema.Message, 0),
	}, nil
}

// loadSession 从 jsonl 文件恢复会话。
// 第一行是头信息，后续每一行是一条消息。
func loadSession(filePath string) (*Session, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	if !scanner.Scan() {
		return nil, fmt.Errorf("empty session file: %s", filePath)
	}

	var header sessionHeader
	if err := json.Unmarshal(scanner.Bytes(), &header); err != nil {
		return nil, err
	}

	sess := &Session{
		ID:        header.ID,
		CreatedAt: header.CreatedAt,
		filePath:  filePath,
		messages:  make([]*schema.Message, 0),
	}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var msg schema.Message
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			// 单条消息损坏时跳过，避免整个会话加载失败。
			continue
		}
		sess.messages = append(sess.messages, &msg)
	}

	return sess, scanner.Err()
}

func main() {
	var sessionID string
	flag.StringVar(&sessionID, "session", "", "session ID")
	flag.Parse()

	ctx := context.Background()

	// 初始化 Qwen 大模型客户端。
	cm, err := qwen.NewChatModel(ctx, &qwen.ChatModelConfig{
		BaseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1",
		APIKey:  mustEnv("DASHSCOPE_API_KEY"),
		Model:   envOrDefault("QWEN_MODEL", "qwen3.5-flash"),
	})
	if err != nil {
		log.Fatalf("new qwen chat model failed: %v", err)
	}

	// 创建一个基于 ChatModel 的简单 Agent。
	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "MemoryDemoAgent",
		Description: "ChatModelAgent with persistent session.",
		Instruction: "你是一个简洁、专业的 Eino 学习助手。",
		Model:       cm,
	})
	if err != nil {
		log.Fatalf("new chat model agent failed: %v", err)
	}

	// Runner 负责执行 Agent，并开启流式输出。
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming: true,
	})

	// Store 负责管理会话持久化目录。
	store, err := NewStore(envOrDefault("SESSION_DIR", "./data/sessions"))
	if err != nil {
		log.Fatalf("new store failed: %v", err)
	}

	// 未传 session 参数时，新建一个会话；否则恢复旧会话。
	if sessionID == "" {
		sessionID = uuid.NewString()
		fmt.Printf("Created new session: %s\n", sessionID)
	} else {
		fmt.Printf("Resuming session: %s\n", sessionID)
	}

	session, err := store.GetOrCreate(sessionID)
	if err != nil {
		log.Fatalf("get or create session failed: %v", err)
	}

	fmt.Printf("Session title: %s\n", session.Title())
	fmt.Println("Enter your message (empty line to exit):")

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("you> ")
		if !scanner.Scan() {
			break
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			break
		}

		// 1. 记录用户输入
		userMsg := schema.UserMessage(line)
		if err := session.Append(userMsg); err != nil {
			log.Fatalf("append user message failed: %v", err)
		}

		// 2. 带上历史消息一起请求模型，实现“记忆”
		history := session.GetMessages()
		events := runner.Run(ctx, history)

		// 3. 一边打印流式输出，一边收集完整回复文本
		content, err := printAndCollectAssistant(events)
		if err != nil {
			log.Fatalf("run agent failed: %v", err)
		}

		// 4. 将助手回复也保存到会话中，便于下次恢复上下文
		assistantMsg := schema.AssistantMessage(content, nil)
		if err := session.Append(assistantMsg); err != nil {
			log.Fatalf("append assistant message failed: %v", err)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("\nSession saved: %s\n", sessionID)
	fmt.Printf("Resume with: go run . --session %s\n", sessionID)
}

// printAndCollectAssistant 处理 Runner 返回的事件流：
// - 流式输出时实时打印内容
// - 同时拼接成完整字符串，便于后续持久化
func printAndCollectAssistant(events *adk.AsyncIterator[*adk.AgentEvent]) (string, error) {
	var sb strings.Builder

	for {
		event, ok := events.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			return "", event.Err
		}
		if event.Output == nil || event.Output.MessageOutput == nil {
			continue
		}

		mv := event.Output.MessageOutput
		if mv.Role != schema.Assistant {
			continue
		}

		if mv.IsStreaming {
			// 流式场景：不断接收分片并实时打印
			mv.MessageStream.SetAutomaticClose()
			for {
				frame, err := mv.MessageStream.Recv()
				if errors.Is(err, io.EOF) {
					break
				}
				if err != nil {
					return "", err
				}
				if frame != nil && frame.Content != "" {
					sb.WriteString(frame.Content)
					fmt.Print(frame.Content)
				}
			}
			fmt.Println()
			continue
		}

		// 非流式场景：直接读取完整消息
		if mv.Message != nil {
			sb.WriteString(mv.Message.Content)
			fmt.Println(mv.Message.Content)
		}
	}

	return sb.String(), nil
}

// mustEnv 读取必填环境变量，缺失则直接退出。
func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("%s is empty", key)
	}
	return v
}

// envOrDefault 读取环境变量；如果为空则返回默认值。
func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
```

### 运行

第一次运行，创建新会话：

```bash
go run .
```

第二次运行，恢复之前的会话：

```bash
go run . --session <session-id>
```

你可以这样试：

```text
Created new session: 083d16da-6b13-4fe6-afb0-c45d8f490ce1
Session title: New Session
Enter your message (empty line to exit):
you> 你好，我是张三
你好，张三，很高兴认识你。
you> 我叫什么名字？
你叫张三。

Session saved: 083d16da-6b13-4fe6-afb0-c45d8f490ce1
Resume with: go run . --session 083d16da-6b13-4fe6-afb0-c45d8f490ce1
```

到这里，其实最核心的事情已经发生了：

- 用户消息被 `session.Append(userMsg)` 追加并写进磁盘
- 下一轮调用前，通过 `session.GetMessages()` 取出完整历史
- 模型返回的 assistant 消息也再次被 append 回会话

你只要把这个闭环看明白，这一章的主线就已经掌握了。

## 4. 一次用户输入，是怎么被保存和恢复的

为了避免你把上面的代码又看成一堆 API，我把它压成一条最关键的主线：

```text
┌──────────────────────────────┐
│ 用户输入一条消息              │
└──────────────────────────────┘
               ↓
┌──────────────────────────────┐
│ session.Append(user)         │
│ 先把用户消息持久化           │
└──────────────────────────────┘
               ↓
┌──────────────────────────────┐
│ session.GetMessages()        │
│ 拿到完整历史                 │
└──────────────────────────────┘
               ↓
┌──────────────────────────────┐
│ runner.Run(ctx, history)     │
│ 把历史交给 Agent 处理        │
└──────────────────────────────┘
               ↓
┌──────────────────────────────┐
│ 收集 assistant 回复          │
└──────────────────────────────┘
               ↓
┌──────────────────────────────┐
│ session.Append(assistant)    │
│ 再把助手回复持久化           │
└──────────────────────────────┘
```

这里最值得注意的是顺序。

### 第一步，先保存用户消息

很多人会下意识先调模型，拿到结果以后再一起存。

但从会话一致性的角度看，先把用户输入落盘更稳。
这样即便中间模型调用失败了，你也至少知道“用户这次问了什么”。

### 第二步，再取完整历史

`session.GetMessages()` 这一步，意义不是“凑个 slice 出来”。

它的含义是：

> 下一次模型调用，不再依赖某个临时变量，而是依赖会话当前的真实状态。

### 第三步，把完整历史交给 `runner.Run`

这里就能看出业务层和框架层的边界了。

- `Session` 不负责生成答案
- `Runner` 不负责存储消息

前者负责状态，后者负责执行。

这也是为什么我前面一直强调：

> Eino 负责处理消息，业务层负责保存和恢复消息。

## 5. 解读
我用到的jsonl这一个文件大概长这样：

```json
{"type":"session","id":"083d16da-6b13-4fe6-afb0-c45d8f490ce1","created_at":"2026-03-24T10:00:00Z"}
{"role":"user","content":"你好，我是张三"}
{"role":"assistant","content":"你好，张三，很高兴认识你。"}
{"role":"user","content":"我叫什么名字？"}
{"role":"assistant","content":"你叫张三。"}
```

这么设计，不是为了“文件格式优雅”。
而是因为：

- 首行 header 记录会话元信息
- 后续消息可以按行追加，无需每次重写整个文件
- 就算某一行损坏，也不至于把整份会话都拖死

这也是为什么 JSONL 这么适合拿来讲“持久化对话”。


### 第一，它没有额外基础设施门槛

你不用先装数据库，不用建表，不用配连接池。
读者只要能跑 Go 程序，就能立刻看到“会话确实被保存下来了”。

### 第二，它天然适合展示追加写

一条用户消息、一条 assistant 消息，本来就是很适合按行落盘的数据。

把这件事用 JSONL 展开，读者很容易理解：

> 原来所谓持久化会话，本质上就是把消息流变成可恢复的数据流。


## 7. 一分钟复盘

如果你读完这篇，一定要记住其中的三点。

第一句：

> 多轮对话，不等于持久化会话。

第二句：

> `Memory / Session / Store` 是业务层概念，不是 Eino 框架核心组件。

第三句：

> Eino 负责处理消息，业务层负责保存和恢复消息。

再把实现闭环压缩成一行，就是：

- 用户输入先 `Append`
- 取完整历史 `GetMessages`
- 交给 `runner.Run`
- 把 assistant 回复再 `Append` 回去

你把这条线真的理解了，后面无论是文件版、数据库版，还是更复杂的 interrupt / resume，其实都是在这个基础上继续扩展。

下一篇如果继续顺着这条线往下拆，我更想回头讲一讲 `Runner / AgentEvent`。
因为你会发现，一旦消息能被稳定保存下来，接下来真正值得深挖的，就是“Runner 到底怎么驱动整个 Agent 执行过程”。

## 参考资料

- Eino 第三章：[Memory 与 Session](https://www.cloudwego.io/zh/docs/eino/quick_start/chapter_03_memory_and_session/)（持久化对话）  