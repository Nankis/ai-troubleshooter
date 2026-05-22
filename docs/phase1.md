# 一期实现说明

本文档是一期 TRD 的公开摘要，保留可实现的工程约束和范围说明；原始内部 TRD 不随开源仓库发布。

## 一期不可变原则

- Agent 不直接拥有生产权限，业务生产证据查询必须经过 Investigation Gateway。
- 一期工具全部只读。
- 信息不足先追问，不先查生产。
- 每次排查在 Agent 平台沉淀 case、message、investigation、AI decision log、tool audit 和 knowledge。
- case 事件化，worker pool 并发处理。
- 优先业务只读 API；直查 DB 后续只能走预注册模板。

## 本轮落地

- 建立 Go 仓库和 TRD 建议目录。
- 实现本地一体化 dev-server，方便先把业务闭环跑起来。
- Tool Server 与 Investigation Gateway 合并实现一期业务只读 Tool API。
- 注册 K线、资产、通用运维共 10 个只读工具。
- 使用 mock connector，后续替换真实业务只读 API。
- 单测优先覆盖状态机、policy、masking、tool registry。
