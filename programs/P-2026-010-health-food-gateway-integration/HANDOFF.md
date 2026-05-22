# HANDOFF

当前状态：本 Program 已完成。`health-food` 本地服务、mock readonly adapter、排障平台 dev-server、Web Chat、Gateway、tool audit 均已验证。

下一步：

1. 如果要接真实 `health-food`，按 `docs/business-service-registration.md` 和 `configs/business-capabilities.health-food.example.yaml` 在业务侧实现 readonly adapter。
2. 清理或固化 `health-food` 本地 schema 初始化顺序，避免历史 DDL 重复建表。
3. 后续如要做动态能力注册，新建 Program，把 manifest 接入 Gateway registry，并保留默认拒绝、scope 审核和审计。
