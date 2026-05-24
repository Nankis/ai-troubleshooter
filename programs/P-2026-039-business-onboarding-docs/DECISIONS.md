# Decisions

## D1. 新手入口以 quickstart 为主

`docs/business-onboarding-quickstart.md` 作为业务方和业务方 AI 的第一入口，详细接口契约仍链接到 `docs/ai-connector-integration.md` 与 `docs/business-service-registration.md`。

## D2. 明确平台和业务边界

文档中明确业务方不提供 LLM、不提供平台 MySQL、不让 Agent/Gateway 直连业务 DB。业务方只提供 readonly adapter 或 MCP readonly adapter。

## D3. 运行说明按职责拆分

文档分别给出 Investigation Gateway、Agent Platform、可选 Decision Engine 调试入口的运行方式，避免误解 Go 侧需要 LLM。
