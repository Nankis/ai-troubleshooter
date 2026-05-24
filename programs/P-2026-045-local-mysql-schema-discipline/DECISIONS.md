# DECISIONS

## D1. 本地平台库固定为 `ai_troubleshooter`

历史上为隔离验证创建 `ai_troubleshooter_*`，短期减少污染，但长期导致环境不可控。之后本地平台持久化验证统一复用 `ai_troubleshooter`，用 case/status/test data 区分验证，而不是每个 Program 建新 schema。

## D2. 非 canonical 本地平台库必须显式开关

如果确实需要隔离实验，必须设置 `ALLOW_NON_CANONICAL_LOCAL_DB=true`，并在 Program 记录 cleanup plan。默认脚本和服务启动都 fail-fast。

## D3. 业务只读 adapter 不默认创造临时库名

health-food readonly adapter 需要业务方显式提供 `HEALTH_FOOD_MYSQL_DATABASE`，并指向已有只读业务库；不再默认 `hf_troubleshoot_codex`。
