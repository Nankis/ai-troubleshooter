# P-2026-051 Direct Agent Chat Routing

## Goal

修复 Web Chat 在同一个 case 内收到“模型状态、Agent 状态、平台咨询、用户纠错/吐槽”等非生产排障输入时，继续复用旧 case 的排障上下文去查 Gateway 的问题。

## Trigger

用户截图指出：咨询“现在是用什么模型”、反馈“我的 Claude Code 都用不了”时，平台仍返回 Gateway/mock-like 排障结果。

## Scope

- Python Agent Platform 意图路由。
- 决策层 Agent 直接回答非排障消息。
- 单测和 Web/MySQL 验证。
