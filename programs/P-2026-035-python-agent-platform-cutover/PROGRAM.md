# P-2026-035 Python Agent Platform Cutover

## 背景

用户确认目标架构：`orchestrator` 属于决策层，应放在 Python；Go 不应承载 LLM、Web Chat、Lark/飞书入口或平台 Agent 编排。Go 只保留 Investigation Gateway，用于下游只读能力接入、安全鉴权、scope、限流、脱敏、审计和超时。

## 目标

- 新增 Python Agent Platform 作为主路径，承接 Web Chat、Case API、平台 MySQL、决策编排、Gateway 调用和进度展示 API。
- Go 主路径收敛为 Investigation Gateway；历史 Go dev-server/decisionbaseline 标记为 legacy，不再作为目标架构。
- 大模型配置收敛到 Python 决策层，预留 Qwen/GPT/Claude/Claude Code 统一入口。
- 更新 README 和业务接入文档，让业务方知道应该启动哪些服务、如何配置 LLM、如何写 readonly 接口并注册 Gateway。
- 重新验证本地流程，记录证据等级和未验证项。

## 非目标

- 本轮不删除所有 Go 历史代码，避免一次性破坏旧 Program 和已有测试；但默认文档、主路径和新增服务不再依赖 Go LLM/baseline。
- 本轮实现并本地验证 Lark/飞书 encrypted callback 解包、token 校验、群 allowlist、消息幂等和图片下载入口；不宣称真实外部平台 L4 验收。
- 本轮不把生产 health-food 接入冒充为真实验收；真实生产接入需另起 Program。
