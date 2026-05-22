# ERRORS

## E1：可空时间字段返回空字符串导致工具失败

- 现象：`get_health_food_recommendation_status` 第一次 Web Chat 联调时返回失败，错误为 Go 侧解析空字符串时间失败。
- 根因：mock adapter 在推荐未生成时把 `generated_at` 写成 `""`，但 connector 类型是 `time.Time`，JSON 解码要求 RFC3339。
- 修复：把 `HealthFoodRecommendationStatus.GeneratedAt` 改为 `*time.Time`，mock adapter 返回 `null`，文档示例同步改为 `null`。
- 防复发：业务 adapter 对可空时间必须返回 `null` 或省略字段，不允许返回空字符串；schema 文档应明确 nullable。

## E2：health-food 历史 DDL 不能盲目全量执行

- 现象：联调准备本地库时发现历史 DDL 中 `tb_payment_order` 有重复建表版本。
- 根因：`20260204/ddl.sql` 和 `ai-model.sql` 都包含 `tb_payment_order`，不是一份可从空库顺序执行的完整 migration。
- 修复：本轮只取 `20260204/ddl.sql` 中 `tb_membership_products` 创建，再执行当前主 DDL 和后续 alter。
- 防复发：业务服务接入排障平台时，应提供一份干净、可重复初始化的 readonly 测试 schema 或 adapter mock 数据，不要让联调依赖人工挑选历史 DDL。
