# KPanel v1.23.0-rc.7 发布验收记录

日期：2026-09-30。发布级别：L3。`releaseChannel=preview`；`releaseTrain=1.23.0`。

- 候选、远端 `main` 与发布标签均指向 `13e17d0f670b479f2112b37f97f48a8033597d16`，注释标签对象为 `46d61c0fb01b1e29b3c59b4377ddc51cd4777d5d`；发布前远端 `main=516030d7b41bec89653ed793c6142b73b5e52c78`，候选 CI 通过后已安全快进主线。
- 上一稳定版及回滚点：`v1.22.0`，Docker `latest` index `sha256:46b854bdf6233e81742986a3bee9e65c6b4a39123c393a51a4ada63783eed7ac`。
- 预览候选 `release/v1.23.0-candidate` 在本发布序列继续保留；本版公开后生产仍未部署。

## 发布画像与范围

- 业务域：登录页、桌面和经典模式共享外观；前端依赖安全补丁。
- 变更面：Panel 匿名外观响应、Web 外观加载与保存、锁定依赖。无 Agent 宿主机动作、Compose、安装脚本或应用市场配置变更。
- 核心旅程：未登录时加载当前主题与壁纸；登录、退出及桌面/经典模式切换时保持同一选择；失败保存可重试，并发冲突后与服务器结果一致；旧浏览器缓存不会在首帧闪出默认图。
- 风险：公开缩略图必须只暴露当前选中图、限定大小并在改变后撤销旧 URL；真实 Nginx 缓存与隔离真机浏览器链路因 `arena-154` 不可达而未验证。本版仅为预览，不作为稳定默认。
- 用户可见内容与升级说明见同标签 `CHANGELOG.md`。`feature/desktop-dynamic-scenes`、`feature/custom-wallpapers-20260925` 等仍有未知本地改动的旧实验工作树不纳入。

| 来源分支 | 精确 tip | 纳入依据与发布后处置 |
| --- | --- | --- |
| `fix/appearance-sync-20260930` | `efe3d709b1a385a4be9be4ed1b499f036a7b5f87` | 七个任务提交合并；已归档为 `archive/fix/appearance-sync-20260930`，远端精确 SHA 已复核 |
| `fix/login-appearance-preload-20260930` | `c117427671f1e1e1e19f3cd22d3ece8b8909ddcb` | 上项祖先；已归档为 `archive/fix/login-appearance-preload-20260930`，远端精确 SHA 已复核 |
| `fix/wallpaper-first-frame-20260930` | `c988e090052be410c627b104a0a2850e67ea7cb3` | 与已纳入的 `7d69aef0` 稳定 patch-id 相同；原 tip 非标签祖先，标记被替代，已归档为 `archive/fix/wallpaper-first-frame-20260930`，远端精确 SHA 已复核 |
| `fix/rc7-dependency-audit-20260930` | `64b8c1cb381b2d827df7348083964b5e664e1c1a` | L3 首轮审计拦截后的安全补丁；已归档为 `archive/fix/rc7-dependency-audit-20260930`，远端精确 SHA 已复核 |

## 外部审计与跨仓库联动

- 安全覆盖：`check-security-audit-coverage.mjs --target HEAD` 为 `decision=scoped-required`，因新增边界包 `internal/backupremote`；最近 full 为 `run-4`，39 个提交、75 个文件未由 CF 账本覆盖。预览版按规则记录，稳定版前须补 scoped 审计；L3 扫描不能替代该审计。
- 外观候选有独立复核与 OCR 记录，末次生命周期修复另由本发布任务检查实际差异、自动测试及浏览器旅程；前端依赖补丁不适用行级代码评审。不能将源分支早于末次修复的 OCR 覆盖宣称为最终增量覆盖。
- `scriptLinkageState=not-required`（无需发布脚本（不适用））：本轮没有脚本协议、运行时动作、宿主机产物、安装路径或内置内容变化。现有脚本 commit `779192048077c130442a64a126d7c0050776d868`，SHA-256 `33010d547355f9bde4067c189a0457f44dc00ec3a72eaa25e5b7101e1fcb9c04`，受管脚本契约检查通过。
- `packaging/kejilion-app/kpanel.conf` 自 rc.6 无变化，仓库与 `kejilion/apps` 本地文件 blob 均为 `fa4b95374ed3b186d920207537f451a5b6d839f7`；无需应用市场提交，默认安装继续指向稳定版。

## 多维质量结论

| 维度 | 状态 | 证据与边界 |
| --- | --- | --- |
| 业务正确性与双端互通 | 已实现未实机验证 | 当前外观由 Panel 状态管理，Go/前端测试及模拟浏览器通过；真实双设备与 Nginx 未测。脚本双端资源管理契约本次未变。 |
| 网络入侵与供应链安全 | 已实现未实机验证 | 匿名响应仅为当前缩略图、大小上限、旧 URL 撤销与安全入口有 Go 测试；L3 govulncheck、npm audit、Trivy 通过，CF scoped 仍待完成。 |
| 稳定性、失败恢复与兼容 | 已实现未实机验证 | 保存冲突、未提交修改、会话重挂载和退出失败有测试；L3 race 与镜像生命周期通过。 |
| 性能与资源预算 | 已实现未实机验证 | 登录配置复用已有 bootstrap；镜像运行约束通过；低配真实浏览器启动时延未测。 |
| 用户体验与可访问性 | 已实现未实机验证 | 精确候选的模拟 API 浏览器验证登录页、窄视口、200% CSS 缩放、桌面首帧、经典通透、自定义缩略图且控制台错误为 0；真实主机未测。 |
| 数据、配置与迁移 | 已实现未实机验证 | 外观旧缓存迁移与服务器版本冲突有回归测试；真实旧数据升级未单独执行。 |

## 自动门禁与隔离验收

- 候选本地测试：213 个前端测试文件共 1926 项通过，类型检查、i18n、构建通过。依赖修复后 `npm audit --audit-level=moderate` 为 0 漏洞。
- `arena-154` SSH 超时；按已授权的 `local-wsl-dr` 在 Ubuntu WSL、root Docker 上用固定 Runner `sha256:a9f708891d1e81f17dd286bc7e9d6124658964716dc9a457b5c73340722f6a7d` 完成 L3。成功 run ID `v1.23.0-rc.7-13e17d0f-l3-r3`，2026-09-30 12:37:04 至 12:44:47（北京时间），`status=passed`、`exit_code=0`，证据目录 `C:/GitHub/_validation/v1.23.0-rc.7-13e17d0f-l3-r3`。
- 固定 bundle SHA-256 `efe92d3b4c90888e40445b0e17c16e72974f6f13736b40ea808defda7a6273f7`，plan `41c4d786cf734b71cc50f71d5778219c1abc021052b69f8a83d5a84cdbfcd3a8`，执行脚本 `21c08b11be3526a0fe766aa606a81d0e4047e8a4e89d79227da4e62914164979`；终态原件 `wsl-evidence/status.txt`。
- 首轮执行将 rc.6 误用为 L3 的稳定基线，入口在准备前拒绝；第二轮对 `26115ba0` 的 `npm audit` 发现 `brace-expansion` 与 `undici` 两项 high，正常阻断。补丁分支将 `markdown-it` 升至 15.0.2、`brace-expansion` 升至 2.1.7、`undici` 升至 8.11.2 后，完整 L3 从头重跑成功，失败证据未覆盖。
- 本地 `acceptance` 模拟预览绑定 `13e17d0f`，六项浏览器检查通过：登录主题/壁纸、390px 窄屏、200% CSS 缩放、当前自定义缩略图、旧缓存覆盖后的桌面首帧和经典模式“通透”；没有请求默认壁纸，页面脚本错误为 0。证据在 `C:/GitHub/_validation/kpanel-v123-rc7-13e17d0f-appearance-preview`，运行进程已精确停止。Mock 不证明真实 API 或 Nginx。
- 候选 [CI](https://github.com/kejilion/KPanel/actions/runs/36670315088) 与 [Dependency freshness](https://github.com/kejilion/KPanel/actions/runs/36670315016)、主线 [CI](https://github.com/kejilion/KPanel/actions/runs/36671113459) 与 [Dependency freshness](https://github.com/kejilion/KPanel/actions/runs/36671113451) 均在精确 SHA `13e17d0f` 上成功。标签 [Release workflow](https://github.com/kejilion/KPanel/actions/runs/36671782891) 与 Dependency freshness `36671782795` 成功。

## 发布产物与公开仓库复核

- [GitHub Release v1.23.0-rc.7](https://github.com/kejilion/KPanel/releases/tag/v1.23.0-rc.7) 于 2026-09-30 13:18:30（北京时间）公开，`prerelease=true`、`draft=false`，共 14 个附件。`SHA256SUMS` 文件 SHA-256 为 `4cc06778778b0de9e2273b999edd76e761c3f600ae4b87790e748039925f624a`；其中 11 条校验值与公开附件的 API 摘要逐一一致。
- Docker Hub `kjlion/kejilion-panel:1.23.0-rc.7` 与 `:preview` 共享多架构 index `sha256:c411cf040ef153c9b6e1ca14ce9f2389df7d481b5e88f448f7922bc8ae12ec1b`；amd64 descriptor `sha256:818c197f70467907739fd918feedbf238ac035ef2dba10eb771f210ad5e17561`、arm64 descriptor `sha256:5809016b184db502c19cab6d56a3dca9cbe18ab36fee25892561220d5916a1e7`。`latest` 保持稳定版 index `sha256:46b854bdf6233e81742986a3bee9e65c6b4a39123c393a51a4ada63783eed7ac`；GitHub Latest 仍为 `v1.22.0`。
- 从 Docker Hub 公开标签复制 amd64 镜像后，在 `local-wsl-dr` 执行仓库 `packaging/tests/image-e2e.sh`，结果 `image_e2e=pass`。镜像 OCI revision 为 `13e17d0f670b479f2112b37f97f48a8033597d16`，版本为 `1.23.0-rc.7`；公开镜像内 `/release/kejilion.sh` SHA-256 为 `33010d547355f9bde4067c189a0457f44dc00ec3a72eaa25e5b7101e1fcb9c04`。公开证据位于 `C:/GitHub/_validation/v1.23.0-rc.7-13e17d0f-l3-r3/public`，`image-e2e.log` 保留终态。

## 自更新、生产与回滚

- 自更新通道契约未改：稳定来源只选择正式 Latest；预览来源可选择规范 RC，加入预览不自动安装，退出预览不自动降级。已有测试和 L3 覆盖自更新选择、备份与失败恢复；本轮未在真实宿主重测。
- 生产部署安全核对：**不适用（预览版禁止生产部署）**。本轮未连接、备份、部署、升级或核对 `prod-108`，也未执行生产写操作；本地 WSL L3 不能代替生产证据。
- 源码稳定回滚点 `v1.22.0`；镜像稳定回滚点为上方 `latest` index。上一预览 `v1.23.0-rc.6` 是不可变预览回退目标；实际回退前仍需备份并核对数据兼容。切换通道不会自动降级。

## 交付节奏数据

<!-- kpanel-release-metrics:start -->
- 首个纳入提交时间：2026-09-30T10:16:25+08:00
- 候选冻结时间：2026-09-30T12:37:04+08:00
- 生产完成时间：不适用
- 提交到生产用时：不适用
- 是否回滚、紧急热修复或重复发布：否
- 若发生失败，发现时间、恢复时间和逃逸门禁：不适用
<!-- kpanel-release-metrics:end -->

<!-- kpanel-release-process-metrics:start -->
- 已记录发布流程异常或无效证据拦截次数：2
- 其中生产写操作开始后异常次数：0
<!-- kpanel-release-process-metrics:end -->

<!-- kpanel-release-process-incidents:start -->
[
  {"fingerprint":"l3/base-tag/preview-as-stable","position":"before-production-write","count":1,"impact":"首次 L3 把上一 RC 当成 --base-tag，入口要求稳定 Tag 并在准备前拒绝；没有执行候选测试。","recoveryEvidence":"改用 v1.22.0 和新的 run ID，成功的 r3 证据目录保留，原失败未改写。","permanentAction":"发布负责人于 2026-10-06 前修正 release-kpanel 工作流的 L3 稳定基线参数说明，并加入发车前参数核对。","historicalReleases":[]},
  {"fingerprint":"browser/playwright/headless-binary-missing","position":"before-production-write","count":1,"impact":"首次模拟浏览器启动缺少 Playwright 自带 headless shell，未产生浏览器结果。","recoveryEvidence":"显式使用已安装 Chrome，精确候选 13e17d0f 的六项浏览器检查通过且进程停止。","permanentAction":"后续固定浏览器命令规格先检查并显式指定已登记 Chrome 可执行文件；发布负责人 2026-10-06 复核。","historicalReleases":[]}
]
<!-- kpanel-release-process-incidents:end -->

## 遗留风险与分支收尾

- `internal/backupremote` 的 CF scoped 审计、真实 Nginx 缓存与隔离真机浏览器验收需在稳定版前补齐；本次公开镜像 E2E 不替代这些场景。
- 本次四项来源分支在公开产物与验收证据确认后按精确 tip 保存到远端 `archive/` 并复核，远端没有对应活跃 `fix/` 引用；本地分支已改为相同归档名。来源工作树中的 `node_modules`、构建目录及本地日志属于被 Git 忽略的文件，暂保留工作树以避免误删唯一证据；其保留不代表仍在发布队列。发布候选 `release/v1.23.0-candidate` 保留以继续同序列预览，其他未知改动工作树原样保留。
