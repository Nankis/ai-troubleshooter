# Workflow Tasks

这里可以存放较长 Program 的可执行 task JSON。每个 JSON 必须符合 `.workflow/task.schema.json`。

规则：

- 不写密钥、真实 token、原始生产日志或完整生产响应。
- `requires_real_dependencies=true` 的 task 不能用 mock/memory/local_rules 作为最终通过证据。
- task 完成后，把命令、截图、MySQL 查询和 API 结果摘要写回对应 Program 的 `EVIDENCE.md`。
