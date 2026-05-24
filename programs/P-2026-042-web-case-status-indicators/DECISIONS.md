# DECISIONS

## D1. 状态来源

排查中状态来自后端 case status：`READY_TO_INVESTIGATE`、`INVESTIGATING`、`WAITING_TOOL_RESULT`。

## D2. 待查看是前端本机读回执

`NEED_HUMAN_CONFIRMATION`、`DONE`、`FAILED` 这类 AI 已经产出结论的状态，如果当前浏览器没有打开过该 case 的最新 `updated_at`，左侧展示“待查看”。点击打开后写入 localStorage，避免误把个人 UI 读回执写进平台业务状态。

## D3. 等待用户补充仍保持业务含义

`NEED_MORE_INFO`、`WAITING_USER_REPLY` 表示需要用户补充，不归类为“AI 结果待查看”。
