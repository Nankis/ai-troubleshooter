# P-2026-006 Architecture Boundary Alignment

## 背景

用户审查一期部署架构图后指出两个边界问题：

- 平台数据、知识沉淀属于 Agent 平台自己的沉淀，不应该被画成业务方外部依赖，也不应该被表达成必须经过查询网关访问。
- LLM/Vision provider 应由 Agent 平台统一提供和配置，不应该要求业务方提供。
- Agent 编排属于决策层，目标实现应该放在 Python decision-engine；Go 侧只能保留本地 baseline/fallback。

上一轮提交 `463cb00` 已完成代码和文档调整。本 Program 作为该变更的执行记录补充。

## 目标

- 明确 Agent 平台边界：平台数据、知识库、LLM/Vision provider 都属于平台内部能力。
- 明确业务方边界：业务方只需要提供 readonly business APIs/adapters 和入口侧必要配置。
- 明确决策层归属：Python `apps/decision-engine` 是目标 Agent Orchestrator。
- 将 Go `orchestrator` 降级为 `decisionbaseline`，只用于本地 smoke/fallback。
- 让 worker 依赖 `CaseProcessor` 接口，为后续切 Python decision-engine 留出结构位置。
- 补齐 README、ADR、部署、安全、决策限制等文档中的边界表述。
- 借鉴 `game` 仓库的 Program/LESSONS 防踩坑机制，补齐本仓库的入口规则、Program 说明和错误复盘。
- 借鉴 `game` 仓库的验证结果写法，补齐 Evidence/Result 的标准模板。

## 非目标

- 不实现 Python decision-engine 接管 worker 的生产调用链。
- 不恢复或重写旧 Program 历史记录。
- 不引入新的向量数据库、队列或模型网关实现。

## 验收标准

- README 架构图中平台数据和 LLM/Vision 位于 Agent 平台内部。
- README 和 ADR 明确业务方只提供 readonly adapter。
- 代码中不存在 `cmd/orchestrator` 和 `internal/orchestrator` 作为目标路径。
- Go baseline 改为 `cmd/baseline-orchestrator` 和 `internal/decisionbaseline`。
- `make test`、`go vet ./...`、`git diff --check` 和 GitHub Actions CI 通过。
- 仓库包含 `AGENTS.md`、`programs/README.md` 和 `docs/LESSONS.md`，并记录本次 `program-history-rewrite` 错误。
- 仓库包含 `docs/VERIFICATION.md`，明确 Full 级 Program 的验证结果记录格式。
