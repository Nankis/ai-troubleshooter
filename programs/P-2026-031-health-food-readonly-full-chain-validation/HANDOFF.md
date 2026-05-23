# Handoff

## 已完成

- health-food 新分支提供只读排障接口，ai-troubleshooter Gateway 已通过 HTTP connector 调用真实服务。
- Web Chat 已验证 4 个真实 case，截图已落到 `evidence/screenshots/`。
- 验证中发现并修复一个显式日期被“今日”覆盖的问题。

## 当前运行

- health-food：`http://127.0.0.1:18080/food-health`
- ai-troubleshooter：`http://127.0.0.1:18088/web`
- 本地启动命令包含 DB 密码和只读 token，Program 中只记录脱敏版本。

## 下一步

- 复跑全量测试和 health-food 编译。
- 运行脱敏扫描和 diff 检查。
- 提交两个仓库的代码和 Program 证据。
