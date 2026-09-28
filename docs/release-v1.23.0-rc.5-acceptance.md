# KPanel v1.23.0-rc.5 发布验收记录

日期：2026-09-29。发布级别：L3。`releaseChannel=preview`；`releaseTrain=1.23.0`。

- 候选、远端 `main` 与标签均指向 `56e223836192dc3a803483e2ab6fc123f7153f9a`；注释标签 `v1.23.0-rc.5` 的 tag object 为 `3bf43d69cd02abdd154063458525c1dd176f0cf9`。
- 发布前 `main=dad6ba9b2a3a62406ad503a1a3f57d85a545a064`；上一稳定版与源码回滚点为 `v1.22.0`，稳定镜像 index 为 `sha256:46b854bdf6233e81742986a3bee9e65c6b4a39123c393a51a4ada63783eed7ac`。上一预览版 `v1.23.0-rc.4` 镜像 index 为 `sha256:c240284c68b00d780ac0ce9c4217ecf626992f4c2e3a26ef2460bc78b6275d8e`。
- 证据目录：`C:/GitHub/_release-evidence/v1.23.0-rc.5-56e22383-l3-r1`。本版只发布预览产物，生产未部署。

## 发布画像与范围

- 业务域：集群主机资料、周期流量、通知记录、到期提醒、服务检测告警、桌面快捷键和默认浅色主题。
- 变更面：Panel 数据与 API、Node/Agent 服务检测回报、桌面与集群 UI；不新增任意宿主命令或对外监听端口。资料、周期统计和通知状态由 Panel 有界保存，宿主真实资源及节点原始流量仍是事实来源。
- 核心旅程：配置到期日、价格、重置日及逐方向流量阈值；从列表、卡片、匿名分享查看白名单资料；按周期接收告警并从本地记录筛选事件；服务检测连续失败及恢复提醒；Tab/Shift+Tab 切换桌面窗口；以默认蓝色主题浏览。公开镜像额外复核了上一 RC 曾出问题的自定义壁纸、经典模式透出及登录页副本。
- 风险：多个来源分支合并了集群通知契约，且有持久状态和节点采样协议。`internal/panel/cluster.go` 与两份翻译目录的三处文本冲突经人工合并；其余自动合并由组合候选 L3、CI 和浏览器预览核验。真实跨月、弱网、外部推送、远端节点重启及付费/耗流量场景没有在本地模拟中发生。
- 未纳入：带未知未提交改动的 `feature/desktop-dynamic-scenes`、旧壁纸源分支和历史治理/验收分支；未按名称或时间批量清理。来源分支中较早的交付记录与单支预览只作为补充证据，不替代本轮组合候选测试。

| 来源分支 | 精确 tip | 纳入方式 |
| --- | --- | --- |
| `feature/cluster-notification-history` | `9ae19fa2a583f497e250158b6d78714cbafd7ec9` | `feature/host-traffic-limits` 的祖先 |
| `fix/activity-tab-spacing` | `1ff9967b6f3b5d10969b24044718913cf3d8a439` | 同上 |
| `feature/cluster-host-billing` | `e276cd6830061d8eea883d538632e13af10d58ca` | 同上 |
| `feature/host-expiry-reminders` | `0a0f181834a6a7c77778d4a67076c63482aaab9c` | 同上 |
| `feature/traffic-reset-accounting` | `83a14f9a9465d8ee0fc6810b3507971380d7e99e` | 同上 |
| `feature/host-traffic-limits` | `4966f4a2f1b41584ab91b7a242d5d3b3d83af8dc` | 无快进合并 `feac1154` |
| `feature/service-monitor-alerts` | `2ebbaadbb8e19ef92568fb57c15bfa5e9f0ac43e` | 无快进合并 `1925e590`，解决三处文本冲突 |
| `feature/desktop-tab-switch` | `1afe417942f3b8482c0864b49e354ac00c9daaef` | 无快进合并 `c563cfbf` |
| `feature/default-blue-theme-20260928` | `36f57967a4aa7a9e935976e65825483b208e0e3f` | 无快进合并 `138e25c5` |

业务上下文刷新提交为 `82c0be88`，发布准备提交为 `56e22383`。上述九个来源 tip 都是不可变版本标签的祖先；没有重放或 squash 的等价性推断。

## 外部审计与跨仓库联动

- 安全审计：`check-security-audit-coverage.mjs --target HEAD` 输出 `decision=ok`，最近 full 为 `run-4`，最近 scoped 覆盖 `run-6/7/9/10`；本候选尚有 27 个提交、55 个文件未被 CF 账本覆盖，最老 2 天，未触发新增边界包或 14/30 天阈值。本 RC 不另行执行 CF scoped/full；L3 的 govulncheck、npm audit、Trivy 源码和最终镜像扫描均通过，不把扫描冒充 CF 审计。
- OCR：来源任务各自保留候选行级评审记录；本轮 Release 组装只新增三处文本冲突的合并，已逐行复核并由组合测试覆盖。发布准备提交记录 `OCR-Review: skipped`，没有声称对 118 个代码路径重新跑过全量 OCR；`constrained-only` 对组装提交不可报告。
- `scriptLinkageState=not-required`（无需发布脚本（不适用））：差异未修改或要求新的 `kejilion.sh` 协议、安装路径、外联配置、镜像内脚本内容。Dockerfile 固定内置 `kejilion/sh` 提交 `2b90b2d2ca56bc954c9328a51bb5571e896f713d`、SHA-256 `806b4715664fad502f7faeccbc75972f2d1a24559d46208221b98f774ef56c99`；L3 的 `managed_script_contract=pass` 与应用生命周期测试证明现有契约兼容。变更集编号、脚本候选和阻断依赖范围均不适用；本版不创建或发布脚本提交。
- 仓库内 `packaging/kejilion-app/kpanel.conf` 与 `kejilion/apps` 远端 `main` 的文件 blob 均为 `fa4b95374ed3b186d920207537f451a5b6d839f7`；无需应用市场提交，默认安装仍指向稳定 `latest`。

## 多维质量结论

| 维度 | 状态 | 证据与边界 |
| --- | --- | --- |
| 业务正确性与双端互通 | 已实现未实机验证 | Go 全套、组合前端 1826 项与模拟浏览器的主机资料、阈值保存回读、通知控件通过；真实多节点服务异常/到期/跨月外部通知未实测 |
| 网络入侵与供应链安全 | 已验证 | 覆盖检查 `ok`、L3 与云端扫描、固定摘要脚本、双架构附件校验；本轮没有新增 CF 审计 run |
| 稳定性、失败恢复与兼容 | 已实现未实机验证 | 特权包 race、应用配置安装/更新备份回滚生命周期通过；真实 systemd/OpenRC 和远端节点重启未重测 |
| 性能与资源预算 | 已实现未实机验证 | L3 镜像运行约束与有界状态测试通过；低配置主机持续采样、外部发送压力未实机测量 |
| 用户体验与可访问性 | 已验证 | 模拟组合页面在 1440×900、720×450 完成资料/通知共存及无横向溢出；公开镜像 1920×920 验证壁纸与登录。真实浏览器 200% 缩放、其他浏览器和用户现场未验证 |
| 数据、配置与迁移 | 已实现未实机验证 | Panel 存储/备份与流量周期测试、模拟保存回读通过；实际旧状态跨版本迁移及长期数据恢复未在真机重测 |

## 自动门禁与隔离浏览器

- 定向组合检查：前端类型检查与 5 个受影响测试文件 123 项通过；模拟 `acceptance` 预览绑定 `56e22383`，主机独立阈值 PUT 返回 200 且重新打开回读，服务异常与本地通知控件同时可见，宽窄弹窗 `scrollWidth=clientWidth`。模拟 API 对 `/api/v1/settings/appearance` 返回 404，是模拟夹具未实现的接口，不构成真实后端结论；已停止该预览。
- L3 外层入口 `scripts/run-release-l3.mjs`：`arena-154` 本轮 SSH 超时，按用户此前选择在 `local-wsl-dr` 执行；run ID `v1.23.0-rc.5-56e22383-l3-r1`，2026-09-29T00:21:02+08:00 至 00:28:44+08:00，`status=passed`、`exit_code=0`。不可变 Runner ID `sha256:a9f708891d1e81f17dd286bc7e9d6124658964716dc9a457b5c73340722f6a7d`，bundle `9fc1b93dc8750ece2360e0d506dd3493e9440adef11f58d3275aaecfa4a01422`，plan `614cd1aca5ad407b9573505449874b3b1a8d19a34a3634bebed5dbcd3451eda1`，执行脚本 `21c08b11be3526a0fe766aa606a81d0e4047e8a4e89d79227da4e62914164979`，证据及 manifest 位于本记录首段目录。
- L3：Go 全套与特权包 race、202 个前端测试文件 1826 项、场景包复现、类型检查、i18n、构建、安装安全、应用配置生命周期和备份恢复、govulncheck、npm audit、Trivy 源码及最终镜像检查均通过。`arena-154` 不可达，WSL 不替代登记真机浏览器、性能、故障注入或生产验收。
- 候选 [CI #1121](https://github.com/kejilion/KPanel/actions/runs/36451367979)、主线 [CI](https://github.com/kejilion/KPanel/actions/runs/36452442137)、[Release](https://github.com/kejilion/KPanel/actions/runs/36453698270) 均为 `success`；三轮对应的 Dependency freshness 检查均成功。

## 发布产物与公开仓库复核

- [GitHub Release v1.23.0-rc.5](https://github.com/kejilion/KPanel/releases/tag/v1.23.0-rc.5) 于 2026-09-29T01:01:18+08:00 公开，`draft=false`、`prerelease=true`、非 GitHub Latest；Latest 仍是 `v1.22.0`。正文包含更新内容、兼容和迁移说明、加入与退出预览、镜像摘要、附件校验、测试及回滚提示。
- 14 个附件含 Agent/Node 双架构、MCP 各平台、元数据归档、`SHA256SUMS` 和许可声明；`SHA256SUMS` 附件自身 SHA-256 为 `bdb10e0e135aeb9178ad9c9167b67f11d84ca27367f3644d6a30c9521a37e736`，其 11 条均与 GitHub Release API 报告的对应附件 digest 一致。
- Docker `1.23.0-rc.5` 与 `preview` 的 OCI index 均为 `sha256:5cae885abd0415163301a253321bcb8adf55b3b678c86250cea7d8a0a8bf2e56`；`linux/amd64` 为 `sha256:5635ef02c6447f436ca10bcea1ef6f6173c02f5c915ed8fe9cbbf501e075b5b1`，`linux/arm64` 为 `sha256:d16a0c1443a4b0d76a49b959553fafed403f022674a7468c213785b50947a147`。额外的 unknown/unknown 条目是镜像证明。Docker `latest` 仍为上方稳定版 index。
- 显式从 Docker Hub 拉取 `kjlion/kejilion-panel:1.23.0-rc.5` 得到相同 index；`packaging/tests/image-e2e.sh` 输出 `image_e2e=pass`。同一公开镜像在本机隔离容器与 Chrome 中上传自定义壁纸并验证普通、减少动态、减少动态加减少透明、强制颜色四种媒体条件：图片请求均为 200、解码宽度 2560，前三者“通透”背景可见，强制颜色隐藏壁纸；退出登录后 `authWallpaper=private` 且品牌背景包含私有图片副本。截图和 JSON 在证据目录 `browser-public/`。隔离容器、网络和模拟数据已清理。

## 自更新、生产与回滚

- 自更新通道逻辑本轮未修改。稳定源只选正式 GitHub Latest，预览源选择规范 RC 并校验官方镜像 digest；加入/退出预览、自动安装与立即安装分离、旧状态迁移、失败隔离、systemd/OpenRC 边界沿用现有自动回归，未在真实宿主重测。退出预览不会自动降级。应用市场默认安装入口仍指向稳定版。
- 生产部署安全核对：不适用（预览版禁止生产部署）。本轮没有连接、备份、部署、升级或核对 `prod-108`；该目标继续禁用全部 KPanel 操作。没有生产前后版本、健康、备份或公网入口数据可报告。
- 源码回滚点为 `v1.22.0`，版本镜像回滚点为稳定 index `sha256:46b854bdf6233e81742986a3bee9e65c6b4a39123c393a51a4ada63783eed7ac`；需要回退预览可选择上一不可变 `v1.23.0-rc.4` index 并先备份数据、核对兼容。切回稳定通道只改变后续更新来源，不自动改写运行版本。GitHub Latest、Docker `latest` 与标准应用市场入口均保持 1.22.0；本轮无需公共默认通道回滚。

## 分支与资源处置

- `release/v1.23.0-candidate` 是 1.23.0 序列唯一远端候选，tip `56e22383`，预览版按规则保留，待稳定版完成后再归档。
- 上表九个来源分支的精确 tip 均由 `v1.23.0-rc.5` 标签可达；远端以一次受 expected-SHA lease 保护的原子推送保存为 `archive/<原分支全名>`，逐项重读确认归档 SHA 与原 tip 一致、原活跃远端 ref 不存在。本地对应分支也已改名为相同 `archive/` 名称，旧 upstream 为空。原任务 clean 工作树保留供对应会话查阅；未删除其他任务目录或声称回收磁盘空间。
- 上表每行的远端归档引用均为 `refs/heads/archive/<来源分支>`，归档 SHA 等于该行精确 tip；处置分类均为“纳入 `v1.23.0-rc.5` 后归档”，远端九项均已复核。归档责任人为本次发布任务。
- 未纳入的动态场景实验分支保留其未知改动，不归档、不计为已发布。验收记录专用分支按 10.2 节在其候选与主线同 SHA CI 成功后归档，归档结果在本次发布交付中复核。

## 交付节奏数据

<!-- kpanel-release-metrics:start -->
- 首个纳入提交时间：2026-09-28T15:07:04+08:00
- 候选冻结时间：2026-09-29T00:21:02+08:00
- 生产完成时间：不适用
- 提交到生产用时：不适用
- 是否回滚、紧急热修复或重复发布：否
- 若发生失败，发现时间、恢复时间和逃逸门禁：不适用
<!-- kpanel-release-metrics:end -->

<!-- kpanel-release-process-metrics:start -->
- 已记录发布流程异常或无效证据拦截次数：4
- 其中生产写操作开始后异常次数：0
<!-- kpanel-release-process-metrics:end -->

<!-- kpanel-release-process-incidents:start -->
[
  {
    "fingerprint": "tag-verification/powershell/revision-escape",
    "position": "before-production-write",
    "count": 1,
    "impact": "标签成功推送后的 git rev-parse ^{} 校验参数被 PowerShell 解析，查询命令退出 1；标签本身未受影响。",
    "recoveryEvidence": "git cat-file -p v1.23.0-rc.5 确认对象指向 56e22383；远端 ls-remote 确认 tag object 3bf43d69。",
    "permanentAction": "Windows 标签校验改用 git cat-file -p 与远端精确引用，避免 shell 特殊字符。",
    "historicalReleases": []
  },
  {
    "fingerprint": "browser-public/wsl-runtime/session-lifetime",
    "position": "before-production-write",
    "count": 1,
    "impact": "首轮公开镜像壁纸浏览器检查通过后 WSL 内隔离容器以退出码 0 停止，登录页后续脚本连接失败，未把该脚本记为通过。",
    "recoveryEvidence": "保留 WSL 会话后重启同一隔离容器，健康接口报告 1.23.0-rc.5，登录页脚本通过；随后执行固定 stop 脚本清理。",
    "permanentAction": "发布负责人在下一次浏览器验收前把 WSL 会话所有权与容器存活检查固化进公开镜像浏览器启动器，2026-10-06 复核；退出条件为跨多个脚本连续运行不再意外停止。",
    "historicalReleases": []
  },
  {
    "fingerprint": "browser-zoom/css-root/invalid-emulation",
    "position": "before-production-write",
    "count": 1,
    "impact": "尝试把根元素 CSS zoom=2 当作浏览器 200% 缩放时导致媒体断点不匹配和关闭按钮不可达；该模拟证据已排除，不能证明真实浏览器 200% 的产品行为。",
    "recoveryEvidence": "重新运行有效的 1440×900 与 720×450 组合预览，主机资料弹窗无横向溢出、阈值回读及通知控件通过；真实 200% 仍列为未验证。",
    "permanentAction": "发布负责人在下一次 UI 验收前提供真实浏览器缩放的固定用例，2026-10-06 复核；取得浏览器缩放状态和同场景截图前不将该维度标记通过。",
    "historicalReleases": []
  },
  {
    "fingerprint": "acceptance-integration/powershell/array-match",
    "position": "before-production-write",
    "count": 1,
    "impact": "验收记录候选 CI 成功后的主线快进预检，把 PowerShell 数组的 -notmatch 当成布尔值，误报远端已移动；预检先于推送终止，远端 main 未变化。",
    "recoveryEvidence": "远端 main 仍为 56e22383、验收分支为 97e21d6c；改用结构化引用逐项核对并重跑同 SHA 验收记录 CI 后再快进。",
    "permanentAction": "Windows Git 引用预检统一对单个精确 ref 值比较，避免把数组筛选结果用作布尔判断；下轮验收前固定该入口并回归。",
    "historicalReleases": []
  }
]
<!-- kpanel-release-process-incidents:end -->

## 遗留风险

- 真实 `arena-154` 不可达，Node/Agent 多主机通知、远端服务检测、实际跨月流量、外部渠道投递、真机重启和 200% 浏览器缩放没有本轮证据；本版是预览通道，不自动进入正式生产。
- 用户现场的浏览器、代理、缓存和实际自定义图片未接入本地隔离试验；公开镜像验证只证明所测图片与媒体状态。下一稳定版前按受影响旅程补真机和真实缩放验收，并再次运行 CF 覆盖准入。
- 本地九个归档来源工作树仍占磁盘，因其他会话可能保留其预览/证据，本轮没有删除目录；远端精确 archive ref 与标签可恢复这些提交。
