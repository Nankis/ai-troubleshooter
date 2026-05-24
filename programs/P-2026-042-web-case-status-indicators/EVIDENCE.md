# EVIDENCE

## Evidence 索引

| Evidence ID | 类型 | 关联任务 | 关联验收标准 | 结论 |
| --- | --- | --- | --- | --- |
| EV-001 | code | 状态 UI 实现 | 左侧排查中显示 spinner，待查看显示结果提示 | pass |
| EV-002 | browser | Web 实际验证 | 页面从真实 `/web/api/overview` 读取 case 状态并渲染 | pass |
| EV-003 | browser | 点击已读验证 | 点击待查看 case 后清除本机待查看状态 | pass |
| EV-004 | command | 静态检查 | inline script 可解析，`git diff --check` 通过 | pass |
| EV-005 | command | 回归 | `make test` 通过 | pass |
| EV-006 | security | 敏感信息 | `make secret-scan` 通过 | pass |

## 命令验证

| Evidence ID | 时间 | 命令 | 结果 | 备注 |
| --- | --- | --- | --- | --- |
| EV-004 | 2026-05-25 01:28 CST | `node -e '... new Function(inlineScript) ...'` | pass | `inline script parse ok` |
| EV-004 | 2026-05-25 01:28 CST | `git diff --check` | pass | 无输出 |
| EV-005 | 2026-05-25 01:28 CST | `PYTHONPATH=apps/agent-platform:apps/decision-engine .venv/bin/python -m unittest discover -s apps/agent-platform/tests -p 'test_*.py'` | pass | 24 tests |
| EV-005 | 2026-05-25 01:28 CST | `make test` | pass | Go 全量、decision-engine 18 tests、agent-platform 24 tests、root 4 tests |
| EV-006 | 2026-05-25 01:28 CST | `make secret-scan` | pass | `Secret scan passed (all).` |

## 现场验证

| Evidence ID | 时间 | 场景 | 证据 | 结论 |
| --- | --- | --- | --- | --- |
| EV-002 | 2026-05-25 01:24 CST | 启动 Agent Platform，打开 `http://127.0.0.1:19091/web` | Browser viewport 1600x950，MySQL fixture case status=`INVESTIGATING` 显示 `case-row running`、`hasSpinner=true`、文本 `排查中` | pass |
| EV-002 | 2026-05-25 01:26 CST | 将一个已在页面出现的 `INVESTIGATING` case 转为 `NEED_HUMAN_CONFIRMATION` 并刷新概览 | DOM 显示 `case-row needs-view`、`hasDot=true`、文本 `待查看`；截图：`programs/P-2026-042-web-case-status-indicators/evidence/web-case-status-before-click.png` | pass |
| EV-003 | 2026-05-25 01:26 CST | 点击待查看 case | DOM 显示该 case 变为 `case-row active`、`hasDot=false`、文本 `待确认`；截图：`programs/P-2026-042-web-case-status-indicators/evidence/web-case-status-after-click.png` | pass |
| EV-003 | 2026-05-25 01:27 CST | 清理验证 fixture | MySQL 将 `case_title LIKE 'UI状态验证%'` 的测试 case 软删除，刷新后页面 fixture 可见数为 0 | pass |

## 覆盖映射

| 验收标准 | 对应任务 | Evidence ID | 状态 |
| --- | --- | --- | --- |
| AI 正在排查中的任务在左侧显示转圈 | spinner 实现、浏览器验证 | EV-001, EV-002 | pass |
| AI 已有结果但用户未查看时显示待查看 | 待查看状态、浏览器验证 | EV-001, EV-002 | pass |
| 用户点击后清除待查看提示 | 本机已读记录、点击验证 | EV-003 | pass |
| 不改变后端 case 状态机 | 代码范围和验证 | EV-001, EV-002 | pass |
| 回归和安全检查通过 | 回归检查 | EV-004, EV-005, EV-006 | pass |

## 未验证项

- 未做服务端多用户已读表；本轮按本地 Web 工作台体验实现，已在 `DECISIONS.md` 记录。

## 已知噪音

- Browser 初始 viewport 为 630px 时左侧因响应式规则隐藏；已通过 Browser viewport capability 切到 1600x950 后验证桌面左侧 UI。
- 验证使用受控 MySQL case fixture，只验证 Web 状态渲染和读回执，不代表真实业务排障。
