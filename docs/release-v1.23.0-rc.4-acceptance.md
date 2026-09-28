# KPanel v1.23.0-rc.4 发布验收记录

日期：2026-09-28。发布级别：L3。`releaseChannel=preview`；`releaseTrain=1.23.0`。

- 候选、远端 `main` 与标签均指向 `595a3910b91b8cced0bb1bb36ef670c005536e17`；注释标签 `v1.23.0-rc.4` 的 tag object 为 `96bb23fce2a9c8ee3fa7df517e874cba349dffa7`。
- 发布前 `main=0c8fcfa1184384459d017e3f4ffb53bc95cd1ccc`；上一稳定版、源码回滚点为 `v1.22.0`，稳定镜像 index 为 `sha256:46b854bdf6233e81742986a3bee9e65c6b4a39123c393a51a4ada63783eed7ac`。
- 证据目录：`C:/GitHub/_release-evidence/v1.23.0-rc.4-20260928`。本版只发布预览产物，生产未部署。

## 发布画像与范围

- 业务域：经典模式壁纸和登录页壁纸、原生发行附件说明、MCP 客户端版本查询。
- 受影响旅程：上传并选择自定义图片、开启“通透”后在经典模式看到图片；浏览器减少动态和减少透明效果下继续显示；退出登录后登录页仍使用该图片；取得发行附件和查询 `kpanel-mcp version`。
- 最小变更：静态经典壁纸恢复 `v1.22.0` 的 CSS 背景图路径；3D 场景继续使用现有 `DesktopWallpaper`。显式选择壁纸时不再让 `prefers-reduced-transparency` 隐藏背景，`forced-colors` 仍使用纯色。发行归档更名为 `kejilion-panel-meta-1.23.0-rc.4.tar.gz`，澄清 Panel 通过 Docker 镜像分发；MCP 增加版本子命令。
- 没有新增 API、数据迁移、端口、Agent 权限、`kejilion.sh` 或应用市场安装契约。风险按上一 RC 已出现可见问题取 L3。

| 来源分支与原 tip | 候选提交 | 纳入证据 |
| --- | --- | --- |
| `fix/classic-wallpaper-reduced-transparency-20260928` `e56018e9` | `e56018e9` | 相同提交 |
| `feature/issue-19-native-distribution-20260928` `96cc1a28`（含 `c1f0298e`） | `b36b4d8c`、`17a32b88` | 两个 patch-id 分别一致 |
| `fix/issue-21-mcp-version-20260928` `65dabd61` | `9f8553db` | patch-id 一致 |

发布准备为 `542b133b`；业务上下文刷新为 `595a3910`。`feature/desktop-dynamic-scenes` 的工作树含其他任务未提交内容，未纳入；`review/issue-20-mcp-acceptance` 停在旧 RC，无新增代码。历史壁纸源分支已经在前版演进纳入，本轮不重复合并。

## 故障定位、修正与证据边界

- 对公开 `v1.22.0` 和 `v1.23.0-rc.3` 的相同浏览器媒体模拟发现：仅开启 `prefers-reduced-motion: reduce` 不会隐藏壁纸；两版的 `prefers-reduced-transparency: reduce` 规则都会隐藏 `.classic-backdrop`。因此不能把该规则说成 RC3 新引入。RC2 起静态经典壁纸从 CSS 背景图改走 `DesktopWallpaper`；本版恢复静态路径，并让用户明确选择的“通透”在减少透明效果下仍显示图片。
- 修复候选和公开 RC4 镜像分别验证普通、减少动态、减少动态加减少透明效果、强制颜色四种状态。前三种均为 `classicLevel=clear`、壁纸层 `display=block`、自定义图片解码宽度 2560、相关图片响应全为 200；强制颜色仍隐藏壁纸层。退出登录后，登录页的私有壁纸副本存在且品牌区背景使用该图片。公开镜像截图在证据目录 `browser-public/`。
- 用户原始 Golden Gate 图片、实际浏览器媒体设置、线上反向代理与缓存未直接取得；隔离容器结果不能证明现场已升级或现场根因唯一。

## 外部审计与跨仓库联动

- 安全覆盖检查为 `decision=ok`，相对最近 full 审计有 11 个未审计提交但无新增信任边界包；本 RC 未另行运行 CF scoped/full 审计。L3 的 Trivy 源码及镜像扫描、govulncheck、npm audit 均通过，不能把扫描等同于 CF 审计。
- OCR 覆盖 `0c8fcfa..a7fd940` 的 13/13 个可审文件，另有 4 个按格式/锁文件规则排除；自由臂无有效发现，`H0/M0/L0`，`constrained-only=0`。后续两个提交为版本说明修订和业务上下文文档，未增加可审代码。
- `scriptLinkageState=not-required`：受管 `kejilion.sh` 基线 `2b90b2d2ca56bc954c9328a51bb5571e896f713d`，SHA-256 `806b4715664fad502f7faeccbc75972f2d1a24559d46208221b98f774ef56c99`，本版不发布脚本。KPanel 内置与 `kejilion/apps` 远端 `main` 的 `kpanel.conf` Git blob 均为 `fa4b95374ed3b186d920207537f451a5b6d839f7`；无需应用市场提交，默认安装仍指向稳定 `latest`。

## 多维质量结论

| 维度 | 状态 | 证据与限制 |
| --- | --- | --- |
| 业务正确性与双端互通 | 已验证 | 公开镜像完成自定义图上传、读取、经典模式与退出登录后品牌背景；真实远端 Agent/Node 未重测 |
| 网络入侵与供应链安全 | 已验证 | L3、候选/主线/Release CI 扫描通过；本版未新增 CF scoped/full 审计 |
| 稳定性、失败恢复与兼容 | 已实现未实机验证 | Go race、应用配置生命周期通过；真实 systemd/OpenRC 更新恢复未执行 |
| 性能与资源预算 | 已实现未实机验证 | 双架构镜像与运行约束检查通过；低端 GPU 的 3D 长测未重复 |
| 用户体验与可访问性 | 已验证 | Chrome 1920×920 的四种媒体模拟及登录页截图；原用户浏览器、其他缩放与浏览器未验证 |
| 数据、配置与迁移 | 已验证 | 隔离容器服务端图片读取和浏览器登录副本通过；本版无格式迁移 |

## 自动门禁与公开产物

- 修复分支 `verify-change` 通过（196 个前端测试文件、1732 项测试及类型检查/构建）；最终 L3 通过 196 个文件、1733 项前端测试、Go 全套与 race、场景包检查、扫描、镜像构建及应用配置备份生命周期。首次 L3 预检只因业务上下文距基线达到 51 个提交而停止，刷新 `docs/product-quality-review-current.md` 后重新生成候选 `595a3910` 的完整 L3 证据并通过。
- L3 外层入口 `scripts/run-release-l3.mjs`，目标 `local-wsl-dr`，run ID `v1.23.0-rc.4-595a3910-l3-r2`，2026-09-28T04:36:45Z 至 04:44:28Z，`status=passed`、`exit_code=0`。不可变 Runner ID `sha256:a9f708891d1e81f17dd286bc7e9d6124658964716dc9a457b5c73340722f6a7d`；离线 Runner 归档 `f533e0db78d328ee5504e511a22e9769e6be55f53cbd96a3a6ae47f04a969509`，bundle `499de806b9df16736ec2581e1498a56fdbee64b67ad09db5dad43d9379787c2c`，plan `5771a5a87527ec3e61ce127329605850b1b433a9b2625e557a34767c592558d3`，执行脚本 `21c08b11be3526a0fe766aa606a81d0e4047e8a4e89d79227da4e62914164979`。
- 候选 [CI #1114](https://github.com/kejilion/KPanel/actions/runs/36379335277)、主线 [CI #1115](https://github.com/kejilion/KPanel/actions/runs/36379851622)、[Release #262](https://github.com/kejilion/KPanel/actions/runs/36380483228) 均为 success；各自相应的 Dependency freshness 检查成功。
- [GitHub Release v1.23.0-rc.4](https://github.com/kejilion/KPanel/releases/tag/v1.23.0-rc.4) 为 `draft=false`、`prerelease=true`，2026-09-28T05:18:47Z 公开；GitHub Latest 仍是 `v1.22.0`。14 个附件包含 Agent 双架构、Node、MCP 各平台、元数据归档和 `SHA256SUMS`；校验文件 SHA-256 `a5aec9a6e18126fa28337ed0339c774510f91498d4c79a15b8dcf767d04246e4`，其 11 条均与附件 digest 一致。
- Docker `1.23.0-rc.4` 与 `preview` 的 OCI index 均为 `sha256:c240284c68b00d780ac0ce9c4217ecf626992f4c2e3a26ef2460bc78b6275d8e`；`linux/amd64` 为 `sha256:e8ec2c42513b668cbba8a7669269141336b8f5c4655d7443af93138849f0bbb8`，`linux/arm64` 为 `sha256:e0955cff57f45125c8ec1c5653a692cdc1358f5e2563e33c1ef75f03db37be71`。Docker `latest` 仍为上方 `v1.22.0` index。
- 显式 `docker pull kjlion/kejilion-panel:1.23.0-rc.4` 得到相同 index，`packaging/tests/image-e2e.sh` 输出 `image_e2e=pass`。同一公开镜像的 Chrome 完成前述壁纸与登录页验证；测试容器、网络和合成数据目录已清理。
- `arena-154` 在本轮前续发布任务仍不可达；用户已选择本地通道，故 L3 使用 `local-wsl-dr`。WSL/Chrome 是本机隔离验证，不冒充登记真机或用户现场。没有执行生产部署。

## 自更新、生产与回滚

- 本版未修改更新通道；稳定来源、预览来源、加入/退出预览和自动安装沿用既有契约，本轮未实机重测 systemd/OpenRC 更新。应用市场默认入口、GitHub Latest 与 Docker `latest` 均保持稳定版。
- 生产部署安全核对：不适用（预览版禁止生产部署）。`prod-108` 禁用全部 KPanel 操作，本次未连接、未备份、未部署、未升级、未核对；没有生产前后版本或健康数据可报告。
- 源码回滚点 `v1.22.0`；镜像回滚点为稳定 index `sha256:46b854bdf6233e81742986a3bee9e65c6b4a39123c393a51a4ada63783eed7ac`。预览用户如需回退，应选用旧版不可变 digest 并备份、复核数据；离开预览不会自动降级。

## 分支与资源处置

- `release/v1.23.0-candidate` 作为 1.23.0 序列唯一远端候选保留，精确 tip `595a3910`，待稳定版按规范归档。
- 本轮三个来源分支的原 tip 已分别保存为远端 `archive/fix/classic-wallpaper-reduced-transparency-20260928=e56018e9`、`archive/feature/issue-19-native-distribution-20260928=96cc1a28`、`archive/fix/issue-21-mcp-version-20260928=65dabd61`；远端复核三者精确一致，原活跃远端引用均不存在。对应本地分支已改为 `archive/`、无旧 upstream，三个 clean 工作树暂保留供原任务核对；归档不代表生产上线。
- 动态 3D 场景实验分支有未知未提交内容，原样保留，不计入本次候选；旧验收/历史分支不凭名称批量删除。本轮临时浏览器容器、网络和合成数据已清理，仓库外证据保留。验收记录分支将在同 SHA 候选与主线 CI 均通过后独立归档。

## 交付节奏数据

<!-- kpanel-release-metrics:start -->
- 首个纳入提交时间：2026-09-28T12:29:25+08:00
- 候选冻结时间：2026-09-28T12:36:45+08:00
- 生产完成时间：不适用
- 提交到生产用时：不适用
- 是否回滚、紧急热修复或重复发布：是（rc.3 的壁纸可见性问题促成 rc.4 重复发布预览）
- 若发生失败，发现时间、恢复时间和逃逸门禁：发现时间: 2026-09-28T12:29:25+08:00; 恢复时间: 2026-09-28T13:18:47+08:00; 逃逸门禁: 已逃逸: rc.3 浏览器验收未覆盖减少透明度与静态自定义壁纸的组合
<!-- kpanel-release-metrics:end -->

发现时间采用本轮最早保留的修正提交时间，用户首次遇到问题的精确时间未记录；恢复时间为公开 Release 时间，用户现场完成升级的时间未验证。

<!-- kpanel-release-process-metrics:start -->
- 已记录发布流程异常或无效证据拦截次数：1
- 其中生产写操作开始后异常次数：0
<!-- kpanel-release-process-metrics:end -->

<!-- kpanel-release-process-incidents:start -->
[
  {
    "fingerprint": "l3-preflight/business-context/stale-baseline",
    "position": "before-production-write",
    "count": 1,
    "impact": "首次 L3 在预检阶段因业务上下文超过 50 个提交而停止，未运行测试；旧 run 不能作为发布证据。",
    "recoveryEvidence": "提交 595a3910 刷新稳定版业务基线；以此 SHA 重新生成的 L3 r2 于 2026-09-28T04:44:28Z 全部通过。",
    "permanentAction": "发布负责人在候选冻结前先执行业务上下文 freshness 预检，并在需要时更新到最近稳定版基线；保留原预检失败记录，不复用旧 L3。",
    "historicalReleases": []
  }
]
<!-- kpanel-release-process-incidents:end -->

## 遗留风险

- 用户原始图片与实际站点的浏览器、反向代理、缓存环境未纳入本地试验。若升级 RC4 后仍不透出，需从该页面获取实际媒体查询、计算样式、图片请求与响应，再定位环境差异。
- 3D 实验分支未纳入；现有 3D 场景路径保留，但低端 GPU 的播放/恢复长测和在线场景库下载未在本轮重复。预览版未部署生产。
