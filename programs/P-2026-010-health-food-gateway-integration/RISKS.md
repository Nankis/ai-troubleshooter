# RISKS

- `health-food` 可能依赖外部 MySQL、Redis、Apollo、Nacos 或 JDK 23，导致不能直接本地启动。
- 若 `health-food` 当前没有 readonly adapter 接口，本轮需要用本地 mock adapter 验证排障平台链路，并把业务侧改造项写清楚。
- 多服务本地联调容易产生端口冲突，需要记录实际端口和启动命令。

