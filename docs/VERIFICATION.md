# 验证结果记录规范

本规范借鉴 `game` 仓库的 Program Evidence 方式。目标是让验证结果可恢复、可审计、可交接，而不是只写“测试通过”。

## Evidence 必备结构

每个 Full 级 Program 的 `EVIDENCE.md` 应至少包含：

```md
# EVIDENCE

## Evidence 索引

| Evidence ID | 类型 | 关联任务 | 关联验收标准 | 结论 |
| --- | --- | --- | --- | --- |
| EV-T1-001 | docs | Task 1 | Program 建立 | pass |
| EV-T5-001 | command | Task 5 | git diff --check 通过 | pass |

## 命令验证

| Evidence ID | 时间 | 命令 | 结果 | 备注 |
| --- | --- | --- | --- | --- |
| EV-T5-001 | 2026-05-23 | `git diff --check` | pass | 无输出 |
| EV-T5-002 | 2026-05-23 | `make test` | pass | Go/Python 单测通过 |

## 现场验证

| Evidence ID | 时间 | 场景 | 证据 | 结论 |
| --- | --- | --- | --- | --- |

## 覆盖映射

| 验收标准 | 对应任务 | Evidence ID | 状态 |
| --- | --- | --- | --- |

## 未验证项

- 无，或说明为什么本轮不需要验证。

## 已知噪音

- 例如 GitHub Actions Node 版本 warning，不影响本轮验证结论。
```

## Result 必备结构

每个 Full 级 Program 的 `RESULT.md` 应至少包含：

```md
# RESULT

## 结果摘要

- 本轮完成了什么。

## 变更范围

- 关键文件或目录。

## 任务完成情况

| Task | 状态 | Evidence ID |
| --- | --- | --- |

## 验证摘要

- `git diff --check`：pass。
- `make test`：pass。

## 验收覆盖

| 验收标准 | 结论 | Evidence ID |
| --- | --- | --- |

## Commit

- `commit_hash commit message`。

## 残留风险

- 明确未完成、未验证或后续要继续的风险。
```

## 任务类型和验证要求

| 类型 | 最低验证 |
| --- | --- |
| docs-only | `git diff --check`，并记录文档覆盖映射 |
| Go code | `gofmt`、`go test ./...`、`go vet ./...`、`git diff --check` |
| Python code | Python 单测、OpenAPI/schema 相关检查、`git diff --check` |
| DB migration | schema 测试、DDL 命名规范检查、回滚/兼容说明 |
| Gateway/security | 鉴权、拒绝路径、scope、限流、脱敏和审计测试 |
| Lark/Feishu | 本地 mock payload、幂等、加密 callback、图片下载路径验证 |

## 记录原则

- 命令必须写完整，包含必要的环境变量或解释。
- 通过和未通过都要记录；失败后修复，应保留失败摘要和最终通过证据。
- 未验证项必须显式写出，不能用沉默代替。
- CI warning 如果不影响结论，放到“已知噪音”。
- 敏感信息、长 payload、完整日志不贴入 Evidence，只记录摘要和路径。
