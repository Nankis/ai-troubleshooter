# DECISIONS

## D1: 本轮先前端推断服务

当前 `tool.Spec` 已有 `backend_handler`、`required_scope` 和 `name`，足够在 Web 工作台中推断服务分组。本轮不改 API，避免影响 Gateway 协议。

## D2: 默认全展开

服务分组默认全展开，方便用户直接看到服务下提供了哪些 tools。后续 tools 数量很大时再加折叠和搜索。

## D3: 服务排序固定

排序为 `health-food`、`asset-service`、`market-service`、`ops/logs`、`platform`、`other`，让业务工具优先于平台工具展示。
