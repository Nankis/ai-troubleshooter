# DECISIONS

## D1：本轮不引入外部 Agent 框架

先用标准库和 dataclass 实现 Supervisor + Specialist + Verifier。原因是当前一期仍强调有限工具计划、Gateway 只读调用和可测试性；等需要 checkpoint、复杂状态图或离线 eval 后，再独立 Program 评估 LangGraph。

## D2：Agent Team 放在 Python decision-engine 内

`apps/decision-engine` 是目标 Agent Orchestrator。Go Gateway 继续负责只读工具鉴权、限流、审计和 timeout，本轮不修改。

## D3：Verifier 是最终出口

所有 specialist 给出的工具计划都必须经过 Verifier 收敛，防止超预算、重复工具、不可用工具或空计划继续下发。
