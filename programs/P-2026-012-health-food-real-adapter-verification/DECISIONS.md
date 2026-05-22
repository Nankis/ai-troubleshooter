# DECISIONS

## D1：mock adapter 不能作为真实业务验收

mock adapter 只能验证平台契约和流程，不能证明业务证据可靠。本轮验收必须来自 health-food 本地真实服务、真实本地 DB、日志或本地代码只读定位。

## D2：本轮允许本地测试 DB 查询

为了验证 readonly adapter 能查到可靠业务信息，本轮 adapter 可以连接 health-food 本地测试库。生产环境仍应由业务方提供 readonly API 或经过审计的 readonly adapter，不允许 Agent 直连生产 DB。

## D3：Web Chat 验收必须看真实证据链

端到端验收不能只看 HTTP 200。必须能在 Web Chat 或 case API 返回中看到真实证据摘要，并在平台库中查到 tool audit 和 AI decision log，证明 Agent 为什么选择这些工具、查到了什么、何时停止。

## D4：本地代码定位只作为 debug-only 补充

当 Gateway 证据不足时，Python Decision Engine 可以在 `debug_local_code=true` 且提供 `service_name` 的情况下读取本地 allowlist 仓库。返回只能包含相对路径、命中词和行号，不能返回源码片段、配置密钥或自动修改业务仓库。
