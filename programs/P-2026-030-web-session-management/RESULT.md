# RESULT

已完成 Web 问题会话管理：

- 草稿会话支持浏览器 localStorage 持久化、重命名和删除。
- 已创建 case 增加 `case_title`，支持 Web/API 重命名。
- 已创建 case 删除采用软删除，列表和直接查询都会隐藏 deleted case。
- UI 使用页面内弹窗，不再依赖浏览器原生 prompt/confirm。
- 正式排查会话仍写 MySQL，便于审计、RAG 索引和经验沉淀；草稿不入库，也不持久化图片。

验证结论：单测、MySQL migration、Web smoke、secret scan、diff check 均通过。
