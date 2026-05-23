# P-2026-017 Health Food Prod Log Readonly

## 背景

用户要求开始把本地排障平台接入 health-food 生产环境，用真实生产日志支撑问题排查。Decision Engine 和 Gateway 仍在本地运行，生产证据只能通过受控 readonly adapter 查询，不能让 Agent 直连生产 DB。

## 目标

- 盘点 health-food 是否已有日志查询能力。
- 在排障平台补齐本地 adapter 到 health-food 生产日志查询的只读桥接。
- 保持 Gateway 鉴权、scope、limit、时间窗、超时、脱敏和审计边界。
- 补测试和文档，明确真实生产验收不能用 mock 冒充。

## 非目标

- 本轮不自动连接生产，除非用户提供生产 base URL、只读密钥和明确问题时间窗。
- 本轮不改 health-food 生产配置、不提交生产部署、不执行生产 DB 操作。
- 本轮不把 health-food 管理接口重构为 Bearer 认证；先在本地 adapter 层做安全桥接。
