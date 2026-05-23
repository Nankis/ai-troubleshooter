# HANDOFF

当前排障平台已具备本地 adapter 到 health-food admin log search 的桥接能力，并已用 fake production log API + 本地 adapter + 本地 Gateway 完成实际 HTTP 链路验证。下一步如果用户提供生产 `HEALTH_FOOD_ADMIN_BASE_URL`、`HEALTH_FOOD_ADMIN_SECRET` 和具体问题时间窗，就执行真实生产验证。

启动参数不要写入文件，全部通过环境变量传入。
