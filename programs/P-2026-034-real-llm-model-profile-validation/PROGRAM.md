# P-2026-034 Real LLM Model Profile Validation

## 目标

接入真实大模型作为排障决策层，并提供统一模型 profile 入口，让本地验证和后续切换模型不再依赖散落的环境变量。

## 范围

- LLM config/profile 加载。
- OpenAI-compatible LLM 严格模式，禁止真实模型失败后静默 fallback 到 local rules。
- health-food 本地配置读取，仅用于本地运行时读取密钥，不写入仓库。
- Web Chat 真实 LLM + 真实 Gateway + 真实 MySQL 验证。

