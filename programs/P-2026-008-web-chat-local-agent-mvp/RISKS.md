# RISKS

- 本地 MySQL 环境可能未启动或没有 schema；通过迁移脚本和 `SKIP/BLOCKED` 记录处理。
- Qwen key 来自本机其他项目，不能输出到日志或提交到仓库。
- Mermaid/文档不属于本轮重点，主要风险在 Web Chat 到 case/agent/gateway/MySQL 的运行闭环。
- LangGraph 暂不引入，后续如果 agent 状态复杂度上升，需要单独 Program 迁移。
