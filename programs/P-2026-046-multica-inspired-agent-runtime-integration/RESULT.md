# RESULT

## 结果摘要

- 已完成 Multica-inspired Agent Run 生命周期升级。系统没有引入 Multica 外部依赖，而是在现有 Python Agent Platform / Python Decision Engine / Go Investigation Gateway 边界内，新增平台侧 runtime 注册、agent run、run event 和 Web/API 可观测轨迹。

## 变更范围

- 新增 `migrations/008_agent_runtime_runs.sql`，创建 `tb_troubleshoot_agent_runtime`、`tb_troubleshoot_agent_run`、`tb_troubleshoot_agent_run_event`。
- Python repository/service/server 新增 runtime 注册、心跳、列表 API，并让 case payload 返回 `agent_runs`。
- `process_case()` 主路径写入 supervisor lifecycle event，并把 Knowledge/specialist/Verifier 报告转为子 Agent Run。
- Agent Run 初始 payload 只保留必要 case 摘要，不复制用户原文或 OCR 原文。
- Web Chat 右侧进度面板新增 Agent Runs 展示。
- README、架构决策、本地运行手册和业务方从零接入手册补充 Agent Run、Local Runtime、Multica 借鉴边界和 API。

## 验证摘要

- `make test` 通过：Go 全量通过；Decision Engine 18 tests OK；Agent Platform 33 tests OK；根目录 4 tests OK。
- `make secret-scan` 通过。
- `git diff --check` 通过。
- `scripts/mysql-migrate.sh` 对本地 canonical schema `ai_troubleshooter` 幂等通过，001-008 全部 skip。
- 启动当前工作树的 Python Agent Platform，调用 `/api/v1/chat` 后返回 `case_20260525_000051 NEED_HUMAN_CONFIRMATION`，payload 包含 4 个 Agent Run；MySQL 查询确认该 case 有 4 个 run、12 个 event。

## Commit

- 提交消息：`P-2026-046 add agent run lifecycle`。

## 残留风险

- 本轮只实现平台契约和可观测生命周期，不实现完整 local runtime daemon。
- 本轮不自动修改业务代码。后续如接 Codex/Claude Code/Cursor runtime，默认应保持只读和 debug-only，并另起 Program 加授权和隔离。
- 冒烟验证使用 `local_rules`，用于证明生命周期落库，不代表真实模型排障质量验收。
