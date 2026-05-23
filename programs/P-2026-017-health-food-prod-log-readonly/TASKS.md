# TASKS

- [x] 检查 health-food 仓库是否已有日志查询接口。
- [x] 确认现有接口：`/food-health/sys/admin/search-logs`。
- [x] 在排障平台 adapter 中接入 health-food admin log upstream。
- [x] 补参数别名兼容、服务名 allowlist、时间窗、limit、超时和脱敏。
- [x] 修复 HTTP connector 查询参数 JSON 为 snake_case。
- [x] 补单元测试覆盖 adapter upstream 和 connector payload。
- [x] 启动本地 fake production log API + adapter + Gateway，实际调用 Gateway 工具。
- [ ] 获得生产 base URL / 只读密钥 / 问题时间窗后，执行真实生产验收。
