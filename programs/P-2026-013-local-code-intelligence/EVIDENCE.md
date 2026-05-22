# EVIDENCE

## Evidence 索引

| Evidence ID | 类型 | 关联任务 | 结论 |
| --- | --- | --- | --- |
| EV-T1-001 | design | T1 | pass |
| EV-T2-001 | implementation | T2 | pass |
| EV-T3-001 | test | T3 | pass |
| EV-T4-001 | command | T4 | pass |

## 命令验证

| Evidence ID | 时间 | 命令 | 结果 | 备注 |
| --- | --- | --- | --- | --- |
| EV-T3-001 | 2026-05-23 | `PYTHONPATH=apps/decision-engine python3.13 -m unittest apps/decision-engine/tests/test_engine.py -v` | pass | Local Code Agent Java/Python 结构化证据测试通过 |
| EV-T4-001 | 2026-05-23 | `make test` | pass | Go + Python 全量测试通过 |
| EV-T4-001 | 2026-05-23 | `git diff --check` | pass | 无空白错误 |
| EV-T4-001 | 2026-05-23 | `python3.13 scripts/secret-scan.py --mode all` | pass | 未发现需阻断的敏感信息 |

## 现场验证

| Evidence ID | 时间 | 场景 | 证据 | 结论 |
| --- | --- | --- | --- | --- |
| EV-T1-001 | 2026-05-23 | 结构化输出设计 | `LocalCodeHit` 增加 `symbols`、`call_edges`、`analysis_modes`，兼容旧字段 | pass |
| EV-T2-001 | 2026-05-23 | health-food 真实仓库试跑 | 扫描 326 个文件，建立 932 个符号、5875 条调用边，命中 `RecommendFoodJob.refreshDailyRecommend -> generateDailyFoodRecommendWithFingerprint` | pass |
| EV-T3-001 | 2026-05-23 | 安全边界 | 单测覆盖 denylist 配置文件、symlink 跳出仓库、无源码片段输出 | pass |
