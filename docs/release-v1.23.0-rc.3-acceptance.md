# KPanel v1.23.0-rc.3 发布验收记录

日期：2026-09-28。发布级别：L3。`releaseChannel=preview`；`releaseTrain=1.23.0`。

- 候选、远端 `main` 与标签均指向 `1e74797eeba26cac4351517c6b2c82f4694f058e`。注释标签 `v1.23.0-rc.3` 的 tag object 为 `8e16b918be2ced04c6eef4c1f1e1ddbdeaca2c7e`。
- 发布前 `main=6e95eca461a86a001c43c011ab899b43a50c9088`；上一稳定版和源码回滚点为 `v1.22.0`，稳定镜像 index `sha256:46b854bdf6233e81742986a3bee9e65c6b4a39123c393a51a4ada63783eed7ac`。
- 证据目录：`C:/GitHub/_release-evidence/v1.23.0-rc.3-20260928`。本预览版仅发布产物，生产未部署。

## 发布画像与范围

- 业务域：浅色经典模式侧边栏与壁纸、3D 场景加载占位层、首次安装 Token 输出。
- 受影响旅程：选择图片并启用“通透”后在经典模式侧边栏透出；切换“氛围/关闭”及深色主题；自定义图片上传后在经典模式显示并在刷新后保留；3D 场景加载期间占位保持暗色；复制首次初始化 Token。
- 变更面：前端样式与加载标记、安装文案及其生命周期断言。没有新增 API、数据库格式、Agent 权限、端口或 `kejilion.sh` 变更。因上一 RC 已发生视觉逃逸，按 L3 验收。

| 来源分支与原 tip | 候选重放 | patch-id |
| --- | --- | --- |
| `fix/initial-token-copy` `276dfb60`（含前一提交 `b26b8f34`） | `7da41fbb`、`871a6258` | `45ca3897`、`86dfb8de`，逐一一致 |
| `fix/wallpaper-loading-dim-20260927` `6ce2bd78` | `6c89ad25` | `f5686628`，一致 |
| `fix/classic-wallpaper-layer-20260928` `3c8d7f81` | `c6fdb443` | `45a5d008`，一致 |

发布准备、两次 OCR 记录和说明提交为 `3a49b53c`、`81f3e07f`、`5bd81e9f`、`1e74797e`。没有纳入其他动态场景实验分支。

## 故障定位与修正

- 公开 `rc.2` 镜像中，浅色主题的侧边栏把 `--classic-wallpaper-sidebar` 强制覆盖为 `94%`，即使“通透”已选中；根定义原本分别为“通透” `70%`、“氛围” `86%`。覆盖来自 `dd1ed6aa` 侧边栏优化提交，而非 `37dd8233` 的共用壁纸切换层。
- 针对公开 `rc.2` 的浏览器回归在 `94% ≠ 70%` 处失败；`rc.3` 删除该覆盖后，候选镜像与公开镜像均验证浅色“通透” `70%`、浅色“氛围” `86%`、关闭时无壁纸层、深色“通透” `58%`，页面脚本异常 0。
- 公开镜像在 1920×920 浅色页面使用自定义图片时，图片请求成功且 `naturalWidth=2560`；隐藏壁纸层会明显改变左右空白区的像素。页面其余区域仍沿用既有 `62%` 遮罩，本版没有调整其视觉强度。用户截图里的 Golden Gate 原图及其线上反向代理环境未直接取得，不能由隔离容器推断其现场状态。
- 曾把壁纸层级误判为根因并设计了会在 `rc.2` 误通过的像素用例。该改动、用例和其后旧候选的 L3 证据均废弃；改用公开 `rc.2` 失败、修正候选及公开 `rc.3` 通过的同一计算样式回归。

## 外部审计与修复交付

- 安全覆盖检查对标签 `v1.23.0-rc.3` 返回 `decision=ok`，相对最近 full run-4 有 10 个边界提交，均在已有 scoped 记录覆盖范围；没有新增边界包。本 RC 没有执行新的 CF scoped/full 审计，也不把覆盖检查称为审计。
- OCR 在 `6e95eca..3a49b53` 已覆盖 6/6 文件；修正后重跑 `6e95eca..5bd81e9` 覆盖 7/7 文件，自由臂 0 个有效发现、`H0/M0/L0`、`constrained-only=0`。自由臂已见前轮记录，`blind=false`，不计作新的盲测。记录位于仓库外 `ocr-r2/`，最终 trailer 为 `1e74797e`。
- 源码、候选、主线、标签、Release 与公开镜像绑定上述精确提交；生产部署不适用。

## 跨仓库联动判定

- `scriptLinkageState=not-required`，内置 `kejilion.sh` 未改；跨仓库脚本变更集不适用。
- 安装输出文案变化要求同步应用市场 `kpanel.conf`。KPanel 内置文件和 `kejilion/apps` 的新文件 Git blob 均为 `fa4b95374ed3b186d920207537f451a5b6d839f7`；`kejilion/apps` 的 `main` 已快进至 `33c7c1cc03fbc7eeee48c2846b3104f060c75416`，Bash 语法通过，KPanel L3 对相同文件执行了应用配置生命周期测试。默认安装镜像仍为稳定 `latest`，未改成 `preview`。

## 多维质量结论

| 维度 | 状态 | 证据与边界 |
| --- | --- | --- |
| 业务正确性与双端互通 | 已验证 | 公开镜像的设置切换、自定义图上传与刷新后服务端读取通过；真实 Agent/远端 Node 未本轮实机复测 |
| 网络入侵与供应链安全 | 已验证 | L3 的 Trivy 源码及最终镜像、govulncheck、npm audit 与候选/主线/Release CI 通过；本 RC 无新 CF 审计 |
| 稳定性、失败恢复与兼容 | 已实现未实机验证 | Go race 与应用配置备份生命周期通过；真实 systemd/OpenRC 更新回滚未执行 |
| 性能与资源预算 | 已实现未实机验证 | 镜像运行约束和双架构构建通过；低端 GPU 的 3D 场景长测未本轮重复 |
| 用户体验与可访问性 | 已验证 | Chrome 1280×800、1920×920 的浅/深色壁纸组合及截图；其他浏览器、200% 缩放和原用户现场未验证 |
| 数据、配置与迁移 | 已验证 | 自定义图在隔离容器刷新后仍被选中并加载；本版没有持久化格式迁移 |

## 自动门禁与浏览器验收

- 版本一致性检查通过；本地浏览器自动回归在公开 `rc.2` 失败，在候选 `1e74797e` 和公开 `rc.3` 通过。模拟预览绑定 `1e74797e` 的用例通过，但其外观 API 读取失败 Toast 仅属 mock 边界，不作为真实持久化证据。
- L3：`scripts/run-release-l3.mjs`，`local-wsl-dr`，run ID `v1.23.0-rc.3-1e74797e-l3-r2`，2026-09-28T03:13:49Z 至 03:21:27Z，`status=passed`、`exit_code=0`。Runner ID `sha256:a9f708891d1e81f17dd286bc7e9d6124658964716dc9a457b5c73340722f6a7d`，离线归档 SHA-256 `f533e0db78d328ee5504e511a22e9769e6be55f53cbd96a3a6ae47f04a969509`；bundle `9f5d98babb2f5ef97b5b6f13aa42384069c13647721f06cc1bef7d278918a2cd`，plan `71b114447837dc62e815ed1c99764b5b43ad2aa97df1f39c3ed3dca6803b10ad`，执行脚本 `21c08b11be3526a0fe766aa606a81d0e4047e8a4e89d79227da4e62914164979`。
- L3 覆盖 Go、race、前端测试与类型检查、Trivy、govulncheck、npm audit、镜像构建及安装配置生命周期。候选 CI [#1110](https://github.com/kejilion/KPanel/actions/runs/36373486325)、主线 CI [#1111](https://github.com/kejilion/KPanel/actions/runs/36373925384)、Release [#261](https://github.com/kejilion/KPanel/actions/runs/36374459039) 均为 success；各自的 Dependency freshness 也成功。
- `arena-154` 此轮未重新连通核验；按用户此前选择仅以 `local-wsl-dr` 承担候选 L3。本地 WSL/Chrome 对候选与公开镜像的浏览器试验是补充证据，不冒充登记隔离真机验收。隔离容器里未部署 Agent，`capabilities`、`automatic-update` 与 `agent/health` 的 503、场景库请求的 502 已在响应清单定位；壁纸上传/图片请求未失败。测试容器、网络、数据目录与本地预览已清理。

## 发布产物与公开仓库复核

- [GitHub Release v1.23.0-rc.3](https://github.com/kejilion/KPanel/releases/tag/v1.23.0-rc.3)：`draft=false`、`prerelease=true`，2026-09-28T03:49:13Z 公开；GitHub Latest 仍为 `v1.22.0`。
- Docker `1.23.0-rc.3` 与 `preview` 的 OCI index 均为 `sha256:d6690b097d9c1a4282132eb56258bd73f67d83ee859bf82099d063d9bd42c6b8`，`linux/amd64` 为 `sha256:23072c16550081f1e61a1cf5bcb2802e3139ee5842f4d1e7e4f05bb89cef46c8`，`linux/arm64` 为 `sha256:c26ee27d444620d57df79b928282f47c177f63423f581437b7c14561409cdc06`。`latest` 与 `1.22.0` 仍为上一稳定版 index `sha256:46b854bd...`。
- Release 有 14 个附件，含 Agent 双架构、部署归档及 `SHA256SUMS`。校验文件 digest 为 `sha256:6d17a8c587e6e7043b9f08714f58924bff733c4e6e00a41f4e616836b70c822a`；其 11 条均与 GitHub 附件 digest 相同。
- 从 Docker Hub 显式拉取 `kjlion/kejilion-panel:1.23.0-rc.3`，拉取摘要与上述 OCI index 相同；`packaging/tests/image-e2e.sh` 输出 `image_e2e=pass`。同一公开镜像的 Chrome 通过浅色通透/氛围/关闭、深色通透、自定义图上传与刷新，以及 1920×920 画面检查。
- 自更新通道、自动安装与退出预览不降级策略本版未改，沿用上一版本的覆盖；没有声称本轮重测真实 systemd 更新。

## 生产适用性与回滚

- 不适用（预览版禁止生产部署）。未连接、备份、部署、升级或核对 `prod-108`，也未在正式环境执行安装；没有生产后版本或健康状态可报告。
- 源码回滚点为 `v1.22.0`，稳定镜像 index `sha256:46b854bdf6233e81742986a3bee9e65c6b4a39123c393a51a4ada63783eed7ac`。预览用户回退需自行选择旧版不可变 digest、备份并验证恢复；离开预览不会自动降级。GitHub Latest、Docker `latest` 与应用市场标准安装入口仍指向稳定版。

## 分支与资源处置

- `release/v1.23.0-candidate` 作为本发布序列唯一候选保留，远端精确 tip `1e74797e`，待 1.23.0 稳定版按规范归档。
- 本轮三个 clean 来源分支的原始 tip 分别保存在远端 `archive/fix/initial-token-copy=276dfb60`、`archive/fix/wallpaper-loading-dim-20260927=6ce2bd78`、`archive/fix/classic-wallpaper-layer-20260928=3c8d7f81`；均已远端核验，旧活跃远端引用不存在，本地原分支已改为 `archive/` 且无旧 upstream。前两条来源工作树归属其他任务，保留不强删；本轮自建的经典壁纸修正工作树已回收。
- 应用市场同步分支原 tip `33c7c1c` 已保存为 `archive/release/kpanel-rc3-app-sync-20260928`，远端 `main` 含同一提交；本地分支无 upstream，同步工作树已回收。验收记录分支在同 SHA 候选与主线 CI 成功后另行归档。仓库外证据保留。
- 先前未解决的其他分支/工作树不属于本轮已核实来源，不凭名称推断已上线或强制清理。

## 交付节奏数据

<!-- kpanel-release-metrics:start -->
- 首个纳入提交时间：2026-09-27T16:33:37+08:00
- 候选冻结时间：2026-09-28T11:10:46+08:00
- 生产完成时间：不适用
- 提交到生产用时：不适用
- 是否回滚、紧急热修复或重复发布：是（rc.2 浅色侧栏透出回归，rc.3 修正并重新发布预览）
- 若发生失败，发现时间、恢复时间和逃逸门禁：发现时间: 2026-09-28T10:42:36+08:00; 恢复时间: 2026-09-28T11:49:13+08:00; 逃逸门禁: 已逃逸: rc.2 浏览器验收没有核对通透模式侧栏的不透明度计算值
<!-- kpanel-release-metrics:end -->

发现时间采用本轮最早留下的诊断提交时间；用户首次看到问题的准确时间未记录。恢复时间采用公开 Release 时间，线上用户自行升级完成时间未验证。

<!-- kpanel-release-process-metrics:start -->
- 已记录发布流程异常或无效证据拦截次数：1
- 其中生产写操作开始后异常次数：0
<!-- kpanel-release-process-metrics:end -->

<!-- kpanel-release-process-incidents:start -->
[
  {
    "fingerprint": "browser-validation/classic-layer/false-positive",
    "position": "before-production-write",
    "count": 1,
    "impact": "初始像素探针在有问题的 rc.2 上也误通过，且由此形成的试验性 z-index 修复使旧候选 L3 证据失效。",
    "recoveryEvidence": "公开 rc.2 上明确失败的侧边栏计算值回归；删除试验改动，在候选 1e74797e 重新完成 L3、CI 及公开镜像验收。",
    "permanentAction": "视觉回归先在已知失败的公开版本上确认能失败，再用同一用例验证修正版本；将侧边栏透明度与背景图加载分别断言。",
    "historicalReleases": []
  }
]
<!-- kpanel-release-process-incidents:end -->

## 遗留风险

- 用户原有 Golden Gate 图片及线上反向代理、浏览器缓存未纳入本地隔离试验；如现场仍无图，需读取该页面实际图片请求、计算样式与缓存版本。场景库在线读取在隔离容器返回 502，未将其等同于已验证的场景在线下载。
- 本版不部署生产；归档引用只是恢复位置，不是生产上线证明。
