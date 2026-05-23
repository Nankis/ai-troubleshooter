# P-2026-022 Web Tools Service Grouping

## 背景

Web 工作台左侧 Gateway tools 原来按 tool 平铺展示。接入真实业务后，一个服务会提供多个 readonly tools，平铺列表会越来越难扫。

## 目标

- 左侧 tools 按服务分组展示。
- 服务标题显示服务名和该服务 tool 数量。
- 保留总 tool 数。
- 不修改 Gateway tools API 协议，优先根据 `backend_handler` / `required_scope` / tool name 推断服务。
- 实际启动 Web 页面验证分组渲染。

## 非目标

- 不新增服务注册元数据字段。
- 不做折叠/展开状态持久化。
