# EVIDENCE

## Evidence 索引

| Evidence ID | 类型 | 关联任务 | 关联验收标准 | 结论 |
| --- | --- | --- | --- | --- |
| EV-T1-001 | docs | Task 1 | Program 和 Scope 建立 | pass |
| EV-T2-001 | code | Task 2 | Web Chat 页面和 API | pass |
| EV-T2-002 | browser | Task 2 | 浏览器加载 Web Chat 并完成文本交互 | pass |
| EV-T2-003 | api | Task 2 | 文本 multipart case 排查闭环 | pass |
| EV-T2-004 | api | Task 2 | 图片 multipart + Qwen-VL 识别 + 排查闭环 | pass |
| EV-T3-001 | mysql | Task 3 | 本地 MySQL migration | pass |
| EV-T3-002 | mysql | Task 3 | case/message/decision/tool audit 落库 | pass |
| EV-T4-001 | code/docs | Task 4 | Qwen 配置、LLM fallback、框架选择文档 | pass |
| EV-T5-001 | security | Task 5 | secret scan all/staged | pass |
| EV-T5-002 | security | Task 5 | pre-commit/pre-push hook 安装 | pass |
| EV-T6-001 | test | Task 6 | Go/Python/格式/扫描全量验证 | pass |

## 命令验证

| Evidence ID | 时间 | 命令 | 结果 | 备注 |
| --- | --- | --- | --- | --- |
| EV-T3-001 | 2026-05-23 | `MYSQL_DATABASE=ai_troubleshooter_p2026008 scripts/mysql-migrate.sh` | pass | 本地 MySQL 应用 `001_initial.sql` 到 `004_case_idempotency.sql`；密码仅通过环境变量传入。 |
| EV-T2-003 | 2026-05-23 | `curl -F message=... http://localhost:18080/web/api/chat` | pass | 文本 case `case_20260523_000003` 进入 kline 排查，调用 `get_internal_kline`、`get_external_kline_compare`，状态 `NEED_HUMAN_CONFIRMATION`。 |
| EV-T2-004 | 2026-05-23 | `curl -F message=... -F images=@/tmp/ai-troubleshooter-ticket.png http://localhost:18080/web/api/chat` | pass | Qwen-VL 识别截图，case `case_20260523_000005` 调用 4 个 mock 工具并生成中文结论。 |
| EV-T3-002 | 2026-05-23 | `mysql ... -e SELECT COUNT(*) ...` | pass | 验证库计数：cases=6、messages=17、ai_decision_logs=42、tool_audits=14。 |
| EV-T5-001 | 2026-05-23 | `python3.13 scripts/secret-scan.py --mode all` | pass | 全仓 tracked 文件未发现疑似真实密钥。 |
| EV-T6-001 | 2026-05-23 | `git diff --check` | pass | 无 whitespace error。 |
| EV-T6-001 | 2026-05-23 | `go test ./...` | pass | Go 全量测试通过。 |
| EV-T6-001 | 2026-05-23 | `PYTHONPATH=apps/decision-engine python3.13 -m unittest discover -s apps/decision-engine/tests -p 'test_*.py'` | pass | Python decision-engine 单测通过。 |

## 文档和代码证据

| Evidence ID | 时间 | 文件/范围 | 证据 | 结论 |
| --- | --- | --- | --- | --- |
| EV-T1-001 | 2026-05-23 | `programs/P-2026-008-web-chat-local-agent-mvp/**` | Program、Scope、Tasks、Decisions、Risks 已建立。 | pass |
| EV-T2-001 | 2026-05-23 | `web/**`、`internal/webchat/**`、`cmd/dev-server/main.go` | 内置 Web Chat、`/web/api/chat`、文本/图片 multipart、case 继续会话、decision logs/tool calls 返回。 | pass |
| EV-T2-002 | 2026-05-23 | Browser `http://localhost:18080/` | 页面标题 `AI Troubleshooter Web Chat`，主界面包含 `AI Troubleshooter`、`本地排障工作台`、`发送并排查`，浏览器文本提交成功。 | pass |
| EV-T4-001 | 2026-05-23 | `internal/llm/**`、`internal/vision/**`、`apps/decision-engine/**`、`docs/agent-framework-selection.md` | Qwen/DashScope OpenAI-compatible 配置可用；LLM 漏字段时回退规则基线；Python decision-engine 增加经验候选；文档记录暂不引入 LangGraph 的原因。 | pass |
| EV-T5-002 | 2026-05-23 | `githooks/**`、`scripts/install-git-hooks.sh`、`.git/hooks/*` | 已安装 pre-commit 和 pre-push，本地提交/推送前会运行 secret scan。 | pass |

## 覆盖映射

| 验收标准 | 对应任务 | Evidence ID | 状态 |
| --- | --- | --- | --- |
| Program 记录目标、Scope、敏感信息禁止提交规则 | Task 1 | EV-T1-001 | pass |
| `GET /` 或 `/web` 返回 Web Chat 页面 | Task 2 | EV-T2-001, EV-T2-002 | pass |
| `POST /web/api/chat` 支持文本和图片 multipart | Task 2 | EV-T2-003, EV-T2-004 | pass |
| 返回 case、reply、messages、decision logs 和 tool call ids | Task 2 | EV-T2-003, EV-T2-004 | pass |
| 本地 MySQL migration 可执行且不写密码 | Task 3 | EV-T3-001 | pass |
| Web Chat case 能落库 | Task 3 | EV-T3-002 | pass |
| Qwen/DashScope 可通过 OpenAI-compatible 配置使用 | Task 4 | EV-T2-004, EV-T4-001 | pass |
| Python decision-engine 文档记录框架选择 | Task 4 | EV-T4-001 | pass |
| 不提交 API key | Task 4/5 | EV-T5-001, EV-T5-002 | pass |
| staged/all tracked 扫描可运行 | Task 5 | EV-T5-001 | pass |
| 本地 `.git/hooks/pre-commit` 和 `pre-push` 已安装 | Task 5 | EV-T5-002 | pass |
| mock K线问题完成 Web Chat 排查闭环 | Task 6 | EV-T2-003, EV-T2-004 | pass |

## 未验证项

- 未接真实业务只读接口；本轮按要求使用 mock Gateway 验证。
- 未接真实 Lark/Feishu bot；Web Chat 优先完成。

## 已知噪音

- 本地验证库为 `ai_troubleshooter_p2026008`，用于避免污染可能已有的一期库。
