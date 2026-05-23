# DECISIONS

## D1: 草稿用浏览器 localStorage

未发送的草稿不需要进入平台 MySQL，也不适合作为 RAG 语料。它只为当前浏览器恢复输入，刷新不丢即可。图片草稿不落本地，避免截图长期留存。

## D2: 正式 case/message 继续入 MySQL

一旦用户发送问题并进入排查，case、message、AI 决策、工具审计和经验沉淀必须进入 MySQL。后续 RAG 可以从 MySQL 的 case/message/root cause/knowledge 做索引，MySQL 是事实源，向量库只是检索索引。

## D3: 删除正式 case 采用软删除

用户在 Web 列表删除 case 后，设置行状态为 deleted，不做物理删除，避免审计链路断裂。
