# DECISIONS

## D1：优先使用 health-workspace 托管副本

`/Users/ginseng/IdeaProjects/health-food` 不是 git 仓库且只包含浅层目录；本轮联调优先使用 `/Users/ginseng/IdeaProjects/health-workspace/repos/health-food` 作为只读验证对象。

## D2：本轮不改 health-food 业务仓库，先用 readonly adapter 接入

原因：`health-food` 当前业务接口依赖登录态和业务上下文，不适合作为 Agent 直接查询入口。排障平台应该只面对 readonly adapter，由 adapter 在业务侧做安全、脱敏、查询聚合和错误映射。

## D3：health-food 工具先静态注册，manifest 先作为接入契约

原因：当前 Gateway 的 registry 是进程内静态注册。为了快速验证完整链路，本轮新增 health-food 工具到默认 registry，同时把未来业务服务注册需要的数据结构沉淀为 manifest 文档和示例配置。后续如接多业务域，再把 manifest 做成动态 registry。

## D4：可空时间字段不能用空字符串

本轮 recommendation status 第一次联调失败，原因是 adapter 对未生成推荐的 `generated_at` 返回空字符串，Go `time.Time` 解析失败。修正为 connector 使用 `*time.Time`，adapter 返回 `null`。
