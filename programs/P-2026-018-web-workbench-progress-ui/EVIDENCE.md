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
