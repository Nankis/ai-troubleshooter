# P-2026-004 Decision Logging And Query Limits

## 背景

用户指出：AI 做出的分类、工具选择、追问和总结需要记录为什么这么做；排查查不到问题时必须有超时、工具调用上限和失败上限，不能死循环或持续查询下游。

## 目标

- 持久化 AI 决策日志，记录 decision type、reason、input/output snapshot、selected tools、latency、status 和 error。
- 给 orchestrator 增加 case 级超时、工具调用上限、工具失败上限。
- 发生超时或内部错误时把 case 和 investigation 安全收敛到 `FAILED`，并写入决策日志。
- 更新 DDL、配置、README 和部署文档。

## 非目标

- 不实现多轮自主 agent 循环；一期仍保持一次有限工具计划。
- 不接入真实 LLM tracing 平台；先落 MySQL / memory store。
- 不替代 Gateway 侧 timeout、rate limit 和下游 connector timeout。

## 验收标准

- AI 决策日志可写入和查询。
- K线完整 case 产生分类、实体抽取、工具计划、工具调用、总结等决策日志。
- 超过工具失败上限会停止继续查询下游。
- case 级超时会失败收敛，不会无限运行。
- `git diff --check`、`go vet ./...`、`make test`、`go test -race ./...` 通过。
