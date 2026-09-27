# KPanel v1.23.0-rc.2 发布验收记录

日期：2026-09-27。发布级别：L3。`releaseChannel=preview`；`releaseTrain=1.23.0`。

- 候选提交：`e2a8b22826dfd1bcacac9d0f28ed461b9bec9705`；注释标签 `v1.23.0-rc.2`，tag object `184a307e75b7d8f80156253cc28be87a00a22e48`，指向同一提交。
- 发布前主线：`468389a599b9007969ae6b81016f447cba5e1293`（rc.1 验收提交）。上一稳定版与回滚点：`v1.22.0`，镜像 index `sha256:46b854bdf6233e81742986a3bee9e65c6b4a39123c393a51a4ada63783eed7ac`。
- 发布候选分支：`release/v1.23.0-candidate`，本 RC 保留至 1.23.0 稳定版。发布工作树：`C:/GitHub/_codex-tasks/kpanel-v123-preview-release`；验收工作树：`C:/GitHub/_codex-tasks/kpanel-v123-preview-acceptance`；证据：`C:/GitHub/_release-evidence/v1.23.0-rc.2-20260927`。

本版只发布预览产物；生产未部署，公共稳定默认入口继续为 1.22.0。

## 发布画像与范围

- 业务域：桌面与经典模式壁纸、3D 场景交接、浅色主题导航。
- 变更面：前端组件和 CSS；无 API、持久化格式、Agent 权限、数据库迁移、端口或安装契约变化。风险等级 L3，因多个视觉层与异步场景状态组合，且前一版曾有壁纸视觉逃逸。
- 用户旅程：静态壁纸与场景互换、连续快速选择时只显示最后目标；场景海报等待与切换；经典模式“通透”；上传自定义图后在另一浏览器、经典模式和登录页显示；浅色/深色侧栏配色。

| 来源分支与原 tip | 纳入候选的提交 | 内容 |
| --- | --- | --- |
| `fix/wallpaper-scene-transitions-20260927` `63d155adf2b7449cc767e79f9e62d26e5c24d2c7` | `acbd3da9` → `37dd8233`；`63d155ad` → `3d30874a` | 桌面与经典模式统一壁纸交接、元数据刷新时保持交接阶段 |
| `feature/light-sidebar-20260927` `c60558f07aa3dcffe4ab86a1d007c6f6ab99f46f` | `c60558f0` → `dd1ed6aa` | 浅色侧栏状态色和设置预览 |

版本准备 `b6c9dfb5`，评审记录与误判更正 `32715345`、`ac80e479`、`e2a8b228`。Vue 模板中的 `&quot;` 会在属性解析时解码；最初假设其使 CSS URL 失效并不成立，试验性改动已撤回。未纳入其他分支或产品改动。

## 外部审计与修复交付

- 安全覆盖检查 `decision=ok`，8 个未审计边界提交、无新增边界包。本 RC 未新跑 CF scoped/full 审计；不把覆盖检查称为审计完成。
- OCR 对 `468389a5..ac80e479` 的 17/17 个可审文件补充逐文件覆盖。首次自由评审的 CSS 实体判断经 Vue 编译器和浏览器核验为误判，保留原始记录并增加更正记录；最终 `free-form=0`、`valid=H0/M0/L0`、`constrained-only=unreported`，不计入有效盲测。来源壁纸分支已有先前 OCR 记录；没有新的有效安全 finding。
- 源码、候选分支、主线和标签绑定 `e2a8b228`；公开产物摘要见下文。生产部署不适用。

## 跨仓库联动判定

- `scriptLinkageState=not-required`（无需发布脚本）。内置 `kejilion.sh` 基线 `2b90b2d2ca56bc954c9328a51bb5571e896f713d`，SHA-256 `806b4715664fad502f7faeccbc75972f2d1a24559d46208221b98f774ef56c99`，本次未改。跨仓库变更集、脚本候选与阻断依赖均不适用。
- 内置 `packaging/kejilion-app/kpanel.conf` 与 `kejilion/apps` 的 `origin/main:kpanel.conf` Git blob 均为 `38f7d9c37a2b55cec6b2292ff46dd22aab120b2c`；无需应用市场提交，默认入口保持 `latest`。

## 多维质量结论

| 维度 | 状态 | 证据与边界 |
| --- | --- | --- |
| 业务正确性与双端互通 | 已验证 | 候选镜像的真实 Panel API、独立浏览器会话、上传与服务端外观状态、场景应用通过；Agent/远端 Node 未实机复测 |
| 网络入侵与供应链安全 | 已验证 | L3 Trivy 源码/镜像、govulncheck、npm audit、CI 通过；本 RC 无新 CF 审计 |
| 稳定性、失败恢复与兼容 | 已实现未实机验证 | Go race 与应用配置生命周期通过；真实 systemd 更新/回滚未执行 |
| 性能与资源预算 | 已实现未实机验证 | 镜像运行时契约通过；低端 GPU、长时间场景资源趋势未测 |
| 用户体验与可访问性 | 已验证 | Chrome 1280×800 的壁纸与场景组合、浅/深色侧栏和截图通过，页面异常 0；其他浏览器与缩放矩阵未测 |
| 数据、配置与迁移 | 已验证 | 另一浏览器读取外观选择；无数据迁移，上传图/场景包仍不随设置备份迁移 |

## 自动门禁

- Windows 本地 typecheck、定向 Vitest 5 文件/74 测试和生产构建通过；更正后另跑 typecheck。OCR 委托入口测试、依赖策略、治理一致性通过。
- L3：`scripts/run-release-l3.mjs`，run ID `v1.23.0-rc.2-e2a8b228-l3-r3`，2026-09-27T07:49:40Z 至 07:58:16Z，`local-wsl-dr`，`status=passed`、`exit_code=0`。固定 Runner ID `sha256:a9f708891d1e81f17dd286bc7e9d6124658964716dc9a457b5c73340722f6a7d`，离线归档 SHA-256 `f533e0db78d328ee5504e511a22e9769e6be55f53cbd96a3a6ae47f04a969509`；完整 Go/race、前端 196 文件、Trivy、govulncheck、npm audit、镜像构建与应用配置备份生命周期通过。
- L3 bundle SHA-256 `1666446de7d19f1324426c365bfa2d62f51e2029db1e99d50468459906c9d0de`；plan `0acd1d435a5a2a0de1afefb081ee0dbca2300eee1ddd576a07db93ffafedb75d`；执行脚本 `21c08b11be3526a0fe766aa606a81d0e4047e8a4e89d79227da4e62914164979`。原始证据位于 `l3-r3/wsl-evidence`。
- 候选 CI [#1105](https://github.com/kejilion/KPanel/actions/runs/36304780617) success；依赖新鲜度 36304780662 success。主线 CI [#1106](https://github.com/kejilion/KPanel/actions/runs/36305207434) success；依赖新鲜度 36305207490 success。
- Release [#260](https://github.com/kejilion/KPanel/actions/runs/36305623659) success，标签依赖新鲜度 36305623548 success；精确源码 SHA 均为 `e2a8b228`。本版未升级 Go/npm 依赖、基础镜像、Action、扫描器或受管脚本；锁文件仅更新产品根版本。

## 隔离浏览器与公开镜像验收

- `arena-154` SSH 连接超时；按先前用户选择，仅由登记的 `local-wsl-dr` 承担候选 L3。本地 WSL/Chrome 浏览器是补充证据，不冒充登记隔离真机的 browser-validation。
- 候选镜像在独立数据目录、Docker 网络与非 root、只读容器中运行。上传 2560×1440 图片，预览成功；服务端外观状态持久化，第二浏览器登录后读取自定义壁纸及经典模式“通透”，退出登录后显示私有图片副本，页面异常 0。
- 场景包 `orbital-station` 的 34 个文件逐项按 catalog SHA-256 校验后注入隔离数据目录；场景在经典模式显示，切成静态图再切回只有一个 iframe，登录页显示场景 WebP 封面，页面异常 0。浅色侧栏表面与文字色等于内容 palette，深色模式切换后表面改变；证据见 `candidate-browser/`。
- WSL 在无常驻进程时关闭 Docker 服务，使首轮场景浏览器试验中断；保持 WSL 会话后，同一候选镜像复测通过。此为验证通道异常，未发现产品场景组件缺失。场景在线下载、真实 Agent、arm64 真机、其他浏览器及全缩放矩阵未测。候选隔离容器和网络已清理。
- 公开镜像 `kjlion/kejilion-panel:1.23.0-rc.2` 已从 Docker Hub 显式拉取，RepoDigest `sha256:1d7b6bfc19ba198e9f77af22ce633b134bb9a39b0f914946e445b69fcfe1d8bb`；`packaging/tests/image-e2e.sh` 输出 `image_e2e=pass`。同一公开镜像的 Chrome 1280×800 再次通过上传、跨浏览器自定义图、经典模式“通透”、场景静态图往返、登录页场景封面与浅/深色侧栏组合，页面异常 0；证据见 `public-preview-browser/`。隔离测试容器和网络已清理。

## 发布产物与公开仓库复核

- [GitHub Release v1.23.0-rc.2](https://github.com/kejilion/KPanel/releases/tag/v1.23.0-rc.2) 于 2026-09-27T08:27:59Z 公开，`draft=false`、`prerelease=true`；GitHub Latest 仍为 `v1.22.0`。
- Docker `1.23.0-rc.2` 与 `preview` 的 OCI index 均为 `sha256:1d7b6bfc19ba198e9f77af22ce633b134bb9a39b0f914946e445b69fcfe1d8bb`；`linux/amd64` 为 `sha256:7f279101f152191c68b6582a815284b0ec68a84f35c140be88596c232807c78e`，`linux/arm64` 为 `sha256:1f2f18914c856bcec346298ff921f1bd547a304e13b66f44148798db25069e63`。Docker `latest` 与 `1.22.0` 仍共用 `sha256:46b854bdf6233e81742986a3bee9e65c6b4a39123c393a51a4ada63783eed7ac`。
- Release 共 14 个附件，含 Agent 双架构、部署归档与 `SHA256SUMS`；校验文件自身 digest 与 GitHub asset digest 一致，其 11 条均与对应附件 digest 相符。证据见 `github-release-api.json`、`SHA256SUMS`、`docker-tags.json`。
- 自更新来源、自动安装开关、退出预览不自动降级等策略本版未修改，沿用 1.22.0 及 rc.1 回归；真实 systemd 更新、OpenRC 和轻量 Node 边界未本轮实机重复验收。

## 生产适用性与回滚

- 不适用（预览版禁止生产部署）。未连接、备份、部署、升级或核对 `prod-108`；`arena-154` 未建立 SSH 连接。没有生产写操作，也没有可报告的生产后版本或健康状态。
- 源码回滚点 `v1.22.0`；稳定镜像 index `sha256:46b854bdf6233e81742986a3bee9e65c6b4a39123c393a51a4ada63783eed7ac`。预览用户回退需独立选择旧版不可变 digest、备份并验证恢复；退出预览不会自动降级。GitHub Latest、Docker `latest` 和应用市场标准入口均保持 1.22.0。

## 分支与资源处置

- `release/v1.23.0-candidate` 保留为本发布序列唯一候选，远端精确 tip `e2a8b228`。
- 本轮两条 clean 来源分支的原 tip 与重放后的净差异 patch-id 一致：壁纸 `b5e927db`，浅色侧栏 `9a7b8519`。远端和本地 `archive/fix/wallpaper-scene-transitions-20260927` 均为原 tip `63d155ad`，`archive/feature/light-sidebar-20260927` 均为原 tip `c60558f0`；远端原活跃引用原先不存在，本地原分支已重命名，两个工作树保持 clean 并保留作恢复入口，没有旧 upstream。归档不代表生产上线。
- 含未提交或归属不清的其他旧工作树不在本轮清理范围；先前 `feature/desktop-3d-scene-packs-20260924` 的归档同名异 SHA 冲突仍待用户确定恢复 ref，不能覆盖既有归档。
- 验收记录分支在同 SHA 的候选与主线 CI 成功后归档。仓库外 L3、浏览器、Release 与公开镜像证据保留。

## 交付节奏数据

<!-- kpanel-release-metrics:start -->
- 首个纳入提交时间：2026-09-27T13:00:42+08:00
- 候选冻结时间：2026-09-27T15:48:22+08:00
- 生产完成时间：不适用
- 提交到生产用时：不适用
- 是否回滚、紧急热修复或重复发布：否
- 若发生失败，发现时间、恢复时间和逃逸门禁：不适用
<!-- kpanel-release-metrics:end -->

<!-- kpanel-release-process-metrics:start -->
- 已记录发布流程异常或无效证据拦截次数：5
- 其中生产写操作开始后异常次数：0
<!-- kpanel-release-process-metrics:end -->

<!-- kpanel-release-process-incidents:start -->
[
  {
    "fingerprint": "candidate-l3/base-tag/invalid-input",
    "position": "before-production-write",
    "count": 1,
    "impact": "首次 L3 入口预检拒绝 rc.1 作为 base-tag，候选代码未执行。",
    "recoveryEvidence": "改用上一稳定标签 v1.22.0；最终 run ID v1.23.0-rc.2-e2a8b228-l3-r3 status=passed。",
    "permanentAction": "后续同发布序列 RC 的 L3 固定使用上一稳定版标签作为 base-tag，在启动前校验标签格式。",
    "historicalReleases": []
  },
  {
    "fingerprint": "candidate-review/vue-entity/false-positive",
    "position": "before-production-write",
    "count": 1,
    "impact": "自由评审把模板属性中的 HTML 实体误判为 CSS 字面量，造成无必要改动并使首轮 L3 证据失效。",
    "recoveryEvidence": "Vue 编译器解码结果和真实浏览器 CSS 核验；撤回该改动，保留原始与更正 OCR 记录，对 e2a8b228 重新完成 L3。",
    "permanentAction": "此类模板属性结论先核对编译输出或浏览器 computed style，再记为有效 finding。",
    "historicalReleases": []
  },
  {
    "fingerprint": "browser-validation/wsl-idle/docker-stop",
    "position": "before-production-write",
    "count": 1,
    "impact": "首轮场景用例期间 WSL 的 Docker 服务随空闲会话停止，场景异步资源中断，浏览器证据无效。",
    "recoveryEvidence": "journalctl 记录 Docker 受 SIGTERM 正常关闭；保持 WSL 会话常驻后同一候选镜像的场景、登录和侧栏旅程通过。",
    "permanentAction": "本地补充浏览器验证期间保持 WSL 会话常驻，并在用例前后核对容器健康与清理。",
    "historicalReleases": []
  },
  {
    "fingerprint": "browser-validation/arena-154/ssh-timeout",
    "position": "before-production-write",
    "count": 1,
    "impact": "登记隔离环境 SSH 超时，无法获取该环境的公开镜像 browser-validation 证据。",
    "recoveryEvidence": "仅将候选 L3 切到登记的 local-wsl-dr；浏览器用本地 WSL/Chrome 补充，不冒充 arena-154。",
    "permanentAction": "1.23.0 稳定版前复核 arena-154 SSH 与 browser-validation；退出条件是在登记隔离环境完成同一公开镜像浏览器 E2E。",
    "historicalReleases": []
  },
  {
    "fingerprint": "tag-identity/powershell/rev-parse",
    "position": "before-production-write",
    "count": 1,
    "impact": "推送标签后的首个 PowerShell rev-parse peel 命令把 ^{} 误解析，标签验证命令退出非零；标签本身已成功推送。",
    "recoveryEvidence": "改用 git cat-file -p 和 git rev-list -n 1 核对 tag object 184a307e 与候选 e2a8b228 一致，远端 tag ref 同为 184a307e。",
    "permanentAction": "Windows PowerShell 中使用 git cat-file/git rev-list 进行注释标签指向核验，避免未加引号的 peel 语法。",
    "historicalReleases": []
  }
]
<!-- kpanel-release-process-incidents:end -->

## 遗留风险与后续准入

- 1.23.0 稳定版前补齐登记隔离环境的公开镜像浏览器验收和场景在线下载链路；当前本地浏览器 E2E 不扩大 `local-wsl-dr` 的登记用途。
- 本版不部署生产。归档 refs 仅为精确恢复位置，不能当成上线证明。
