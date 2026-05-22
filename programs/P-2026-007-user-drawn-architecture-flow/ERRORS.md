# ERRORS

mistake_count: 0

## Incidents

暂无。

## Repeat Rules

- 独立架构图修正必须新增 Program，不回写旧 Program 历史。
- 架构图必须明确平台数据不走 Gateway，Gateway 只查业务证据。
- 决策层和 Agent 不直接访问业务 DB、日志 MCP 或业务服务。
