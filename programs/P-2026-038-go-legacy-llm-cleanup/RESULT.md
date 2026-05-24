# RESULT

# RESULT

## 结果摘要

- Go 侧 legacy LLM、Vision、Web Chat、Lark bot、worker、baseline decision、caseflow/evolution/queue/chatplatform 代码已删除。
- Go `internal/config` 不再解析或暴露 LLM/Vision/Lark bot/Queue/ToolGateway 配置，只保留 Gateway 需要的 server、database、connectors、gateway、control API 和 tool timeout。
- Go storage 已收敛为 Gateway 所需的 audit sink 和 dynamic capability store；平台 case/knowledge 继续由 Python Agent Platform 负责。
- README、AGENTS、runbook、架构、安全、MCP、health-food、部署和 compose 文档已同步为 Python Agent Platform + Go Investigation Gateway 主路径。

## 变更范围

- 删除：`cmd/baseline-orchestrator`、`cmd/dev-server`、`cmd/lark-bot`、`cmd/worker`。
- 删除：`internal/llm`、`internal/vision`、`internal/decisionbaseline`、`internal/webchat`、`internal/lark`、`internal/worker`、`internal/queue`、`internal/caseflow`、`internal/evolution`、`internal/chatplatform`。
- 保留：`cmd/investigation-gateway` 以及 Gateway 必需的 `audit/capability/config/connectors/gateway/httpauth/masking/policy/ratelimit/storage/tool`。

## 任务完成情况

| Task | 状态 | Evidence ID |
| --- | --- | --- |
| T1 | done | EV-T1-001, EV-T1-002 |
| T2 | done | EV-T2-001, EV-T2-002 |
| T3 | done | EV-T3-001 |
| T4 | done | EV-T4-001, EV-T4-002, EV-T4-003, EV-T4-004, EV-T4-005, EV-T4-006 |
| T5 | done | 8f66fc2 |

## 验证摘要

- `go test ./...`：pass。
- `make test`：pass。
- `make secret-scan`：pass。
- `git diff --check`：pass。
- `go vet ./...`：pass。
- `go build ./cmd/investigation-gateway`：pass。

## 验收覆盖

| 验收标准 | 结论 | Evidence ID |
| --- | --- | --- |
| `cmd/` 只保留正式 Go Gateway 入口 | pass | EV-T2-001 |
| Go 代码中不再存在旧 LLM/Vision/decisionbaseline 路径 | pass | EV-T2-002 |
| Go config 不再持有模型或 Lark bot 配置 | pass | EV-T2-002, EV-T4-001 |
| 主链路测试和扫描通过 | pass | EV-T4-001, EV-T4-002, EV-T4-003, EV-T4-004, EV-T4-005, EV-T4-006 |

## Commit

- `8f66fc2 P-2026-038 remove legacy go llm paths`
- `672233e P-2026-038 record go cleanup handoff`

Push status:

- `8f66fc2` and `672233e` pushed to `origin/main`.

## 残留风险

- 无本轮阻塞。真实 Lark/飞书和真实业务 adapter 验收不属于本轮范围。
