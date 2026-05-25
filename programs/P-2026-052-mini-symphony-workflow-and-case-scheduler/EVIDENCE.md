# EVIDENCE

## Evidence 索引

| Evidence ID | 类型 | 关联任务 | 关联验收标准 | 结论 |
| --- | --- | --- | --- | --- |
| EV-P052-001 | docs | workflow contract | `WORKFLOW.md`、schema、scheduler design 已建立 | pass |
| EV-P052-002 | command | Program 验证脚本 | 5 个 Program 骨架可被脚本校验 | pass |
| EV-P052-003 | command | workflow schema | JSON schema 格式有效 | pass |

## 命令验证

| Evidence ID | 时间 | 命令 | 结果 | 备注 |
| --- | --- | --- | --- | --- |
| EV-P052-002 | 2026-05-25 | `python3.13 scripts/validate_program.py programs/P-2026-052-mini-symphony-workflow-and-case-scheduler programs/P-2026-053-investigation-brief-observable programs/P-2026-054-brief-bound-tool-verifier programs/P-2026-055-brief-driven-decision-planning programs/P-2026-056-case-scheduler-state-machine` | pass | 输出 `validated 5 program(s)` |
| EV-P052-003 | 2026-05-25 | `python3.13 -m json.tool .workflow/task.schema.json` | pass | JSON schema 可解析 |

## 现场验证

| Evidence ID | 时间 | 场景 | 证据 | 结论 |
| --- | --- | --- | --- | --- |
| EV-P052-001 | 2026-05-25 | 文档/schema 落地 | 新增 `WORKFLOW.md`、`.workflow/task.schema.json`、`docs/investigation-brief.md`、`docs/case-scheduler-design.md`、`scripts/validate_program.py` | pass |

## 覆盖映射

| 验收标准 | 对应任务 | Evidence ID | 状态 |
| --- | --- | --- | --- |
| 文档和 schema 可以被后续 Program 引用 | workflow contract | EV-P052-001 | pass |
| 验证脚本能检查 Program 基本文件和 Evidence 结构 | Program 验证脚本 | EV-P052-002 | pass |
| 不引入 mock/memory 验收口径 | workflow contract | EV-P052-001 | pass |

## 未验证项

- 真实排障链路不在 P-052 范围；后续 P-053 到 P-056 负责 L3 全链路验证。

## 已知噪音

- 无。
