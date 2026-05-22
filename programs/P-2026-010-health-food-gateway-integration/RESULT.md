# RESULT

## 结果摘要

- 本地启动了 `health-food`，使用独立 MySQL 库和 18080 端口完成探活。
- 在排障平台新增 `health_food` 故障域、4 个 health-food 只读工具、HTTP/mock connector、默认 policy scope 和 Web Chat 规则编排。
- 新增本地 `mock-health-food-readonly-adapter.py`，可模拟 `recommendation_missing` 和 `quota_exhausted` 两类故障，并通过标准 readonly envelope 接入 Gateway。
- 补齐业务服务注册 manifest 文档和 health-food 示例配置。
- 用 curl 和浏览器页面分别跑通 Web Chat / Case / Decision / Gateway / Adapter / Audit 链路。

## 变更范围

- `internal/caseflow`：新增 `health_food` domain 和必要字段检查。
- `internal/llm`、`internal/decisionbaseline`：新增 health-food 分类、实体抽取、工具选择和参数构造。
- `internal/connectors`、`internal/gateway`、`internal/policy`：新增 health-food connector、工具注册和授权 scope。
- `scripts/mock-health-food-readonly-adapter.py`：本地 readonly adapter。
- `docs/business-service-registration.md`、`configs/business-capabilities.health-food.example.yaml`：业务服务能力注册数据结构。
- `README.md`、`docs/ai-connector-integration.md`、`configs/config.example.yaml`：同步入口文档和配置。

## 任务完成情况

| Task | 状态 | Evidence ID |
| --- | --- | --- |
| T1 | done | EV-T1-001, EV-T1-002 |
| T2 | done | EV-T2-001 |
| T3 | done | EV-T3-001 |
| T4 | done | EV-T4-001, EV-T4-002, EV-T4-003, EV-T4-004, EV-T4-005, EV-T4-006 |
| T5 | done | EV-T5-001 |
| T6 | done | EV-T6-001, EV-T6-002, EV-T6-003 |

## 验证摘要

- `health-food` 本地 JAR：pass，18080 `/food-health/sys/alive` 返回 `0`。
- 推荐缺失 mock：pass，定位 `meal_data_fingerprint` 未刷新。
- AI 配额耗尽 mock：pass，定位 `daily_chat_count=30/30`。
- 缺 uid：pass，Agent 追问且 `tool_count=0`。
- 浏览器 Web Chat：pass，页面展示 case、工具调用和决策日志。
- `make test`：pass。
- `git diff --check`：pass。
- `python3.13 scripts/secret-scan.py --mode all`：pass。

## 验收覆盖

| 验收标准 | 结论 | Evidence ID |
| --- | --- | --- |
| health-food 可本地启动 | pass | EV-T1-001 |
| 业务服务注册数据结构已梳理 | pass | EV-T5-001 |
| 可通过 Gateway 调 health-food 只读能力 | pass | EV-T4-001, EV-T4-002 |
| mock 故障可完整排查 | pass | EV-T4-001, EV-T4-002, EV-T4-004 |
| 缺字段不查下游 | pass | EV-T4-003 |
| 鉴权、审计、测试和 secret scan 覆盖 | pass | EV-T4-005, EV-T4-006, EV-T6-001, EV-T6-003 |

## Commit

- 本 Program 随本次提交交付；最终 hash 以 `git log` 为准。

## 残留风险

- 本轮没有修改 `health-food` 仓库本身；真实接入生产前，业务方仍需按 manifest 提供正式 readonly adapter。
- `health-food` 历史 DDL 存在重复 `tb_payment_order` 版本，联调时需要明确初始化顺序或补一份干净的本地 schema。
- health-food 当前业务接口依赖登录态，不适合作为 Agent 直接查询入口；排障平台应继续通过 readonly adapter 访问证据。
- 当前 health-food 工具是静态注册，后续可以把 manifest 做成运行时 registry，但仍要保持默认拒绝和 scope 审核。
