# HANDOFF

## 当前状态

- Program：`P-2026-008-web-chat-local-agent-mvp`
- 阶段：完成
- 最新 commit：见最终推送结果
- 本地验证 URL：`http://localhost:18080/`

## 已完成

- Web Chat 页面和 `/web/api/chat`。
- 文本/图片 multipart 输入，图片走 Vision provider。
- 本地 MySQL migration 和落库验证。
- Qwen/DashScope OpenAI-compatible 本地 smoke。
- Mock Gateway 工具链排查闭环。
- Secret scan、pre-commit、pre-push、安装脚本。
- Program Evidence/Result 回写。

## 下次继续

1. 业务方按 `docs/interface-contract.md` 和 Gateway 工具契约封装真实只读接口。
2. 若公司先接 Lark/飞书，复用当前 case processor、vision client 和 MySQL store。
3. 若要更强 Agent 状态编排，新建 Program 将 Python decision-engine 迁到 LangGraph。
