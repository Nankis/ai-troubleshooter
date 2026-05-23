# EVIDENCE

## 索引

| 编号 | 类型 | 说明 |
| --- | --- | --- |
| EV-001 | code inspection | health-food 已有 `/food-health/sys/admin/search-logs` 和 `/log-dates`。 |
| EV-002 | unit | `go test ./internal/connectors`。 |
| EV-003 | unit | `python3.13 -m unittest tests/test_real_health_food_readonly_adapter.py tests/test_mcp_readonly_adapter.py`。 |
| EV-004 | runtime | fake production health-food log API + local adapter + local Gateway 三进程实际调用成功。 |
| EV-005 | runtime-negative | 无 Bearer、超时间窗、非法 service_name 负向 case 实际调用成功拦截。 |
| EV-006 | regression | `make test` 全量通过。 |
| EV-007 | hygiene | `git diff --check` 通过。 |
| EV-008 | security | `python3.13 scripts/secret-scan.py --mode all` 通过。 |

## 已执行

```text
go test ./internal/connectors
PASS
```

```text
python3.13 -m unittest tests/test_real_health_food_readonly_adapter.py tests/test_mcp_readonly_adapter.py
PASS
```

```text
make test
PASS
```

```text
git diff --check
PASS
```

```text
python3.13 scripts/secret-scan.py --mode all
PASS
```

```text
fake production health-food log API: http://127.0.0.1:19191
local readonly adapter: http://127.0.0.1:19084
local Gateway: http://127.0.0.1:18088

POST /tools/search_logs_by_service/invoke
agent_id=health-food-readonly-agent
chat_id=oc_health_food_oncall
service_name=health-food
keyword=generateDailyFoodRecommend

PASS:
- status=success
- total=1
- trace_id=trace_prod_demo_1
- password/email/phone/token 均未出现在 Gateway 响应中
```

```text
negative runtime checks:
- no bearer -> 401
- time range > 30 minutes -> 400
- service_name not in adapter allowlist -> failed with readonly BAD_REQUEST
PASS
```

## 待补

- 用户提供生产凭据后，执行真实生产查询验收。
