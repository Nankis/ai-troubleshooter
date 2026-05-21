# STANDARDIZATION

- 生产环境必须 fail-closed：缺少 Gateway token、控制面 token、Lark token 或 allowed chat 时不得启动对应 HTTP 服务。
- 控制面 API 和 Tool Gateway API 分开鉴权，不能复用 Lark 入口信任。
