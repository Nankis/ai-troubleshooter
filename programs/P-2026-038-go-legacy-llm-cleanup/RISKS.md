# RISKS

| 风险 | 状态 | 缓解 |
| --- | --- | --- |
| 删除 legacy Go 入口后文档仍引用旧命令 | closed | 已用 `rg` 扫描 README/AGENTS/docs/configs/Docker/deploy/Makefile/internal/cmd，旧入口引用清理完毕；仅保留说明 Go 不再承接这些职责的边界文字。 |
| 误删 Gateway 依赖包 | closed | `go list ./...` 和 `go list -deps ./cmd/investigation-gateway` 仅剩 Gateway 必需包；`go test ./...` 通过。 |
| 测试覆盖被误删导致真实能力下降 | closed | 删除只覆盖 legacy 路径的测试；保留 Gateway/security/capability/storage 测试，`make test` 通过。 |
