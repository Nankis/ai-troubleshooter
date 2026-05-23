# Result

## 当前结果

- 已在 health-food 新分支实现只读排障接口，并通过 ai-troubleshooter Gateway 真实接入。
- 已启动两个本地服务并从 Web Chat 完整提交 4 个真实 case。
- 已从 `ai_troubleshooter` 反查到 case、消息、AI 决策日志和工具调用决策。
- 已从 `meow_pas` 反查真实业务数据，排查结论以实际表数据为准。

## Web 验证 case

| Case | 输入 | 结果 |
| --- | --- | --- |
| `case_20260524_000008` | `uid:2054603630081875968 今日没有每日推荐` | 当天无餐食记录，推荐缺失符合真实数据。 |
| `case_20260524_000005` | `uid:999999999999 推荐数据不准` | 用户不存在，要求反馈方确认正确 uid 后继续。 |
| `case_20260524_000007` | `uid:2054603630081875968 2026-05-23 推荐数据不准` | 推荐记录存在，但 `source_meal_ids` 引用了 `2026-05-14` 餐食，判定 `source_date_mismatch`。 |
| `case_20260524_000009` | `uid:2054603630081875968 今日 token 消耗数量不对` | token 账户真实余额和 daily_chat 数据显示账户健康，需业务 owner 对用户反馈继续确认。 |
| `case_20260524_000010` | `uid:2054603630081875968 今日没有每日推荐，验证uid回写` | 主表 `tb_troubleshoot_case.uid` 已回写为业务 uid，不再保留 `web_user`。 |

## 真实数据断言

- `tb_user_info`：`2054603630081875968` 存在，`999999999999` 不存在。
- `tb_daily_food_recommend`：`2054603630081875968 / 2026-05-23` 有推荐记录。
- `tb_meal_record`：相关餐食 ID 位于 `2026-05-14`，与 `2026-05-23` 推荐日期不一致。
- `tb_user_asset_account`：token 账户余额和 daily_chat_count 为真实 MySQL 数据。

## 截图

- `evidence/screenshots/case-a-daily-missing.png`
- `evidence/screenshots/case-b-missing-uid.png`
- `evidence/screenshots/case-c-source-date-mismatch.png`
- `evidence/screenshots/case-d-token-quota.png`
- `evidence/screenshots/case-e-uid-persistence.png`

## 待完成

- 无。
