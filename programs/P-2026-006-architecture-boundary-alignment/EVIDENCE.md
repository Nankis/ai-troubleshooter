# EVIDENCE

## Evidence 索引

| Evidence ID | 类型 | 关联任务 | 关联验收标准 | 结论 |
| --- | --- | --- | --- | --- |
| EV-T1-001 | docs | Task 1 | Program 建立 | pass |
| EV-T2-001 | docs | Task 2 | 平台和业务边界修正 | pass |
| EV-T3-001 | diff | Task 3 | Go baseline 命名调整 | pass |
| EV-T4-001 | docs | Task 4 | README、ADR 和部署文档同步 | pass |
| EV-T5-001 | command | Task 5 | `git diff --check` 通过 | pass |
| EV-T5-002 | command | Task 5 | `make test` 通过 | pass |
| EV-T5-003 | command | Task 5 | `go vet ./...` 通过 | pass |
| EV-T5-004 | ci | Task 5 | GitHub Actions CI 通过 | pass |
| EV-T6-001 | docs | Task 6 | 防复发机制补齐 | pass |
| EV-T7-001 | docs | Task 7 | 验证结果规范补齐 | pass |
| EV-T7-002 | command | Task 7 | 本轮 docs-only diff 检查通过 | pass |

## 命令验证

| Evidence ID | 时间 | 命令 | 结果 | 备注 |
| --- | --- | --- | --- | --- |
| EV-T5-001 | 2026-05-23 | `git diff --check` | pass | 无 whitespace error。 |
| EV-T5-002 | 2026-05-23 | `make test` | pass | Go/Python 单测通过。 |
| EV-T5-003 | 2026-05-23 | `go vet ./...` | pass | Go vet 通过。 |
| EV-T5-004 | 2026-05-23 | GitHub Actions run `26299605632` | pass | `Clarify platform architecture boundaries` CI 通过。 |
| EV-T7-002 | 2026-05-23 | `git diff --check` | pass | 本轮新增验证规范和 Program 记录，无 whitespace error。 |

## 文档和代码证据

| Evidence ID | 时间 | 文件/范围 | 证据 | 结论 |
| --- | --- | --- | --- | --- |
| EV-T1-001 | 2026-05-23 | `programs/P-2026-006-architecture-boundary-alignment` | Program 文档已建立，包含 PROGRAM/STATUS/SCOPE/TASKS/EVIDENCE/DECISIONS/RISKS/HANDOFF/ERRORS/RESULT。 | pass |
| EV-T2-001 | 2026-05-23 | `README.md`、`docs/architecture-decisions.md` | README 架构图已把 `Platform Data`、`Knowledge`、`LLM / Vision Provider` 放入 `排障 Agent 平台`；业务侧只保留 `Readonly Business APIs`。 | pass |
| EV-T3-001 | 2026-05-23 | `cmd/*`、`internal/*` | `cmd/orchestrator` 改为 `cmd/baseline-orchestrator`；`internal/orchestrator` 改为 `internal/decisionbaseline`；`internal/worker` 依赖 `CaseProcessor` 接口。 | pass |
| EV-T4-001 | 2026-05-23 | `docs/*`、`api/openapi/decision-engine.yaml`、`apps/decision-engine/README.md` | ADR 记录 Python `apps/decision-engine` 是目标 Agent Orchestrator；部署检查清单说明 LLM/Vision 属于平台统一配置。 | pass |
| EV-T6-001 | 2026-05-23 | `AGENTS.md`、`programs/README.md`、`docs/LESSONS.md` | 参考 `game` 的入口规则、Program 说明和反复错误计数器，记录 `program-history-rewrite` 错误。 | pass |
| EV-T7-001 | 2026-05-23 | `docs/VERIFICATION.md`、当前 `EVIDENCE.md` / `RESULT.md` | 参考 `game` 的 Evidence/Result 写法，补齐索引、命令验证、覆盖映射、未验证项和已知噪音。 | pass |

## 覆盖映射

| 验收标准 | 对应任务 | Evidence ID | 状态 |
| --- | --- | --- | --- |
| README 架构图中平台数据和 LLM/Vision 位于 Agent 平台内部 | Task 2 | EV-T2-001 | pass |
| README 和 ADR 明确业务方只提供 readonly adapter | Task 2, Task 4 | EV-T2-001, EV-T4-001 | pass |
| 代码中不存在 `cmd/orchestrator` 和 `internal/orchestrator` 作为目标路径 | Task 3 | EV-T3-001 | pass |
| Go baseline 改为 `cmd/baseline-orchestrator` 和 `internal/decisionbaseline` | Task 3 | EV-T3-001 | pass |
| `make test`、`go vet ./...`、`git diff --check` 和 GitHub Actions CI 通过 | Task 5 | EV-T5-001..EV-T5-004 | pass |
| 仓库包含 `AGENTS.md`、`programs/README.md` 和 `docs/LESSONS.md` | Task 6 | EV-T6-001 | pass |
| 仓库包含 `docs/VERIFICATION.md`，并按新格式记录验证结果 | Task 7 | EV-T7-001, EV-T7-002 | pass |

## 未验证项

- 本轮 P-2026-006 后续补充为文档/流程规范，不涉及服务运行态，未重新启动本地 dev-server。
- Python decision-engine 接管 worker 的真实调用链尚未实现，后续应单独开 Program。

## 已知噪音

- GitHub Actions 出现 Node.js 20 deprecation warning，但 workflow 已设置 `FORCE_JAVASCRIPT_ACTIONS_TO_NODE24=true`，不影响本轮 CI 结论。
