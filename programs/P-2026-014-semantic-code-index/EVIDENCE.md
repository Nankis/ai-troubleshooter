# EVIDENCE

## Evidence 索引

| Evidence ID | 类型 | 关联任务 | 结论 |
| --- | --- | --- | --- |
| EV-T1-001 | implementation | T1 | completed：Local Code Agent 输出 resolved call edge |
| EV-T2-001 | implementation | T2 | completed：Java receiver type 和 implements 关系进入证据 |
| EV-T3-001 | test | T3 | passed：Python semantic 单测覆盖接口和实现类 |
| EV-T4-001 | command | T4 | passed：完整测试、diff check、secret scan |

## 命令验证

| Evidence ID | 时间 | 命令 | 结果 | 备注 |
| --- | --- | --- | --- | --- |
| EV-T3-001 | 2026-05-23 | `PYTHONPATH=apps/decision-engine python3.13 -m unittest apps/decision-engine/tests/test_engine.py -v` | passed | 14 个 Python decision-engine 单测通过 |
| EV-T4-001 | 2026-05-23 | `make test` | passed | Go 全量测试 + Python unittest discover 通过 |
| EV-T4-002 | 2026-05-23 | `git diff --check` | passed | 无 whitespace 错误 |
| EV-T4-003 | 2026-05-23 | `python3.13 scripts/secret-scan.py --mode all` | passed | Secret scan passed |

## 现场验证

| Evidence ID | 时间 | 场景 | 证据 | 结论 |
| --- | --- | --- | --- | --- |
| EV-T1-001 | 2026-05-23 | 临时 health-food Java 结构：`RecommendationJob -> IFoodService -> FoodServiceImpl` | `receiver_type=IFoodService`，resolved 到 `IFoodService.generateDailyFoodRecommendWithFingerprint` 和 `FoodServiceImpl.generateDailyFoodRecommendWithFingerprint` | 跨模块调用解析可用 |
| EV-T2-001 | 2026-05-23 | `FoodServiceImpl implements IFoodService` | `implement_relation_count=1`，`analysis_modes` 包含 `interface_implementation` | 接口实现关系可用 |
