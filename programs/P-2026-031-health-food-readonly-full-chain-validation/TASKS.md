# Tasks

## Task 1: [x] health-food 新分支只读接口

- 新分支：`feature/P-2026-009-health-food-readonly`
- 接口路径兼容 Gateway HTTP connector。
- 底层只使用固定 SELECT 查询。

## Task 2: [x] 决策规则补齐

- 推荐不准确归类为 health-food 推荐问题。
- 不存在 uid 时，Agent 输出要求反馈方确认正确 uid。

## Task 3: [x] 本地服务启动

- health-food `18080`。
- ai-troubleshooter `18088`。

## Task 4: [x] Web 端真实验证

- Case A：今日每日推荐缺失。
- Case B：不存在 uid。
- Case C：推荐不准/未按健康目标。
- Case D：token 配额/日志辅助证据。

## Task 5: [ ] 证据回写和提交

- 截图、命令、真实数据断言写入 Evidence/Result。
