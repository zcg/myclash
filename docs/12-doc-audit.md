# 文档审计与动工前检查

> 日期：2026-11-14。范围：仓库内全部设计与决策文档。目的：在写第一行业务代码前，
> 确认没有自相矛盾、没有漏项、没有把「做不到」写成「已支持」。

## 1. 文档清单

| 文档 | 状态 | 说明 |
| --- | --- | --- |
| `README.md` | 定稿 | 入口、平台、设计要点、文档索引、构建命令 |
| `GLOSSARY.md` | 定稿 | 配置/内核/网络/界面四组术语与 `_Avoid_` |
| `docs/00-research.md` | 定稿 | 参考产品、mihomo API、MyGo 能力、风险 R1–R8 |
| `docs/01-prd.md` | 定稿 | 149 条功能、平台矩阵、NFR、里程碑出口 |
| `docs/02-ui.md` | 定稿 | 信息架构、页面规格、控件对账、托盘/命令面板/悬浮窗 |
| `docs/03-architecture.md` | 定稿 | 进程模型、Host、配置管线、三端集成、测速、安全/测试 |
| `docs/04-roadmap.md` | 定稿 | M1–M3 工作流、依赖、风险闸门、DoD |
| `docs/05-config-pipeline.md` | 定稿 | 覆写语法、注入、脚本契约、回滚、黄金测试 |
| `docs/06-parity.md` | 定稿 | 与 Verge/Party 的功能矩阵；`?` 为实现前复核项 |
| `docs/07-security.md` | 定稿 | 威胁 T1–T14、安全默认值、验收 |
| `docs/08-quality.md` | 定稿 | 内存/性能/体验预算、评审清单、反模式 |
| `docs/09-appearance.md` | 定稿 | 系统材质 + 窗口内玻璃 + 亚克力纪律 |
| `docs/10-ui-libraries.md` | 定稿 | 组件库/图标评估、本项目本地层、准入清单 |
| `docs/11-references.md` | 定稿 | 官方文档、插件、mihomo、平台、设计参考 |
| `docs/12-doc-audit.md` | 定稿 | 文档审计、一致性检查、动工前清单（本文） |
| `docs/13-code-style.md` | 定稿 | 包结构、依赖方向、命名、代码/UI/测试规范 |
| `docs/adr/0001–0012` | 定稿 | 0009 为 superseded；0012 为当前内核/模式决策 |

## 2. 本轮锁定、不再摇摆的决策

1. **三端同等**：Windows 10 1809+、macOS 12+、Linux（GTK 3、systemd/polkit）。
2. **内核只 bundled**：随应用更新；不下载外部 mihomo；重启内核可用（ADR-0012）。
3. **两种运行模式都实现、运行时互斥**：
   - standard：单进程，GUI 内嵌内核，内存最低（默认）。
   - lightweight：Agent 持有内核 + 托盘；GUI 关窗即退，再次打开附着。
   - 切换重启内核；TUN 先停；不允许两个内核/两个托盘。
4. **配置模型**：Subscription → Profile → Runtime Config；原子切换 + Last Known Good。
5. **外观三层**：系统材质（透桌面） / 窗口内玻璃（只透应用内容） / 亚克力纪律（Ground/Card/洗色）。
   四预设：混合（默认）/ 原生 / 液态玻璃 / 实色。
6. **UI 依赖**：不引入第三方组件库；官方 `plugins/glass` + 本地 `mygo-acrylic` 组件层 + Lucide 图标。
7. **安全默认**：Controller 仅 loopback；无遥测；密钥进系统凭据库；提权子进程只走 stdin。
8. **明确不做**：内置 Sub-Store 后端、MCP、多内核并存、独立内核更新、Tailscale 引导登录、移动端。

## 3. 自动检查

| 检查 | 结果 |
| --- | --- |
| 本地 Markdown 链接 | 0 断链 |
| 表格列数一致性 | 0 不匹配 |
| 残留术语（Glass/Solid/Compact、core_source、F-CORE-09、F-ST-11、update-core） | 0 |
| 外部内核/External Host 残留 | 仅出现在明确标注「已推迟/不做」的说明（00/03/11、ADR-0009/0012） |
| 里程碑引用 | PRD/UI/Roadmap 对 lightweight=M3、speedtest=M2、command palette=M2 一致 |
| 三端平台矩阵 | PRD §6.0 与 09/02 一致；Win10/Linux 无系统材质已标注 |
| 亚克力资产 | MyGo v0.9.0 临时模块编译 + 自带测试通过（2026-11-14） |

## 4. 人工复核

- **功能覆盖**：PRD 20 个域、149 条；对照 `06-parity.md`，除明列例外外均有里程碑。
- **页面覆盖**：02 的 6 主页面 + 9 分组页面 + 首次启动/命令面板/托盘/悬浮窗；每页有加载/空/错误态。
- **进程/状态**：standard/lightweight/TUN 的所有权、状态单写者、托盘归属、更新顺序均已写明。
- **安全**：提权握手、Agent socket、面板 secret、订阅限制、脚本沙箱、测速端点均有威胁与验收。
- **质量**：内存/性能预算、亚克力 guard/洋红压力、玻璃采样边界、三端验收矩阵。
- **外观**：原生材质/窗口内玻璃/亚克力三者的能力边界与「不伪造桌面磨砂」已写死。
- **参考**：官方文档、插件目录、mihomo、平台、设计参考均给出地址与用途；本地模块 docs 优先。

## 5. 动工前需要准备（不是文档矛盾）

| 项 | 现状 | 建议 |
| --- | --- | --- |
| 应用标识 | `mygo.json` 仍是 `com.example.myclash` | 改成你拥有的反向域名；没有域名可用 `io.github.<用户名>.myclash` |
| 显示名/图标 | 占位 `icon.png` | 提供 logo/wordmark 与托盘亮/暗图标；没有则先用占位 |
| 版权署名 | README/NOTICE 已写 `myclash contributors` | 发布前可替换为作者/组织名；补全第三方全文 |
| 更新签名 | 未生成 | `mygo keygen`；私钥进 CI secret，公钥进配置 |
| 亚克力组件 | 资产在 skill 内，未入库 | 已用 v0.9.0 编译并通过自带测试；仍需复制到 `internal/ui/acrylic` |
| 亚克力许可 | 来自 Gloss/mygo-acrylic，未见独立 LICENSE | 发布前确认许可与 Gloss 署名；在 NOTICE/关于页列出 |
| mihomo 版本 | 未写入 `go.mod` | 固定一个版本；三端编译 + REST 冒烟 + TUN 手工清单 |
| `wintun.dll` | 未入库 | 选稳定版本，x64/arm64 各一份，随安装包分发 |
| 运行时 | WebView2/WebKitGTK 未验证 | 面板页做运行时探测与安装指引 |
| 签名/公证 | 未接 | 发布前准备 Windows 代码签名与 macOS 公证账号 |
| 参考产品 `?` 行 | 06 中仍有 `?` | 实现对应功能前，用当时 Verge/Party 源码复核并更新 |

## 6. 建议的首次实现顺序（M1 纵向切片）

1. 仓库、CI、`mygo.json`、图标、更新密钥、三端构建。
2. 外壳：`acrylic.Window/Theme/Ground/Card` + 侧栏/路由 + 三预设 + 明暗/强调色。
3. Embedded Host + REST 客户端 + traffic/connections/logs 流 + 心跳。
4. Profile 库、订阅添加/同步、合并覆写、Runtime 校验、原子切换。
5. 概览（真实速率）+ 代理页（选择/测速）+ 系统代理（三端手动）。
6. 配置/连接/规则/日志页 + 设置（常规/外观/内核）。
7. guard/洋红测试、三端安装包、M1 演示脚本。

## 7. 文档变更规则

- 影响范围/验收：改 `01-prd.md`。
- 影响进程、数据、接口：改 `03-architecture.md`，必要时新增 ADR。
- 影响外观/控件：改 `02-ui.md`、`09-appearance.md`，跑 guard/压力。
- 新参考链接：先核对版本与可达性，再写入 `11-references.md`。
- 术语冲突：先改 `GLOSSARY.md`，再改文档与代码。

## 8. 结论

文档之间没有已知的自相矛盾；功能、页面、进程、外观、安全、质量、里程碑均有对应条目。
剩余事项是**开工前的配置与资产确认**（§5），不是设计缺口。可以按 §6 的顺序开始实现。
