# Agent 决策层框架选择

## 结论

一期本地 MVP 继续使用轻量有限状态编排，不立即引入 LangGraph / LangChain 作为运行时强依赖。

目标形态仍然是 Python `apps/decision-engine` 承接 Agent Orchestrator。当前 Go `decisionbaseline` 只用于本地 smoke/fallback，并复用同一套限制：必要字段检查、平台经验优先、有限工具计划、Gateway 只读调用、决策日志和停止条件。

## 调研摘要

- LangGraph：适合长时程、有状态、可恢复的 agent workflow。官方定位是低层 orchestration framework/runtime，适合后续把排障流程表达为显式状态图。
- LangChain Agents：上手更快，适合通用 tool-calling agent；但一期我们工具边界、安全和审计规则已经固定，不需要先引入大依赖。
- Web Chat UI：开源项目很多，但大多引入 Next.js/Node/多 provider 管理。当前项目本地 Web Chat 先用纯 HTML/CSS/JS 内置页面，降低私有化和本地 smoke 成本。

参考：

- LangGraph docs: https://docs.langchain.com/oss/python/langgraph
- LangChain agents docs: https://docs.langchain.com/oss/python/langchain/agents

## 当前实现原则

- Agent loop 不无限循环：单 case 有总超时、工具数上限、工具失败上限。
- 工具不是模型自由决定后直连：只能调用 Gateway 已注册只读工具。
- 经验优先，但不盲信：高置信经验可直接返回，必须记录来源和原因；低置信或需实时状态时查 Gateway。
- 所有关键决策写入 `tb_troubleshoot_ai_decision_log`。
- 模型 provider 通过 OpenAI-compatible 接口接入，Qwen/DashScope 不需要 SDK。

## 后续迁移 LangGraph 的触发条件

满足以下任意条件，再新建 Program 迁移：

- 需要多轮工具调用、反证验证、重试/恢复等显式状态图。
- 需要持久化 checkpoint，支持 case 处理中断后恢复到某个节点。
- 需要把本地代码检查、日志分析、知识检索拆成可观测子图。
- 需要离线 eval 对比多个决策策略。

迁移时保留现有外部契约：Web Chat / Lark / Case API 不变，Gateway 工具契约不变，MySQL 表不变。
