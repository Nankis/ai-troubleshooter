# Evidence

## 索引

| ID | 类型 | 证据 | 结论 |
| --- | --- | --- | --- |
| E1 | 静态检查 | `git diff --check` | PASS |
| E2 | 敏感信息扫描 | `make secret-scan` | PASS |
| E3 | 测试范围说明 | 文档级变更，未改 Go/Python 代码 | 未运行 `make test` |

## 覆盖映射

| 需求 | 证据 |
| --- | --- |
| 新手能知道如何使用平台 | `docs/business-onboarding-quickstart.md` |
| LLM/Vision 配置位置和方式 | `docs/business-onboarding-quickstart.md` 第 5 节 |
| 建表和运行服务说明 | `docs/business-onboarding-quickstart.md` 第 4、6 节 |
| 下游接口和 Gateway 接入方式 | `docs/business-onboarding-quickstart.md` 第 8、9 节 |
| 给业务方 AI 的执行提示 | `docs/business-onboarding-quickstart.md` 第 12 节 |

## 未验证项

- 本次为文档变更，未启动 Web/Gateway/业务 adapter 做 L2-L4 验收。
- 未调用真实 LLM、真实 Lark/飞书或真实业务生产接口。
