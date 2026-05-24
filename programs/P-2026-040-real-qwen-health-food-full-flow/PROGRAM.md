# P-2026-040 真实 Qwen + health-food 全链路复验

## 背景

用户要求重新跑一次完整流程，必须启动所有相关服务：

- Python Agent Platform。
- Go Investigation Gateway。
- 用于测试的真实本地 health-food 服务。

验收必须覆盖：

- Python 端 LLM 实际走 Qwen，key 参考本机 health-food 配置。
- 平台经验查询：查得到和查不到都要覆盖。
- Gateway 实际调用下游 health-food readonly 接口排查问题。
- 设计多个此前未问过的问题，覆盖数据查询问题和需要本地代码辅助的问题。
- 以实际运行和实际结果作为验收；发现问题要记录并修复。

## 目标

完成一次 L3 级本地真实依赖验收：真实 MySQL、真实本地 health-food 服务、真实 Gateway、真实 Python Agent Platform、真实 Qwen 文本模型。

## 非目标

- 不接真实 Lark/飞书。
- 不连接生产环境。
- 不触发 health-food 写操作、支付、退款、任务重跑或生产迁移。

## 关键路径

1. 初始化/确认平台 MySQL。
2. 启动 health-food 本地服务，开启 readonly token。
3. 启动 Go Gateway，`CONNECTOR_MODE=http` 指向 health-food readonly endpoint。
4. 启动 Python Agent Platform，配置 Qwen 和本地代码仓 allowlist。
5. 录入一条高置信平台经验。
6. 发起多个真实排障问题，并核对 case、decision log、context ledger、tool audit、health-food 数据和必要截图。
