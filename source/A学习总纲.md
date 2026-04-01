基于当前 Eino 官方中文文档结构，我更推荐按下面这条主线学习。这个顺序是我整理出来的“推荐学习路径”，不是对官方 Quick Start 的原样照搬；它更适合先建立从 `ChatModel` 到 `Agent / Runner` 的执行骨架，再补状态、工具和可观测性。

截至 `2026-03-25`，官方 Quick Start 的完整主线还包括 `第五章：Middleware`、`第七章：Interrupt/Resume`、`第八章：Graph Tool`、`第九章：Skill`、`第十章：A2UI`。我把其中一部分放到了后续阶段或按需选修里，是为了让主线更顺，不代表这些章节不重要。

总入口：

- [Eino 快速开始总入口](https://www.cloudwego.io/zh/docs/eino/quick_start/)

## 第一阶段：入门必学

这一组先建立从 `ChatModel` 到 `Agent / Runner` 的执行骨架，再补 `Memory`、`Tool`、`Callback / Trace`。如果你只想先抓住主线，先把这五篇跑通。

1. [第一章：ChatModel 与 Message（Console）](https://www.cloudwego.io/zh/docs/eino/quick_start/chapter_01_chatmodel_and_message/)
2. [第二章：ChatModelAgent、Runner、AgentEvent（Console 多轮）](https://www.cloudwego.io/zh/docs/eino/quick_start/chapter_02_chatmodelagent_runner_agentevent/)
3. [第三章：Memory 与 Session（持久化对话）](https://www.cloudwego.io/zh/docs/eino/quick_start/chapter_03_memory_and_session/)
4. [第四章：Tool 与文件系统访问](https://www.cloudwego.io/zh/docs/eino/quick_start/chapter_04_tool_and_filesystem/)
5. [第五章：Callback 与 Trace（可观测性）](https://www.cloudwego.io/zh/docs/eino/quick_start/chapter_06_callback_and_trace/)

## 第二阶段：组件核心

这一组在“核心模块 -> Components 组件”下面，适合你补齐 `Chat`、`Tool`、`RAG` 这三条能力线。

1. [ChatModel 使用说明](https://www.cloudwego.io/zh/docs/eino/core_modules/components/chat_model_guide/)
2. [ChatTemplate 使用说明](https://www.cloudwego.io/zh/docs/eino/core_modules/components/chat_template_guide/)
3. [ToolsNode&Tool 使用说明](https://www.cloudwego.io/zh/docs/eino/core_modules/components/tools_node_guide/)
4. [Document Loader 使用说明](https://www.cloudwego.io/zh/docs/eino/core_modules/components/document_loader_guide/)
5. [Embedding 使用说明](https://www.cloudwego.io/zh/docs/eino/core_modules/components/embedding_guide/)
6. [Indexer 使用说明](https://www.cloudwego.io/zh/docs/eino/core_modules/components/indexer_guide/)
7. [Retriever 使用说明](https://www.cloudwego.io/zh/docs/eino/core_modules/components/retriever_guide/)


## 第三阶段：编排进阶

这一组在“Chain & Graph & Workflow 编排功能”下面，适合在前两阶段跑顺以后继续往运行时和复杂流程走。

1. [Chain/Graph 编排介绍](https://www.cloudwego.io/zh/docs/eino/core_modules/chain_and_graph_orchestration/chain_graph_introduction/)
2. [Workflow 编排框架](https://www.cloudwego.io/zh/docs/eino/core_modules/chain_and_graph_orchestration/workflow_orchestration_framework/)
3. [Callback 用户手册](https://www.cloudwego.io/zh/docs/eino/core_modules/chain_and_graph_orchestration/callback_manual/)
4. [Interrupt & CheckPoint 使用手册](https://www.cloudwego.io/zh/docs/eino/core_modules/chain_and_graph_orchestration/checkpoint_interrupt/)
5. [CallOption 能力与规范](https://www.cloudwego.io/zh/docs/eino/core_modules/chain_and_graph_orchestration/call_option_capabilities/)

## 第四阶段：ADK 体系

这一组在“ADK - Agent Development Kit”下面，适合在理解组件和编排之后系统化学习 Agent。

1. [ADK Quickstart](https://www.cloudwego.io/zh/docs/eino/core_modules/eino_adk/agent_quickstart/)
2. [Agent 抽象](https://www.cloudwego.io/zh/docs/eino/core_modules/eino_adk/agent_interface/)
3. [Agent 协作](https://www.cloudwego.io/zh/docs/eino/core_modules/eino_adk/agent_collaboration/)
4. [ChatModelAgent](https://www.cloudwego.io/zh/docs/eino/core_modules/eino_adk/agent_implementation/chat_model/)
5. [Workflow Agents](https://www.cloudwego.io/zh/docs/eino/core_modules/eino_adk/agent_implementation/workflow/)
6. [Agent Runner 与扩展](https://www.cloudwego.io/zh/docs/eino/core_modules/eino_adk/agent_extension/)

== 第一版本，先学习到这里。
7. [Agent Callback](https://www.cloudwego.io/zh/docs/eino/core_modules/eino_adk/adk_agent_callback/)
8. [Supervisor Agent](https://www.cloudwego.io/zh/docs/eino/core_modules/eino_adk/agent_implementation/supervisor/)
9. [Plan-Execute Agent](https://www.cloudwego.io/zh/docs/eino/core_modules/eino_adk/agent_implementation/plan_execute/)

可选入口：

- [ADK 概述](https://www.cloudwego.io/zh/docs/eino/core_modules/eino_adk/agent_preview/)

## 第五阶段：按需选修

这一组偏产品化、工具化和生态集成，也包含了官方 Quick Start 里我没有放进主线的章节。

1. [第五章：Middleware（中间件模式）](https://www.cloudwego.io/zh/docs/eino/quick_start/chapter_05_middleware/)
2. [第七章：Interrupt/Resume（中断与恢复）](https://www.cloudwego.io/zh/docs/eino/quick_start/chapter_07_interrupt_resume/)
3. [第八章：Graph Tool（复杂工作流）](https://www.cloudwego.io/zh/docs/eino/quick_start/chapter_08_graph_tool/)
4. [第九章：Skill（Console）](https://www.cloudwego.io/zh/docs/eino/quick_start/chapter_09_skill_console/)
5. [第十章：A2UI 协议（流式 UI 组件）](https://www.cloudwego.io/zh/docs/eino/quick_start/chapter_09_a2ui_protocol/)
6. [应用开发工具链（总入口）](https://www.cloudwego.io/zh/docs/eino/core_modules/devops/)
7. [Eino Dev 插件安装指南](https://www.cloudwego.io/zh/docs/eino/core_modules/devops/ide_plugin_guide/)
8. [Eino Dev 可视化编排插件功能指南](https://www.cloudwego.io/zh/docs/eino/core_modules/devops/visual_orchestration_plugin_guide/)
9. [Eino Dev 可视化调试插件功能指南](https://www.cloudwego.io/zh/docs/eino/core_modules/devops/visual_debug_plugin_guide/)
10. [ReAct Agent 使用手册](https://www.cloudwego.io/zh/docs/eino/core_modules/flow_integration_components/react_agent_manual/)
11. [Host Multi-Agent](https://www.cloudwego.io/zh/docs/eino/core_modules/flow_integration_components/multi_agent_hosting/)
12. [组件集成（总入口）](https://www.cloudwego.io/zh/docs/eino/ecosystem_integration/)
13. [发布记录 & 迁移指引（总入口）](https://www.cloudwego.io/zh/docs/eino/release_notes_and_migration/)

## 你最适合的实际阅读顺序

如果你不想完全照目录走，我更建议按下面这条最短主线读：

1. [ChatModel 与 Message](https://www.cloudwego.io/zh/docs/eino/quick_start/chapter_01_chatmodel_and_message/)
2. [ChatModelAgent、Runner、AgentEvent](https://www.cloudwego.io/zh/docs/eino/quick_start/chapter_02_chatmodelagent_runner_agentevent/)
3. [Memory 与 Session](https://www.cloudwego.io/zh/docs/eino/quick_start/chapter_03_memory_and_session/)
4. [Tool 与文件系统访问](https://www.cloudwego.io/zh/docs/eino/quick_start/chapter_04_tool_and_filesystem/)
5. [Callback 与 Trace](https://www.cloudwego.io/zh/docs/eino/quick_start/chapter_06_callback_and_trace/)
6. [ChatTemplate 使用说明](https://www.cloudwego.io/zh/docs/eino/core_modules/components/chat_template_guide/)
7. [ToolsNode&Tool 使用说明](https://www.cloudwego.io/zh/docs/eino/core_modules/components/tools_node_guide/)
8. [Document Loader 使用说明](https://www.cloudwego.io/zh/docs/eino/core_modules/components/document_loader_guide/)
9. [Embedding 使用说明](https://www.cloudwego.io/zh/docs/eino/core_modules/components/embedding_guide/)
10. [Indexer 使用说明](https://www.cloudwego.io/zh/docs/eino/core_modules/components/indexer_guide/)
11. [Retriever 使用说明](https://www.cloudwego.io/zh/docs/eino/core_modules/components/retriever_guide/)
12. [Workflow 编排框架](https://www.cloudwego.io/zh/docs/eino/core_modules/chain_and_graph_orchestration/workflow_orchestration_framework/)
13. [Interrupt & CheckPoint 使用手册](https://www.cloudwego.io/zh/docs/eino/core_modules/chain_and_graph_orchestration/checkpoint_interrupt/)
14. [ADK Quickstart](https://www.clou dwego.io/zh/docs/eino/core_modules/eino_adk/agent_quickstart/)
15. [ChatModelAgent](https://www.cloudwego.io/zh/docs/eino/core_modules/eino_adk/agent_implementation/chat_model/)
16. [Workflow Agents](https://www.cloudwego.io/zh/docs/eino/core_modules/eino_adk/agent_implementation/workflow/)

如果你想把官方 Quick Start 也完整补齐，再接着看：

- [第五章：Middleware（中间件模式）](https://www.cloudwego.io/zh/docs/eino/quick_start/chapter_05_middleware/)
- [第七章：Interrupt/Resume（中断与恢复）](https://www.cloudwego.io/zh/docs/eino/quick_start/chapter_07_interrupt_resume/)
- [第八章：Graph Tool（复杂工作流）](https://www.cloudwego.io/zh/docs/eino/quick_start/chapter_08_graph_tool/)
- [第九章：Skill（Console）](https://www.cloudwego.io/zh/docs/eino/quick_start/chapter_09_skill_console/)
- [第十章：A2UI 协议（流式 UI 组件）](https://www.cloudwego.io/zh/docs/eino/quick_start/chapter_09_a2ui_protocol/)




// 第一步学习途径：
adk体系：
1. [ADK Quickstart](https://www.cloudwego.io/zh/docs/eino/core_modules/eino_adk/agent_quickstart/)


额外学习：
- [第五章：Middleware（中间件模式）](https://www.cloudwego.io/zh/docs/eino/quick_start/chapter_05_middleware/)
- [第九章：Skill（Console）](https://www.cloudwego.io/zh/docs/eino/quick_start/chapter_09_skill_console/)
- [ReAct Agent 使用手册](https://www.cloudwego.io/zh/docs/eino/core_modules/flow_integration_components/react_agent_manual/)