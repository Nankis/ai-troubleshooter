# EVIDENCE

## 索引

| 编号 | 类型 | 说明 |
| --- | --- | --- |
| EV-001 | unit | `go test ./internal/webchat ./internal/caseflow ./internal/storage/mysql` 通过。 |
| EV-002 | browser | 启动 `go run ./cmd/dev-server`，打开 `http://127.0.0.1:18088/web`。 |
| EV-003 | browser | 提交 health-food 问题后，页面显示 case、agent 输出、右侧进度和 10 条决策日志摘要。 |
| EV-004 | browser | 手动录入知识后左侧可见，删除后列表隐藏。 |
| EV-005 | browser | 390x844 移动视口可用，主输入和对话不重叠。 |
| EV-006 | regression | `make test` 全量通过。 |
| EV-007 | hygiene | `git diff --check` 通过。 |
| EV-008 | security | `python3.13 scripts/secret-scan.py --mode all` 通过。 |

## 命令

```text
go test ./internal/webchat ./internal/caseflow ./internal/storage/mysql
PASS
```

```text
make test
PASS
```

```text
git diff --check
PASS
```

```text
python3.13 scripts/secret-scan.py --mode all
PASS
```

## 浏览器验证摘要

- 左侧显示 14 个 Gateway tools。
- 输入 `health-food uid 123 今日推荐没有生成...` 后生成 `case_20260523_000001`。
- 中间显示用户消息和 Agent 排查结果。
- 右侧显示 `NEED_HUMAN_CONFIRMATION`、7 个进度步骤和环境信息。
- 手动知识 `health-food 推荐缺失排查` 保存后可见，删除后隐藏。

## 2026-05-23 二次真实点击复验

启动命令：

```text
APP_ENV=dev HTTP_PORT=18088 DB_DRIVER=memory CONNECTOR_MODE=mock GATEWAY_AUTH_ENABLED=false CONTROL_API_AUTH_ENABLED=false LLM_PROVIDER=local_rules VISION_PROVIDER=local_rules go run ./cmd/dev-server
```

已通过浏览器打开 `http://127.0.0.1:18088/web` 并逐项点击：

- 点击 `刷新工作台`：左侧展示 recent cases、14 个 Gateway tools、平台知识为空。
- 点击 `录入` 并保存知识：`Web validation knowledge daily recommendation missing` 保存后左侧和右侧知识计数变为 1。
- 点击 `删除经验`：知识列表恢复 `暂无沉淀经验`，右侧知识计数变为 0。
- 在输入框提交 health-food 文本问题：生成 `case_20260523_000001`，状态进入 `NEED_HUMAN_CONFIRMATION`。
- 点击 case 列表重新打开 case：中间消息区展示 agent 结论，右侧展示 7 个进度步骤和最新决策日志摘要。
- 点击 `返回`：消息区滚动位置回到顶部。
- 点击 `新对话`：主输入区重置为 ready，左侧历史 case 保留。
- 图片 case 通过同一 `POST /web/api/chat` multipart 链路上传测试图片生成 `case_20260523_000002`；随后在 Web UI 中点击该 case、补充缺失时间、点击发送，最终进入 `NEED_HUMAN_CONFIRMATION`。
- 390x844 移动视口复验：主输入和内容无横向溢出。
- 桌面 1440x900 视口复验：三栏可见，无横向溢出。
- 浏览器 console error/warning 均为空。

图片上传说明：当前 Browser 自动化运行时不能接管 macOS 原生文件选择器，也不暴露 `setInputFiles/File/DataTransfer`。因此本次对图片链路采用真实 multipart 后端上传，再回到 Web UI 点击 case、补充信息和查看进度；后端和 UI 展示链路通过，原生文件选择器需要人工或更完整 Playwright/Chrome 自动化继续覆盖。
