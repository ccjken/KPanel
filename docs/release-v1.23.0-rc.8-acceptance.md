# KPanel v1.23.0-rc.8 发布验收记录

日期：2026-09-30。发布级别：L3。`releaseChannel=preview`；`releaseTrain=1.23.0`。

- 产品候选和标签：`82eeaf76e3bf47355ed8f4d945a562b22865f713`；发布前远端 `main` 基线为 `784317476fbba4e2c8cfa988ee5c268922584b27`。候选 CI 通过且基线未移动后已快进主线，主线 CI 通过并三次核对一致后推送注释标签 `v1.23.0-rc.8`，tag object `eaaf16cef3adfc6c2118e8c1db317eda52134e97`。
- 上一稳定版及回滚点：`v1.22.0`，Docker `latest` index `sha256:46b854bdf6233e81742986a3bee9e65c6b4a39123c393a51a4ada63783eed7ac`。上一不可变预览为 `v1.23.0-rc.7`。
- 同序列候选 `release/v1.23.0-candidate` 继续保留；本轮仅公开预览产物，生产未部署。

## 发布画像与精确范围

- 业务域：经典模式背景连续性；管理员自定义站点名称与图标；登录页开源地址。
- 变更面：Panel 外观状态及匿名品牌白名单、Web 表单与导航/登录/浏览器品牌、内置静态壁纸缓存与绘制。无 Agent 动作、宿主机资源、Compose、脚本协议或安装配置变化。
- 核心旅程：名称和图标保存、刷新回读、恢复默认、冲突重试；主题保存不清除旧客户端省略的品牌；经典通透有缓存时从启动层连续交接主界面，服务端选择覆盖旧缓存且缓存不自动回写；桌面和登录页使用同一服务器外观选择。
- 图标仅允许 PNG/JPG/WebP、上传上限 2 MiB，浏览器转换为 128px PNG，服务端限制 64 KiB、128×128 并完整解码；名称最多 64 字符，禁止控制字符。Session/CSRF/Origin 和资源版本边界继续执行。
- 启动位图缓存只覆盖公开内置静态壁纸。自定义图与 3D 场景继续等待登录后的服务器状态；真实 Nginx、隔离真机浏览器及 PWA 未验收。

| 来源分支 | 精确 tip | 纳入与处置依据 |
| --- | --- | --- |
| `fix/classic-refresh-background-20260930` | `13e26899bbc17880c25fddb0e4f64561f4c4208b` | 三个提交合并；恢复静态背景、缓存和视口构图；已归档至 `archive/fix/classic-refresh-background-20260930`，远端回读 SHA 一致 |
| `feature/site-branding-20260930` | `7bb41f12bda659020a2f0f8bcf06face641ac985` | 十一个提交合并；站点品牌及图标上传交互；已归档至 `archive/feature/site-branding-20260930`，远端回读 SHA 一致 |

历史已在主线的文档和 MCP 验收分支不是新增候选；旧壁纸/3D 实验工作树及旧审计分支的独有提交、未知本地内容保持原样，复用 rc.7 已记录的处置依据，不作为本轮发布内容。当前活跃品牌实现为上述 7bb41f12，旧的另一份站点信息实现不重复组装。

## 外部审计与跨仓库联动

- 安全覆盖在精确候选上为 `decision=scoped-required`，40 个提交、75 个文件未由 CF 账本覆盖；新增边界包 `internal/backupremote`，最近 full 为 `run-4`。预览按规则记录；稳定版前补 scoped，L3 扫描不替代 CF 审计。
- 来源提交保留 OCR 和独立复核记录：经典连续性 10/10，最终视口 CSS 增量注明 skipped；品牌 29/29，最终上传与无障碍增量已有来源复核。品牌 constrained-only 因先前规则暴露为 unreported，不能当作盲审通过。发布任务另核对实际组合差异和八项浏览器旅程。
- `scriptLinkageState=not-required`（无需发布脚本（不适用））：本轮不改变脚本、安装契约或镜像内脚本内容。固定脚本 commit `779192048077c130442a64a126d7c0050776d868`，SHA-256 `33010d547355f9bde4067c189a0457f44dc00ec3a72eaa25e5b7101e1fcb9c04`，L3 受管契约检查通过。
- `Dockerfile` 和 `packaging/kejilion-app/kpanel.conf` 相对 rc.7 无变化；本仓库与 `kejilion/apps` 配置 blob 均为 `fa4b95374ed3b186d920207537f451a5b6d839f7`。无需应用市场提交，默认入口继续稳定版。

## 多维质量与证据边界

| 维度 | 状态 | 证据与边界 |
| --- | --- | --- |
| 业务正确性与双端互通 | 已实现未实机验证 | 品牌 CAS、旧客户端兼容、并发保存队列和缓存覆盖有自动测试；组合 mock-ui 通过。脚本双端资源契约未变；真实双设备未测。 |
| 网络入侵与供应链安全 | 已实现未实机验证 | 图标/名称验证、Session/CSRF/Origin 与匿名白名单有 Go 测试；govulncheck、npm audit、源码及镜像 Trivy 通过。CF scoped 待补。 |
| 稳定性、失败恢复与兼容 | 已实现未实机验证 | 保存失败重试、冲突、旧客户端和坏缓存回归通过；完整 race 与镜像生命周期通过。 |
| 性能与资源预算 | 已实现未实机验证 | warm 经典刷新壁纸请求 0、额外外观读取 0、回写 0，启动/主界面背景有效区域像素一致；真实低配启动性能未测。 |
| 用户体验与可访问性 | 已实现未实机验证 | 保存/上传/错误、窄宽屏、深色、中英文及繁体、200% CSS zoom 通过；来源键盘操作有证据。CSS zoom 不是浏览器原生缩放，原生 200% 与 PWA 待补。 |
| 数据、配置与迁移 | 已实现未实机验证 | 品牌复用现有外观持久化和备份；旧请求省略品牌时保留当前值；真实旧数据升级未单独执行。 |

## 自动门禁与本地验收

- `arena-154` SSH 超时，沿用用户已选择的 `local-wsl-dr`。固定 Runner 为 `kpanel-release-gate:go1.26.7-node24`，image ID `sha256:a9f708891d1e81f17dd286bc7e9d6124658964716dc9a457b5c73340722f6a7d`。
- L3 run `v1.23.0-rc.8-82eeaf76-l3-r1`，2026-09-30 16:00:23 至 16:09:06（北京时间），`status=passed`、`exit_code=0`。完整 Go、215 个前端文件/1940 项测试、typecheck、i18n、构建、race、漏洞扫描、镜像构建与安装/备份恢复生命周期通过；npm audit 为 0 漏洞。
- 自包含 bundle SHA-256 `18562c70c44f37c3323a7535b9a29b748d672674162821e71f43985960bc88cd`，plan `c80f4299588271af95857e9d57b9c7b75ae440cdf5201d1a4014a81a7abd6c4c`，执行脚本 `21c08b11be3526a0fe766aa606a81d0e4047e8a4e89d79227da4e62914164979`。原件在 `C:/GitHub/_validation/v1.23.0-rc.8-82eeaf76-l3-r1`，终态为 `wsl-evidence/status.txt`。
- 精确候选 acceptance/visual-composition 预览完成八项 mock-ui 检查：品牌保存、无效图保留、保存失败重试、warm 经典刷新像素衔接、服务端覆盖旧缓存、深色宽/窄屏与 CSS zoom、英文/繁体窄屏、公开登录品牌/开源链接/壁纸。页面错误 0；证据 `C:/GitHub/_validation/kpanel-v123-rc8-82eeaf76-preview/browser-result.json`，预览 mockApi/web 已精确停止。首轮错误选择器超时保留在 `browser-result-r1.json`。
- 候选 [CI 36688010079](https://github.com/kejilion/KPanel/actions/runs/36688010079) 与 [Dependency freshness 36688010309](https://github.com/kejilion/KPanel/actions/runs/36688010309)、主线 [CI 36689351127](https://github.com/kejilion/KPanel/actions/runs/36689351127) 与 Dependency freshness 36689351135 均在精确提交 `82eeaf76` 上成功。

## 标签与公开产物

- [Release](https://github.com/kejilion/KPanel/releases/tag/v1.23.0-rc.8) 于 2026-09-30 16:46:05（北京时间）公开，`prerelease=true`、`draft=false`；[Release workflow 36690459460](https://github.com/kejilion/KPanel/actions/runs/36690459460) 与标签 Dependency freshness 36690459700 成功。
- API 列出 14 个发布附件。下载的 `SHA256SUMS` 文件 SHA-256 为 `5bf0020098ce4ad3c7caa9ffd226864a23a287009411377491e23ff72b0c8ecf`，其中 11 条校验值逐一匹配 GitHub API 公布的附件 SHA-256 digest；未将此写作全部二进制下载校验。证据 `public/release-verification.json`。
- Docker `1.23.0-rc.8` 与 `preview` index 一致：`sha256:7ce0d347d5f1672a95c25f2caa969e7562ee94685ff4f5e2c51410699d3a5ef8`。amd64 descriptor `sha256:6c3039538c5320f44b15f2ab18fea02853f95de658dd02a58bd187e2df09d58a`，arm64 descriptor `sha256:5aff52638df33bc4e61cf6c7baa9dd91ea2021708ad8bb058ff364c2963744ad`。Docker `latest` 和 GitHub Latest 仍为首段记录的 `v1.22.0` 稳定版本。
- 从 Docker Hub 拉取 amd64 公开镜像，config `sha256:bcaf152ab2cff472945c3eeae9830e462ad7a7c1b67bcfb6efef6c5b8f95447f`；OCI revision 和版本与候选一致。WSL 临时容器执行标准镜像 E2E 及外部附加断言，`image_e2e=pass`、`public_appearance_api=pass`，退出码 0，临时容器、网络及数据目录由原脚本清理。
- 真实公开镜像 API 验证：名称与有界 PNG 保存，省略品牌的旧主题请求保留品牌，SVG 拒绝 422，旧 CAS 拒绝 409，容器重启后匿名登录引导回读名称、图标和选定壁纸且无私有 resourceVersion。公开 CSP 允许 data/blob 图片；HTML 和 `appearance-init.js` 为 no-cache，启动脚本字节与候选一致。镜像内脚本实际 SHA-256 与固定脚本摘要一致。
- 上述公开证据均在 `C:/GitHub/_validation/v1.23.0-rc.8-82eeaf76-l3-r1/public`，包括镜像日志、`appearance-api-result.json` 和三个原始 index。这是公开 amd64 镜像运行及 API 证据；arm64 已核对发布 descriptor，未执行 arm64 容器，也未将 API 结果当作真实 Nginx/PWA 浏览器验收。

## 自更新、生产与回滚

- 自更新通道契约未改，测试覆盖稳定来源排除 RC、预览选择、手动安装、通道切换和失败恢复；本轮未在真实宿主重测。
- 生产部署安全核对：不适用（预览版禁止生产部署）。`prod-108` 禁用全部 KPanel 操作；本次未连接、未部署、未核对。WSL 候选和公开镜像验收不是生产证据。
- 稳定源码与镜像回滚点见首段，上一不可变预览为 rc.7；实际回退前须备份和检查数据兼容，退出预览不会自动降级。

## 交付节奏数据

<!-- kpanel-release-metrics:start -->
- 首个纳入提交时间：2026-09-30T13:55:23+08:00
- 候选冻结时间：2026-09-30T16:00:23+08:00
- 生产完成时间：不适用
- 提交到生产用时：不适用
- 是否回滚、紧急热修复或重复发布：否
- 若发生失败，发现时间、恢复时间和逃逸门禁：不适用
<!-- kpanel-release-metrics:end -->

<!-- kpanel-release-process-metrics:start -->
- 已记录发布流程异常或无效证据拦截次数：1
- 其中生产写操作开始后异常次数：0
<!-- kpanel-release-process-metrics:end -->

<!-- kpanel-release-process-incidents:start -->
[
  {"fingerprint":"browser/appearance-fixture/button-label-mismatch","position":"before-production-write","count":1,"impact":"首轮组合浏览器脚本使用过时的保存按钮文案，选择器超时，未取得完整浏览器证据；产品候选没有改变。","recoveryEvidence":"核对候选实际 i18n 文案后修正外部脚本，精确提交 82eeaf76 的八项检查通过，原失败 JSON 保留。","permanentAction":"发布负责人于 2026-10-07 前核对同类外观浏览器测试入口，复用稳定定位器或在用例预检精确名称与唯一性。","historicalReleases":[]}
]
<!-- kpanel-release-process-incidents:end -->

## 遗留风险与分支收尾

- CF scoped 审计、真实 Nginx/隔离主机浏览器、原生浏览器 200% 缩放与 PWA 验收留待稳定版前；当前公开镜像验收不能替代这些场景。
- 两项来源 tip 均已确认是发布标签祖先，原任务空闲且来源工作树干净；公开镜像通过后，以 expected-absent leases 原子创建上述 archive refs，远端回读 SHA 一致，原远端活动分支不存在。原任务仍供用户体验的本地预览、对应本地分支与工作树继续保留，待预览结束后再清理；不会再次纳入新增候选。未知内容工作树保留，发布候选继续供同序列后续 RC 使用。
