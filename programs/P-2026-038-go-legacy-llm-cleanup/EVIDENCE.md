# EVIDENCE

## Evidence 索引

| Evidence ID | 类型 | 关联任务 | 关联验收标准 | 结论 |
| --- | --- | --- | --- | --- |
| EV-T1-001 | scan | T1 | 确认 Go Gateway 正式依赖边界 | pass |
| EV-T1-002 | scan | T1 | 清理后 Gateway 依赖只剩正式包 | pass |
| EV-T2-001 | code | T2 | 删除 legacy Go 决策/模型/入口代码 | pass |
| EV-T2-002 | scan | T2 | Go 代码中无旧 LLM/Vision/decision 配置和包名 | pass |
| EV-T3-001 | docs | T3 | 文档不再指向旧 Go LLM/入口 | pass |
| EV-T4-001 | tests | T4 | `go test ./...` 通过 | pass |
| EV-T4-002 | tests | T4 | `make test` 通过 | pass |
| EV-T4-003 | security | T4 | `make secret-scan` 通过 | pass |
| EV-T4-004 | lint | T4 | `git diff --check` 通过 | pass |
| EV-T4-005 | static | T4 | `go vet ./...` 通过 | pass |
| EV-T4-006 | build | T4 | `go build ./cmd/investigation-gateway` 通过 | pass |

## 命令验证

| Evidence ID | 时间 | 命令 | 结果 | 备注 |
| --- | --- | --- | --- | --- |
| EV-T1-001 | 2026-05-24 | `go list -f '{{.ImportPath}} {{join .Imports "\\n"}}' ./cmd/investigation-gateway ./internal/...` | 初始扫描完成 | 发现正式 Gateway 不直接依赖 Go LLM；legacy 包仍存在 |
| EV-T1-002 | 2026-05-24 | `go list -deps ./cmd/investigation-gateway \| rg '^github.com/Nankis/ai-troubleshooter/' \| sort` | pass | 清理后依赖只剩 `audit/capability/config/connectors/gateway/httpauth/masking/policy/ratelimit/storage/tool` |
| EV-T2-001 | 2026-05-24 | `find cmd internal -maxdepth 3 -type f \| sort` | pass | `cmd/` 仅剩 `cmd/investigation-gateway/main.go`；`internal/` 仅剩 Gateway 必需包 |
| EV-T2-002 | 2026-05-24 | `rg -n "package (llm\|vision\|decisionbaseline\|webchat\|lark\|worker\|queue\|caseflow\|evolution\|chatplatform)\\b\|LLMConfig\|VisionConfig\|LarkConfig\|ValidateForLLM..." internal cmd --glob '*.go'` | pass | 无输出，命令返回 1 表示未命中 |
| EV-T3-001 | 2026-05-24 | `rg -n "cmd/(dev-server\|worker\|baseline-orchestrator\|lark-bot)\|go run ./cmd/dev-server\|internal/(llm\|vision\|decisionbaseline...)" README.md AGENTS.md docs configs Dockerfile deploy Makefile internal cmd` | pass | 无旧入口/旧包引用；仅保留“Go 不再承接这些职责”的边界说明 |
| EV-T4-001 | 2026-05-24 | `go test ./...` | pass | Go Gateway 相关包全部通过 |
| EV-T4-002 | 2026-05-24 | `make test` | pass | Go tests + decision-engine 17 tests + agent-platform 17 tests + root 4 tests passed |
| EV-T4-003 | 2026-05-24 | `make secret-scan` | pass | `Secret scan passed (all).` |
| EV-T4-004 | 2026-05-24 | `git diff --check` | pass | 无输出 |
| EV-T4-005 | 2026-05-24 | `go vet ./...` | pass | 无输出 |
| EV-T4-006 | 2026-05-24 | `go build ./cmd/investigation-gateway` | pass | 构建通过；临时二进制已删除，未纳入提交 |

## 现场验证

| Evidence ID | 时间 | 场景 | 证据 | 结论 |
| --- | --- | --- | --- | --- |

## 覆盖映射

| 验收标准 | 对应任务 | Evidence ID | 状态 |
| --- | --- | --- | --- |
| Go legacy 决策/模型/入口删除 | T2 | EV-T2-001, EV-T2-002 | pass |
| Gateway 编译测试不受影响 | T4 | EV-T1-002, EV-T4-001, EV-T4-002, EV-T4-006 | pass |
| 安全扫描和 diff check | T4 | EV-T4-003, EV-T4-004, EV-T4-005 | pass |

## 未验证项

- 无。本轮是代码清理与文档同步，不涉及真实 Lark/飞书或真实业务 adapter 验收。

## 已知噪音

- 暂无。
