# STANDARDIZATION

- 平台数据、知识库、AI 决策日志、工具审计和模型配置属于 Agent 平台边界。
- 业务方接入只要求 readonly business APIs/adapters，不要求提供平台 MySQL、LLM 或 Vision provider。
- Decision Engine / Agent Orchestrator 的目标实现归 Python 3.13；Go baseline 只能用于本地 smoke、fallback 或兼容验证。
- worker 应依赖 case processor 接口，而不是依赖具体 Go baseline 实现。
- 后续每个独立需求、新架构调整或较大修正都新增 Program；旧 Program 保留历史上下文，不为新命名或新边界反复回写。
- 每次用户指出 AI 自身流程错误，必须写入当前 Program 的 `ERRORS.md`，并同步沉淀到 `docs/LESSONS.md` 的反复错误计数器。
- 新任务启动时先读 `AGENTS.md` 和 `docs/LESSONS.md`；命中历史错误场景时，先按复盘规则设计方案，再改文件。
