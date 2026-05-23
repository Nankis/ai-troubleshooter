# Result

## 完成内容

- 已把 Go/Python/SQL 安全规范写入 `AGENTS.md`，并明确 raw SQL 的准入条件。
- 已把 `scripts/real-health-food-readonly-adapter.py` 从 MySQL CLI/f-string SQL 改为 PyMySQL 参数绑定。
- 已把 Go MySQL 层的动态列表查询抽出 builder，并新增单测证明注入 payload 不进入 SQL 文本。
- 已补 Python adapter 单测，防止回归到字符串拼 SQL。
- 已补本地 runbook 的 PyMySQL 安装步骤。

## 审计结论

- 用户指出的 `CreateCase` 插入语句本身使用 `?` 参数绑定，不属于注入点。
- 真正不符合规范的是 Python 真实 adapter 里旧的 f-string SQL，已修复。
- Go 层保留 raw SQL 但限制在 repository 层，所有外部输入使用占位符；动态 SQL 只拼接代码内白名单片段，并有测试覆盖。

## 验证

- `make test`：pass。
- `python3.13 -m py_compile scripts/real-health-food-readonly-adapter.py && python3.13 -m unittest tests/test_real_health_food_readonly_adapter.py`：pass。
- `make secret-scan`：pass。
- `go vet ./...`：pass。
- `git diff --check`：pass。

## 待完成

- 无。
