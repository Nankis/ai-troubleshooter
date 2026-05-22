# STANDARDIZATION

- 架构图以后固定使用三段边界：Agent 平台、Investigation Gateway、业务服务/业务 DB。
- 单 case 流程图必须体现经验评分、高置信直接返回、低置信查 Gateway、统一回复出口。
- Gateway 文档必须同时写平台侧鉴权和业务服务侧内部身份校验。
