# Handoff

## 当前目标

P-2026-038：清理 Go 侧遗留 LLM/Vision/决策/入口代码，让 Go 正式职责收敛到 `cmd/investigation-gateway`。

## 已完成

- 已按接手规则读取 `README.md`、`AGENTS.md`、`docs/LESSONS.md`、`docs/VERIFICATION.md`、`programs/README.md`。
- 初步扫描发现 legacy Go 入口和包：`cmd/baseline-orchestrator`、`cmd/dev-server`、`cmd/lark-bot`、`cmd/worker`、`internal/decisionbaseline`、`internal/llm`、`internal/vision`、`internal/webchat` 等。
- 初步确认 `cmd/investigation-gateway` 不直接依赖 Go LLM/decisionbaseline。
- 已删除 Go 侧 legacy LLM/Vision/决策/入口/worker/caseflow/evolution/queue/chatplatform 包。
- 已清理 Go config，不再解析模型或 Lark bot 配置。
- 已更新 README、AGENTS、runbook、架构、安全、MCP、health-food、部署和 compose 文档。
- 已通过 `go test ./...`、`make test`、`make secret-scan`、`git diff --check`、`go vet ./...`、`go build ./cmd/investigation-gateway`。
- 已提交 `8f66fc2 P-2026-038 remove legacy go llm paths`。
- 已提交 `672233e P-2026-038 record go cleanup handoff`。
- `8f66fc2` 和 `672233e` 已推送到 `origin/main`；本文件当前改动是最终 push 状态补记。

## 证据路径

- `programs/P-2026-038-go-legacy-llm-cleanup/EVIDENCE.md`
- `programs/P-2026-038-go-legacy-llm-cleanup/RESULT.md`

## 已运行命令

- `git status --short && git branch --show-current`
- `sed -n ... README.md AGENTS.md docs/LESSONS.md docs/VERIFICATION.md programs/README.md`
- `find cmd internal -maxdepth 3 -type f | sort`
- `rg -n "internal/llm|decisionbaseline|baseline-orchestrator|cmd/dev-server|cmd/worker|cmd/lark-bot|llm|LLM|Vision|vision|orchestrator" ...`
- `go list -f '{{.ImportPath}} {{join .Imports "\\n"}}' ./cmd/investigation-gateway ./internal/...`
- `go list ./...`
- `go list -deps ./cmd/investigation-gateway | rg '^github.com/Nankis/ai-troubleshooter/' | sort`
- `rg` 旧 Go LLM/decision/Lark/Web/worker 包和文档引用扫描。
- `gofmt -w $(find cmd internal -name '*.go')`
- `go test ./...`
- `make test`
- `make secret-scan`
- `git diff --check`
- `go vet ./...`
- `go build ./cmd/investigation-gateway`
- 删除 `go build` 产生的临时二进制 `investigation-gateway`。

## 工作树

- 仅剩最终 push 状态补记等待提交；主实现和第一版 handoff 已推送。

## 下一步

1. 提交最终 push 状态补记。
2. push `main`。

## 风险/阻塞

- 无当前阻塞。临时构建产物已删除，未纳入 Git。
