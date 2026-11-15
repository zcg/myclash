# 参考索引：MyGo、mihomo、平台与设计

> 用法：先查**本地模块文档**（与 `go.mod` 的 v0.9.0 完全一致），再用官网/插件目录导航。
> 外部链接会随版本变化，采纳前先核对；引用时记下版本与日期。

## 1. 开工前必读顺序

1. [GLOSSARY.md](../GLOSSARY.md) —— 术语
2. [01-prd.md](./01-prd.md) —— 做什么
3. [03-architecture.md](./03-architecture.md) —— 怎么放
4. [09-appearance.md](./09-appearance.md) + `.agents/skills/mygo-acrylic/` —— 外观
5. [02-ui.md](./02-ui.md) + `.agents/skills/mygo-components/` —— 控件对账
6. [06-parity.md](./06-parity.md)、[07-security.md](./07-security.md)、[08-quality.md](./08-quality.md)
7. 本文 —— 需要查 API/插件/平台细节时

## 2. MyGo 官方

| 资源 | 地址 | 用途 |
| --- | --- | --- |
| 文档首页 | https://mygo.run/zh/docs | 入口、快速开始、CLI |
| 原生 UI 概览 | https://mygo.run/zh/docs/ui | 控件总览与导航 |
| 样式与主题 | https://mygo.run/zh/docs/ui/styling | token、材质、主题 |
| 布局 | https://mygo.run/zh/docs/ui/layout | Row/Column/Grid/Split/Scroll |
| 页面与导航 | https://mygo.run/zh/docs/ui/navigation | Router / Sidebar |
| 浮层 | https://mygo.run/zh/docs/ui/overlays | Dialog / Popover / Menu |
| 表格 | https://mygo.run/zh/docs/ui/table | 列宽、排序、选择、持久化 |
| 代码编辑器 | https://mygo.run/zh/docs/ui/code-editor | YAML/JS 编辑、`LanguageFor` |
| 差异查看器 | https://mygo.run/zh/docs/ui/diff-viewer | 覆写预览 |
| 图表 | https://mygo.run/zh/docs/ui/chart | 速率曲线、流量柱状图 |
| WebView | https://mygo.run/zh/docs/ui/web-view | Web 面板 |
| 自定义控件 | https://mygo.run/zh/docs/ui/custom-widgets | base + part 组合 |
| 动效 | https://mygo.run/zh/docs/ui/motion | `ui.Animation`、`AnimateTo` |
| 绘制 | https://mygo.run/zh/docs/ui/drawing | `Painter`、路径、着色器 |
| 辅助功能 | https://mygo.run/zh/docs/ui/accessibility | 焦点、标签、对比度 |
| 原生 UI 性能 | https://mygo.run/zh/docs/ui/performance | 帧预算、虚拟化、分配 |
| 测试 | https://mygo.run/zh/docs/ui/testing | `ui.NewTester` |
| 窗口 | https://mygo.run/zh/docs/ui/windows | `WindowOptions`、Vibrancy、面板 |
| 应用生命周期 | https://mygo.run/zh/docs/app | 单实例、深链、登录启动、路径 |
| 菜单与托盘 | https://mygo.run/zh/docs/menus | `NewTray`、`OnClick`/`OnRightClick`、`SetTitle` |
| 桌面 API | https://mygo.run/zh/docs/native | 对话框、通知、剪贴板、Secrets、快捷键 |
| 多语言 | https://mygo.run/zh/docs/i18n | ICU 消息、复数、类型化函数 |
| 分发 | https://mygo.run/zh/docs/distribution | 三端打包、签名、安装器 |
| 自动更新 | https://mygo.run/zh/docs/updates | 签名更新、增量、频道 |
| GitHub Actions | https://mygo.run/zh/docs/github-actions | 三端构建/签名/公证 |
| 配置与 CLI | https://mygo.run/zh/docs/configuration · https://mygo.run/zh/docs/cli | `mygo.json`、`mygo build` |
| Go API | https://pkg.go.dev/github.com/egoist/mygo | 精确签名 |
| 示例 | https://github.com/egoist/mygo/tree/main/examples | vibrancy、liquid-glass、menubar、table 等 |
| 基准 | https://mygo.run/zh/benchmarks | 性能基线 |

**版本提醒**：官网跟最新版；本项目锁 MyGo v0.9.0。冲突时以
`go list -m -f '{{.Dir}}' github.com/egoist/mygo` 下的 `docs/` 为准。

## 2.1 材质与玻璃（必读）

- 官方 glass 插件：https://github.com/egoist/mygo/blob/main/docs/plugins/glass.md
- 官方材质/Vibrancy：https://mygo.run/zh/docs/ui/windows（Vibrancy 一节）
- 亚克力纪律与组件：`.agents/skills/mygo-acrylic/SKILL.md` · `references/tokens.md` · `references/components.md`
- 结论：**系统材质与窗口内玻璃是两个层次**。前者透桌面/其他窗口，只有 macOS/Win11 有；
  后者只模糊窗口内已绘制内容，三端一致。不要把两者互称，也不要在 Linux/Win10 承诺桌面磨砂。

## 2.2 Awesome MyGo 目录

| 页面 | 地址 | 用途 |
| --- | --- | --- |
| 总览 | https://awesomemygo.dev/ | 应用、模板、组件、插件索引 |
| UI 组件 | https://awesomemygo.dev/ui | 组件库、主题、图标 |
| 插件 | https://awesomemygo.dev/plugins | 官方插件与第三方插件 |
| 模板 | https://awesomemygo.dev/templates | 起步模板 |
| 状态管理 | https://awesomemygo.dev/state | 状态库对比 |
| 文档/教程 | https://awesomemygo.dev/learn | 文章与指南 |
| 作品 | https://awesomemygo.dev/showcase | 参考应用 |
| 源码文档 | https://github.com/egoist/mygo/tree/main/docs | 与版本绑定的 Markdown（v0.9.0 本地副本优先） |

## 3. 官方插件

目录：https://awesomemygo.dev/plugins · 文档：https://mygo.run/zh/docs/plugins

| 插件 | 本项目用途 | 结论 |
| --- | --- | --- |
| Updater | 应用更新、验签 | **用**（ADR-0005） |
| Fetch | 订阅/Geo 下载 | 评估；标准库 `net/http` 足够时不用 |
| WebSocket | 不需要（mihomo 用 chunked HTTP 流） | 不用 |
| SQLite | 流量历史可选 | M3 评估；当前 JSONL |
| Watch | 监听 Profile/覆写文件变化 | 可选（外部编辑后重载） |
| Terminal | 不需要 | 不用 |
| Glass | 窗口内玻璃（工具条/浮层/液态玻璃预设） | **用**（ADR-0011） |
| SF Symbols | macOS 图标 | 可选；主图标用 Lucide |
| Automation | 不需要 | 不用 |
| PermissionFlow | macOS 权限引导 | 可选（通知/辅助功能） |
| Loading | 骨架/加载态 | 可选 |
| Game | 不需要 | 不用 |

## 4. UI 库与图标

完整评估与准入清单见 [10-ui-libraries.md](./10-ui-libraries.md)。

| 资源 | 地址 | 结论 |
| --- | --- | --- |
| Onevra | https://github.com/oneo-me/onevra · https://oneo-me.github.io/onevra/ | 参考（Shell/命令面板/Lucide 组织）；锁 MyGo v0.8.3 |
| Lobe MyGo | https://github.com/lobehub/lobe-mygo · https://mygo.lobehub.com | 参考 token 架构 |
| uix | https://github.com/smallnest/uix | 不采用（依赖网深） |
| CrychicUI | https://github.com/ELY-Open-Source/CrychicUI | 参考覆盖清单 |
| Mayu | https://github.com/harukizmoe/mayu | Material 3，暂不采用 |
| 8bitMyGoUI | https://github.com/ELY-Open-Source/8bitMyGoUI | 不做 |
| MujicaUI | https://github.com/ELY-Open-Source/MujicaUI | 不做 |
| mygo-native-ui | https://github.com/xingkaixin/mygo-native-ui | 未评估，参考 |
| mygo-platinum | https://github.com/firede/mygo-platinum | 未评估，参考 |
| Sunbeam | https://github.com/AstraFoundry/design | 未评估，参考 |
| NuxtUI Go | https://github.com/josexy/nuxt-ui-go | 未评估，参考 |
| Kubai Icons | https://github.com/yldm-tech/mygo-kubai-icons · https://kubai.yldm.ai | 图标候选；默认 Lucide |
| AI Logo | https://github.com/yldm-tech/ai-logo | 品牌/占位 logo，参考 |

**本项目采用的本地层**：`.agents/skills/mygo-acrylic/assets/acrylic/`（Ground/Card/洗色、控件、guard 与压力测试）。
落地时复制进 `internal/ui/acrylic`，保留来源说明；许可与 Gloss 署名在发布前确认。

图标：
[Lucide](https://lucide.dev)（ISC，默认）·
[Tabler](https://tabler.io/icons)（MIT）·
[Heroicons](https://heroicons.com)（MIT）·
[Material Symbols](https://fonts.google.com/icons)（Apache-2.0）·
[SF Symbols](https://developer.apple.com/sf-symbols/)（Apple 许可，仅 macOS）。

## 5. mihomo

| 资源 | 地址 | 用途 |
| --- | --- | --- |
| 仓库 | https://github.com/MetaCubeX/mihomo | 源码、issue、版本 |
| 文档 | https://wiki.metacubex.one/ | 配置项、入站/出站、DNS、TUN |
| 配置参考 | https://wiki.metacubex.one/config/ | 字段语义与默认值 |
| API | https://wiki.metacubex.one/api/ | REST 路由与 payload |
| 发布 | https://github.com/MetaCubeX/mihomo/releases | 版本与资产（查阅用；应用不下载外部内核） |
| Geo 数据 | https://github.com/MetaCubeX/meta-rules-dat | GeoIP/GeoSite/ASN |
| Web 面板 | https://github.com/MetaCubeX/metacubexd · https://github.com/haishanh/yacd | 面板资源 |
| 规则集 | https://github.com/MetaCubeX/meta-rules-dat/tree/geo | 规则提供者来源示例 |

本地核对：`hub/route/*.go`（路由）、`config/config.go`（字段）、`main.go`（启动顺序）。

## 6. 参考客户端

| 资源 | 地址 | 用途 |
| --- | --- | --- |
| Clash Verge Rev | https://github.com/clash-verge-rev/clash-verge-rev | 页面/配置模型/服务模式对照 |
| Clash Party | https://github.com/mihomo-party-org/clash-party | 网络页拆分/覆写/统计对照 |
| 功能矩阵 | [06-parity.md](./06-parity.md) | 已整理，不重复翻源码 |

## 7. Go 依赖与库

| 资源 | 地址 | 用途 |
| --- | --- | --- |
| goja | https://github.com/dop251/goja | 脚本覆写沙箱（M2） |
| yaml.v3 | https://pkg.go.dev/gopkg.in/yaml.v3 | 配置解析与行号 |
| x/sys | https://pkg.go.dev/golang.org/x/sys | Windows 注册表/服务、Unix 权限 |
| purego | https://github.com/ebitengine/purego | MyGo 原生绑定（间接） |
| bbolt | https://github.com/etcd-io/bbolt | mihomo `cache.db`（内部） |
| Go pprof | https://pkg.go.dev/runtime/pprof | 内存/CPU 测量 |

## 8. 平台细节

### Windows

- WinINet 代理设置与刷新：`InternetSetOptionW`（`INTERNET_OPTION_SETTINGS_CHANGED`、`REFRESH`）。
- 注册表：`HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings`。
- wintun：https://www.wintun.net/ （随包分发，注意架构）。
- Windows 11 材质：Mica/Acrylic 由 MyGo 的 `Vibrancy` 映射，不直接写 DWM。
- 安装器：NSIS（MyGo 生成）；签名工具 `signtool`。

### macOS

- `networksetup`（web/secure proxy、PAC、bypass）与 `scutil --proxy`。
- `SMAppService`（macOS 13+）或 `launchctl`（12）安装提权助手。
- utun 由 mihomo 创建；DNS 用 `networksetup -setdnsservers`。
- 签名/公证：`codesign`、`notarytool`、`stapler`。

### Linux

- GNOME：`gsettings` 的 `org.gnome.system.proxy`（mode/http/https/autoconfig-url/ignore-hosts）。
- KDE：`kwriteconfig6`/`5` 写 `kioslaverc`。
- DNS：`resolvectl dns`；无 systemd-resolved 时 `/etc/resolv.conf`。
- TUN/路由：`ip`、`nftables`；提权 `pkexec` + polkit。
- 包：`deb`/`rpm`/`PKGBUILD`；依赖 GTK 3、WebKitGTK 4.1（面板）。

## 9. 设计参考

| 资源 | 地址 | 用途 |
| --- | --- | --- |
| Apple HIG | https://developer.apple.com/design/human-interface-guidelines/ | macOS 控件/材质/间距 |
| Microsoft Fluent | https://fluent2.microsoft.design/ | Windows 11 材质与密度 |
| GNOME HIG | https://developer.gnome.org/hig/ | Linux 桌面惯例 |
| Material 3 | https://m3.material.io/ | 仅作 token 对照 |
| Lucide | https://lucide.dev | 默认图标与线宽 |
| 亚克力 skill | `.agents/skills/mygo-acrylic/SKILL.md` | 三层/洗色/guard/压力截图 |
| 官方 glass 插件 | https://github.com/egoist/mygo/blob/main/docs/plugins/glass.md | 窗口内玻璃、ScrollEdge、Blur |

## 10. 核对规则

- 引用外部链接时，在 PR/文档中写明**访问日期**与**版本**（如 MyGo v0.9.0、mihomo v1.19.34）。
- 组件库采纳前跑 `10-ui-libraries.md` §5 的准入清单；不允许把 MyGo 降级。
- 官方插件先用 `go list -m` 确认版本；插件 API 与 MyGo 主版本绑定。
- 平台命令先在目标系统验证，再写进文档；不把「某博客说可以」当事实。
