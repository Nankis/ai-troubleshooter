# Decisions

## D1. 下游使用 health-food 原生 readonly endpoint

本轮不使用 `scripts/mock-health-food-readonly-adapter.py`，也不优先使用 `scripts/real-health-food-readonly-adapter.py`。Go Gateway 直接通过 `HEALTH_FOOD_READONLY_BASE_URL=http://127.0.0.1:<port>/food-health` 调用 health-food 服务暴露的 `/food-health/v1/readonly/**`。

理由：这更接近业务方真实接入形态，证据来自 health-food 服务和 `meow_pas`。

## D2. 真实模型走 Python Agent Platform 的 Qwen profile

LLM/Vision 配置只放 Python Agent Platform。Qwen key 从本机 health-food `application-local.yml` 读取，不写入仓库、不写入 Program。

## D3. 代码辅助只做 debug-only 验证

本地代码检查必须显式启用 `debug_local_code=true` 且证据状态为 insufficient/no_match。返回证据只允许相对路径、命中词、符号、调用边和行号，不返回源码片段。
