# DECISIONS

## D1: 粘贴图片写回原 file input

前端不维护第二套附件状态。粘贴事件把 clipboard image 转成 `File`，通过 `DataTransfer` 合并进 `#images.files`。提交时继续遍历 `el.images.files`，后端无需改动。

## D2: 保留手动选择图片能力

`change` 事件仍只更新计数。粘贴图片与手动选择图片可共存，计数展示合并后的文件总数。

## D3: 不在页面增加长说明

只在粘贴成功时用 toast 反馈，避免把工作台变成说明页。
