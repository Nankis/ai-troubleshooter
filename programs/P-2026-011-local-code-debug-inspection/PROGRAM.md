# P-2026-011 Local Code Debug Inspection

## 背景

用户指出调试阶段可能需要让 Python 决策层查看本地业务代码仓库，并且 Gateway / readonly adapter 至少要提供 `service_name`、`repo_hint` 或 `suspect_area`，决策层才能知道该查哪个服务的代码。

## 目标

- 在 Python decision-engine 中增加 debug-only Local Code Agent。
- 通过本地 allowlist registry 做 `service_name -> repo_path` 映射，不信任 Gateway 下发本地路径。
- 本地代码检查只在 Gateway 证据不足且显式开启 debug 时触发。
- 只读搜索本地仓库，返回相对路径、命中词和行号，不返回源码片段、不读取敏感文件、不改代码。
- 用临时 mock 仓库验证正向命中、敏感文件跳过、无 mapping 失败收敛。

## 非目标

- 不改 Go Gateway。
- 不接真实生产服务或生产 DB。
- 不提交本地真实仓库路径、API key、密码、token。
- 不自动修复业务代码。
