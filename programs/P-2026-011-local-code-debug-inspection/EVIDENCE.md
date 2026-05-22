# EVIDENCE

## Evidence 索引

| Evidence ID | 类型 | 关联任务 | 关联验收标准 | 结论 |
| --- | --- | --- | --- | --- |
| EV-T1-001 | design | T1 | 安全边界清楚 | pass |
| EV-T2-001 | implementation | T2 | 本地 inspector 可用 | pass |
| EV-T3-001 | implementation | T3 | Local Code Agent 接入 | pass |
| EV-T4-001 | test | T4 | 正负向测试覆盖 | pass |
| EV-T5-001 | command | T5 | 全量验证通过 | pass |
| EV-T5-002 | smoke | T5 | HTTP debug local code flow 通过 | pass |
| EV-T5-003 | security | T5 | 敏感内容未出现在 smoke 输出 | pass |

## 命令验证

| Evidence ID | 时间 | 命令 | 结果 | 备注 |
| --- | --- | --- | --- | --- |
| EV-T4-001 | 2026-05-23 | `PYTHONPATH=apps/decision-engine python3.13 -m unittest discover -s apps/decision-engine/tests -p 'test_*.py'` | pass | 12 个 Python decision-engine 单测通过，包含 local code 正负向、敏感文件 deny、越界 symlink 跳过。 |
| EV-T5-001 | 2026-05-23 | `make test` | pass | Go 全仓测试和 Python decision-engine 单测通过。 |
| EV-T5-001 | 2026-05-23 | `python3.13 -m py_compile apps/decision-engine/decision_engine/local_code.py apps/decision-engine/decision_engine/agent_team.py apps/decision-engine/decision_engine/models.py` | pass | Python 语法检查通过。 |
| EV-T5-001 | 2026-05-23 | `git diff --check` | pass | 无空白错误。 |
| EV-T5-001 | 2026-05-23 | `python3.13 scripts/secret-scan.py --mode all` | pass | Secret scan passed。 |
| EV-T5-003 | 2026-05-23 | `grep -R "should_not_be_returned\\|token:" /tmp/ai_troubleshooter_local_code_debug.json /tmp/ai_troubleshooter_local_code_no_mapping.json || true` | pass | 无输出，敏感 mock 配置没有进入 Agent 输出。 |

## 现场验证

| Evidence ID | 时间 | 场景 | 证据 | 结论 |
| --- | --- | --- | --- | --- |
| EV-T1-001 | 2026-05-23 | 设计边界 | `DECISIONS.md` | Gateway 不下发本地路径；本地 registry 才能映射 repo。 |
| EV-T2-001 | 2026-05-23 | 实现 `LocalCodeInspector` | `apps/decision-engine/decision_engine/local_code.py` | 支持 env JSON registry、allowed/deny globs、相对路径命中结果。 |
| EV-T3-001 | 2026-05-23 | 接入 Local Code Agent | `apps/decision-engine/decision_engine/agent_team.py` | 只有 debug 显式开启且 Gateway 证据不足时触发。 |
| EV-T5-002 | 2026-05-23 | HTTP 正向 smoke | `/tmp/ai_troubleshooter_local_code_debug.json` | 返回 `local_code_inspection`，命中 `src/main/java/com/example/RecommendationJob.java`，跳过 1 个 denied 文件。 |
| EV-T5-002 | 2026-05-23 | HTTP 无 mapping smoke | `/tmp/ai_troubleshooter_local_code_no_mapping.json` | 返回 `need_human`，风险为 `local_repo_mapping_missing`。 |

## 覆盖映射

| 验收标准 | 对应任务 | Evidence ID | 状态 |
| --- | --- | --- | --- |
| 不信任 Gateway 本地路径 | T1 | EV-T1-001 | pass |
| 只读 allowlist 搜索且不返回源码片段 | T2 | EV-T2-001, EV-T5-003 | pass |
| 证据不足且 debug 显式开启才查代码 | T3 | EV-T3-001, EV-T4-001 | pass |
| 敏感文件和无 mapping 负向测试通过 | T4 | EV-T4-001, EV-T5-002, EV-T5-003 | pass |
| 全量验证和安全扫描通过 | T5 | EV-T5-001 | pass |

## 未验证项

- 未接入真实 health-food 仓库做源码搜索；本轮使用临时 mock 仓库验证安全和流程。
- 未做 AST/调用链分析；当前是关键词级只读检索。

## 已知噪音

- HTTP smoke 结束时用 Ctrl-C 停止服务，终端输出 `KeyboardInterrupt`，不影响验证结论。
