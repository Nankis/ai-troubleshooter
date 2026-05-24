# Program: P-2026-038 Go Legacy LLM Cleanup

## 背景

当前架构已经明确：Python Agent Platform / Decision Engine 负责 Web Chat、Lark/飞书、图片、LLM/Vision 和决策编排；Go 正式职责只保留 Investigation Gateway。用户指出代码里仍有 Go LLM 等历史实现，需要检查并删除无用代码，避免后续接入方误用旧路径。

## 目标

- 删除 Go 侧历史 LLM、Vision、Go Web Chat、Go Lark bot、baseline orchestrator 和 worker 代码。
- 清理 Go config 中 LLM/Vision/Lark bot 专用配置和校验。
- 保留 Go Gateway 必需的鉴权、scope、限流、timeout、脱敏、审计、动态只读工具注册能力。
- 同步 README、架构文档和本地运行文档，明确 Go 只运行 `cmd/investigation-gateway`。
- 用依赖扫描和测试证明删除后主链路仍可编译、测试和扫描通过。

## 非目标

- 不改 Python Agent Platform 的 LLM/Vision provider。
- 不改数据库 DDL。
- 不做真实 Lark/飞书回调验收。
- 不做真实 health-food 生产 adapter 验收。

## 验收标准

- `cmd/` 下只保留正式 Go Gateway 入口。
- Go 代码中不再存在 `internal/llm`、`internal/vision`、`internal/decisionbaseline` 这类 Go 决策/模型路径。
- Go `config.Config` 不再解析或暴露 LLM/Vision/Lark bot 配置。
- `go test ./...`、`make test`、`make secret-scan`、`git diff --check` 通过。
- Program `EVIDENCE.md` 记录删除依据、命令和覆盖映射。
