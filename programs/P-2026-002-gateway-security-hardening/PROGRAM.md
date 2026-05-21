# P-2026-002 Gateway Security Hardening

## 背景

当前 investigation gateway 已具备工具级默认拒绝、scope/tool 校验、参数边界、timeout、审计和脱敏，但 HTTP 入口还缺少生产级硬门禁。业务方开箱部署前，需要补齐 Gateway HTTP 鉴权、请求身份绑定和基础限流。

## 目标

- 为 Gateway HTTP 入口增加 Bearer token 鉴权。
- 将认证身份绑定到 tool invocation 的 `agent_id`，防止调用方伪造其它 agent。
- 增加网关级 agent/user/tool 固定窗口限流。
- 更新配置、部署文档和单元测试。

## 非目标

- 不实现 mTLS 证书校验；预留给 Ingress/Service Mesh。
- 不实现分布式限流；当前为单实例内存限流，后续 Redis/Envoy 网关可替换。
- 不实现持久化 tool audit sink；另开 Program。

## 验收标准

- 未带 token 调用 `/tools/{tool}/invoke` 时返回 401。
- token 对应 agent 与 body agent_id 不一致时返回 403。
- token 对应 agent 与 body agent_id 一致时通过后续 policy。
- 超过限流返回 429。
- `make test` 通过。
