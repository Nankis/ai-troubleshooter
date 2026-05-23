# Decisions

## D1: DB 安全标准

- ORM 不是唯一安全条件；安全底线是参数绑定、白名单动态 SQL、最小权限和测试/扫描。
- 业务 CRUD 优先 ORM/Query Builder；当前 Go MySQL 层暂保留 repository raw SQL，但所有外部输入必须使用 `?` 参数绑定。
- Python adapter 禁止 `mysql -e` 和 f-string SQL，统一使用 PyMySQL DB-API 参数化执行。

## 参考来源

- OWASP SQL Injection Prevention Cheat Sheet: https://cheatsheetseries.owasp.org/cheatsheets/SQL_Injection_Prevention_Cheat_Sheet.html
- GitHub CodeQL Go SQL injection: https://codeql.github.com/codeql-query-help/go/go-sql-injection/
- Uber Go Style Guide: https://github.com/uber-go/guide/blob/master/style.md
- Google Python Style Guide: https://github.com/google/styleguide/blob/gh-pages/pyguide.md
