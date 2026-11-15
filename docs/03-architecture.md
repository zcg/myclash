# 架构

> 决策背景见 `docs/adr/`；术语见 `GLOSSARY.md`。本文描述进程、模块、数据流与三端落地方式。

## 1. 约束

- 语言 Go 1.27，UI 用 MyGo v0.9.0 原生控件，内核用 mihomo（锁定版本，纯 Go、`CGO_ENABLED=0`）。
- 三端同等支持：Windows 10 1809+、macOS 12+、Linux（GTK 3、glibc 2.28+）。
- 界面进程永不以管理员/root 运行；TUN 由提权后的独立内核进程承载。
- 界面只通过 mihomo REST API 访问内核，不调用 mihomo 内部包（`hub.Parse` 仅用于启动内嵌内核）。

## 2. 进程模型

**标准模式（默认，单进程）**

```
┌──────────────────────────────────────┐
│ myclash（GUI，普通权限）              │
│  MyGo UI / state / config pipeline   │
│  Embedded Core（同进程 hub.Parse）    │
│  sysproxy / tray / updater           │
└──────────────┬───────────────────────┘
               │ REST 127.0.0.1:port + secret
        ┌──────┴──────┐
        │ TUN 时切换   │
        ▼             ▼
  Embedded Core   Privileged Host
  （同进程）       myclash --core --parent
```

**轻量模式（Agent 常驻）**

```
┌──────────────────┐   控制 socket + REST   ┌───────────────────────────┐
│ GUI 前端（按需）  │ ◄────────────────────► │ myclash --agent           │
│ 关闭即退出       │                        │ Core + tray + sysproxy/TUN │
└──────────────────┘                        └───────────────────────────┘
```

- 同一时刻只有一个 Core 所有者（GUI 或 Agent），也只有它写 `state.yaml` 与流量采样。
- 两种运行模式都实现，但**运行时互斥**：同一时刻只运行一种，不能同时存在两个内核/两个托盘。
- 标准模式切换 TUN：停 Embedded → 提权启动 → 等 ready；关闭 TUN 反向。切换期间 UI 显示「正在切换内核」。
- 轻量模式：GUI 关闭即退出；再次启动先探测 Agent，存在则附着为前端，不新建内核。
- 内核只做 bundled（ADR-0012）；没有外部内核与独立更新。

| 模式 | 窗口打开 | 窗口关闭 | 内存 |
| --- | --- | --- | --- |
| standard（默认） | GUI 内嵌内核，1 进程 | 托盘保留或退出 | 最低 |
| lightweight | GUI + Agent，2 进程 | 仅 Agent，1 进程 | 关窗后释放 UI 内存 |

### 2.1 提权内核的握手与控制

1. 界面生成一次性 `token`、`nonce`，构造 JSON 信封 `{parent_pid, parent_exe, nonce, token, config}`；配置与 token 经 **stdin** 传给子进程，不写临时文件、不进 `argv`（配置里可能有密钥）。
2. 提权启动 `myclash --core --parent <pid>`。
3. 子进程校验：父进程存在、可执行文件路径与 `parent_exe` 一致、token 长度与格式合法；随后初始化内核、绑定 `127.0.0.1`、以 token 为 controller secret，向 stdout 输出 `ready`。
4. 界面读 stdout（15s 超时）并轮询 `/version`；失败则终止子进程并恢复 Embedded。
5. 子进程监视父进程管道；EOF 或父进程消失即退出并清理网卡/DNS。
6. 控制命令（重载配置、切转发模式、更新 Geo、停止）沿用同一条 stdin/stdout JSON Lines 管道；不新开本地端口，不暴露到 REST。

### 2.2 Agent 控制通道

- 位置：Windows `\\.\pipe\myclash-<用户哈希>`；macOS/Linux `<userData>/agent.sock`（0600）。
- 首次启动生成 token，写入 0600 文件；GUI 附着时携带 token，Agent 校验后才接受命令。
- JSON Lines，请求/响应带 `id`；单条 30s 超时；命令串行执行。
- 方法：`status`、`reload`、`set-mode`、`restart-core`、`set-sysproxy`、`set-tun`、`speedtest`、`open-window`、`quit`。
- 协议版本不匹配时拒绝附着，提示升级应用，不尝试兼容旧字段。

## 3. 分层与包结构

```
cmd/myclash/            入口；--core / --agent / --parent / --portable 模式
internal/app/           组合根 + 页面（page_*.go）+ 托盘/动作/生命周期
internal/agent/         Agent 生命周期、控制 socket、附着/脱离、托盘
internal/ui/            外壳、主题、格式化、widgets（无业务、无页面）
internal/ui/acrylic/    Ground/Card/洗色/控件（来自 mygo-acrylic，含 guard/压力测试）
internal/ui/icons/      Lucide 生成图标（只含使用中的）
internal/core/          Host 接口、Embedded/Privileged、REST 客户端、流消费者
internal/config/        Profile 库、订阅同步、覆写链、Runtime 生成与校验
internal/platform/      系统代理、TUN 助手、开机启动、DNS、防火墙、路径
internal/settings/      设置 schema、默认值、迁移
internal/state/         运行期状态、流量采样、持久化
internal/i18n/          zh-CN / en 资源与格式化
internal/diag/          诊断包、脱敏、版本清单
```

依赖方向：`main → app → {ui, core, config, settings, platform}`；`ui` 可导入 `core`/`config` 的只读类型，但不导入 `app`；`core`/`config` 不导入 `ui`/`app`；`platform` 不导入其他内部包（只依赖标准库与 MyGo 的路径/Secrets）。详见 `13-code-style.md`。

## 4. 关键接口

```go
// core
type Kind int
const (
    KindEmbedded Kind = iota
    KindPrivileged
)

type Endpoint struct {
    Addr   string // 127.0.0.1:9090
    Secret string
}

type Host interface {
    Start(ctx context.Context, cfg []byte) error    // 首次启动（Embedded: hub.Parse）
    Reload(ctx context.Context, cfg []byte) error   // 原子切换配置，不重启进程
    SetMode(ctx context.Context, mode Mode) error   // 运行时切转发模式
    Stop(ctx context.Context) error                 // 停内核并释放端口
    Endpoint() Endpoint
    Kind() Kind
    WaitReady(ctx context.Context) error
}
```

`Reload` 与 `SetMode` 不经过 `PUT/PATCH /configs`：mihomo 的配置写路由在 embed 模式下被关闭
（原因见 ADR-0007）。Embedded 与 Privileged 在进程内使用同一组 mihomo 内部调用，Privileged Host 经控制管道转发。

```go
// platform/sysproxy
type Controller interface {
    Snapshot(ctx context.Context) (Snapshot, error) // 原始系统设置
    Enable(ctx context.Context, cfg Config) error
    Disable(ctx context.Context) error
    Status(ctx context.Context) (Status, error)     // 读系统真实值
    Restore(ctx context.Context, s Snapshot) error
}
```

```go
// config
type Pipeline struct {
    Store    ProfileStore
    Settings Settings
    Validate func([]byte) error // mihomo 解析器包装
}

func (p *Pipeline) Build(profileUID string) (Runtime, error)
func (p *Pipeline) Apply(ctx context.Context, host core.Host, rt Runtime) error
```

## 5. 内核宿主

### 5.1 Embedded

启动前先完成 mihomo 要求的初始化（顺序与 mihomo `main.go` 一致）：

```go
os.MkdirAll(<userData>/core, 0o700) // SetHomeDir 之前先建目录
constant.SetHomeDir(<userData>/core) // cache.db、geo、provider 的相对路径都以此为准
config.Init(<userData>/core)         // 与 mihomo main.go 相同的初始化
route.SetEmbedMode(true)             // 关闭 /restart、/upgrade、/configs 写

hub.Parse(cfg,
    hub.WithExternalController(addr),
    hub.WithSecret(secret),
    hub.WithExternalUI(<cache>/panel),
)
```

- **必须设置 HomeDir**。否则 `cache.db`、GeoIP/GeoSite、相对路径的 provider 会落到用户主目录，
  且不同 Profile 的 provider 缓存互相覆盖。
- 重载：`cfg, err := executor.ParseWithBytes(payload)` → `executor.ApplyConfig(cfg, false)`；
  端口未变时监听器复用（mihomo 的 `ReCreate*` 自行判断），失败不改变运行中的配置；
  若面板路径变化，随后调用 `route.SetUIPath`。
- 切转发模式：`tunnel.SetMode(...)`；日志级别：`log.SetLevel(...)`；不必重生成整份配置。
- Geo：设置 `geox-url`/`geodata-mode`，并调用 `updater.RegisterGeoUpdater()`（若开启自动更新）；
  手动更新调用 updater 的导出函数，而不是 `/upgrade/geo`（embed 模式已禁用）。
- 停止：`executor.Shutdown()`；随后等待端口释放（带超时）。
- Controller 绑定 `127.0.0.1`；端口默认 9090，占用时回退到临时端口并在设置页提示。
- REST 在 embed 模式下保留读与选择能力（代理、规则、连接、日志、提供者、DNS、缓存、存储），
  只关闭「改配置 / 重启 / 自升级」三类危险路由。

### 5.2 Privileged

| 平台 | 授权 | 网卡 | 子进程额外要求 |
| --- | --- | --- | --- |
| Windows | `ShellExecuteEx` + `runas` | wintun.dll（随包，exe 同级） | 无控制台窗口；继承 job object，父进程退出即终止 |
| macOS | `osascript ... with administrator privileges`（M2）→ LaunchDaemon（M3） | utun | macOS 12 用 `launchctl` 安装 LaunchDaemon；13+ 用 `SMAppService` |
| Linux | `pkexec`（M2）→ systemd + polkit（M3） | tun | 需要 `CAP_NET_ADMIN`；无 systemd 环境降级 |

- 子进程只做三件事：读 stdin 信封、校验并启动内核、报告就绪；不提供任意命令执行。
- 所有控制走 stdin/stdout JSON Lines，命令串行执行、单条 30s 超时；没有配置文件交接，避免 TOCTOU 与路径注入。
- 失败路径：授权被拒 → 开关回弹并提示；启动超时 → 终止子进程并恢复 Embedded。
- 子进程同样 `route.SetEmbedMode(true)`；否则 root 进程上的 `/upgrade` 会尝试覆盖应用可执行文件。

### 5.3 Agent（轻量模式）

- 角色：内嵌的 bundled 内核所有者、托盘、系统代理/TUN、全局快捷键、单实例锁、控制 socket。无窗口、无 Skia。
- GUI 附着：探测 socket → 校验 token/协议版本 → 读取 `status` → 构建页面；不启动第二个内核。
- 关闭主窗口：GUI 发 `quit` 给自己，退出；Agent 继续运行。托盘「显示窗口」拉起 GUI（带 `--attach`）。
- 模式切换：停止当前所有者 → 按新 `run_mode` 启动 → 断流一次；不允许两个所有者并存。
- 状态：Agent 写 `state.yaml` 与流量采样；GUI 写 `settings.yaml`/Profile 后发 `reload`。
- 崩溃：Agent 退出则内核停止；GUI 在运行时提示并可重启 Agent；无 GUI 时下次启动修复。
- Linux 无托盘时 Agent 仍可运行；用户再次启动应用即附着。

## 6. REST 客户端与流

| 流 | 路径 | 频率 | 消费者 |
| --- | --- | --- | --- |
| 流量 | `/traffic` | 1s | 概览、托盘、悬浮窗、流量采样 |
| 内存 | `/memory` | 1s | 概览 |
| 连接 | `/connections` | 1s（可暂停） | 连接页 |
| 日志 | `/logs?level=` | 实时 | 日志页、诊断 |

- 每条流一个 goroutine，解析失败即重连（指数退避 0.5s→8s，上限）。
- 流数据写入不可变快照（`atomic.Pointer`），UI 在 view 中读取；不把锁带进渲染。
- 日志按 100ms 批量合并；连接按 1s 整包替换；速率做 60 点环形缓冲。
- 暂停只停 UI 消费，不中断底层流（避免重连风暴）。
- 所有请求带 `Authorization: Bearer <secret>`；`http.Client` 的 `Proxy` 设为 `nil`，避免订阅/内核请求被系统代理再次代理。
- 连接表进程名依赖 `find-process-mode`：默认 `off`；用户开启「显示进程」时注入 `always`，失败时该列显示 `-`。
- 规则命中次数来自 `/rules` 的 `extra.hitCount`/`hitAt`（mihomo 提供）；规则提供者的条目内容不在 `/rules` 中，预览需读 provider 文件。
- Controller 的写路由在 embed 模式下不可用；任何配置/模式变更只能经 `Host`。

## 7. 配置管线

```
profiles/<uid>.yaml           Base
        │
        ▼
overrides/<uid>.yaml          Merge Override（深合并 / 列表前插后插 / 删除）
        │
        ▼
profiles/<uid>.js             Script Override（M2，goja，2s 超时）
        │
        ▼
应用注入                       端口、controller、secret、mode、log-level、
                              allow-lan、ipv6、tun、dns（按设置）
        │
        ▼
校验                          mihomo executor.ParseWithBytes
        │
        ├─ 失败 → 保留 last-good，返回带行号的错误
        ▼
runtime/candidate.yaml  →  Host.Reload  →  成功则写 runtime/last-good.yaml
```

### 7.1 合并语义（M1 固定）

完整语法、注入优先级、脚本契约与黄金测试见 [`05-config-pipeline.md`](./05-config-pipeline.md)。

- 映射：递归深合并，覆写文档优先。
- 列表：默认整体替换；`prepend-rules` / `append-rules` / `prepend-proxies` / `append-proxies` /
  `prepend-proxy-groups` / `append-proxy-groups` / `prepend-listeners` / `append-listeners` 做前插/后插。
- 删除：`delete: ["dns.fallback", "hosts"]`，按点路径删除。
- 未识别的键原样保留，并在预览中高亮提示（不阻断）。

### 7.2 应用注入的优先级

应用设置对以下字段**始终优先于** Profile 与覆写：`external-controller`、`secret`、`mixed-port`/`port` 等入站端口、
`mode`、`log-level`、`allow-lan`、`bind-address`、`ipv6`、`profile.store-selected`、`tun.*`。

DNS 分两种情况：TUN 开启时强制 `dns.enable=true` 且追加 `dns-hijack`；否则尊重 Profile 的 DNS 段，
仅当用户显式保存过 DNS 页才注入。

### 7.3 原子切换与回滚

1. 生成 candidate → 校验 → 写盘（临时文件 + rename）。
2. `Host.Reload(candidate)`；成功则 `last-good := candidate`。
3. 校验器：bundled 用内置 mihomo 解析器；Privileged 转发给子进程执行。
4. 失败：若 Host 仍可用，保持内存中的旧配置继续运行；若 Host 已不可用，用 `last-good` 重启。
5. 连续失败 2 次：停止自动重试，概览显示错误条与「查看诊断」。

## 8. 系统代理（三端）

| 平台 | 写入 | 校验 | PAC | 绕过 |
| --- | --- | --- | --- | --- |
| Windows | `HKCU\...\Internet Settings` 的 `ProxyEnable`/`ProxyServer`/`ProxyOverride`/`AutoConfigURL`，`InternetSetOptionW` 刷新 | 读回注册表 + 刷新成功 | 本机 HTTP 服务 + `AutoConfigURL` | `;` 分隔，支持 `*` 通配 |
| macOS | `networksetup -setwebproxy/-setsecurewebproxy/-setproxybypassdomains/-setautoproxyurl` | `scutil --proxy` | 同上 | 域名/IP/CIDR |
| Linux | GNOME `gsettings`；KDE `kwriteconfig6`（回退 5） | 读回设置 | 仅 GNOME | GNOME 忽略列表 |

- 开启前 `Snapshot()` 原值写入 `state.yaml`；退出、崩溃恢复、手动「还原」都走 `Restore()`。
- macOS 逐个活动网络服务（Wi‑Fi/以太网/雷雳）设置与快照；只改当前 `scutil --proxy` 生效的服务集合。
- 还原前先比对：若当前系统代理值既不等于我们写入的值、也不等于快照值，说明被其他程序改过，弹窗让用户选择「保留现状 / 还原快照」。
- Linux 只承诺 GNOME（`gsettings`）与 KDE（`kioslaverc`）；其他桌面隐藏入口，引导用 TUN。
- 崩溃会把系统代理留在失效端口上，用户会短暂断网：启动自检优先修复；设置页提供「安全模式启动（不改系统代理）」说明。
- PAC 服务：`127.0.0.1:<随机端口>/pac/<token>`，只响应本机；关闭即停。
- 不支持 PAC 的桌面隐藏 PAC 选项并说明；不支持的桌面整体隐藏系统代理页，引导使用 TUN。
- 端口变更时若系统代理开启，立即重写并校验。

## 9. TUN（三端）

- 运行时配置由 `tun.*` 设置 + Profile 合成；`stack` 默认 `mixed`，`device` 默认 `myclash`，
  `auto-route: true`，`auto-detect-interface: true`，`dns-hijack: ["any:53"]`。
- 开启顺序：停 Embedded → 授权 → 启动 Privileged → 等 ready → 校验网卡/路由/DNS → 标记就绪。
- 关闭顺序：停子进程 → 恢复 DNS → 清理路由 → 重启 Embedded → 恢复系统代理（若之前开启）。
- **与系统代理互斥**：开启 TUN 时自动关闭系统代理并记住其状态；关闭 TUN 时恢复。避免双代理与 PAC 失效。
- 「自动设置系统 DNS」写入 `1.1.1.1`/`8.8.8.8`（可配），由提权进程执行并在关闭时恢复；DNS 恢复必须在子进程退出前完成，否则 GUI 没有权限改回。
- DNS 原值在开启前快照；异常退出由下次启动的自检恢复。
- 防火墙默认不改；仅 Windows 提供「重设防火墙」按钮（`netsh advfirewall` 规则限定本程序）。
- 冲突检测：Windows Hyper-V/WSL 虚拟网卡、Linux `docker0`/其他 tun；只提示，不自动改路由。
- Linux 可选 `auto-redirect`（nftables）作为 M3 能力；默认关闭。
- 轻量模式下 TUN 由 Agent 启动提权子进程；GUI 关闭不影响 TUN，Agent 退出时按上面的顺序恢复。

## 10. 数据模型

`settings.yaml` 保存用户意图（默认值与策略），不保存「当前是否已开启」：

```yaml
schema: 1
app: { run_mode: standard }        # standard | lightweight
ui: { preset: mixed, blur: subtle, card_opacity: 0.8, theme: system, accent: system, density: comfortable, scale: 1, language: auto }
core: { source: bundled, mixed_port: 7890, controller_port: 9090, mode: rule, log_level: info, allow_lan: false, ipv6: false }
proxy:
  system: { method: manual, bypass: ["<local>", "127.*", "10.*"], restore_on_exit: true }
  tun: { stack: mixed, device: myclash, auto_route: true, strict_route: false, set_dns: true, dns: ["1.1.1.1", "8.8.8.8"] }
subscription: { auto_update: true, interval_hours: 24, user_agent: "clash-verge/v2.4.2" }
speedtest: { download_url: "", upload_url: "", duration_seconds: 8, max_bytes: 52428800 }
```

`profiles.yaml`：

```yaml
schema: 1
selected: p_7f3a
items:
  - uid: p_7f3a
    name: 示例订阅
    source: subscription          # subscription | local
    subscription:
      url: https://example.com/sub
      interval_hours: 24
      updated_at: 2026-11-14T10:00:00+08:00
      userinfo: { upload: 0, download: 0, total: 0, expire: 0 }
    chain: { merge: overrides/p_7f3a.yaml, script: profiles/p_7f3a.js }
```

`state.yaml` 保存事实（系统真实状态与上次运行痕迹）：

```yaml
schema: 1
last_good: runtime/last-good.yaml
runtime:
  host: embedded                  # embedded | privileged
  mode: rule                      # 当前生效的转发模式
  sysproxy: { enabled: true, snapshot: { ... }, applied: { ... } }
  tun: { enabled: false, dns_snapshot: { ... } }
```

- **边界**：设置是「用户想要」，状态是「现在是什么」。UI 开关读状态；写操作后读回系统/内核确认，再更新状态。
- **不重复**：当前 Profile 只存在 `profiles.yaml.selected`；`state.yaml` 不再存一份。
- 所有写入走「临时文件 + rename」，避免半写。
- `schema` 升级在启动时执行一次迁移，备份原文件到 `backup/migrate-<ts>/`；损坏文件隔离为 `.corrupt-<ts>` 并以默认值启动，同时提示。

## 11. 生命周期

**启动**：单实例/Agent 探测 → 路径/便携检测 → 迁移 → 加载设置/状态 → 系统代理与 TUN 残留自检 →
按 `run_mode` 启动/附着内核 → 启动流 → 建窗口/托盘 → 心跳 → 延迟任务（订阅同步、检查更新）。

- 先探测 Agent 控制 socket：存在且协议匹配则附着，**不新建内核**；不存在则按 `run_mode` 启动 GUI 内嵌内核
  或先启动 Agent 再附着。
- 无可用 Profile 时不启动内核，系统代理与 TUN 开关禁用，概览显示导入引导。
- 心跳：每 10s 调 `/version`；连续 3 次失败视为内核无响应，自动重启一次（用 last-good），仍失败则停核并报错。
- `--no-proxy` 安全模式：启动时不读状态、不碰系统代理/TUN，只用于修复残留。

**应用更新**：检查与下载由 GUI 发起（轻量模式下托盘「显示窗口」后操作）；安装前停止 Agent/内核、
系统代理与 TUN；替换后按原 `run_mode` 启动并恢复上次 Profile。deb/rpm 安装不执行应用内替换。

**退出**：

- 标准模式：停止订阅/更新任务 → 关系统代理（按设置）→ 停 TUN/恢复 DNS → 停内核 → 保存状态 → 释放单实例。
- 轻量模式：GUI 只关闭自己，向 Agent 发 `detach`；Agent 继续代理。托盘「退出」才执行上面的完整清理。
- Agent 退出：关系统代理 → 停 TUN/恢复 DNS → 停内核 → 写 `state.yaml` → 删 socket → 释放单实例。

- 退出时**不恢复**指向本应用的代理设置；`restore_on_exit` 指的是恢复「应用接管前」的系统快照（若快照本来就是关闭，则保持关闭）。
- 关 TUN 且应用仍运行时，才恢复到 TUN 开启前的系统代理状态。

**崩溃恢复**：启动时若 `state.runtime.sysproxy.enabled` 为真但与当前系统值不匹配，按快照恢复；
若 TUN 标记为开启但无子进程，清理 `myclash` 网卡并复位开关；若设置/索引 YAML 损坏，隔离后以默认值启动。

## 12. 并发模型

- MyGo 的 view 在主线程执行；view 内禁止网络、磁盘、锁等待。
- 每条流、每个下载任务、每个测速任务各有归属 goroutine；通过 channel/atomic 发布快照。
- 持久化状态由 `state.Store` 单写者模型保护（mutex + 去抖 500ms）；UI 只读快照。
- **单写者 = 内核所有者**：标准模式是 GUI，轻量模式是 Agent。GUI 在轻量模式下只写 settings/profiles，
  运行状态经控制通道读取；避免两个进程同时写 `state.yaml`。
- 从 goroutine 请求重绘用 `win.Update`，合并到 ≤10Hz；禁止在 goroutine 触碰 `ui.Element`。
- 对话框、通知、剪贴板等按 MyGo 规则从绑定方法或应用自己的 goroutine 调用。

## 13. 错误模型

```go
type Kind string

const (
    KindCoreUnavailable  Kind = "core_unavailable"
    KindConfigInvalid    Kind = "config_invalid"
    KindPortInUse        Kind = "port_in_use"
    KindPrivilegeNeeded  Kind = "privilege_required"
    KindPermissionDenied Kind = "permission_denied"
    KindSubscription     Kind = "subscription_failed"
    KindNetwork          Kind = "network"
    KindUnsupported      Kind = "unsupported"   // 当前平台/桌面不支持
    KindBusy             Kind = "busy"          // 正在切换，拒绝并发操作
    KindAgentUnavailable Kind = "agent_unavailable"
    KindInternal         Kind = "internal"
)

type Error struct {
    Kind    Kind
    Op      string
    Cause   error
    Message string // 已本地化：发生了什么
    Hint    string // 下一步动作
    Action  Action // retry | open_settings | elevate | open_logs | none
}
```

UI 映射：可恢复错误 → 概览错误条 + 重试；不可恢复 → `AlertDialog`；后台任务 → 通知。
错误日志写 app log，诊断包中脱敏。

## 14. 安全

完整威胁模型与对策见 [`07-security.md`](./07-security.md)。实现时必须守住：

- Controller 仅 loopback；secret 每次启动随机；`allow-lan` 开启时二次确认并显示风险。
- `route.SetEmbedMode(true)`：关闭 `/restart`、`/upgrade`、`/configs` 写，避免面板或本机进程把应用/内核换成别的二进制。
- 订阅下载只允许 http/https；重定向上限 5；响应体上限 32MB；YAML 别名/节点数限制；不执行订阅内容中的脚本。
- 脚本覆写（M2）在 goja 沙箱运行：无文件/网络/进程访问，2s 超时，异常不落盘。
- Web 面板资源校验 SHA-256；secret 经 URL 参数注入仅在本地资源面板；自定义远程面板需显式确认并默认不给 secret。
- 更新资产必须验签（`mygo keygen` 密钥对）；失败拒绝安装。
- 提权子进程校验父进程 PID + 可执行文件路径 + 一次性 token；配置与命令只走 stdin，不接受外部路径。
- macOS 助手需签名；Linux polkit 规则限定可执行文件路径与参数。
- 遥测：无。崩溃与诊断只写本地，导出前脱敏。

## 15. 测试策略

| 层 | 方式 |
| --- | --- |
| 配置管线 | 黄金文件：Base/Merge/Script/注入/最终 YAML；含失败与回滚 |
| 解析与格式化 | 表驱动：`subscription-userinfo`、绕过列表、字节/速率、时间 |
| REST 客户端 | `httptest` 假内核，覆盖分页、流断线、401、5xx |
| Host | Embedded 集成测试：临时端口 + 最小配置，验证启动/重载/停止；Privileged 用手工子进程模拟握手 |
| UI | `ui.NewTester`：页面三态、键盘路径、过滤/排序；不依赖截图，断言结构与文本 |
| 外观 | 亚克力 guard 测试（禁止卡片上的禁用控件/层色）+ 洋红压力截图（深/浅、0% 不透明度） |
| 平台 | 命令构造单测；三端 CI 跑单元与构建；系统代理/TUN 需真机手工清单 |
| 发布 | 三端安装包在干净虚拟机验收；升级/回滚路径演练 |

CI 矩阵：`windows-latest`、`macos-latest`、`ubuntu-latest`；`go vet`、`go test ./...`、
`mygo build -platform ...`（各自本机目标）；发布工作流按平台签名/打包。

## 16. 构建与发布

- `mygo.json`：名称、identifier、版本、图标；`mygo keygen` 生成更新密钥，公钥入库，私钥只进 CI secret。
- 资源：Windows `wintun.dll`；许可与第三方声明；托盘图标（亮/暗/动态速率）。
- 内核版本随应用更新（ADR-0012）；安装包只带内置内核，不下载外部二进制。
- 构建：`mygo build -platform darwin/universal,windows/amd64,windows/arm64,linux/amd64,linux/arm64`。
- 资产：Windows `setup.exe` + portable zip；macOS dmg（签名+公证）；Linux tar.gz/deb/rpm/PKGBUILD。
- 版本：语义化；内核升级单独记录在 release notes，附带回归清单。

## 17. 性能与资源的架构护栏

当前不设内存门槛，但以下约束先立住，避免后期返工：

- 表格/列表只渲染可见行（MyGo `Table`/`List` 已虚拟化）。
- 日志环形缓冲上限 5000 行 / 8MB；连接快照只保留当前集合，不累积历史。
- 流解析复用缓冲；JSON 解码不产生整包字符串拼接。
- UI 重绘合并到 ≤10Hz；速率曲线只保留 60 点。
- 材质与玻璃：系统材质只用于 Ground；玻璃 pane ≤6/屏、Group ≤8；工作卡不透明度唯一入口；guard 测试 + 洋红压力截图（见 09）。

### 17.1 流量统计

- `/traffic` 给的是瞬时字节/秒，不是累计值。采样器每秒读一帧，用**实际经过时间**积分成累计量，不假设恰好 1s。
- 每 60s 把差值写入 `traffic/YYYY-MM.jsonl`，记录 `{ts, profile_uid, up, down}`；当前 Profile 在采样时确定。
- 内核重启时累计量归零，采样器随之归零，但已落盘的历史保留；用单调时钟计算间隔。
- 日/月汇总由 1 分钟样本聚合；保留 90 天，按月压缩；时钟回拨时丢弃异常样本。
- 不用连接表求和做总流量：连接关闭后其计数消失，会低估。

### 17.2 规则命中

- `/rules` 的 `extra.hitCount` / `extra.hitAt` 提供规则命中信息；仅在规则页可见时每 10s 刷新一次，不在后台轮询。

## 18. 内核升级与兼容

- `go.mod` 固定 mihomo 版本；升级单独 PR，必须跑：
  1. 全部配置黄金测试；
  2. REST 契约冒烟（版本、代理、连接、日志、规则、提供者、DNS、存储）；
  3. 内部 API 契约检查：`constant.SetHomeDir`、`config.Init`、`route.SetEmbedMode`、`executor.ParseWithBytes`、`executor.ApplyConfig`、`tunnel.SetMode`、`updater.RegisterGeoUpdater`；
  4. 三端 TUN/系统代理手工清单；
  5. 生成配置字段 diff，记录新增/废弃字段。
- 应用注入字段若被内核重命名，编译期/测试期失败，不静默丢弃。
- 内核许可与版本文本进「关于」页。

## 19. 网速测试

与「延迟测试」分开：延迟只测 TTFB（`/proxies/{name}/delay`），网速测吞吐（Mbps）。

### 19.1 方法

- 客户端用 `http.Transport{Proxy: http://127.0.0.1:<mixed-port>}`，直接连本机混合端口，不依赖系统代理开关。
- **当前节点测试**：不改变任何选择，流量按当前规则走；结果反映用户实际路径。
- **指定节点测试**：仅当节点所属组是 `select` 时可用。记录当前选择 → `PUT /proxies/<group>` → 等 100ms
  → 测速 → 在 `defer` 中恢复原选择（失败/取消也恢复）；恢复失败时明确报错。
- auto/fallback/load-balance 组不允许强制指定，只提供「测当前」。
- 下载：GET 可配 URL，按 `duration`（默认 8s）或 `max_bytes`（默认 50MB）先到者停止；
  Mbps = bytes×8 / 实际秒数；TTFB 作为延迟参考。
- 上传：POST 可配 URL，同规则；默认关闭，用户显式勾选才测。
- 抖动：连续 5 次小请求的 TTFB 标准差；与吞吐同一次会话完成。
- 并发：全局同时只允许 1 个网速测试；延迟测试并发上限 16；两者可同时存在但互不阻塞。

### 19.2 默认端点

- 下载/上传默认使用 Cloudflare 的测速端点（`speed.cloudflare.com/__down`、`__up`），可在设置中替换。
- 端点只允许 `http(s)`；默认拒绝回环/内网地址，避免被当成端口扫描器；用户改为内网地址时二次确认。
- 端点不可达时给出「换一个测速地址」的可执行建议，不把失败归因为节点慢。

### 19.3 与运行模式的关系

- 无论 standard（GUI 内嵌）还是 lightweight（Agent 内嵌）或 TUN（Privileged），都通过本机 mixed-port 测速，不需要内核内部 API。
- TUN 开启时仍走 mixed-port；系统代理关闭不影响。
- Agent 模式下由 GUI 发起测速，经控制通道让 Agent 执行选择切换与恢复（避免两个进程争抢选择）。

### 19.4 结果与隐私

- 结果：下行/上行 Mbps、TTFB、抖动、已用流量、时长、端点、节点、时间。
- 仅保留本次会话（M2）；历史记录 M3 可配，默认关闭。
- 不上传结果、不带遥测；测速流量计入节点流量，UI 明示会消耗流量。
- 取消：context 取消 → 关闭连接 → 恢复选择 → 释放缓冲区；目标 200ms 内响应。
