# EVIDENCE

## 命令与场景

| Evidence ID | 时间 | 验证 | 结果 |
| --- | --- | --- | --- |
| EV-T1-001 | 2026-05-23 | `go test ./internal/gateway ./internal/decisionbaseline ./internal/masking ./internal/httpauth` | pass |
| EV-T2-001 | 2026-05-23 | Web API K线完整问题 | pass：`case_20260523_000008`，`NEED_HUMAN_CONFIRMATION`，domain=`kline`，tool_count=2 |
| EV-T2-002 | 2026-05-23 | Web API 资产完整问题 | pass：`case_20260523_000009`，`NEED_HUMAN_CONFIRMATION`，domain=`asset`，tool_count=2 |
| EV-T2-003 | 2026-05-23 | Web API 缺字段问题 | pass：`case_20260523_000010`，`WAITING_USER_REPLY`，missing=`symbol, interval, abnormal_time`，tool_count=0 |
| EV-T2-004 | 2026-05-23 | Web API 图片上传 + Qwen-VL OCR | pass：`case_20260523_000011`，OCR 识别到 `BTCUSDT`，tool_count=4 |
| EV-T2-005 | 2026-05-23 | Browser 页面提交 | pass：`case_20260523_000012` 页面输入并点击提交成功；`case_20260523_000013` 坐标点击提交成功 |
| EV-T2-006 | 2026-05-23 | MySQL 落库计数 | pass：cases=13、messages=37、ai_decision_logs=90、tool_audits=29 |
| EV-T3-001 | 2026-05-23 | `git diff --check` | pass |
| EV-T3-002 | 2026-05-23 | `go test ./...` | pass |
| EV-T3-003 | 2026-05-23 | `PYTHONPATH=apps/decision-engine python3.13 -m unittest discover -s apps/decision-engine/tests -p 'test_*.py'` | pass |
| EV-T3-004 | 2026-05-23 | `python3.13 scripts/secret-scan.py --mode all` | pass |

## 说明

- 坐标级 `type` 受当前 Browser 插件虚拟剪贴板能力限制；最终采用页面输入框填充后坐标点击提交，验证真实 Web 页面提交链路。
- 本地服务使用 mock Gateway 和本地 MySQL 验证；未调用真实业务下游。
