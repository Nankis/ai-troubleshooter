# STANDARDIZATION

- Web Chat 内置页面优先保持无构建依赖，便于私有化开箱验证。
- 本地敏感配置统一通过 env 注入，不提供带真实密码的 committed 示例。
- Secret scan 和 git hook 后续应作为所有新项目模板的默认门禁。
