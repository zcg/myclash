# 调研：myclash 参考产品、内核与框架事实

> 本文只记录事实与来源，不做决策。决策见 `docs/adr/`，术语见 `GLOSSARY.md`。
> 状态：初稿（2026-11-14）。标注「待核对」的条目尚未逐行读源码确认。

## 1. 目标

用 Go + MyGo 原生 UI 做一款 mihomo 桌面客户端：界面与功能对齐 Clash Verge Rev 与 Clash Party，
单进程、低占用、功能不缺项。

## 2. 参考产品

### 2.1 Clash Verge Rev

- 技术栈：Tauri（Rust + React），内核为独立进程（mihomo 系）。
- 页面（`src/pages`，已核对）：`home`、`proxies`、`profiles`、`connections`、`rules`、`logs`、`settings`、`unlock`（解锁测试）。
- 布局：左侧固定侧边栏 + 内容区；设置页为多分组长表单。
- 内核模式：内置内核二进制，可切换版本；Windows/macOS 提供「服务模式」以获得 TUN 所需权限（待核对具体实现）。
- 配置模型：自有配置库（`profiles.yaml` 索引 + 每个配置一个 YAML），支持订阅、合并（merge）、脚本（script）三段式覆写。
- 系统代理：手动 / PAC，含绕过列表（待核对 PAC 编辑是否内置）。
- 其他：托盘、热键、开机自启、便携模式、深链、自动更新、i18n（含简中）。

### 2.2 Clash Party

- 技术栈：Electron（React + TypeScript），GPL-3.0，默认分支 `smart_core`。
- 页面（`src/renderer/src/pages`，已核对）：`proxies`、`profiles`、`connections`、`rules`、`logs`、
  `settings`、`traffic`、`dns`、`tun`、`sysproxy`、`sniffer`、`override`、`resources`、`substore`、`network`、`mihomo`（Web 面板）。
- 设置拆分：系统代理、TUN、DNS、嗅探各自成页，而非挤在一个设置页。
- 系统代理（据 `zh-CN.json`，已核对）：手动 / PAC / 绕过列表 / UWP 工具。
- TUN（已核对）：堆栈、网卡名、严格路由、自动设置系统 DNS、重设防火墙、手动授权内核。
- 嗅探（已核对）：HTTP/TLS/QUIC、端口、跳过/强制域名、按源/目标地址跳过。
- 其他已验证线索：Tailscale 节点认证、流量历史统计、悬浮窗（`FloatingApp`）、窗口置顶。
- 覆写页：结构化覆写（非纯文本），另有 `substore` 页面。

### 2.3 两者共同的「功能全集」（对照用）

| 域 | 条目 |
| --- | --- |
| 概览 | 内核状态/版本/运行时长、上下行速率与累计、连接数、内存、模式切换（规则/全局/直连） |
| 代理 | 代理组卡片、节点列表、延迟测速、自动选择组、排序/搜索、当前节点定位 |
| 配置 | 订阅（URL/名称/自动更新/流量与到期）、本地文件导入、编辑、合并、脚本、覆写、分享链接 |
| 连接 | 实时列表（域名/IP/进程/规则/链路/流量/时长）、搜索、排序、关闭单条/全部 |
| 规则 | 生效规则列表、搜索、分页、规则提供者更新 |
| 日志 | 实时流、级别、过滤、暂停、清空、导出 |
| DNS | 查询工具、缓存清理、DNS 设置 |
| 网络 | 系统代理（手动/PAC/绕过）、TUN、嗅探配置 |
| 资源 | GeoIP/GeoSite、规则/代理提供者、内核与 UI 更新 |
| 系统 | 托盘、悬浮窗、热键、开机启动、单实例、便携模式、主题、语言、备份/恢复、更新 |

## 3. mihomo 的事实

### 3.1 可嵌入（已实测）

`github.com/metacubex/mihomo` 不是 `main` 独占：`hub.Parse(configBytes, options...)` 可在自己的进程里
启动完整内核，并暴露 REST API：

```go
hub.Parse(cfg, hub.WithExternalController("127.0.0.1:9090"), hub.WithSecret(secret))
```

- 实测：Go 1.27、`CGO_ENABLED=0`、Windows/amd64 可编译；`-s -w` 后二进制 53.6 MB（含内核，mihomo v1.19.34，2026-11-14）。
- 实测：`GET /version`、`GET /configs` 在嵌入模式下可访问。
- 许可：GPL-3.0。嵌入分发会要求本应用同为 GPL-3.0。
- 结论：**同进程可行**；默认内嵌以降低内存。独立外部内核已评估并推迟（ADR-0012），当前只做 bundled。

### 3.2 外部控制 REST API（读自 `hub/route/*.go`，已核对）

| 方法 | 路径 | 用途 |
| --- | --- | --- |
| GET | `/version` | 版本 |
| GET | `/traffic`、`/memory` | 速率 / 内存（流式） |
| GET | `/logs?level=` | 日志流（流式） |
| GET | `/connections` | 连接列表（流式，支持 `interval`） |
| DELETE | `/connections`、`/connections/{id}` | 关闭全部 / 单条 |
| GET | `/proxies`、`/proxies/{name}` | 代理与组 |
| PUT | `/proxies/{name}` | 选择节点 |
| DELETE | `/proxies/{name}` | 清除固定选择 |
| GET | `/proxies/{name}/delay` | 节点延迟测试 |
| GET | `/group`、`/group/{name}`、`/group/{name}/delay` | 组与组测速 |
| GET | `/providers/proxies`、`/providers/proxies/{name}` | 代理提供者 |
| PUT | `/providers/proxies/{name}` | 更新提供者 |
| GET | `/providers/proxies/{name}/healthcheck` | 提供者健康检查 |
| GET | `/providers/rules`、`PUT /providers/rules/{name}` | 规则提供者 |
| GET | `/rules`、`PATCH /rules/disable` | 规则与禁用 |
| GET | `/configs`、`PATCH /configs`、`PUT /configs` | 读/改/重载配置 |
| POST | `/configs/geo`、`/upgrade/geo` | 更新 Geo 数据 |
| POST | `/restart`、`/upgrade` | 重启内核 / 自更新 |
| GET | `/dns/query?name=&type=` | DNS 查询 |
| POST | `/cache/dns/flush`、`/cache/fakeip/flush` | 清缓存 |
| GET/PUT/DELETE | `/storage/{key}` | 内核侧键值存储 |
| GET | `/group` | 组列表（另一形状） |

要点：
- 流量、连接、日志、内存均为**流式**响应，UI 需增量读取（HTTP chunked 或 WebSocket）。
- `PUT /configs` 支持 `{"path": ...}` 与 `{"payload": ...}`；但 `route.SetEmbedMode(true)` 会关闭改配置/重启/自升级三类路由。
  本应用因此把配置写入放到进程内（见 ADR-0007），REST 只用于读与选择。
- 内核**不做**系统代理设置，也不做订阅下载/解析——这些是 GUI 的职责。

### 3.3 内核能力面（决定功能上限）

- 入站：mixed/http/socks/redir/tproxy/tun（wintun/system 栈）、listeners、DoH server。
- 出站：ss/ssr/vmess/vless/trojan/hysteria2/tuic/wireguard/tailscale/ssh/http/socks 等。
- 组：select/url-test/fallback/load-balance/relay；`profile.store-selected` 持久化选择。
- 规则：domain/geoip/ipcidr/process/rule-set/sub-rules/logic 等；规则提供者。
- DNS：fake-ip、nameserver-policy、fallback、respect-rules；嗅探。
- 其他：`external-ui` 托管 Web 面板、`storage` 键值、`/upgrade` 自更新。

### 3.4 内核不提供、必须由 GUI 实现的部分

- 系统代理（Windows 注册表 / WinINet 刷新、PAC、绕过列表、UWP 回环豁免）。
- TUN 的提权与 wintun.dll 分发、防火墙规则。
- 订阅下载、解析、`subscription-userinfo` 流量/到期、自动更新策略。
- 配置库、合并/脚本覆写、备份、导入导出。
- 托盘、悬浮窗、热键、开机启动、单实例、自动更新、i18n、主题。

## 4. MyGo 的事实（v0.9.0，Go 1.27）

- 原生 UI（Skia 渲染，无 WebView 主界面）；`ui` 包控件齐备。

### 4.1 可用控件（`docs/ui/*.md` 全量清单，已核对）

结构：`Row`/`Column`/`Grid`/`Split`/`Scroll`/`Sidebar`/`Tabs`/`Toolbar`/`Inspector`/`Outline`。
数据：`Table`、`List`、`GridView`、`Tree`、`Chart`、`Meter`、`Badge`、`Progress`、`Spinner`。
输入：`TextInput`、`SearchField`、`Select`、`Combobox`、`Autocomplete`、`Checkbox`、`Radio`、
`Switch`、`Slider`、`RangeSlider`、`Stepper`、`NumberInput`、`DecimalInput`、`DateInput`、`TimeInput`、
`ColorPicker`、`TokenField`、`CodeEditor`、`RichTextEditor`、`DiffViewer`。
覆盖层：`Dialog`、`AlertDialog`、`Popover`、`ContextMenu`、`MenuButton`、`Toast`、`Tooltip`、`FindBar`。
内容：`Text`、`Markdown`、`Image`、`Video`、`WebView`、`Link`、`Avatar`、`Icon`。
- `CodeEditor` 可直接用于 YAML 编辑；`Chart` 用于流量曲线；`WebView` 可托管 metacubexd/yacd。

### 4.2 桌面能力

- 托盘图标与菜单、全局快捷键、通知、文件对话框、剪贴板、Shell、Secrets、WatchFiles、电源、媒体会话。
- 单实例锁、二次启动回调、深链、文件关联、登录启动、退出钩子。
- 更新：`plugins/updater`；分发：NSIS 安装器（Windows）、`.dmg`（macOS）、`deb`/`rpm`/`PKGBUILD`/`tar.gz`（Linux）；安装钩子、可随包携带**辅助可执行文件**与原生库。
- 玻璃风格：`plugins/glass`，`glass.Glass/Group`、`glass.Blur`、`glass.ScrollEdge`，跨平台同一套着色器，
  尊重「减少透明度/提高对比度」辅助功能设置。

### 4.3 已测体积（仅记录，不作为当前目标）

| 项 | 值 |
| --- | --- |
| 空壳应用（dev） | 15.5 MB |
| 嵌入 mihomo 后（`-s -w`） | 53.6 MB |
| 全量构建耗时 | ~33 s |

## 5. 差距与风险清单

| # | 风险 | 影响 | 备注 |
| --- | --- | --- | --- |
| R1 | TUN 需要提权，三端机制不同 | 决定进程/服务架构 | Windows wintun + UAC；macOS utun + 授权；Linux tun + polkit |
| R2 | 系统代理需自实现 | WinINet 刷新、PAC 服务、绕过语法 | 三端各一套后端；Linux 仅覆盖 GNOME/KDE |
| R3 | 嵌入内核 = GPL-3.0 | 许可与分发 | 需明确接受 |
| R4 | 覆写脚本（JS）无现成 Go 实现 | 功能对齐度 | `goja` 纯 Go 可选 |
| R5 | Sub-Store 依赖 Node 后端 | 体积/复杂度 | 可先只支持外部 URL |
| R6 | 流量历史统计需自建存储 | 额外功能 | SQLite 插件或简单文件 |
| R7 | 内核流式接口需稳定消费 | 实时性/内存 | 单连接多路复用 |
| R8 | 自动更新签名与回滚 | 发布安全 | MyGo updater（`mygo keygen`）+ GitHub Releases |

## 6. 已核对与待核对

本次已核对（已写入架构/ADR）：

- `route.SetEmbedMode(true)` 关闭 `PUT/PATCH /configs`、`/restart`、`/upgrade`；本应用因此改用内部 API 写配置（ADR-0007）。
- 必须先 `constant.SetHomeDir` 再 `config.Init`/`hub.Parse`；`cache.db`、Geo、provider 相对路径都以 HomeDir 为准。
- `/rules` 的 `extra` 提供 `hitCount`/`hitAt`；规则提供者的条目内容不在 `/rules`，预览需读 provider 文件。
- MyGo `Table` 支持列宽拖拽、排序、横向滚动与列状态持久化；`CodeEditor` 有 `LanguageFor`；`WebView` 有 `Eval`/`LoadURL`；托盘支持 `SetTitle`/`SetIcon`。

仍需核对（实现到对应功能前）：

- Verge 服务模式的进程模型与提权细节（不影响本方案，仅作对照）。
- Party 的「覆写」数据模型（字段级 override vs merge 文档）与 Sub-Store 后端形态。
- 默认订阅 UA 在真实供应商上的接受度（需要回归样本）。
- 自定义入站监听/redir 在参考产品中的暴露程度。
