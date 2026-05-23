# EVIDENCE

## 索引

| 编号 | 类型 | 说明 |
| --- | --- | --- |
| EV-T1-001 | code | `web/static/index.html` 增加 `groupToolsByService`、`toolService` 和分组 UI。 |
| EV-T2-001 | unit | `go test ./...` 通过。 |
| EV-T2-002 | browser | 本地启动 dev-server，打开 `http://127.0.0.1:18088/web`，左侧 tools 显示 `health-food(4)`、`asset-service(2)`、`market-service(4)`、`ops/logs(3)`、`platform(1)`，总数仍为 14，无横向溢出。 |

## 浏览器验证

启动命令：

```text
APP_ENV=dev HTTP_PORT=18088 DB_DRIVER=memory CONNECTOR_MODE=mock GATEWAY_AUTH_ENABLED=false CONTROL_API_AUTH_ENABLED=false LLM_PROVIDER=local_rules VISION_PROVIDER=local_rules go run ./cmd/dev-server
```

截图：`/tmp/ai-troubleshooter-tools-grouped.png`
