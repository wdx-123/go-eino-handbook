声明：本文数据源于官方文档与官方实现。
@[TOC](一文读懂 Eino 的 ChatModel 和 Message)

很多人学 Eino，一上来就盯着 `Agent`、`Tool`、`Memory` 这些词。

虽然它们听起来更像“真正的 AI 应用开发”。
但 `ChatModel` 和 `Message` 这两个才是最基础的边界。
它的之所以重要，是因为他将与你之后的所有AI开发形影不离。

## 1. 初入 Eino

很多 Go 后端工程师第一次接触 Eino，路径大概都是这样的：

- 先搜“Go 怎么写 Agent”
- 然后看到一堆工作流、工具调用、多轮会话
- 最后一头扎进了复杂编排

你以为自己在学“AI 应用开发”，其实常常只是在拼装更大的调用链。只要底层那次最简单的模型请求没理解，后面的抽象层级越多，人就越容易飘。

说白了， ChatModel 和 Message 在整个Eino框架中，不是拿来炫功能的。

它是在回答一个更底层的问题：

> 在 Eino 里，一次最基础的大模型对话，到底是怎么被表达、组织和执行的？

这个问题不解决，后面所有“高级能力”都会变成悬空建筑。

## 2. ChatModel 与 Message 到底是什么

你可以这样理解
> 用最小的 Go 代码，把一次用户输入变成一次模型调用<small>（ChatModel）</small>，并且`Message` 不是普通字符串，而是对话协议。

注意，这里有两个重点。

**第一，`ChatModel`。**

它解决的是“怎么和模型说话”。

**第二，`Message`。**

它解决的是“你说的话，怎么被组织成模型能理解的上下文”。


## 3. `ChatModel` 对我们而言到底有什么价值

如果你只把 `ChatModel` 理解成“调一下模型接口”，那就低估它了。

通常来说，它真正的价值至少有三层。

**第一层，统一模型调用边界。**

你今天接 OpenAI，明天接 Ark，后天换千问。业务最怕的不是换模型，而是换模型就要改一大片调用代码。

`ChatModel` 的意义，就是先把“和模型交互”这件事抽象成一个稳定接口。你上层的业务逻辑关心的是“输入一组消息，拿回模型响应”，而不是每家厂商的参数细节。

**第二层，给后续编排留接口。**

后面你看到的 `Agent`、`Runner`、`Graph`、`Chain`，本质上都不是凭空长出来的。它们之所以能组合，是因为底下先有一层统一的 `Component` 抽象。

如果没有 `ChatModel` 这种边界，后面的编排层就会变成一堆和供应商 SDK 强耦合的胶水代码。

**第三层，让测试可以做的更自然。**

接口一旦稳定，mock 就有了位置。你做单测时，不必每次真打外部模型。

所以 `ChatModel` 的价值，不只是“能调模型”，而是：

> 它把模型能力从“某个厂商的 HTTP 调用”提升成了“业务里可替换、可编排、可测试的一类能力”。

## 4. `Message` 的作用是什么

大多人会下意识觉得：

“不就是传一段 prompt<small>（提示词）</small> 给模型吗？”

真要这么理解，问题就来了。那系统指令放哪？用户问题放哪？模型回复怎么回灌进上下文？后面工具调用结果又怎么拼回对话链路？

这就是 `Message` 存在的原因。

在 Eino 里，一次对话不是一段字符串，而是一组有角色的消息。

- `system`：系统指令
- `user`：用户输入
- `assistant`：模型回复
- `tool`：工具返回结果

它本质上是在表达一种“对话协议”，而不是一段散装文本。

你只要把 `Message` 理解成协议，就会自然明白下面这些事：

- 为什么系统指令通常放在最前面
- 为什么多轮对话不是简单字符串拼接
- 为什么后面接 Tool Calling 时，`tool` 角色必须单独存在

所以接触 Message 的时候，真正要你建立的，不只是 API 用法，而是一个认知转换：

> 在 Eino 里组织上下文，操作的核心单位不是 prompt 字符串，而是 `schema.Message`。

## 5. 用千问把第一轮 Eino 对话跑通
注：<small>我之所以选择用千问，而非OpenAI，是因为他送的有免费额度，适合学习的时候用</small>



先准备依赖和环境变量：

```bash
go mod init eino-ch01-demo
go get github.com/cloudwego/eino@latest
go get github.com/cloudwego/eino-ext/components/model/qwen@latest

export DASHSCOPE_API_KEY="你的百炼 API Key"
export QWEN_MODEL="qwen3.5-flash"
```

如果你在 Windows PowerShell 下，环境变量改成 `$env:DASHSCOPE_API_KEY="..."` 和 `$env:QWEN_MODEL="qwen-flash"` 就行。

然后把下面这份完整代码保存成 `main.go`：

```go
package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/cloudwego/eino-ext/components/model/qwen"
	"github.com/cloudwego/eino/schema"
)

func main() {
	ctx := context.Background()

	query := "用一句话解释 Eino 的 Component 设计解决了什么问题？"
	if len(os.Args) > 1 {
		query = strings.Join(os.Args[1:], " ")
	}

	cm, err := qwen.NewChatModel(ctx, &qwen.ChatModelConfig{
		BaseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1",
		APIKey:  mustEnv("DASHSCOPE_API_KEY"),
		Model:   envOrDefault("QWEN_MODEL", "qwen3.5-flash"),
	})
	if err != nil {
		log.Fatalf("new qwen chat model failed: %v", err)
	}

	messages := []*schema.Message{
		schema.SystemMessage("你是一个简洁、专业的 Go AI 框架助手。"),
		schema.UserMessage(query),
	}

	stream, err := cm.Stream(ctx, messages)
	if err != nil {
		log.Fatalf("stream chat failed: %v", err)
	}
	defer stream.Close()

	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("recv stream failed: %v", err)
		}

		fmt.Print(chunk.Content)
	}

	fmt.Println()
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("%s is empty", key)
	}
	return v
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
```

直接执行 `go run . -- "用一句话解释 Eino 的 Component 设计解决了什么问题？"`，你就能看到千问按流式把回复一段段打出来。

## 6. 这段代码执行时到底发生了什么

按执行顺序，它其实做了四件事。

**第一，初始化 `ChatModel`。**

`qwen.NewChatModel(...)` 这一层的意义，不只是创建一个客户端对象，而是把“千问模型能力”接成 Eino 能识别的 `ChatModel` 组件。

从这一步开始，你的代码面对的就是 Eino 抽象，而不再是散落的 HTTP 参数。

**第二，构造 `messages`。**

这里不是简单传一个字符串，而是明确传入两条消息：

- 一条 `system`，告诉模型你希望它以什么方式回答
- 一条 `user`，表达当前用户问题

这就是第一章最关键的认知点之一。对话不是一段文本，而是一组有角色的消息。

**第三，调用 `Stream`。**

这一点也很重要。这个代表流式生成的意思。

**第四，逐块读取并打印。**

`chunk.Content` 看起来只是一个字段，但它意味着模型回复不必等到全部生成完再展示。前端可以边收边显示，后端也可以边收边处理。后面你做可观测、会话管理、回调链路时，这种流式思维会很自然。

所以本demo的意义，不只是“打印了一行字”，而是搭建起来了一个 Eino 的最小对话闭环。

## 7. 一分钟复盘

如果你读完这篇，只记住一句话，我希望是这一句：

> Eino 本章不只是在教你写一个最简单的聊天 Demo，而是在教你建立“模型调用抽象”和“对话消息协议”这两个最基础的认知。

再压缩一点，就是三件事：

- `ChatModel` 解决的是模型能力的统一接入
- `Message` 解决的是对话上下文的结构化表达
- `Stream` 让这次调用更接近真实产品里的交互方式（流式输出）

本篇是为了之后的多轮对话、Runner、AgentEvent 打下坚实的根基。

## 参考资料

- Eino 第一章：ChatModel 与 Message（Console）  
  https://www.cloudwego.io/zh/docs/eino/quick_start/chapter_01_chatmodel_and_message/
- Eino ChatModel 使用说明  
  https://www.cloudwego.io/zh/docs/eino/core_modules/components/chat_model_guide/
- Eino Qwen 组件文档  
  https://pkg.go.dev/github.com/cloudwego/eino-ext/components/model/qwen
- Eino Qwen 免费额度页面
- https://bailian.console.aliyun.com/cn-beijing/?tab=model#/model-usage/free-quota
