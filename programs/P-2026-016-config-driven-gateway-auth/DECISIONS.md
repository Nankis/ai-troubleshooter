# DECISIONS

## D1：Gateway agent 权限使用 JSON 配置

为了避免引入额外 YAML 依赖，先支持 `GATEWAY_AGENT_CONFIG_JSON` 和 `GATEWAY_AGENT_CONFIG_FILE`。配置源为 JSON object，结构为 `{ "agents": [...] }`。

## D2：token 通过环境变量注入

agent 配置中只写 `bearer_token_env`，运行时从该 env 读取 token。旧版 `GATEWAY_BEARER_TOKENS=agent_id:token` 继续可用，便于兼容和本地快速启动。

## D3：默认 agent 保持兼容但 agent id 可配置

没有提供 agent 配置时，仍使用原来的默认 scopes/tools。`GATEWAY_AGENT_ID` 可以改变默认 agent id，worker/dev-server/baseline runner 会使用同一个配置。
