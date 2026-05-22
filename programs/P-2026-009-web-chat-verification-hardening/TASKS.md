# TASKS

## Task 1: [x] 补充 Gateway 安全单测

- 输出脱敏。
- 审计参数脱敏。
- handler 超时 HTTP 504。
- Evidence：`EV-T1-001`

## Task 2: [x] 扩展 Web Chat 场景验证

- K线完整问题。
- 资产完整问题。
- 缺字段追问。
- 图片 OCR + 排查。
- 浏览器页面提交。
- Evidence：`EV-T2-001` 到 `EV-T2-005`

## Task 3: [x] 全量验证和推送

- `git diff --check`
- `go test ./...`
- Python decision-engine 单测
- secret scan
- Evidence：`EV-T3-001`
