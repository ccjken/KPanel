# KPanel v1.23.0-rc.1 发布验收记录

日期：2026-09-27。发布级别：L3。`releaseChannel=preview`；`releaseTrain=1.23.0`。

- 候选提交：`f8d4a42e51180fa409e2f3a5889b115c4dea13e7`；注释标签 `v1.23.0-rc.1`，tag object `b4449ff4439044a22ece2afa1de28ace9e248342`，指向同一提交。
- 发布前主线：`3ebd4e64c4627381b32bff42b92632bdab1c13c8`（v1.22.0 验收提交）。上一稳定版与回滚点：`v1.22.0`，镜像 index `sha256:46b854bdf6233e81742986a3bee9e65c6b4a39123c393a51a4ada63783eed7ac`。
- 发布候选分支：`release/v1.23.0-candidate`，远端和主线均为 `f8d4a42e`；预览序列期间保留，待 1.23.0 稳定版后按 10.2 归档。
- 发布工作树：`C:/GitHub/_codex-tasks/kpanel-v123-preview-release`；验收工作树：`C:/GitHub/_codex-tasks/kpanel-v123-preview-acceptance`。证据：`C:/GitHub/_release-evidence/v1.23.0-rc.1-20260927`。

预览产物已公开，生产未部署。本版不改变公共稳定默认入口。

## 发布画像与范围

- 业务域：外观同步、3D 场景切换、终端交互、应用安装生命周期。
- 变更面：认证后的外观设置 API、Panel 持久化与备份字段、前端渲染、应用配置锁兼容；无新增 Agent 权限、端口或数据库迁移。
- 用户旅程：既有浏览器偏好首次同步到面板；另一浏览器登录后读取壁纸、主题与经典模式透出；自定义图上传后在经典模式和登录页显示；3D 场景切换与登录封面；终端右键复制/粘贴；BusyBox `flock -n` 环境下安装/更新/卸载互斥。
- 边界：登录页私有图片副本只保存在该浏览器；新浏览器需要登录后取得外观选择并生成自己的副本。上传图和下载的场景包不随设置备份迁移。
- 风险等级：L3。认证写路径、场景加载状态与应用生命周期受影响。

| 来源分支与原 tip | 纳入候选的提交 | 内容 |
| --- | --- | --- |
| `fix/busybox-lifecycle-flock` `6d4d8c54` | `6d4d8c54` → `b8347e7c` | BusyBox 生命周期锁兼容 |
| `feature/wallpaper-server-sync-20260926` `b7f18d74` | `555871e7` → `b37219f5`；`b8b0b2b0` → `e7f7322c` | 外观服务端同步；试验提交 `72a6e9cd` 与其撤销提交 `b7f18d74` 未重放 |
| `fix/scene-loading-poster-20260927` `ddf78065` | `dd57af42..ddf78065` 九个提交，结果 `ea4a1950..5583a612` | 场景海报、加载反馈与过渡 |
| `fix/terminal-clipboard-direct-20260926` `cc83e302` | `cc83e302` → `7b91b1e4` | 终端右键复制粘贴 |

集成版本准备提交 `70f17ea3`，应用市场元数据对齐提交 `f8d4a42e`。`feature/desktop-dynamic-scenes` 工作树含未提交改动，未纳入或归档；`feat/terminal-file-transport-v3` 的关键安全修复与既有主线 patch 等价，其余提交未纳入，保持原样供后续判断。

## 外部审计与修复交付

- 冻结候选的安全覆盖检查 `decision=ok`，8 个未审计边界提交、无新增边界包。本 RC 未执行新的 CF scoped/full 审计；不把覆盖检查说成安全审计完成。
- OCR 行审覆盖候选变更的 39/39 个可审文件，自由审查无发现；候选状态核验记录 `ocr_line_review=recorded`、34 个代码路径、1100 行。最后的应用元数据 URL 对齐单独记录为有理由的 skipped，不改变受审代码路径。本轮没有另一模型提供商的独立复核记录。
- 源码、候选分支、主线、标签与公开版本镜像均绑定 `f8d4a42e`；生产部署不适用。没有新增安全 finding 或修复 fingerprint。

## 跨仓库联动判定

- `scriptLinkageState=not-required`（无需发布脚本）。内置 `kejilion.sh` 基线 `2b90b2d2ca56bc954c9328a51bb5571e896f713d`，SHA-256 `806b4715664fad502f7faeccbc75972f2d1a24559d46208221b98f774ef56c99`，相对主线未变；无脚本候选与跨仓库变更集。
- `kejilion/apps` 的 `origin/main` 已包含 BusyBox 修复。本版内置 `packaging/kejilion-app/kpanel.conf` 与该仓库 `origin/main:kpanel.conf` 的 Git blob 同为 `38f7d9c37a2b55cec6b2292ff46dd22aab120b2c`，无需新应用市场提交；默认入口继续为 `latest`。

## 多维质量结论

| 维度 | 状态 | 证据与边界 |
| --- | --- | --- |
| 业务正确性与双端互通 | 已验证 | 公开镜像的真实 Panel API、两独立浏览器会话、上传和外观持久化通过；Agent/远端 Node 未实机验收 |
| 网络入侵与供应链安全 | 已验证 | L3 Trivy 源码/镜像、govulncheck、npm audit、CI 与 Release 扫描通过；本 RC 无新 CF 审计 |
| 稳定性、失败恢复与兼容 | 已实现未实机验证 | Go race、应用配置生命周期及 BusyBox 定向测试通过；真实 systemd 更新/回滚未执行 |
| 性能与资源预算 | 已实现未实机验证 | 镜像运行时契约通过；低端 GPU 与长期场景资源趋势未测 |
| 用户体验与可访问性 | 已验证 | 本地 Chrome 1280×800 的公开镜像浏览器旅程与场景组合截图通过、页面异常 0；其他浏览器/缩放矩阵未测 |
| 数据、配置与迁移 | 已验证 | 外观服务端状态及跨会话读取通过；无数据库迁移，上传图/场景包备份边界按既有设计保留 |

## 自动门禁

- Windows 本地 `npm ci`、`npm run typecheck`、定向 Vitest 7 文件/76 测试及 `app-conf-lock-compat.sh` 通过。Windows 无 Go/gofmt，完整发行验证由固定 Linux Runner 完成。
- L3：`scripts/run-release-l3.mjs`，run ID `v1.23.0-rc.1-f8d4a42e-l3-r2`，2026-09-27T02:47:15Z 至 02:55:51Z，`local-wsl-dr`，`status=passed`、`exit_code=0`。固定 Runner ID `sha256:a9f708891d1e81f17dd286bc7e9d6124658964716dc9a457b5c73340722f6a7d`；Runner 归档 SHA-256 `f533e0db78d328ee5504e511a22e9769e6be55f53cbd96a3a6ae47f04a969509`。
- L3 bundle SHA-256 `90ed3ab4def0332039773f6c7600402f3d5bf693aa1de6f6836d81ec0ad1b870`；plan `bf598156338cf38fff1338480d2d42c5a6078efed56d5ad5c367d615384b6763`；执行脚本 `21c08b11be3526a0fe766aa606a81d0e4047e8a4e89d79227da4e62914164979`。证据在 `l3-r2/wsl-evidence`。Go 全套/race、前端 196 文件/1728 测试、Trivy、govulncheck、npm audit、镜像构建和应用生命周期均通过。
- 候选 CI [36289959935](https://github.com/kejilion/KPanel/actions/runs/36289959935) success、依赖新鲜度 36289959884 success；主线 CI [36290430233](https://github.com/kejilion/KPanel/actions/runs/36290430233) success、依赖新鲜度 36290430246 success。
- Release [36290933934](https://github.com/kejilion/KPanel/actions/runs/36290933934) success，标签依赖新鲜度 36290933951 success；精确源码 SHA 均为 `f8d4a42e`。
- 本版未升级 Go/npm 依赖、基础镜像、Action、扫描器或受管脚本；锁文件仅更新产品根版本。候选和主线依赖新鲜度检查通过。

## 隔离浏览器与公开镜像验收

- `arena-154` SSH 连接超时；按先前用户选择，仅让登记的 `local-wsl-dr` 承担候选 L3。下述本地 WSL/Chrome 浏览器验收是补充证据，不冒充登记隔离真机的 browser-validation。
- 候选构建镜像和公开镜像均在独立数据目录、Docker 网络和非 root、只读容器中运行。公开镜像从 Docker Hub 显式拉取，RepoDigest 为 `sha256:f2eabc4249a8f3f0562d31bd8e3eeb896a2130f806b7962d36db2bf8ed8d470f`；`packaging/tests/image-e2e.sh` 输出 `image_e2e=pass`。
- 公开镜像的 Chrome 1280×800：上传 2560×1440 图片，预览成功；服务端外观设置持久化，第二个浏览器登录后读取自定义壁纸与经典模式“通透”，退出登录后显示私有图片副本，页面异常 0。场景包 `orbital-station` 34 个文件按 catalog SHA-256 校验后注入隔离数据目录；场景在经典模式显示，切为静态壁纸再切回只有一个 iframe，登录页显示场景 WebP 封面，页面异常 0。证据见 `public-preview-browser/`。
- 隔离环境的场景仓库曾不可达，故本次未证明在线下载链路；未执行 systemd 宿主机升级、真实 Agent、arm64 真机、其他浏览器与 125%/200% 全视口矩阵。隔离测试容器和网络已清理。

## 发布产物与公开仓库复核

- [GitHub Release v1.23.0-rc.1](https://github.com/kejilion/KPanel/releases/tag/v1.23.0-rc.1) 于 2026-09-27T03:28:26Z 公开，`draft=false`、`prerelease=true`；GitHub Latest 仍为 `v1.22.0`。
- Docker `1.23.0-rc.1` 与 `preview` OCI index 均为 `sha256:f2eabc4249a8f3f0562d31bd8e3eeb896a2130f806b7962d36db2bf8ed8d470f`。`linux/amd64` 为 `sha256:898ebb4c4ccd0fb21ce34ca63945996852fbfaf99fb55961ffadfa1b8da2f1e9`，`linux/arm64` 为 `sha256:cd34013c8dbe3e3c34b6fee57d8f7aa2bceca20d321aa1dd53fbbf3b3a8db088`；两标签的平台摘要一致。
- Docker `latest` 与稳定版本 `1.22.0` 仍共用 `sha256:46b854bdf6233e81742986a3bee9e65c6b4a39123c393a51a4ada63783eed7ac`。Release 共 14 个附件，`SHA256SUMS` 的 11 条均与 GitHub 附件 digest 一致；证据为 `github-release-api.json`、`SHA256SUMS`、`docker-tags.json`。
- 自更新：预览来源只发现规范 RC/稳定版，加入预览与自动安装互相独立；本版未修改策略代码，沿用 1.22.0 回归。退出预览不会自动降级。systemd 更新事务、OpenRC 和轻量 Node 边界未在本轮实机重复验收。

## 生产适用性与回滚

- 不适用（预览版禁止生产部署）。本次未连接、备份、部署、升级或核对 `prod-108`，亦未执行任何生产写操作；`arena-154` 未建立 SSH 连接。
- 源码回滚点为 `v1.22.0`；稳定镜像 index 为 `sha256:46b854bd…ed7ac`。预览用户若需回到旧版，须独立选择旧版不可变 digest 与备份并验证恢复；退出预览开关不会降级。GitHub Latest、Docker `latest` 与应用市场默认入口保持 1.22.0。

## 分支与资源处置

- `release/v1.23.0-candidate` 保留为本发布序列唯一候选，远端精确 tip `f8d4a42e`。
- 本轮四个 clean 的来源工作树原 tip 已逐一保存到远端和本地 `archive/<原分支全名>`，远端复核均与原 tip 完全一致，活跃原分支在远端不存在、在本地已重命名；原先跟踪 `origin/main` 的两个归档分支已解除 upstream。映射：`fix/busybox-lifecycle-flock` → `archive/fix/busybox-lifecycle-flock` `6d4d8c54`；`feature/wallpaper-server-sync-20260926` → `archive/feature/wallpaper-server-sync-20260926` `b7f18d74`；`fix/scene-loading-poster-20260927` → `archive/fix/scene-loading-poster-20260927` `ddf78065`；`fix/terminal-clipboard-direct-20260926` → `archive/fix/terminal-clipboard-direct-20260926` `cc83e302`。归档 refs 保留试验/撤销历史，不将其冒充已发布净变更。
- 来源 worktree 保留作恢复入口；用户未要求删除工作树。未纳入且含未提交内容的 `feature/desktop-dynamic-scenes`、归属/差集尚未清晰的其他历史分支均保留，不归类为本版待发布候选。本验收记录分支将在同 SHA 的候选与主线 CI 通过后归档。

## 交付节奏数据

<!-- kpanel-release-metrics:start -->
- 首个纳入提交时间：2026-09-26T21:15:38+08:00
- 候选冻结时间：2026-09-27T10:45:01+08:00
- 生产完成时间：不适用
- 提交到生产用时：不适用
- 是否回滚、紧急热修复或重复发布：否
- 若发生失败，发现时间、恢复时间和逃逸门禁：不适用
<!-- kpanel-release-metrics:end -->

本版为首个 1.23.0 RC，不计稳定发布或生产部署频率。

<!-- kpanel-release-process-metrics:start -->
- 已记录发布流程异常或无效证据拦截次数：4
- 其中生产写操作开始后异常次数：0
<!-- kpanel-release-process-metrics:end -->

<!-- kpanel-release-process-incidents:start -->
[
  {
    "fingerprint": "candidate-l3/app-metadata/drift",
    "position": "before-production-write",
    "count": 1,
    "impact": "首轮 L3 在预检后发现内置应用配置的 app_url 与已发布应用市场配置不同，主动中断；首轮没有有效 L3 终态。",
    "recoveryEvidence": "候选提交 f8d4a42e 对齐 Git blob 38f7d9c3；新 run ID v1.23.0-rc.1-f8d4a42e-l3-r2 的完整 L3 status=passed，旧证据保留。",
    "permanentAction": "发布任务在下次候选冻结前比较内置配置与应用市场 origin/main 的精确 Git blob；退出条件是 L3 首轮前完成一致性核验。",
    "historicalReleases": []
  },
  {
    "fingerprint": "browser-validation/arena-154/ssh-timeout",
    "position": "before-production-write",
    "count": 1,
    "impact": "登记隔离环境 SSH 超时，未取得该环境的公开镜像 browser-validation 证据。",
    "recoveryEvidence": "仅将候选 L3 切到登记的 local-wsl-dr；公开镜像在本地 WSL/Chrome 补充验证，明确不冒充 arena-154。",
    "permanentAction": "下一次稳定版发布前复核 arena-154 SSH 与 browser-validation；退出条件是在登记隔离环境对同一公开镜像完成浏览器 E2E。",
    "historicalReleases": []
  },
  {
    "fingerprint": "browser-validation/scene-catalog/unavailable",
    "position": "before-production-write",
    "count": 1,
    "impact": "隔离浏览器首次读取场景仓库失败，无法直接完成本轮场景切换旅程。",
    "recoveryEvidence": "对仓库内 orbital-station 34 个文件逐项校验 catalog SHA-256 后注入隔离数据目录；重启容器，公开镜像场景、切换和登录封面验收通过。在线下载链路仍未验证。",
    "permanentAction": "下次场景下载改动时在可访问场景仓库的登记环境另测下载线路；退出条件是公开镜像从仓库完成下载和安装。",
    "historicalReleases": []
  },
  {
    "fingerprint": "acceptance-ci/metrics/recovery-sentinel",
    "position": "before-production-write",
    "count": 1,
    "impact": "验收记录首轮 CI 的指标校验拒绝带解释的“不适用”恢复字段，文档候选未能直接进入主线。",
    "recoveryEvidence": "将无产品失败时的恢复字段改为精确“不适用”，本地 report-release-metrics 验证通过；重新提交并等待同 SHA 候选 CI。",
    "permanentAction": "后续验收记录提交前先运行 report-release-metrics --validate-acceptance；退出条件是文档首轮 CI 不因已知指标格式失败。",
    "historicalReleases": []
  }
]
<!-- kpanel-release-process-incidents:end -->

## 遗留风险与后续准入

- 1.23.0 稳定版前补足登记环境的公开镜像浏览器验收，并复核场景在线下载线路。当前本地公开镜像 E2E 已通过，但不扩大 `local-wsl-dr` 的登记用途。
- 本轮未部署生产。归档 refs 只作精确恢复位置，不作为生产上线证明。
