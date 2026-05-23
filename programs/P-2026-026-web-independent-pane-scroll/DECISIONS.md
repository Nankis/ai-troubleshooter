# DECISIONS

## D1: 外层使用 fixed viewport 高度

`html/body/.app` 使用 `height: 100vh` 和 `100dvh`，并设置 `overflow: hidden`，避免浏览器页面本身参与滚动。

## D2: 滚动只交给内容容器

左侧 `.left-scroll`、中间 `.conversation`、右侧 `.right` 承担滚动；`.main` 使用 grid 保持 topbar 和 composer 固定。

## D3: 使用 overscroll containment

滚动容器设置 `overscroll-behavior: contain`，减少触控板或滚轮到边界后把滚动传给其它区域的情况。
