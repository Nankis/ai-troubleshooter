# EVIDENCE

## Evidence 索引

| Evidence ID | 类型 | 关联任务 | 关联验收标准 | 结论 |
| --- | --- | --- | --- | --- |
| EV-P057-001 | docs | 编辑路径铁律 | `AGENTS.md` 明确写入前确认仓库根和 `apply_patch` 绝对路径 | pass |
| EV-P057-002 | docs | 错误复盘 | `docs/LESSONS.md` 新增 `wrong-repo-apply-patch` 计数和复盘 | pass |
| EV-P057-003 | command | 静态验证 | 文档变更通过 diff/secret/Program 校验 | pass |

## 命令验证

| Evidence ID | 时间 | 命令 | 结果 | 备注 |
| --- | --- | --- | --- | --- |
| EV-P057-003 | 2026-05-25 | `git diff --check` | pass | 无输出 |
| EV-P057-003 | 2026-05-25 | `make secret-scan` | pass | 无敏感信息 |
| EV-P057-003 | 2026-05-25 | `python3.13 scripts/validate_program.py programs/P-2026-057-repo-root-edit-discipline` | pass | Program 结构有效 |

## 现场验证

| Evidence ID | 时间 | 场景 | 证据 | 结论 |
| --- | --- | --- | --- | --- |
| EV-P057-001 | 2026-05-25 | 编辑入口规则 | `AGENTS.md` 新增“编辑路径铁律” | pass |
| EV-P057-002 | 2026-05-25 | 历史错误记录 | `docs/LESSONS.md` 计数器新增 `wrong-repo-apply-patch`，count=1 | pass |

## 覆盖映射

| 验收标准 | 对应任务 | Evidence ID | 状态 |
| --- | --- | --- | --- |
| 下次编辑前能看到强制确认仓库根的规则 | `AGENTS.md` | EV-P057-001 | pass |
| 错误被纳入复盘计数器 | `docs/LESSONS.md` | EV-P057-002 | pass |
| 文档变更没有空白/敏感信息问题 | 静态验证 | EV-P057-003 | pass |

## 未验证项

- 本 Program 不涉及业务链路、MySQL、Gateway 或 Web UI。

## 已知噪音

- 无。
