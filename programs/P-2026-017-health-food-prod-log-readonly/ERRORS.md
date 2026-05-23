# ERRORS

## E1. 现有 adapter 只支持本地日志文件

之前 `real-health-food-readonly-adapter.py` 的日志搜索只读 `HEALTH_FOOD_LOG_PATH`，无法查询生产 health-food 日志。已补 `HEALTH_FOOD_ADMIN_BASE_URL` + `HEALTH_FOOD_ADMIN_SECRET` 的 upstream 查询。

## E2. HTTP connector 查询参数默认 PascalCase

Go `LogQuery`、`KlineQuery`、`AssetQuery` 原先没有 JSON tag，业务方 adapter 会收到不符合规范的字段名。已补 snake_case JSON tag，并为日志查询补测试。
