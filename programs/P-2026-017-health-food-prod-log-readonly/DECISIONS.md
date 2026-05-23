# DECISIONS

## D1. 不让 Agent 直连生产 DB

生产排障优先查询生产接口和日志证据。DB 只允许由业务方提供的受控 readonly API 间接访问，本项目不在本地保存生产 DB 凭据。

## D2. 复用 health-food 现有日志查询能力

health-food 已有 `/food-health/sys/admin/search-logs`，本轮不急着改业务仓库。排障平台用本地 adapter 把它转换成标准 `POST /v1/readonly/ops/logs/search`。

## D3. Gateway contract 使用 snake_case

公司 readonly adapter 规范统一使用 snake_case。Go 查询结构体补 JSON tag，避免普通 HTTP connector 发出 `ServiceName/StartTime` 这种 PascalCase 字段。

## D4. mock 只能证明链路，不等于生产验收

本地 fake production log API 可以验证 adapter/gateway 的 HTTP 链路、安全边界和脱敏，但生产验收必须实际命中生产 health-food 日志接口。
