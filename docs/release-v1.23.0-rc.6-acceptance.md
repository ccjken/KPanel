# KPanel v1.23.0-rc.6 发布验收记录

日期：2026-09-30。发布级别：L3。`releaseChannel=preview`；`releaseTrain=1.23.0`。

- 候选、远端 `main` 与标签均指向 `967167a106476eb65d6ec6df0300272abc94f888`；注释标签对象为 `6300d7c6e28c78fb337965532f27f10fb3bebba2`。
- 发布前 `main=ffc9358b226b0af48f96ad6dbc7219c53aa75335`，上一稳定版及源码回滚点为 `v1.22.0`。候选分支 `release/v1.23.0-candidate` 继续供本发布序列使用。
- L3 证据：`C:/GitHub/_release-evidence/v1.23.0-rc.6-967167a1-l3-r1`。本版产物已公开，生产未部署。

## 发布画像与范围

- 业务域：远程及定时备份、集群资料与分享、应用脚本终端、AI 助手。
- 变更面：Panel 持久数据与 API、备份传输、集群展示和通知、Web 交互、镜像内受管 `kejilion.sh`。不增加任意宿主命令入口、对外监听端口或 Agent 权限。
- 核心旅程：配置 S3 兼容存储或 WebDAV 与每日/每周/每月备份及保留份数；查看月流量、剩余价值、分享主题与到期提醒；缩放和重连应用脚本终端；复制 AI 对话并处理流式工具输出。
- 风险：远程备份新增 `internal/backupremote` 安全边界包，云端 S3/NAS 实际互通尚未验收；真实 Codex CLI 会话、长期跨月统计、外部通知、弱网重试和 200% 浏览器缩放仍需现场验证。预览版不进入生产。
- 用户可见更新及升级注意事项见同标签的 `CHANGELOG.md`。旧壁纸/3D 实验分支和带未知改动的其他工作树未纳入或清理。

| 来源分支 | 精确 tip | 纳入与处置 |
| --- | --- | --- |
| `feature/cluster-share-themes` | `98a56956f8eddaa42e161fbe54bfed3a27b35f32` | 合并；归档 |
| `feature/cluster-monthly-traffic` | `1b072a0f59bcdd51de240eaf00e241647eeed765` | 上行祖先；归档 |
| `feature/cluster-share-metrics` | `f35a9f2cda95e18efbe302671d1489884b8951d5` | 上行祖先；归档 |
| `feature/backup-remote-schedule` | `6546f21feac3ec3f2ee837506d539766229ab93a` | 合并；归档 |
| `feature/cluster-globe-wheel-zoom` | `626bbbe469b9008f1d5b148f46b63b4dd6d229f6` | 合并；归档 |
| `fix/ai-chat-copy` | `003da8fc6abe0d277eec721f5fa551468c7d7ce1` | 合并；归档 |
| `fix/ai-stream-state` | `6ce1f1650025cb59f2c5fcec7a159ca03f8cfcb1` | 合并；归档 |
| `fix/ai-tool-output` | `ec4e5c0616cab38c3137bcec480a32934902e657` | 合并；归档 |
| `fix/app-terminal-resize` | `fd686dfff2c79225140ad713b342b99ad2b2d754` | 合并；归档 |
| `feature/cluster-remaining-value` | `0d51ff5b81ecba6a077c5a8f1c70e85b5d037174` | 由 `70a2cfd8` 与后续可用配置修正替代；未合并旧实现，归档 |

前九个 tip 均为发布标签祖先。全部十个原 tip 已按精确 SHA 原子推送到远端 `refs/heads/archive/<原分支名>` 并重读确认；相应干净本地工作树已改名为 `archive/<原分支名>`，未删除工作树或其他会话数据。第十项的剩余价值功能在组合分支重新实现并增加缺省配置保护，不能将旧 tip 称为标签祖先。远端不存在这十项对应的活跃来源分支；预览候选继续保留。

## 外部审计与跨仓库联动

- 安全覆盖：`check-security-audit-coverage.mjs --target HEAD` 输出 `decision=scoped-required`，新增边界包 `internal/backupremote`；最近 full 为 `run-4`，37 个提交、73 个文件尚未由 CF 账本覆盖。预览版记录此项，稳定版前必须按 scoped 审计补齐；L3 扫描不冒充 CF 审计。
- OCR：各来源分支保留各自的评审记录；本次七个顶层分支合并没有文本冲突，发布准备提交按规则记录 `OCR-Review: skipped`，未宣称重新完成 133 文件的行级全审。本周期没有充分证据得出 constrained-only 工具效果结论。
- `scriptLinkageState=coupled`，变更集 `ai-cli-20260929`。配套 `kejilion/sh` 已先发布到 `779192048077c130442a64a126d7c0050776d868`，根脚本 SHA-256 为 `33010d547355f9bde4067c189a0457f44dc00ec3a72eaa25e5b7101e1fcb9c04`。Dockerfile、外部来源清单、OCI 标签与公开镜像内脚本文件均一致；新版脚本含 Claude Code/Codex 应用入口。未发布新的 `kpanel.conf`，仓库及 `kejilion/apps` 主线文件 blob 均为 `fa4b95374ed3b186d920207537f451a5b6d839f7`，默认安装仍走稳定通道。

## 多维质量结论

| 维度 | 状态 | 证据与边界 |
| --- | --- | --- |
| 业务正确性与双端互通 | 已实现未实机验证 | Go 全套、1911 项前端测试、组合模拟界面；真实 S3/WebDAV 服务及 Codex 会话未测 |
| 网络入侵与供应链安全 | 已实现未实机验证 | L3 的 govulncheck、npm audit、Trivy 与镜像脚本摘要通过；CF scoped 审计仍待稳定版前完成 |
| 稳定性、失败恢复与兼容 | 已实现未实机验证 | 特权包 race、应用配置备份回滚生命周期及备份单测通过；真实远程故障和长期计划未测 |
| 性能与资源预算 | 已实现未实机验证 | 镜像运行约束通过；低资源持续运行和大备份远传压力未测 |
| 用户体验与可访问性 | 已实现未实机验证 | 组合模拟预览可见集群分享主题、月流量、剩余价值和备份计划/远程存储；真实 200% 缩放及跨浏览器未测 |
| 数据、配置与迁移 | 已实现未实机验证 | 现有 `.kpb` 与新计划/远程目标自动测试通过；真实旧数据升级和云端取回未测 |

## 自动门禁与隔离验收

- 合并后定向类型检查与 75 项受影响前端测试通过。模拟 `acceptance` 预览绑定提交 `967167a1`，在本地浏览器检查集群分享、月流量和备份设置入口；模拟接口的外观设置读取失败未作为真实后端结论，预览进程已停止。
- `arena-154` SSH 超时；按已获授权的 `local-wsl-dr` 在 Ubuntu WSL、root Docker 运行 L3。run ID `v1.23.0-rc.6-967167a1-l3-r1`，2026-09-30 00:59:30 至 01:08:29（北京时间），`status=passed`、`exit_code=0`。Runner ID `sha256:a9f708891d1e81f17dd286bc7e9d6124658964716dc9a457b5c73340722f6a7d`；bundle `11d270099545e1d96baafc6a597552aa9381405ab6ffac674bbac69cbafce73e`，plan `c2424ad532684b5b3f25b574e49454090a6660d87f3177080f08fc3d151cb734`，执行脚本 `21c08b11be3526a0fe766aa606a81d0e4047e8a4e89d79227da4e62914164979`。
- L3 包含 Go 全套、特权包 race、211 个前端测试文件共 1911 项、场景包复现、类型检查、i18n、构建、安装安全、受管脚本契约、应用配置生命周期、备份恢复、govulncheck、npm audit 和 Trivy 源码/最终镜像扫描，均通过。首次 L3 被外部来源清单旧脚本摘要正确拦截；提交 `967167a1` 修正并重新执行完整 L3，不能把首次失败写成通过。
- 候选 [CI](https://github.com/kejilion/KPanel/actions/runs/36602913122)、主线 [CI](https://github.com/kejilion/KPanel/actions/runs/36604105013)、[Release](https://github.com/kejilion/KPanel/actions/runs/36605322876) 均为 `success`，各自的 Dependency freshness 检查也成功。
- 本地 Windows 依赖报告执行后检测源不完整（`go` 缺失及部分网络 fetch 失败），保留在证据目录 `public/dependency-report.md`，不作为完整依赖新鲜度结论；云端上述检查是本版通过依据。EOL 最后记录为 2026-07-28；本轮未单独验证每日通告。依赖、工具链与基座仍以仓库锁文件、固定 Dockerfile 和 L3/Release 构建结果为准，未来升级另开候选和回滚验证。

## 发布产物与公开仓库复核

- [GitHub Release v1.23.0-rc.6](https://github.com/kejilion/KPanel/releases/tag/v1.23.0-rc.6) 于 2026-09-30 01:43:08（北京时间）公开，`draft=false`、`prerelease=true`、非 Latest；GitHub Latest 仍为 `v1.22.0`。
- 14 个附件包括 Agent/Node 双架构、MCP 各平台、元数据归档、许可与 `SHA256SUMS`。`SHA256SUMS` 自身 SHA-256 为 `20ffb014224c13317d08bf7fe78390e612736dcb2aedb9d742b55ea807a91746`，其 11 条与 GitHub Release API 对应资产摘要逐一相同。
- Docker `1.23.0-rc.6` 与 `preview` 的 OCI index 均为 `sha256:aa9bfb47d3d1ef4ef299bfd7b9531c26feffccd5cffe792e4b7745a6d9d54b7f`；amd64 为 `sha256:1dbb2a83bc1ced5fc78dcca09f90dc760cdb914c69d1df9c5c2a3bd45f1b2e56`，arm64 为 `sha256:07e369ca130080c7e5226e8b4b6458535e7bd7dd57d94c8de9d886c8befd2c52`，其余 unknown/unknown 为 attestation。Docker `latest` 仍为稳定版 index `sha256:46b854bdf6233e81742986a3bee9e65c6b4a39123c393a51a4ada63783eed7ac`。
- 从 Docker Hub 公开版本镜像复制到隔离 Docker 后，OCI revision、版本、受管脚本 revision 和镜像内实际脚本摘要均匹配；`packaging/tests/image-e2e.sh` 输出 `image_e2e=pass`，覆盖启动、健康、公开静态资源、首次设置和网络/权限约束。此项不等于真实云存储或 Codex 会话验收。

## 自更新、生产与回滚

- 自更新通道契约未修改：稳定来源指向正式 Latest，预览来源选择规范 RC 与唯一镜像摘要；加入预览不自动安装，退出预览不自动降级。systemd/OpenRC 重启、失败隔离和数据备份沿用已有自动测试，本轮未在真实宿主重测。
- 生产部署安全核对：**不适用（预览版禁止生产部署）**。本轮未连接、备份、部署、升级或核对 `prod-108`；`arena-154` 不可达，隔离 WSL 测试不能代替生产部署证据。生产前后版本、健康和公网入口均无可报告数据。
- 源码回滚点为 `v1.22.0`；稳定镜像回滚点为上方 `latest` index。上一预览版 `v1.23.0-rc.5` 可作为不可变预览镜像回退目标，但操作前必须备份并核对数据兼容。切换稳定通道只影响后续更新来源，不自动降级当前实例。公共默认通道无需恢复。

## 交付节奏数据

<!-- kpanel-release-metrics:start -->
- 首个纳入提交时间：2026-09-29T21:26:13+08:00
- 候选冻结时间：2026-09-30T00:59:30+08:00
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
  {"fingerprint":"image-e2e/wsl-user/docker-permission","position":"before-production-write","count":1,"impact":"首次公开镜像拉取以 WSL 默认用户执行，在读取 Docker Socket 前被拒绝；未产生拉取或 E2E 结果。","recoveryEvidence":"改用已登记 local-wsl-dr 的 root Docker，随后公开镜像 image_e2e=pass。","permanentAction":"发布命令规格固定 WSL -u root 并先预检 Docker Socket 权限。","historicalReleases":[]},
  {"fingerprint":"image-e2e/docker-pull/layer-stall","position":"before-production-write","count":1,"impact":"Docker CLI 拉取一个公开镜像层持续重试并停滞，未将该次拉取视为通过。","recoveryEvidence":"skopeo 从同一 Docker Hub 版本标签复制所有层到本地 Docker，核对公开 OCI 摘要、镜像标签与脚本实际摘要，随后 image_e2e=pass。","permanentAction":"公开镜像 E2E 入口加入受摘要约束的 skopeo 下载回退及有界超时，发布负责人 2026-10-06 复核。","historicalReleases":[]},
  {"fingerprint":"script-hash/powershell/shell-quoting","position":"before-production-write","count":1,"impact":"首次通过嵌套 shell 抽取镜像脚本时参数转义失败，没有得到有效摘要。","recoveryEvidence":"改用 PowerShell 保存 Docker create 返回的容器 ID，docker cp 精确文件后计算 SHA-256 与 OCI 标签一致，并删除临时容器。","permanentAction":"Windows 侧镜像文件核验固定为独立参数的 create/cp/hash 步骤，避免嵌套 shell 插值。","historicalReleases":[]},
  {"fingerprint":"dependency-report/windows-runtime/incomplete-sources","position":"before-production-write","count":1,"impact":"Windows 本地依赖报告缺少 go，部分远端检测源 fetch 失败；该报告不作为完整通过证据。","recoveryEvidence":"原始报告保留于 public/dependency-report.md；候选、主线、标签对应云端 Dependency freshness 均成功。","permanentAction":"下次发布从具备 Go 和完整网络源的固定 Runner 生成报告，负责人 2026-10-06 复核全部检测源为 ok。","historicalReleases":[]}
]
<!-- kpanel-release-process-incidents:end -->

## 遗留风险与后续准入

- 稳定版前须对 `internal/backupremote` 做 CF scoped 安全审计，并完成真实 S3/WebDAV/NAS 上传、重试、取回、计划清理和旧数据迁移验收；真实 Codex CLI、弱网与低配资源也未通过本轮模拟证明。
- 本地原任务工作树与 L3 证据保留供回溯；归档只更名分支，不宣称已释放工作树磁盘空间。发布任务产生的短期中间文件可在验收记录并入主线后按目录核对清理。
