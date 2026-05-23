# P-2026-031 health-food Readonly Full Chain Validation

## 背景

用户要求不再用 mock 自欺，必须从 health-food 主分支拉新分支作为真实下游服务，新增只读接口查询本地 MySQL 表数据，并让排障平台 Web 端通过 Gateway 实际咨询问题、实际调用接口、实际得到有用结论，且截图留证。

## 目标

- health-food 新分支提供符合 Gateway 接入规范的 readonly APIs。
- 排障平台通过 `CONNECTOR_MODE=http` 调用 health-food 本地服务，不使用 mock adapter。
- Web Chat 至少验证：
  - 今日每日推荐缺失。
  - 不存在 uid 时要求反馈方确认正确 uid。
  - 推荐不准/未按健康目标推荐。
  - 额外补充 token 配额或日志证据 case。
- 证据以真实 MySQL 数据、真实 HTTP 调用和 Web 截图为准。

## 非目标

- 不连接生产数据库。
- 不写入或暴露密钥。
- 不新增任意 SQL 或危险写接口。

## 完成标准

- health-food 编译通过并启动。
- 排障平台测试通过并启动。
- Web 端提交问题后能展示 Gateway 工具进度和结论。
- Program 记录命令、真实数据断言、截图路径和未验证项。
