# ERROR SUMMARY

## 已归类错误

| 错误类型 | 来源 | 影响 | 当前防复发 |
| --- | --- | --- | --- |
| 先改代码后补 Program | P-2026-001 | 流程记录不完整 | `program-first-discipline` 计数器；Full 任务先建/读 Program。 |
| 回写旧 Program 历史 | P-2026-006 | 历史证据被新理解污染 | `program-history-rewrite` 计数器；独立变更新增 Program。 |
| mock 冒充真实接入 | P-2026-012 | 业务证据可靠性被夸大 | `mock-as-real-evidence` 计数器；结论不得高于证据等级。 |
| memory 冒充持久化验收 | P-2026-027 | 用户在 MySQL 查不到 Web 录入经验 | `memory-as-persistence-evidence` 计数器；mysql 缺 DSN fail-fast；持久化验收必须查表和重启读取。 |
| 浏览器输入能力没说清 | P-2026-009、P-2026-027 | UI 验收过程不透明 | `browser-input-capability-gap` 计数器；记录操作方式和已知噪音。 |
| 模型输出过信任 | P-2026-008 | 分类/字段缺失阻断排障或误分类 | `model-output-overtrust` 计数器；关键字段加规则 fallback。 |
| adapter 契约不一致 | P-2026-010、P-2026-015、P-2026-017 | 时间字段、参数名、历史 DDL 导致接入失败 | `adapter-contract-mismatch` 计数器；字段归一化、nullable、schema 测试。 |

## 本轮新增约束

- `AGENTS.md` 新增 L0-L4 证据等级。
- mock/fake/memory/local_rules 必须在结论中显式降级。
- 平台持久化验收固定为 MySQL migration + UI/API 写入 + 查表 + 重启读取。
- UI 验收必须说明是浏览器页面操作、CDP/DOM 触发，还是 API/curl。
- README 状态表避免把单测/mock 实现写成真实 bot 或生产链路已验收。
