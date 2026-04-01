前段时间，由于回家过年，躺在床上实在感觉无聊，
所以就在网上搜罗了相关资料，整理了学习内容，方便以后温故。

进来各种模型频繁迭代，好像光是闻着claude、gpt、deepseek、豆包这些模型升级的声音，就已经让我们热血澎湃。
但你真的了解他们吗？你知道如何用好他们吗？
**如：**
* user prompt
* system prompt
* AI Agent
* function calling
* MCP
* RAG
* 上下文窗口

可能你零星的知道些皮毛，不过没关系，现在让我带着你深入学习一番。

---
@[TOC](大纲)

# 一、什么是所谓的user prompt

最早的 GPT，其实只是个“高级点的聊天机器人”。

你给它一句话（user prompt），它给你一句话回答。
![在这里插入图片描述](https://i-blog.csdnimg.cn/direct/dac435477b834556a3a89d5be410b66a.png)
它能聊天、能写文章、能解释代码

但它**不能真的帮你做事**。

比如你说：

> 帮我把 C 盘的 hello_world.cpp 移动到 D 盘，并总结内容

它最多告诉你“应该怎么做”，但不会真的帮你操作文件。

于是问题来了：

> 能不能让 AI 真正去执行任务？

这就引出了 —— **AI Agent**。

---

# 二、user prompt 和 system prompt

在讲 Agent 之前，我们先把基础打牢。

## 1、 user prompt（用户提示词）

就是你在对话框里输入的内容。

例如：

```text
你好
```

早期 GPT 只有 user prompt。

模型没有人格设定、没有角色设定，只是普通问答。

---

## 2、 system prompt（系统提示词）

后来人们发现，可以给模型“设定人设”。

比如：

```text
你是一个傲娇的程序员，说话尽量傲娇，最好带 emoji。
```

这个提示不让用户看到，但每次请求都会和 user prompt 一起发给模型。

于是模型有了：

* 性格
* 风格
* 行为约束

本质上：

> user prompt = 你说的话
> system prompt = 模型的隐藏设定

---

# 三、AI Agent 是怎么让 AI 干活的？

现在进入核心。

## 1、AI 的问题

AI 本身：

* 只能输出文本
* 不能操作系统
* 不能读文件
* 不能访问数据库

所以它只能“动脑”，不能“动手”。

---

## 2、Agent 的出现

AI Agent 本质上就是一段程序。

它的作用是：

> 在 用户、AI、工具 之间做协调。

你可以理解为：

| 角色    | 职责    |
| ----- | ----- |
| AI    | 思考和决策 |
| Agent | 协调和调度 |
| Tool  | 实际执行  |

---

## 3、举个完整流程例子

用户说：

> 读取 C 盘 hello_world.cpp，移动到 D 盘，并总结内容

流程是这样的：

**第一步：Agent 告诉 AI 可以用哪些工具**

例如：

* read_file
* move_file

**第二步：AI 决定调用 read_file**

```text
调用 read_file，路径：C://hello_world.cpp
```

**第三步：Agent 真正执行工具**

* 读取文件
* 把内容返回给 AI

**第四步：AI 决定调用 move_file**

**第五步：Agent 执行移动**

**第六步：AI 输出总结**

**第七步：Agent 返回结果给用户**

这就是一个完整的循环。

> 规划 → 执行 → 反馈 → 再规划 → 交付

---

# 四、Function Calling：工具调用的标准化革命

早期 Agent 有个问题：

AI 是“猜”怎么调用工具的。

比如天气查询工具：

```text
check_weather(city, date)
```

AI 可能会写：

```text
上海 明天
```

问题来了：
* 参数顺序错了？
* 明天不是标准日期？
* 少传字段？

于是就出现了 **Function Calling**。


**Function Calling**： 把工具描述从 system prompt中剥离，用JSON格式统一定义函数名、函数介绍、参数字段，并规范AI调用工具的回复格式。这就是Function Calling的核心： 用标准化格式让AI理解怎么调用工具，而不是猜。

---

## 1、工具定义（标准 JSON）

```json
{
  "name": "check_weather",
  "parameters": {
    "type": "object",
    "properties": {
      "city": {
        "type": "string"
      },
      "date": {
        "type": "string",
        "format": "YYYY-MM-DD"
      }
    },
    "required": ["city"]
  }
}
```

---

## 2、AI 必须按格式调用

```json
{
  "function_call": {
    "name": "check_weather",
    "parameters": {
      "city": "上海",
      "date": "2025-11-14"
    }
  }
}
```

## 3、Function Calling的好处：

1. **告别猜谜语**：以前靠System Prompt用自然语言描述工具，AI可能听不懂；现在用JSON格式，AI一看就会。
2. **降低开发难度**：开发者不用自己写代码检测AI回复是否正确，若AI回复错误，AI的服务器端可检测并自动重试，降低用户开发难度和token开销。
3. **跨场景通用**：无论是ChatGPT还是开源模型，只要支持**Function Calling**，就能用同一套工具。

| 对比项  | System Prompt（传统方式） | Function Calling（标准化方式）               |
| ---- | ------------------- | ------------------------------------- |
| 工具描述 | 自然语言随意写（如你可以用查天气工具） | JSON格式强制规范（必须包含name/parameters）       |
| 调用格式 | 等AI猜（可能返回散文式回复）     | 固定JSON结构（如 {`"function_name":"..."`}） |
| 错误处理 | 开发者自己写代码重试          | 大模型服务端自动重试                            |

---
# 五、MCP：AI 世界的 USB-C

上文提到的Agent和Tool是怎么进行交互的？最简单的做法就是把Agent和Tool写在同一个程序里面，直接通过函数调用来完成，这也是现在大多数agent的做法。

但其实有些tool的功能其实挺**通用**的，可能多个agent都需要，但总不能在每个agent里面都拷贝一份相同的代码吧。

---

我们把tool变成服务，统一的托管，让所有的agent都来调用，这就是 **mcp server**。mcp是一个通信协议，专门用来规范agent和tool服务之间是怎么交互的。运行tool的服务叫做mcp server，调用它的agent叫做mcp client。mcp规定了mcp server如何和mcp client通信，以及mcp server有哪些接口。

mcp server既可以和agent跑在同一台机器上，通过标准输入输出进行通信。也可以被部署在网络上，通过http进行通信。虽然mcp是为了通用定制出来的标准，但实际上mcp本身却和ai模型没有关系，他并不关心agent用的是哪个模型，mcp只负责帮agent托管工具、资源。

---

你可以把 MCP 想象成电脑的 **USB-C 接口**：

* 各种外设（如键盘、U盘、显示器）就是不同的 **MCP Server**，它们提供各自独特的功能。
* 电脑就是 **AI Agent**，它作为 **MCP Client**，通过统一的 **USB-C 接口（即 MCP 协议）** 来连接和使用所有外设（MCP Server）。
* 这样一来，无论你更换电脑还是外设，只要都支持 USB-C 标准，就能即插即用，非常方便。MCP 协议正是为 AI 世界带来了这种即插即用的便利性。


如果说 Function Calling 解决的是：

> “怎么调用工具”

那么 MCP 解决的是：

> “工具怎么统一接入”

---

## 1、所以说，什么是 MCP？

MCP = Model Control Protocol
它把工具变成一个服务（MCP Server）。
Agent 不再直接调用工具，而是通过 MCP 协议访问。

---

## 2、完整流程示例

用户问：

> 女朋友肚子疼怎么办？

流程：

1. Agent 通过 MCP 获取可用工具（如网页搜索）
2. 转换为 Function Calling 格式
3. AI 选择 web_browse
4. Agent 通过 MCP 调用搜索服务
5. 返回结果
6. AI 生成建议

---

# 六、大模型的上下文窗口

很多人忽略这个概念，但它非常关键。

## 什么是上下文窗口？

就是：

> 模型一次对话能记住多少内容

你可以把它想象成一块黑板。

* 黑板大：能写很多
* 黑板小：写几行就满

当写满时：

> 模型会“擦掉最前面的内容”

这就是为什么：

* 对话太长会“失忆”
* 输入太大成本会上升
* 回答会变慢

---

# 七、RAG：检索增强生成
（这个等后面，会单独在写一篇博客细讲）

最后讲一个企业级必备技术 —— RAG。

RAG = Retrieval-Augmented Generation

简单说就是：

> 先查资料，再生成回答。

---

## 为什么不直接把资料丢给模型？

问题：

* 有上下文窗口限制
* 推理成本高
* 输入越大越慢
* 容易幻觉

---

## RAG 怎么做？

1. 用户提问
2. 向知识库检索相关片段
3. 只把相关内容发给模型
4. 模型基于检索结果回答

比如：

用户问产品维修政策。

RAG 不会发 200 页手册。

而是：

* 精准找 3 段相关内容
* 送给模型
* 生成答案

优点：

* 成本低
* 更准确
* 更快
* 可扩展

---

# 八、整套体系串起来是什么样？

我们把今天讲的全部串起来：

```
用户
  ↓
Agent
  ↓（MCP 获取工具）
工具列表
  ↓（Function Calling 格式）
AI 模型
  ↓
调用工具
  ↓
RAG 检索知识
  ↓
生成答案
  ↓
返回用户
```

---

# 九、最终总结

| 名词               | 作用      |
| ---------------- | ------- |
| user prompt      | 用户输入    |
| system prompt    | 模型隐藏设定  |
| Agent            | 调度协调    |
| Tool             | 实际执行    |
| Function Calling | 标准化工具调用 |
| MCP              | 工具接入协议  |
| 上下文窗口            | 模型记忆容量  |
| RAG              | 检索增强生成  |

---

# 结语

AI 正在从“会聊天”进化为“能做事”。

这背后不是一个技术，而是一整套体系：

- Agent 负责调度
- Function Calling 负责规范调用
- MCP 负责统一接入
- RAG 负责精准知识增强

理解了这些，你基本就理解了当前 AI 应用的核心架构。


