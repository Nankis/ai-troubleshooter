# RISKS

- 如果配置允许空 scopes，可能误放大权限；本轮校验会要求每个配置 agent 至少有一个 scope。
- 如果把 token 写进配置文件，会有泄露风险；本轮要求用 `bearer_token_env`，旧 env token 仅保留兼容。
- Tool Registry 仍是代码注册，本轮只解决 agent 权限配置，不解决全量动态工具注册。
