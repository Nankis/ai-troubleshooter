# TASKS

## Task 1: [x] 建立 Program 和真实验收标准

- 明确上一轮 mock 验证不等于业务真实验收。
- Evidence：`EV-T1-001`

## Task 2: [x] 梳理 health-food 真实 API / DB / 日志 / 代码证据

- 注册/登录接口。
- 关键业务表。
- 日志位置和关键字。
- 推荐/餐食/AI 配额相关代码路径。
- Evidence：`EV-T2-001`

## Task 3: [x] 实现真实 readonly adapter

- 查询 health-food 本地 DB。
- 探活真实 health-food 服务。
- 可读日志摘要。
- 可返回 service_name / suspect_area 供本地代码检查。
- Evidence：`EV-T3-001`

## Task 4: [x] 启动真实服务并跑端到端验证

- health-food 本地服务。
- real adapter。
- ai-troubleshooter dev-server。
- 通过平台 case API 或 Web Chat 完整排查。
- Evidence：`EV-T4-001`

## Task 5: [x] 记录证据、测试、提交推送

- `make test`
- `git diff --check`
- `python3.13 scripts/secret-scan.py --mode all`
- Evidence：`EV-T5-001`
