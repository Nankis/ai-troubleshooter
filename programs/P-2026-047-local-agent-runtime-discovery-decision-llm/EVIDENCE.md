# EVIDENCE

## Evidence 索引

| Evidence ID | 类型 | 关联任务 | 关联验收标准 | 结论 |
| --- | --- | --- | --- | --- |
| EV-001 | 单测 | Local Agent Discovery / local_agent provider / advisor verifier | API、provider、Verifier 不绕过边界 | PASS |
| EV-002 | 回归 | 全仓 Go/Python 测试 | 现有主路径未回归 | PASS |
| EV-003 | 安全 | secret scan / diff whitespace | 无密钥入仓，diff 无空白错误 | PASS |
| EV-004 | 浏览器现场 | Web 工作台本地 Agent 发现/启用 | UI 可见、可启用 llm-capable provider、editor-only provider 禁用 | PASS |
| EV-005 | API smoke | local-agents discover/probe | 本机 provider 被发现，probe 不执行模型也能返回能力状态 | PASS |

## 命令验证

| Evidence ID | 时间 | 命令 | 结果 | 备注 |
| --- | --- | --- | --- | --- |
| EV-001 | 2026-05-25 | `PYTHONPATH=apps/agent-platform:apps/decision-engine .venv/bin/python -m unittest discover -s apps/agent-platform/tests -p 'test_agent_platform_fastapi.py'` | PASS，25 tests | 覆盖 discovery/enable/local_agent command override |
| EV-001 | 2026-05-25 | `PYTHONPATH=apps/decision-engine .venv/bin/python -m unittest discover -s apps/decision-engine/tests -p 'test_engine.py'` | PASS，19 tests | 覆盖 advisor 输出先过 Verifier |
| EV-002 | 2026-05-25 | `make test` | PASS | Go 全包、decision-engine 19 tests、agent-platform 36 tests、root tests 4 tests |
| EV-003 | 2026-05-25 | `make secret-scan` | PASS | `Secret scan passed (all).` |
| EV-003 | 2026-05-25 | `git diff --check` | PASS | 无输出 |
| EV-005 | 2026-05-25 | `curl -s http://127.0.0.1:19147/api/v1/local-agents/discover` | PASS | 发现 Claude Code、Codex CLI、Cursor、Cursor Agent missing；未读取密钥内容 |
| EV-005 | 2026-05-25 | `curl -s -X POST http://127.0.0.1:19147/api/v1/local-agents/probe -d '{"provider_id":"codex","execute":false}'` | PASS | 返回 Codex installed / llm_capable / probe_status=installed |

## 现场验证

| Evidence ID | 时间 | 场景 | 证据 | 结论 |
| --- | --- | --- | --- | --- |
| EV-004 | 2026-05-25 | 启动临时 Agent Platform，打开 `/web`，点击“发现”，启用 Codex | `programs/P-2026-047-local-agent-runtime-discovery-decision-llm/web-local-agent-discovery.png` | PASS：Claude Code/Codex 可见，Codex 可启用，Cursor editor 禁用 |

## 覆盖映射

| 验收标准 | 对应任务 | Evidence ID | 状态 |
| --- | --- | --- | --- |
| 自动发现本地 Claude Code / Codex / Cursor | Discovery API + Web | EV-004, EV-005 | 通过 |
| 显式启用，不能自动把 editor 当 LLM | Enable API + Web disabled state | EV-001, EV-004 | 通过 |
| 本地 agent 可作为 LLM provider | `local_agent` command override 单测 | EV-001 | 通过 |
| Decision Engine 中本地 LLM 只做 advisor | `llm_decision_agent` + Verifier 单测 | EV-001 | 通过 |
| 不破坏现有 Go Gateway / Python 主路径 | 全量回归 | EV-002 | 通过 |
| 不提交密钥 | secret scan | EV-003 | 通过 |

## 未验证项

- 未执行真实 Claude Code / Codex 模型推理 probe（`execute=true`），因为会消耗本机模型额度且需要当前登录态；本轮只验证 discover、enable、非交互 fake provider 和 `execute=false` probe。
- 未验证 Cursor Agent 真实执行；本机未安装 `cursor-agent`，仅验证 Cursor editor 被发现但禁用 LLM 启用。
- 未用生产 MySQL 做 runtime 落库验收；本轮代码路径使用现有 `register_agent_runtime` 仓库方法，自动化用 memory repository 和临时 Web 服务验证。生产 MySQL schema 未变更。

## 已知噪音

- 浏览器现场发现本机真实版本号会随本地安装变化；证据只证明 2026-05-25 当前机器的发现链路。
