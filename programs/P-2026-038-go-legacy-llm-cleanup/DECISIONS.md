# DECISIONS

## D1: Go 只保留 Investigation Gateway 主路径

- 决策：删除 Go 侧 Web Chat、Lark bot、worker、baseline orchestrator、LLM 和 Vision 实现。
- 理由：当前架构已把入口、模型和决策全部迁到 Python；保留旧 Go 实现会让接入方误以为 Go 仍可配置 LLM 或承接对话入口。
- 影响：Go Gateway 继续负责任只读工具边界；Python 负责所有 Agent 平台能力。

## D2: Go config 不再解析模型配置

- 决策：从 Go `internal/config` 移除 `LLMConfig`、`VisionConfig`、模型 profile 文件读取和对应校验。
- 理由：模型配置属于 Python Agent Platform；Go Gateway 不应读取或持有模型 API key。
