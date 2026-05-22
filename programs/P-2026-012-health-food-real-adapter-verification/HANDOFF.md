# HANDOFF

## 当前状态

- Program 已完成。真实 health-food 服务、本地 MySQL、real readonly adapter、排障平台 dev-server、Python decision-engine local code inspection 都已跑通过。
- 关键验证 case：`case_20260523_000002`，Web Chat 页面显示 `NEED_HUMAN_CONFIRMATION`、`health_food`、`每日推荐缺失`。
- 关键证据：真实 DB 有 1 条餐食记录和 1 条餐食明细；当天 `tb_daily_food_recommend` 为 0 行；平台 tool audit 和 AI decision log 已落库。
- 浏览器验证截图：`/tmp/hf_real_webchat_result.png`。

## 下一步

- 接真实 Lark / 飞书 bot 后，用同一个 health-food readonly adapter 跑群聊事件链路。
- 如果公司有日志平台，替换 `HEALTH_FOOD_LOG_PATH` 或实现真实 logs readonly adapter。
- 如需生产接入，业务方应把本地 DB 查询逻辑改造成服务内 readonly API；Agent 平台仍只接 adapter，不直连业务 DB。
