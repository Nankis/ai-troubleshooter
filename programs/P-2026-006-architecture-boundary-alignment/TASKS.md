# TASKS

## Task 1: [x] 建立 Program

- Evidence：`EV-T1-PROGRAM`

## Task 2: [x] 修正平台和业务边界

- 文件：`README.md`、`docs/architecture-decisions.md`
- 验收：
  - 平台数据、知识库、LLM/Vision provider 属于 Agent 平台。
  - 业务方只提供 readonly APIs/adapters。

## Task 3: [x] 调整 Go baseline 命名和 worker 依赖

- 文件：`cmd/baseline-orchestrator/*`、`internal/decisionbaseline/*`、`internal/worker/*`
- 验收：
  - Go baseline 不再占用 `orchestrator` 目标命名。
  - worker 通过 `CaseProcessor` 接口调用 case 处理器。

## Task 4: [x] 同步文档

- 文件：`README.md`、`docs/*`、`api/openapi/decision-engine.yaml`、`apps/decision-engine/README.md`
- 验收：
  - 文档中不再要求业务方提供平台 MySQL 或 LLM provider。
  - 文档明确 Python decision-engine 是目标 Agent Orchestrator。

## Task 5: [x] 验证并推送

- Evidence：`EV-T5-FINAL`

## Task 6: [x] 补齐防复发机制

- 文件：`AGENTS.md`、`programs/README.md`、`docs/LESSONS.md`、当前 Program
- 验收：
  - 根入口规则明确“不要回写旧 Program”。
  - `docs/LESSONS.md` 记录 `program-history-rewrite` 错误和防复发规则。
  - `programs/README.md` 明确新需求、新架构调整、新错误复盘要新建 Program。

## Task 7: [x] 补齐验证结果规范

- 文件：`docs/VERIFICATION.md`、`AGENTS.md`、`programs/README.md`、当前 Program
- 验收：
  - 明确 Evidence 索引、命令验证、覆盖映射、未验证项和已知噪音。
  - 明确 Result 必须包含验证摘要、验收覆盖、commit 和残留风险。
  - 当前 Program 的 `EVIDENCE.md` / `RESULT.md` 按新格式补齐。
