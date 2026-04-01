# Source Manuscripts

`source/` 是原始稿件区和维护者入口，不是普通读者的起点。

如果你是第一次阅读这个仓库，请优先回到 [`../README.md`](../README.md) 或从 [`../docs/00-学习总纲.md`](../docs/00-学习总纲.md) 开始。

## 这里放什么

- 原始中文稿件，便于继续写作和增量维护。
- 还没有整理成 GitHub 发布版之前的工作内容。

## 维护说明

- GitHub 成品文档生成到仓库根目录下的 `docs/`。
- 请在仓库根目录执行重建命令：`powershell -NoProfile -ExecutionPolicy Bypass -File .\tools\rebuild-docs.ps1`
- 文章映射清单位于仓库根目录：`tools/article-manifest.json`
