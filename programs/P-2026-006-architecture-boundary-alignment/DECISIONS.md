# DECISIONS

## D1：平台数据属于 Agent 平台

平台自己的 case、message、AI decision log、tool audit、root cause 和 knowledge item 都是 Agent 平台沉淀，不属于业务方提供的下游证据源。

## D2：LLM/Vision 由平台统一提供

业务方不需要提供 LLM 或视觉模型接口。私有化部署时可以由部署方填写模型配置，但它仍然属于 Agent 平台配置，不属于业务 adapter 契约。

## D3：业务方只提供 readonly adapter

业务方接入面收敛为行情、资产、日志、缓存、发布记录等 readonly business APIs/adapters。Agent 和 decision-engine 不能绕过 Investigation Gateway 直连业务生产系统。

## D4：Agent 编排归 Python decision-engine

分类、实体抽取、追问、工具预算、工具计划、证据总结、停止条件和本地代码辅助排查都归 Python decision-engine。Go baseline 只作为本地 smoke/fallback。

## D5：后续变更追加 Program

后续独立需求或架构调整应新增 Program 记录执行过程和证据。旧 Program 原则上保留当时上下文，不再为了新命名或新架构回写历史记录。

## D6：引入 LESSONS 反复错误计数器

借鉴 `game` 仓库的 `cocos-project/docs/LESSONS.md`，本仓库新增 `docs/LESSONS.md`。以后用户指出流程错误或重复错误时，必须写入反复错误计数器，并在后续任务启动时先检查。

## D7：引入验证结果标准模板

借鉴 `game` 仓库的 Evidence/Result 写法，本仓库新增 `docs/VERIFICATION.md`。以后 Full 级 Program 不允许只写“测试通过”，必须记录 Evidence 索引、命令验证、覆盖映射、未验证项、已知噪音和 Result 验收覆盖。
