$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent $PSScriptRoot
$sourceRoot = Join-Path $repoRoot "source"
$docsRoot = Join-Path $repoRoot "docs"
$manifestPath = Join-Path $PSScriptRoot "article-manifest.json"

if (-not (Test-Path $sourceRoot)) {
    throw "source directory not found: $sourceRoot"
}

if (-not (Test-Path $manifestPath)) {
    throw "manifest not found: $manifestPath"
}

$manifest = Get-Content -LiteralPath $manifestPath -Raw -Encoding UTF8 | ConvertFrom-Json

if (Test-Path $docsRoot) {
    Remove-Item -LiteralPath $docsRoot -Recurse -Force
}

New-Item -ItemType Directory -Path $docsRoot | Out-Null

function Remove-LeadingNoise {
    param([string]$Text)

    $normalized = $Text -replace "`r`n", "`n"
    $lines = $normalized -split "`n"
    $start = 0

    while ($start -lt $lines.Count) {
        $line = $lines[$start].Trim()
        if ($line -eq "") {
            $start++
            continue
        }
        if ($line -like "声明：*") {
            $start++
            continue
        }
        if ($line -like "@[TOC]*") {
            $start++
            continue
        }
        break
    }

    return (($lines[$start..($lines.Count - 1)] -join "`n").Trim() + "`n")
}

function Normalize-Body {
    param([string]$Text)

    $body = Remove-LeadingNoise -Text $Text
    $body = [regex]::Replace($body, "(?m)^\s*@\[TOC\].*\r?\n?", "")
    $body = $body -replace "https://www\.clou dwego\.io", "https://www.cloudwego.io"
    $body = $body -replace "https://www\.cloudwego\.io/docs/eino/", "https://www.cloudwego.io/zh/docs/eino/"
    return $body.Trim() + "`n"
}

function Render-Bullets {
    param([object[]]$Items)

    return (($Items | ForEach-Object { "- $_" }) -join "`n")
}

foreach ($entry in $manifest) {
    $sourcePath = Join-Path $sourceRoot $entry.source
    if (-not (Test-Path $sourcePath)) {
        throw "source article not found: $sourcePath"
    }

    $destPath = Join-Path $docsRoot $entry.dest
    $destDir = Split-Path -Parent $destPath
    if (-not (Test-Path $destDir)) {
        New-Item -ItemType Directory -Path $destDir -Force | Out-Null
    }

    $body = Normalize-Body -Text (Get-Content -LiteralPath $sourcePath -Raw -Encoding UTF8)
    $selfLink = "./" + (Split-Path -Leaf $destPath)

    $header = (
        @(
            "# $($entry.title)",
            "",
            "> GitHub 主文：[当前文章]($selfLink)",
            "> CSDN 跳转：[$($entry.title)]($($entry.csdn_url))",
            "> 官方文档：[$($entry.official_label)]($($entry.official_url))",
            ">",
            "> 最新版以 GitHub 仓库为准，CSDN 作为分发入口，官方文档作为权威参考。",
            "",
            "**一句话摘要**：$($entry.summary)",
            "**适合谁看**：$($entry.audience)",
            "**前置知识**：$((($entry.prerequisites) -join '、'))",
            "**对应 Demo**：[$($entry.demo_label)]($($entry.demo_link))",
            "",
            "**面试可讲点**",
            (Render-Bullets -Items $entry.interview_points),
            "",
            "---",
            ""
        ) -join "`n"
    )

    $footer = (
        @(
            "",
            "---",
            "",
            "## 发布说明",
            "",
            "- GitHub 主文：[$($entry.title)]($selfLink)",
            "- CSDN 跳转：[$($entry.title)]($($entry.csdn_url))",
            "- 官方文档：[$($entry.official_label)]($($entry.official_url))",
            "- 最新版以 GitHub 仓库为准。"
        ) -join "`n"
    )

    $final = ($header + $body + $footer).Trim() + "`n"
    Set-Content -LiteralPath $destPath -Value $final -Encoding UTF8
}

$interviewDocPath = Join-Path $docsRoot "面试速览.md"
$interviewDoc = (
    @(
        '# Eino 面试速览',
        '',
        '这个仓库的文章不只是阅读材料，也可以直接拿来组织面试表达。最稳的讲法不是“我看过哪些 API”，而是“我怎么把 Eino 拆成四层能力”。',
        '',
        '## 四层表达法',
        '',
        '1. `模型调用层`：用 [`ChatModel 与 Message`](02-入门必学/01-ChatModel和Message.md) 解释模型能力如何被稳定接入。',
        '2. `组件协议层`：用 [`ChatTemplate`](03-组件核心/02-ChatTemplate为什么不是字符串拼接.md)、[`ToolsNode`](03-组件核心/04-为什么很多人会写Tool，却没真正看懂ToolsNode.md)、[`Retriever`](03-组件核心/07-为什么很多人会用Retriever，却没真正看懂Retrieve.md) 解释输入输出协议。',
        '3. `编排运行时层`：用 [`Chain / Graph`](04-编排进阶/01-一文讲透编排（Chain与Graph）.md) 和 [`Workflow`](04-编排进阶/02-既然有了Chain、Graph，为何还需要Workflow.md) 解释复杂链路如何建模。',
        '4. `Agent 抽象层`：用 [`什么是 Eino ADK`](05-ADK体系/01-什么是Eino ADK？.md) 和 [`为什么一定要有 Agent 抽象`](05-ADK体系/02-为什么一定要有Agent这层抽象.md) 解释为什么 Agent 不只是 Prompt 包装器。',
        '',
        '## 推荐回答框架',
        '',
        '- `我怎么开始学 Eino`：先跑 ChatModel，再补 Runner、Memory、Tool、Callback，最后才进入编排和 ADK。',
        '- `我怎么看 Agent`：Agent 不是大模型突然会做事，而是模型决策、工具能力、状态管理、运行时协议被系统性组织起来。',
        '- `我怎么看 RAG`：不要直接讲“向量库 + LLM”，先讲文档加载、解析、Embedding、Indexer、Retriever 的职责分层。',
        '- `我怎么看编排`：Chain 解决线性主链表达，Graph 解决关系显式化，Workflow 解决更细颗粒度的字段映射和控制流。',
        '- `我怎么看生产化`：先补 Callback/Trace、Interrupt/Resume、CheckPoint，再谈 Agent 真正进业务。',
        '',
        '## 高频追问点',
        '',
        '- 为什么 `Message` 不是字符串。',
        '- 为什么 `ChatModel` 要单独抽成组件。',
        '- `Tool` 和 `ToolsNode` 有什么区别。',
        '- `Chain` 和 `Graph` 到底怎么选。',
        '- `Runner`、`AgentEvent`、`AsyncIterator` 为什么不直接返回字符串。',
        '- `Interrupt/Resume` 解决的是暂停，还是治理。',
        '',
        '## 对应示例',
        '',
        '- 最小模型调用：[`examples/chatmodel-message`](../examples/chatmodel-message/README.md)',
        '- 会话持久化：[`examples/memory-session`](../examples/memory-session/README.md)',
        '- 文件系统工具调用：[`examples/tool-filesystem`](../examples/tool-filesystem/README.md)',
        '- 可观测性：[`examples/callback-trace`](../examples/callback-trace/README.md)',
        '- 最小编排：[`examples/chain-graph`](../examples/chain-graph/README.md)'
    ) -join "`n"
)
Set-Content -LiteralPath $interviewDocPath -Value ($interviewDoc + "`n") -Encoding UTF8

$repoSettingsDocPath = Join-Path $docsRoot "仓库发布设置.md"
$repoSettingsDoc = (
    @(
        '# 仓库发布设置',
        '',
        '这些内容不能直接靠本地文件自动设置到 GitHub，但已经在这里固定下来，创建远端仓库时照着填即可。',
        '',
        '## 基本信息',
        '',
        '- 仓库名：`go-eino-handbook`',
        '- 描述：`面向中文开发者的 CloudWeGo Eino 教程仓，含可运行 Go 示例和面试导向笔记。`',
        '- 主页：`https://blog.csdn.net/2302_80067378/category_13132166.html`',
        '- 默认分支：`main`',
        '- 许可证：`Apache-2.0`',
        '',
        '## Topics',
        '',
        '- `go`',
        '- `golang`',
        '- `eino`',
        '- `ai-agent`',
        '- `llm`',
        '- `rag`',
        '- `workflow`',
        '',
        '## Social Preview 文案',
        '',
        '- 标题：`Go Eino 中文学习手册`',
        '- 副标题：`GitHub 主文站 / CSDN 分发站 / 官方文档参考源`',
        '- 标语：`19 篇正文 + 1 篇总纲 + 5 个首批可运行 demo`',
        '',
        '## Release 节奏',
        '',
        '1. `v1.0-eino-learning-path`',
        '   - README',
        '   - 学习总纲',
        '   - 前置基础篇',
        '   - 入门必学 5 篇',
        '   - 3 个 demo',
        '2. `v1.1-core-components`',
        '   - 组件核心',
        '   - 编排进阶',
        '   - 补齐 `callback-trace` 与 `chain-graph`',
        '3. `v1.2-adk-and-interview`',
        '   - ADK 体系',
        '   - 面试速览',
        '   - 仓库首页文案和 social preview 最终版',
        '',
        '## Publish Checklist',
        '',
        '- 仓库描述和 topics 填完。',
        '- README 首页渲染正常，文章数量写成 `19 篇正文 + 1 篇总纲`。',
        '- `docs/` 与 `examples/` 的相对链接点检一轮。',
        '- CSDN 文内加上 GitHub 主文优先级说明。',
        '- 选 3 个最稳定 demo 先录屏或截图，作为仓库首发素材。'
    ) -join "`n"
)
Set-Content -LiteralPath $repoSettingsDocPath -Value ($repoSettingsDoc + "`n") -Encoding UTF8

