# HANDOFF

## 当前状态

- 分支：`codex/framework-readiness-hardening`
- 当前任务：已完成，等待合并。
- 已完成：控制面 API Bearer 鉴权、生产配置 fail-closed、policy `chat_id` 边界修复、MySQL tool audit 持久化、默认限流阈值调整。
- 验证：`git diff --check`、`go vet ./...`、`make test`、`go test -race ./...`、prod smoke。
