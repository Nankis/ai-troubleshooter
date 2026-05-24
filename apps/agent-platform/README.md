# Agent Platform

Python FastAPI 主服务，承接 Agent 平台入口和排障主路径：

- Web Chat：`GET /web`，`POST /web/api/chat`。
- 正式 API：`/api/v1/chat`、`/api/v1/cases/*`、`/api/v1/knowledge/*`、`/api/v1/capabilities/*`。
- Lark / 飞书：`POST /lark/events`，`POST /feishu/events`，支持 encrypted callback、verification token、群 allowlist、消息幂等和图片下载入口。
- Case API：`GET/PATCH/DELETE /web/api/cases/{case_no}`。
- 平台经验：`/web/api/knowledge`。
- 能力接入：`/web/api/capabilities/*`。
- 决策主路径：内嵌 `apps/decision-engine`，通过 Go Investigation Gateway 调用只读 tools。

## Run

```bash
python3.13 -m venv .venv
.venv/bin/python -m pip install -e apps/agent-platform

export PYTHONPATH=apps/agent-platform:apps/decision-engine
export DB_DRIVER=mysql
export DB_DSN="$LOCAL_DB_DSN"
export GATEWAY_ENDPOINT=http://127.0.0.1:18080
export AGENT_PLATFORM_PORT=19091
make dev
```

Open:

```text
http://localhost:19091/web
```

API smoke:

```bash
curl -s -X POST http://localhost:19091/api/v1/chat \
  -F 'message=health-food uid hf-user-001 today token quota wrong' \
  -F 'async=0'
```

## Model Profiles

LLM/Vision 配置属于 Python Agent Platform，不属于 Go Gateway。

```bash
export AI_MODEL_PROFILE=qwen
export DASHSCOPE_API_KEY="$DASHSCOPE_API_KEY"
export QWEN_MODEL=qwen-plus
export QWEN_VISION_MODEL=qwen-vl-plus

export AI_MODEL_PROFILE=gpt
export OPENAI_API_KEY="$OPENAI_API_KEY"
export OPENAI_MODEL=gpt-4.1-mini
export OPENAI_VISION_MODEL=gpt-4.1-mini

export AI_MODEL_PROFILE=claude
export ANTHROPIC_API_KEY="$ANTHROPIC_API_KEY"

export AI_MODEL_PROFILE=claude_code
export CLAUDE_CODE_BASE_URL=http://127.0.0.1:19093
export CLAUDE_CODE_API_KEY="$LOCAL_PROXY_TOKEN"
```

也可以只读本机已有 Spring AI YAML，例如 health-food 的 `application-local.yml`：

```bash
export AI_MODEL_PROFILE=qwen
export AI_MODEL_CONFIG_FILE="$HEALTH_FOOD_LOCAL_CONFIG"
```

`AI_MODEL_PROFILE=qwen` 默认把文本模型接到 DashScope OpenAI-compatible，把图片理解接到 `qwen-vl-plus`；`AI_MODEL_PROFILE=gpt` 默认文本和图片都走 OpenAI。`LLM_PROVIDER`、`LLM_BASE_URL`、`LLM_API_KEY`、`LLM_MODEL` 可以覆盖主模型；`VISION_PROVIDER`、`VISION_BASE_URL`、`VISION_API_KEY`、`VISION_MODEL` 可以单独覆盖图片模型，用于“主模型 GPT、图片 Qwen-VL”这类组合。

没有真实 key 时可用 `LLM_PROVIDER=local_rules VISION_PROVIDER=local_rules` 做页面 smoke，但不能把结果记录成真实大模型或真实 Vision 验收。
