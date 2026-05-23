# TASKS

## Task 1: [x] 实现 MCP readonly adapter

- Evidence：`EV-T1-001`

## Task 2: [x] 补 health-food MCP 实验服务

- Evidence：`EV-T2-001`

## Task 3: [x] 文档和配置样例

- Evidence：`EV-T3-001`

## Task 4: [x] 实际启动验证

- 启动 MCP server。
- 启动 MCP readonly adapter。
- 启动 dev-server。
- 通过 Gateway tool invoke 调用 health-food MCP tool。
- Evidence：`EV-T4-001`

## Task 5: [x] 测试、提交、推送

- `make test`
- `git diff --check`
- `python3.13 scripts/secret-scan.py --mode all`
- Evidence：`EV-T5-001`
