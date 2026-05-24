# RESULT

## 结果摘要

- 已完成 Web 左侧问题列表状态增强：排查中 case 显示 spinner，AI 已产出但当前浏览器未查看的 case 显示“待查看”，点击进入后自动清除待查看提示。

## 变更范围

- `web/static/index.html`：新增 case status chip、spinner/dot 样式、本机已读 localStorage 和状态渲染逻辑。
- `programs/P-2026-042-web-case-status-indicators/`：记录决策、证据、结果和交接。

## 验证摘要

- `node -e` inline script parse：pass。
- Browser 打开 `http://127.0.0.1:19091/web`，通过 MySQL fixture 验证 `排查中` spinner、`待查看` dot、点击清除待查看：pass。
- `git diff --check`：pass。
- `make test`：pass。
- `make secret-scan`：pass。

## Commit

- `P-2026-042 add web case status indicators`（最终 hash 以 `git log` 为准）

## 残留风险

- 当前“待查看”是浏览器本机 localStorage 读回执，跨设备/跨浏览器不共享。后续如需要多人协同读回执，应新增服务端 read state 表。
