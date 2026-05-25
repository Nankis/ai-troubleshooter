# P-2026-046 Multica Inspired Agent Runtime Integration

## 背景

用户要求根据 Multica 的 managed agents 思路升级本项目。我们不照搬 Multica 的项目管理系统，而是借鉴其 Runtime / Daemon、Task lifecycle、Run messages、Skills 和上下文分页思想，用于增强 ai-troubleshooter 的排障 Agent 可观测性和后续本地 coding agent 接入能力。

## 目标

- 在平台侧引入 Agent Run 生命周期，记录 specialist / local runtime 的 enqueue、start、event、complete、fail。
- 预留 Local Runtime 注册与心跳抽象，后续可接 Codex / Claude Code / Cursor Agent 做本地代码辅助排查。
- Web / API 可查看某个 case 的 agent runs 和 run events。
- Decision Engine 主路径写入 agent run 事件，避免排查过程只剩最终答案。
- 文档说明和 Multica 的映射关系、边界和后续接入计划。

## 非目标

- 本轮不直接接入 Multica 作为外部依赖。
- 本轮不让 agent 自动改业务代码。
- 本轮不实现完整本地 daemon 二进制，只做平台契约和本机可扩展抽象。
- 本轮不改变 Go Gateway 职责。
