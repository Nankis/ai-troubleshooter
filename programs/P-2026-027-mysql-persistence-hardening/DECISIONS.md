# DECISIONS

## D1: 禁止隐式 memory fallback

默认配置是 `DB_DRIVER=mysql`。如果没有 `DB_DSN`，服务必须失败启动，避免把“本该落 MySQL 的验收”悄悄变成内存验证。

## D2: memory 只能用于显式 smoke

`DB_DRIVER=memory` 只适合一次性前端交互或快速 smoke。任何涉及平台经验沉淀、case、消息、tool audit、AI decision log 的验收都必须连接 MySQL 并查询表验证。

## D3: Program 记录失败，不回写旧历史

旧 Program 中用 memory 验证 UI 交互的记录保留历史上下文。本 Program 记录本次错误和强约束修正，后续类似需求必须新增 MySQL 现场验证 evidence。
