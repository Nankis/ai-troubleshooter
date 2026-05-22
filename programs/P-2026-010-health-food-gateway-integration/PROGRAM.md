# P-2026-010 Health Food Gateway Integration

## 背景

用户要求本地启动 `health-food`，把它作为业务服务接入问题排查平台，通过 mock 故障走完整排障流程，并梳理业务服务注册到 Investigation Gateway 时需要提供的数据结构。

## 目标

- 确认可用的 `health-food` 本地工程路径、启动方式和当前阻塞项。
- 让排障平台通过标准 readonly adapter 契约连接一个 `health-food` 本地服务或 mock adapter。
- 构造至少 2 类业务故障，走 Web Chat / Case / Gateway / Connector / 审计链路。
- 补齐业务服务能力注册的数据结构规范和示例，形成可复用接入文档。
- 记录真实验证证据、失败原因、残留风险和下一步。

## 非目标

- 不提交任何 `health-food` 私有配置、API key、MySQL 密码或 token。
- 不直接改写旧 Program 历史记录。
- 不把 Agent 直接连到业务 DB。
- 不做真实生产服务调用。

