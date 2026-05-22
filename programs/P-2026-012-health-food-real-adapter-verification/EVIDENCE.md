# EVIDENCE

## Evidence 索引

| Evidence ID | 类型 | 关联任务 | 关联验收标准 | 结论 |
| --- | --- | --- | --- | --- |
| EV-T1-001 | design | T1 | 真实验收标准明确 | pass |
| EV-T2-001 | discovery | T2 | health-food 真实证据来源明确 | pass |
| EV-T3-001 | implementation | T3 | real readonly adapter 可用 | pass |
| EV-T4-001 | e2e | T4 | 端到端查到真实证据 | pass |
| EV-T5-001 | command | T5 | 测试和安全扫描通过 | pass |

## 命令验证

| Evidence ID | 时间 | 命令 | 结果 | 备注 |
| --- | --- | --- | --- | --- |
| EV-T3-001 | 2026-05-23 | `python3.13 -m py_compile scripts/real-health-food-readonly-adapter.py` | pass | adapter 语法校验通过 |
| EV-T3-001 | 2026-05-23 | `GET /healthz` + Bearer/unauthorized adapter requests | pass | `source=real-local`；无 token 调用返回 401 |
| EV-T4-001 | 2026-05-23 | health-food 真实注册、添加餐食、查询 profile/meals/recommend API | pass | 通过真实 API 创建测试用户并写入餐食 |
| EV-T4-001 | 2026-05-23 | health-food 本地 DB 查询 | pass | 用户 1 行、餐食 1 行、餐食明细 1 行、当天推荐 0 行、token 账户存在 |
| EV-T4-001 | 2026-05-23 | Web Chat `case_20260523_000002` | pass | 浏览器逐键输入问题并点击发送，返回真实排查结论 |
| EV-T4-001 | 2026-05-23 | Python Decision Engine local code inspection | pass | `service_name=health-food` 命中 `RecommendFoodJob.java`、`FoodServiceImpl.java`、`TbDailyFoodRecommend.java` 等相对路径和行号 |
| EV-T5-001 | 2026-05-23 | `make test` | pass | Go 测试和 Python decision-engine 12 个单测通过 |
| EV-T5-001 | 2026-05-23 | `git diff --check` | pass | 无空白错误 |
| EV-T5-001 | 2026-05-23 | `python3.13 scripts/secret-scan.py --mode all` | pass | 未发现需阻断的敏感信息 |

## 现场验证

| Evidence ID | 时间 | 场景 | 证据 | 结论 |
| --- | --- | --- | --- | --- |
| EV-T2-001 | 2026-05-23 | health-food 真实证据梳理 | 注册接口、登录态接口、餐食接口、推荐接口、AI 配额接口、推荐任务代码路径 | 可以构造真实本地测试账号并查真实 DB |
| EV-T3-001 | 2026-05-23 | real readonly adapter | `scripts/real-health-food-readonly-adapter.py` | 返回 envelope 来自本地 DB / 探活 / 可选日志文件，不合成 mock 故障数据 |
| EV-T4-001 | 2026-05-23 | 平台端到端 | Web Chat 回复中包含 `registered=true`、`meal records=1`、`recommendation exists=false/job_status=missing` | 真实查到“有餐食但当天推荐表无记录” |
| EV-T4-001 | 2026-05-23 | 平台审计 | `tb_troubleshoot_tool_call_audit` 5 条 allowed 调用；`tb_troubleshoot_ai_decision_log` 10 条成功决策记录 | 工具调用和 AI 决策原因均可追溯 |
| EV-T4-001 | 2026-05-23 | UI 验证 | `/tmp/hf_real_webchat_result.png`、`/tmp/hf_real_webchat_dom_snapshot.txt` | 浏览器真实打开 Web Chat 并提交问题 |

## 覆盖映射

| 验收标准 | 对应任务 | Evidence ID | 状态 |
| --- | --- | --- | --- |
| 真实服务启动 | T4 | EV-T4-001 | pass |
| 真实 adapter 不返回 mock 故障数据 | T3 | EV-T3-001 | pass |
| 通过平台查到真实 DB/日志/代码证据 | T4 | EV-T4-001 | pass |
| 验证失败项如实记录 | T5 | EV-T5-001 | pass |

## 未验证项

- 未接真实 Lark / 飞书 bot，本轮聚焦 Web Chat 和真实 health-food adapter。
- 未接公司真实日志平台；本地 adapter 已支持 `HEALTH_FOOD_LOG_PATH`，但本轮未提供独立日志文件，因此日志查询返回 0 条并给 warning。
- 未触发 health-food 定时任务真实执行；本轮通过真实 DB 证明“当天有餐食但推荐表无记录”，并通过本地代码定位到推荐任务与生成逻辑。

## 已知噪音

- 浏览器自动化批量填充时本地虚拟剪贴板不可用，已改用逐键输入完成 UI 验证。
- health-food 启动日志可能打印本地配置，证据记录中不保存密钥、token 或完整日志内容。
