# RISKS

- 风险：把 Multica 的通用项目管理模型照搬进来，导致本项目复杂化。
  - 缓解：只实现 Agent Run 生命周期和 Runtime 抽象，case 仍是主实体。
- 风险：本地 coding agent 能力被误解为可以直接修改生产代码。
  - 缓解：文档和契约标记 debug-only / readonly；代码修改能力不在本轮范围。
- 风险：新增事件表导致写入噪音过大。
  - 缓解：事件记录结构化 summary 和 payload，避免塞完整工具结果；后续可做 TTL 或压缩。
