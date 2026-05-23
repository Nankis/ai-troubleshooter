# ERRORS

## E1. 首次编译发现 AddMessage 返回值处理错误

`processCaseDetached` 中把 `AddMessage` 当作单返回值使用，Go 编译失败。已修复为 `_, _ = h.store.AddMessage(...)`。
