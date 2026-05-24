# P-2026-041 决策层拥有 Gateway 读取策略

## 背景

用户指出 README 架构图中 `Python Agent Platform -> Go Investigation Gateway` 的箭头会误导为平台入口决定是否读取 Gateway。正确边界应是：Decision Engine 决定是否需要实时业务证据、生成工具计划并由 Verifier 校验；Agent Platform runtime 只执行已验证计划。

## 目标

- 修正 README 和架构文档表达。
- 把“是否需要实时查 Gateway”的判断从 Agent Platform service 移到 Decision Engine。
- 保持 Gateway 仍只做业务只读工具安全边界，不引入 LLM/决策职责。

## 非目标

- 不重构 Gateway 协议。
- 不改变 Web/Lark/Case API 入口。
- 不改变平台 MySQL schema。
