# Errors

## E1: 非排障输入继承旧 case 排障上下文

- 现象：同一个 case 中用户追加“现在是用什么模型”“瞎说，我的 Claude Code 都用不了”，平台仍按旧 health-food case 继续查 Gateway 或总结 mock 证据。
- 根因：意图判断使用聚合后的 `original_text`，其中包含旧生产问题；没有针对最新消息做 meta/chat 分流。
- 防复发：进入 Gateway/Knowledge/Tool 之前，必须先根据最新用户消息判断是否是非排障咨询；命中后只能 direct answer 或阻断。
