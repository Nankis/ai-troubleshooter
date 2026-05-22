# 经验沉淀与自进化闭环

本系统的一期自进化不是“自动改代码”，而是把每次排障的人工确认结果沉淀为可查询、可复用、可持续更新的知识条目。业务方只要回填最终根因，平台就会自动更新经验库。

## 数据表

核心表：

- `tb_troubleshoot_root_cause`：人工确认的最终根因。
- `tb_troubleshoot_case_feedback`：业务方对 AI 排障结果的反馈。
- `tb_troubleshoot_knowledge_item`：可复用排障经验。
- `tb_troubleshoot_knowledge_evolution_run`：每次知识演进的运行记录。
- `tb_troubleshoot_ai_decision_log`：AI 分类、实体抽取、工具选择、工具调用、总结和失败收敛的决策轨迹。

DDL：

- `migrations/001_initial.sql`
- `migrations/002_knowledge_evolution.sql`
- `migrations/003_ai_decision_logs.sql`
- `migrations/004_case_idempotency.sql`

## 直接 SQL 示例

生产推荐通过 API 写入，以下 SQL 用于 DBA、数据校验或临时排查。

插入/更新人工根因：

```sql
INSERT INTO tb_troubleshoot_root_cause (
  case_id,
  ai_predicted_reason,
  human_confirmed_reason,
  root_cause_category,
  owner_service,
  owner_team,
  is_external_source_issue,
  prevention_action,
  confirmed_by,
  confirmed_at,
  create_time,
  update_time
) VALUES (
  ?,
  ?,
  ?,
  ?,
  ?,
  ?,
  TRUE,
  ?,
  ?,
  NOW(),
  NOW(),
  NOW()
) ON DUPLICATE KEY UPDATE
  ai_predicted_reason = VALUES(ai_predicted_reason),
  human_confirmed_reason = VALUES(human_confirmed_reason),
  root_cause_category = VALUES(root_cause_category),
  owner_service = VALUES(owner_service),
  owner_team = VALUES(owner_team),
  is_external_source_issue = VALUES(is_external_source_issue),
  prevention_action = VALUES(prevention_action),
  confirmed_by = VALUES(confirmed_by),
  confirmed_at = VALUES(confirmed_at),
  update_time = VALUES(update_time);
```

查询某 case 的自进化结果：

```sql
SELECT
  r.run_no,
  r.decision,
  r.created_knowledge_item,
  r.updated_knowledge_item,
  k.title,
  k.confidence,
  k.observed_case_count
FROM tb_troubleshoot_knowledge_evolution_run r
LEFT JOIN tb_troubleshoot_knowledge_item k ON k.id = r.knowledge_item_id
WHERE r.case_id = ?
ORDER BY r.id DESC;
```

查询某 case 的 AI 决策轨迹：

```sql
SELECT
  decision_type,
  reason,
  selected_tools_json,
  decision_status,
  latency_ms,
  error_message,
  create_time
FROM tb_troubleshoot_ai_decision_log
WHERE case_id = ?
ORDER BY id;
```

查询可复用知识：

```sql
SELECT
  id,
  title,
  issue_domain,
  issue_type,
  last_root_cause_category,
  confidence,
  observed_case_count,
  recommended_steps_json,
  useful_tools_json
FROM tb_troubleshoot_knowledge_item
WHERE knowledge_status = 'active'
  AND issue_domain = ?
  AND (? IS NULL OR issue_type = ?)
ORDER BY confidence DESC, observed_case_count DESC, update_time DESC
LIMIT 20;
```

## 自进化触发器

当前默认触发器：

```text
业务 owner 回填 root cause
  -> 写入 tb_troubleshoot_root_cause
  -> 生成/更新 tb_troubleshoot_knowledge_item
  -> 写入 tb_troubleshoot_knowledge_evolution_run
  -> case 标记为 DONE
```

后续可新增定时触发器：

```text
每日扫描 DONE cases
  -> 聚合相同 issue_domain / issue_type / root_cause_category
  -> 调整 confidence、recommended_steps、useful_tools
  -> 产出候选 SOP 或工具改进建议
```

## API

以下接口在 `cmd/dev-server` 的一体化服务中已实现。生产拆分时建议放到 Agent 平台的 case API 服务，由 Python decision-engine 读取并写回决策结果。

### 查询 case

```text
GET /cases/{case_no}
```

返回：

```json
{
  "case": {},
  "entities": [],
  "messages": [],
  "root_cause": {},
  "evolution_runs": [],
  "tb_troubleshoot_ai_decision_log": []
}
```

### 回填根因并触发自进化

```text
POST /cases/{case_no}/root-cause
```

请求：

```json
{
  "ai_predicted_reason": "行情源延迟或聚合补偿未完成",
  "human_confirmed_reason": "Binance 行情源 20:02-20:04 延迟，补偿任务完成前用户看到旧 high",
  "root_cause_category": "external_source_delay",
  "owner_service": "market-service",
  "owner_team": "market-team",
  "is_cache_issue": false,
  "is_data_sync_issue": false,
  "is_external_source_issue": true,
  "is_frontend_display_issue": false,
  "is_user_misunderstanding": false,
  "fix_action": "补偿该时间段 K线并刷新缓存",
  "prevention_action": "增加行情源延迟监控和补偿任务告警",
  "confirmed_by": "owner_1"
}
```

返回：

```json
{
  "case": {},
  "root_cause": {},
  "knowledge_item": {},
  "evolution_run": {}
}
```

字段要求：

- `human_confirmed_reason` 必填。
- `root_cause_category` 必填，建议使用稳定枚举。
- `owner_service`、`owner_team` 建议填写，便于后续统计责任域。
- 多个 `is_*` 布尔字段可并存，但不要滥填。

### 查询根因

```text
GET /cases/{case_no}/root-cause
```

### 写入人工反馈

```text
POST /cases/{case_no}/feedback
```

请求：

```json
{
  "rating": 4,
  "ai_useful": true,
  "wrong_conclusion": false,
  "missing_key_information": "",
  "missing_tools_json": "[\"get_order_detail\"]",
  "comment": "方向对，但缺少订单工具",
  "created_by": "owner_1"
}
```

### 查询人工反馈

```text
GET /cases/{case_no}/feedback
```

### 查询自进化运行记录

```text
GET /cases/{case_no}/evolution-runs
```

### 查询 AI 决策日志

```text
GET /cases/{case_no}/ai-decisions?limit=100
```

### 查询知识库

```text
GET /knowledge?issue_domain=kline&issue_type=价格不一致&root_cause_category=external_source_delay&limit=20
```

返回：

```json
{
  "items": [
    {
      "id": 1,
      "title": "kline / 价格不一致 / external_source_delay",
      "issue_domain": "kline",
      "issue_type": "价格不一致",
      "required_fields_json": "[\"symbol\",\"interval\",\"abnormal_time\",\"issue_type\",\"compare_exchange\"]",
      "recommended_steps_json": "[]",
      "common_causes_json": "[]",
      "useful_tools_json": "[]",
      "success_case_ids_json": "[1]",
      "confidence": 0.55,
      "observed_case_count": 1,
      "last_root_cause_category": "external_source_delay",
      "last_confirmed_reason": "Binance 行情源延迟",
      "status": "active"
    }
  ]
}
```

## 知识条目演进规则

当前代码规则在 `internal/evolution`：

1. 以 `issue_domain + issue_type + root_cause_category` 作为知识条目聚合 key。
2. 如果不存在知识条目，则创建新条目。
3. 如果已存在，则追加成功 case id、常见原因、推荐步骤、有效工具。
4. `observed_case_count` 随成功 case 数增加。
5. `confidence` 按样本数分段提升：
   - 1 个 case：0.55
   - 2-4 个 case：0.65
   - 5-9 个 case：0.78
   - 10 个及以上：0.90
6. 每次演进都写 `tb_troubleshoot_knowledge_evolution_run`，便于审计和回滚。

## 建议 root_cause_category 枚举

K线/行情：

- `external_source_delay`
- `external_source_gap`
- `kline_aggregation_delay`
- `kline_cache_stale`
- `frontend_chart_render`
- `user_compare_exchange_mismatch`

资产/余额：

- `asset_freeze_pending`
- `asset_event_delay`
- `asset_snapshot_stale`
- `trade_settlement_delay`
- `deposit_withdraw_pending`
- `frontend_balance_display`
- `user_misunderstanding`

通用：

- `recent_deployment_regression`
- `config_change`
- `data_sync_delay`
- `unknown_need_human_followup`

## 给后续 AI 的要求

后续 AI 继续开发时必须保持这个闭环：

- 新增业务域时，同步更新：
  - 最小必要字段；
  - tool registry；
  - connector 接口规范；
  - root cause category；
  - knowledge evolution recommended steps；
  - 单元测试和 smoke。
- 任何 AI 结论不能直接写 root cause，root cause 必须来自人工确认或明确可信系统。
- 任何自进化结果必须可审计、可查询、可回滚。
- 如果修改 DDL，必须同时更新 migration、Go model、store、API 文档和测试。
