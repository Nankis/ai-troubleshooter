# HANDOFF

Web 问题会话重命名、删除和草稿本地持久化已完成。

后续如果继续优化：

- 增加 deleted case 恢复入口。
- 增加草稿导出/清空全部草稿。
- RAG 索引层需要跳过 `tb_troubleshoot_case.status = 0` 的软删除 case。
