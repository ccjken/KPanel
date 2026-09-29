# 集群分享主题库

与 `scene-packs/` 一样，这是 KPanel 仓库内开放源码的独立包目录。面板只内置默认样式；管理员在「集群 → 公开分享 → 分享主题」下载、应用或删除可选主题。删除当前主题自动恢复默认样式。已安装主题无需再次访问 GitHub。

## 提交自己的主题

1. 复制 `_template/` 为自己的短名称目录，例如 `my-fleet/`，更新 `manifest.json` 的 ID、名称、版本、作者及许可证。
2. 修改 `src/index.html`、`src/theme.css` 和 `src/theme.js`。无需打包器；可用自己的框架，但须将完整源码、构建方法和所有运行依赖放在仓库中，禁止 CDN、统计脚本、外链字体、混淆代码。
3. 从仓库根目录运行 `node scripts/build-share-themes.mjs`，同时提交源码、生成的 `dist/` 与 `catalog.json`。校验命令为 `node scripts/build-share-themes.mjs --check` 和 `go test ./internal/scenepacks`。
4. 向本仓库提交 PR，附 390px / 768px / 1280px、浅色 / 深色、200% 缩放截图。维护者复核后目录才对用户可见。

目录来源固定为 `https://raw.githubusercontent.com/kejilion/KPanel/main/share-themes/`，自动使用现有 GitHub 镜像回退。主题文件逐个校验 SHA-256 和字节数，完整下载后原子安装。每包最多 40 个文件、5 MiB；最多安装 10 包，总存储遵循共享包引擎 150 MiB 上限。`index.html` 和 `manifest.json` 各不超过 64 KiB。文件名小写，目录最多 4 层，不允许软链接、路径穿越或任意下载地址。

## 运行协议 `kpanel-share-theme@1`

主题运行于 `sandbox="allow-scripts"` 的独立 iframe，HTTP CSP 也强制隔离。不能访问面板 DOM、Cookie、存储、管理 API、任意网络、弹窗、表单或顶层导航。所有 JS、CSS、图片和字体使用包内相对路径；JS 不得内联。父页面保留刷新、浅深色切换和「使用默认样式」入口。12 秒内未就绪自动回退。

安装 `message` 监听后发送：

```js
parent.postMessage({ source: 'kpanel-share-theme', type: 'ready' }, '*')
```

父页面发送 `{ source: 'kpanel-share', type: 'snapshot', schema: 1, locale, mode, data }`。先验证 `event.source === parent`，再校验 source/type/schema。`mode` 为 `light` / `dark`；`locale` 为 `zh-CN` / `zh-TW` / `en-US`。数据更新时再次发送，无需主题自行轮询。可选发送 `{ source: 'kpanel-share-theme', type: 'resize', height }` 调整内容高度；只接受 320–32768 的整数像素，避免嵌套滚动条，超出范围保持原高度。

`data` 包含 `title`、`description`、`generatedAt`、`total`、`online`、`attention`、`value` 和 `hosts`。主机字段及完整示例见 `_template/src/theme.js`。`value.groups` 按原币种分别列出已格式化估算金额；`included` / `excluded` 标明资料覆盖。`hosts[].traffic` 提供 `monthly`、`percent`、`tone`、`received`、`sent`；不得把上行和下行分别除以配额再当成两个总百分比。未知数值使用 `—`，不算成零。剩余价值是按已公开价格、到期日估算的预付价值，不代表退款金额。

`traffic.hint` 是核心提供的本地化说明，包含周期、等待数据、不完整/估算状态、用量/配额、计费方向和接近/超过配额提示，必须展示或提供可访问详情。另有 `available`、`partial`、`estimated`、`startedAt`、`endsAt` 供布局使用，不得隐藏影响解读的状态。

所有指标沿用 KPanel 核心计算，主题只负责展示。协议不包含分享令牌、面板地址、真实节点 ID 或管理资料。对名称、介绍等用户数据必须使用 `textContent`，不能插入 HTML。提供搜索、空状态、超长名称换行、可见键盘焦点；文字最小 12px，正文和操作最小 14px，辅助文字最小 13px。颜色不能独自表达状态。

新增/删除/更新已安装主题立即影响分享设置；已经打开的公开页会在下一次轮询（通常 15 秒）更新。访客的「使用默认样式」只影响本次浏览，不修改管理员设置。关闭/重置分享仍由现有分享令牌生命周期控制。

仓库中的主题包需要随候选代码合入主线后，线上目录才能下载；本地预览从当前检出读取包，不能据此声称 GitHub 已发布。
