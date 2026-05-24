# DECISIONS

## D1. 正常 Web Chat 不单独启动外部 Decision Engine 服务

Agent Platform 作为入口进程内嵌 Python `apps/decision-engine` 包，调用 `DecisionEngine.plan()` 做排查决策。`make decision-engine` 只用于协议/CLI/HTTP 调试，不是业务方必需服务。

## D2. Decision Engine 不能被 mock 掉当作完整排查验收

允许 mock Gateway adapter 或使用 `local_rules` 做某个局部链路验证，但如果验证目标是“排查流程”，必须证明 `DecisionEngine.plan()` 实际被调用，并记录 mock 边界。
