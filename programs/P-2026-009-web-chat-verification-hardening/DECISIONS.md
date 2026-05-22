# DECISIONS

## D1：补 Gateway 直接单测

已有鉴权、限流和决策层超时测试，但 Gateway 输出脱敏和 HTTP 超时状态缺少直接覆盖。本轮补齐，避免只靠文档或间接行为。

## D2：Web smoke 记录业务场景结果

只说“跑过”不够，Program 证据里记录 case_no、状态、domain、tool_count，便于后续回溯。
