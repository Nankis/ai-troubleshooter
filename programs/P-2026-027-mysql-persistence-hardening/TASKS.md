# TASKS

## Task 1: [x] 修正 storage 打开策略

- `DB_DRIVER=mysql` 缺少 `DB_DSN` 时返回错误。
- `DB_DRIVER=memory` 需要显式设置。
- memory driver 携带 DSN 时返回错误，避免误以为会落库。

## Task 2: [x] 补充单元测试

- 覆盖 mysql 缺 DSN、显式 memory、memory 携带 DSN、未知 driver。

## Task 3: [x] 更新文档强约束

- README、CONTRIBUTING、Gateway security、Web workbench 不再宣传“未配置自动内存”。

## Task 4: [x] 本地 MySQL 现场验证

- 应用 migration。
- 以 MySQL store 启动 Web 工作台。
- 通过 UI 手动录入经验。
- 查询 MySQL 表确认落库。
- 重启服务后确认记录仍可读。

## Task 5: [x] 提交前检查

- secret scan 通过。
- `git diff --check` 通过。
- commit/push 到 main 由本轮最终操作完成。
