# EVIDENCE

## E1 单元测试

命令：

```bash
make test
```

结果：Go 全包测试、Python decision-engine 测试、根目录 Python 测试均通过。

新增覆盖：

- `ServeChat` 可接收草稿标题并写入 case title。
- `ServeCaseStatus` 支持 PATCH rename 和 DELETE soft delete。
- 删除后 `FindCaseByNo` 返回 not found。

## E2 MySQL migration

命令：

```bash
MYSQL_HOST=127.0.0.1 MYSQL_PORT=3306 MYSQL_USER=root MYSQL_PASSWORD=*** MYSQL_DATABASE=ai_troubleshooter make migrate-mysql
```

结果：

```text
apply 006_web_case_session_management.sql
migrations applied to ai_troubleshooter
```

确认：

```text
tb_troubleshoot_case.case_title column exists = 1
schema_migrations contains 006_web_case_session_management.sql
```

## E3 Web smoke

启动：

```bash
APP_ENV=dev HTTP_PORT=18088 DB_DRIVER=mysql DB_DSN=*** CONNECTOR_MODE=mock GATEWAY_AUTH_ENABLED=false CONTROL_API_AUTH_ENABLED=false LLM_PROVIDER=local_rules VISION_PROVIDER=local_rules go run ./cmd/dev-server
```

浏览器验证：

- 点击“新对话”，草稿数增加。
- 刷新页面后草稿仍在，证明草稿本地持久化生效。
- 点击草稿重命名按钮，页面内弹窗标题为“重命名问题”。
- 点击草稿删除按钮，页面内弹窗标题为“删除问题会话”，确认后草稿消失。
- 创建正式 case `case_20260524_000002`，PATCH 重命名为“会话管理验证标题-已重命名”，刷新 Web 后左侧可见新标题。
- 在 Web 点击正式 case 删除并确认后，case 从左侧列表消失。
- MySQL 确认删除时 `tb_troubleshoot_case.status = 0`。

验证结束后已清理本地 smoke case 和草稿，避免污染工作台。

## E4 静态检查

命令：

```bash
make secret-scan
git diff --check
```

结果：全部通过。
