# RISKS

- health-food 当前 admin 日志接口使用 `password` 查询参数，生产侧更理想的形态是 Bearer 或内网签名请求。
- 生产日志可能包含用户隐私，请求和响应必须严格脱敏、截断并受 limit/time range 控制。
- health-food admin 日志接口按日期和文件类型搜索，不是完整日志平台；如果需要跨服务、跨 pod、trace 聚合，后续应接 SLS/ELK/Loki 等日志平台 readonly API。
- 真实生产验证缺少 base URL 和只读密钥，当前只能完成本地链路自测。
