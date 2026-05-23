# EVIDENCE

## E1 单元测试

命令：

```bash
make test
```

结果：Go 全包测试、Python decision-engine 测试、根目录 Python 测试全部通过。

覆盖点：

- `internal/capability`：HTTP manifest readonly candidate、危险能力 rejected、Claude/Cursor `mcpServers` pending discovery、MCP route readonly path、YAML manifest。
- `internal/gateway`：动态 capability 注册后可通过 Gateway 调用 readonly adapter；非 readonly safety 不能注册。
- `internal/webchat`：overview 返回 capabilities；import/publish API 触发 reload。
- `internal/storage` / `internal/tool`：memory capability store 注入、registry unregister。

## E2 MySQL migration

命令：

```bash
MYSQL_HOST=127.0.0.1 MYSQL_PORT=3306 MYSQL_USER=root MYSQL_PASSWORD=*** MYSQL_DATABASE=ai_troubleshooter make migrate-mysql
```

结果：

```text
apply 005_dynamic_capability_registry.sql
migrations applied to ai_troubleshooter
```

确认：

```text
schema_migrations contains 005_dynamic_capability_registry.sql
tb_troubleshoot_tool_registry dynamic columns count = 4
```

## E3 Web/API/Gateway 实跑

启动：

```bash
APP_ENV=dev HTTP_PORT=18088 DB_DRIVER=mysql DB_DSN=*** CONNECTOR_MODE=mock GATEWAY_AUTH_ENABLED=false CONTROL_API_AUTH_ENABLED=false LLM_PROVIDER=local_rules VISION_PROVIDER=local_rules go run ./cmd/dev-server
```

外部 readonly adapter 使用本地 mock server 暴露 `/v1/readonly/demo/status`。

验证链路：

- Web 工作台打开 `http://127.0.0.1:18088/web`，能看到“能力接入”和“已注册 Gateway Tools”。
- 通过 import API 导入 `get_demo_status_smoke_202605240018`，状态为 `draft + readonly_candidate`。
- 在 Web 端点击发布后，工具数从 14 变为 15，工具列表 `other` 分组出现 `get_demo_status_smoke_202605240018`。
- 通过 Gateway 调用：

```bash
curl -sS -X POST http://127.0.0.1:18088/tools/get_demo_status_smoke_202605240018/invoke \
  -H 'Content-Type: application/json' \
  -d '{"case_id":"case_dynamic_smoke","agent_id":"business-troubleshooter-v1","caller_user_id":"web_user","chat_id":"web-local","arguments":{"uid":"u-smoke"}}'
```

结果：

```json
{
  "status": "success",
  "summary": "get_demo_status_smoke_202605240018 returned from dynamic-demo-adapter",
  "data": {
    "path": "/v1/readonly/demo/status",
    "status": "ok",
    "uid": "u-smoke"
  }
}
```

MySQL 确认：

```text
tb_troubleshoot_tool_registry: enabled / readonly_candidate / approved / /v1/readonly/demo/status
tb_troubleshoot_tool_call_audit allowed count = 1
```

验证结束后已清理本地 demo/danger smoke 数据，避免污染用户后续本地工作台。

## E4 危险能力负向验证

导入 `delete_demo_cache_smoke`：

```json
{
  "tool_status": "rejected",
  "safety_status": "rejected",
  "safety_reasons_json": [
    "dangerous action keyword: delete",
    "readonly http path must be under /readonly/"
  ]
}
```

发布被拒绝：

```text
HTTP 400
capability safety status is rejected; only readonly_candidate can be published
```

Web 刷新后能看到 rejected 能力，Gateway tools 数仍为 15，危险工具未发布。

## E5 静态检查

命令：

```bash
make secret-scan
git diff --check
```

结果：全部通过。
