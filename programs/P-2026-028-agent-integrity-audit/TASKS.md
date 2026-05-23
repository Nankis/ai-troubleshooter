# TASKS

## Task 1: [x] 盘点入口规则和历史错误

- 读取 `AGENTS.md`、`docs/LESSONS.md`。
- 扫描所有 Program `ERRORS.md`。
- 汇总重复错误类型。

## Task 2: [x] 扫描 mock/memory/未验证项

- 扫描 Program Evidence/Result 中的 `DB_DRIVER=memory`、`mock`、`fake`、`未验证项`。
- 区分可接受的本地 smoke 和会污染结论的持久化/真实验收。

## Task 3: [x] 补跑核心 MySQL-backed 现场验证

- Web UI 经验创建、预览、编辑、删除。
- Web Chat 创建 case 并写入消息、AI 决策日志、tool audit。
- Root cause 回填触发 knowledge evolution。

## Task 4: [x] 更新规则文档

- 更新 `AGENTS.md`。
- 更新 `docs/LESSONS.md`。
- 修正 README 组件状态中容易过度承诺的表述。

## Task 5: [x] 最终验证

- `make test`
- `go vet ./...`
- `make secret-scan`
- `git diff --check`
- commit + push main 由本轮最终操作完成。
