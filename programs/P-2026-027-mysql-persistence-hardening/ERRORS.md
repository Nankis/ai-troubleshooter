# ERRORS

## E1: 使用 memory store 验证了需要持久化的 Web 功能

### 现象

Web 工作台可以录入平台经验，但用户手动检查本地 MySQL 时找不到对应数据。

### 根因

- 本地服务多次以 `DB_DRIVER=memory` 启动，UI 写入只进入内存 store。
- `storage.Open` 的历史逻辑允许 `DB_DRIVER=mysql` 但 `DB_DSN` 为空时静默回退到 memory。
- README 和 CONTRIBUTING 中也写了“未配置 DB_DSN 自动使用内存”的误导性说明。

### 防复发

- 代码层 fail-fast：mysql 缺 DSN 直接报错。
- 文档层明确：持久化验收必须用 MySQL。
- Evidence 层要求：涉及平台数据沉淀，必须包含 MySQL 表查询结果或迁移结果。
