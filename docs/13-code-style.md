# 代码组织与风格

> 这份文档约束实现，不重复产品设计。包职责以 `03-architecture.md` 为准；本文规定**文件怎么放、依赖往哪走、代码长什么样**。
> 目标：半年后加功能不用翻旧账，新页面不制造第二套状态，审查时能一眼看出边界。

## 1. 一条原则

**组合根知道所有事，叶子包只知道自己的事。**
`main` 只做参数与 `app.Run`；`app` 装配一切；`ui` 只画；`core` 只管内核；`config` 只处理配置；`settings` 只读写设置。
任何「顺手在这里也做一点」都是包边界破口。

## 2. 实际包结构

```
main.go                     入口：flags、单一实例、app.Run
internal/
  app/                      组合根 + 页面 + 生命周期
    app.go                  依赖装配、窗口、路由、主循环接线
    page_overview.go        每页一个文件，方法挂在 *App 上
    page_proxies.go
    page_profiles.go
    page_connections.go
    page_rules.go
    page_logs.go
    page_settings.go
    page_core.go
    page_network.go         系统代理/TUN/DNS/嗅探（可再拆）
    page_tools.go           覆写/资源/流量/解锁/面板
    tray.go                 托盘菜单与动态图标
    actions.go              跨页动作（启停内核、切换配置、测速）
  ui/                       可复用外壳与控件，无业务
    shell.go                侧栏 + 内容区 + 底栏
    theme.go                acrylic 主题、明暗、强调色
    format.go               字节/速率/时间格式化
    widgets/                StatCard、NodeRow、EmptyState、ErrorBar
    acrylic/                从 mygo-acrylic 复制（逐字保留，仅改 import）
    icons/                  Lucide 生成文件（只含使用中的）
  core/                     内核宿主、REST 客户端、流、快照
    host.go                 Host 接口、Kind、Endpoint
    embedded.go             hub.Parse / ApplyConfig / Shutdown；mihomo 全局初始化
    client.go               REST：version/configs/proxies/select/delay
    stream.go               traffic/connections/logs 流与重连
    snapshot.go             不可变快照与发布
    types.go                对外只读类型（Version、Proxy、Rate）
  config/                   Profile 库与 Runtime 生成
    store.go                profiles.yaml、文件、选择
    subscribe.go            下载、userinfo、命名
    merge.go                合并覆写
    inject.go               应用注入、provider 路径重写
    pipeline.go             生成、校验、原子切换
    script.go               goja 沙箱（M2）
  settings/                 settings.yaml schema、默认值、原子读写
  state/                   流量采样/历史持久化（M3）
  platform/                 系统能力（接口 + 三端实现）
    sysproxy.go / sysproxy_windows.go / sysproxy_darwin.go / sysproxy_linux.go
    tun.go / ...
    autostart.go / dns.go / paths.go
  agent/                    轻量模式（M3）：控制 socket、附着/脱离
  i18n/                     语言资源与类型化消息
  diag/                     诊断包、脱敏
```

页面放在 `app` 而不是 `ui`：页面知道业务状态，`ui` 不知道。这样 `ui` 可以被任何页面复用，也不会反过来 import `app`。

## 3. 依赖方向

```
main → app
app  → ui, core, config, settings, state, platform, i18n, agent
ui   → core(只读类型), config(只读类型), i18n
core → mihomo, settings? (不；由 app 传入)
config → yaml, settings? (不；由 app 传入 Options)
platform → 标准库
i18n → 无内部依赖
```

硬规则：

- **禁止循环依赖**。`ui` 不得 import `app`；`core`/`config` 不得 import `ui`/`app`。
- `core` 不读设置文件；启动参数由 `app` 组装后传入。
- `config` 不直接读 `settings.Settings`；用 `config.Options`（端口、模式、日志级别、TUN 开关）避免耦合。
- `platform` 不 import 其他内部包；需要 i18n 错误时返回带 `Kind` 的错误，由 app 翻译。
- 只有 `app` 与 `main` 知道所有包；其他包只看得到自己的邻居。

## 4. 命名与文件

- 包名：小写单词，不用下划线、不用 `util`/`common`/`helpers`/`base`/`misc`。
- 文件：`snake` 不需要；用 `feature.go`、`feature_test.go`；平台实现 `feature_windows.go`、`feature_darwin.go`、`feature_linux.go`。
- 类型：`Host`、`Embedded`、`Client`、`Snapshot`、`Profile`、`Store`、`Pipeline`、`Options`；避免 `Manager`、`Service`、`Helper` 这类空词。
- 构造函数：`NewStore(dir)`、`NewClient(endpoint)`；只在需要替换实现时返回接口。
- 接口定义在**使用方**附近；接口只含调用方需要的方法，通常 ≤5 个。
- 导出标识符必须有文档注释，首词是名字；注释解释「为什么/约束」，不复述代码。
- 代码注释、标识符用英文；文档与用户可见文案用中文（文案走 i18n）。这样与 Go 生态一致，搜索也方便。

## 5. Go 风格

- `gofmt` 必须；CI 跑 `gofmt -l`、`go vet`、`staticcheck`（可用时）。
- 错误：`fmt.Errorf("load profile %s: %w", uid, err)`；只在调用方需要分支时定义哨兵错误；不把错误字符串当 API。
- 不 `panic`（除 `init`/测试）；不在库代码 `log.Fatal`；`recover` 只在 goroutine 顶层记录并退出该任务，不吞掉。
- `context.Context` 第一个参数；所有网络/下载/测速可取消；不用 `context.TODO()` 进生产路径。
- 并发：谁创建 goroutine 谁负责结束；用 `errgroup` 或 `sync.WaitGroup` + `ctx`；禁止无主 `time.Ticker`。
- 锁：只在状态所有者内；不在持锁时做 IO/回调；UI 线程不拿锁（读快照）。
- 全局可变状态：只允许 MyGo app 与 mihomo 要求的全局（`constant.SetHomeDir` 等），集中在一个文件并注释原因。
- 时间/随机：生产代码用 `time.Now`，测试用容差或注入的最小接口；不要为可测性过度抽象。
- 切片/映射：返回副本或只读快照；不把内部 map 直接交给 UI。
- 字符串：路径用 `filepath`；URL 用 `net/url`；不用 `+` 拼用户可见句子（i18n）。
- 文件 IO：写入临时文件 + `rename`；权限 0600/0700；错误带路径。
- 依赖：标准库 → MyGo → mihomo → yaml → goja（M2）；新增依赖要说明用途、许可、体积。

## 6. UI 代码规则

- 每个页面一个文件，方法挂在 `*App` 上：`func (a *App) overview(c *ui.Context)`。
- `view` 只读状态、只建控件；禁止网络、磁盘、`time.Sleep`、阻塞等待。
- 页面本地状态放页面 struct（搜索词、选中行、展开组），不要塞进全局；跨页共享的才进 `App`。
- 数据来自 `core.Snapshot` / `config.Store` 的只读快照；不直接调 REST。
- 动作走 `App` 的方法（`a.startCore()`），页面不拼 URL、不做业务分支。
- 卡片/控件遵守 `09-appearance.md`：Ground/Card/浮层分层、洗色、单一不透明度、禁用清单。
- 不手拼按钮/输入框/分栏；用 `acrylic` 组件或 MyGo 语义控件；新增组合写进 `02-ui.md` 对账表。
- 图标只从 `ui/icons` 取；不内联 SVG 字符串，不在页面里 `MustParseSVG`。
- 文案全部 `i18n`；数字/字节/时间用 `ui/format.go`，不在页面里手写格式。
- 列表/表格虚拟化交给 MyGo；不自己 for 循环建 1000 个 Row。
- 重绘从 goroutine 用 `win.Update` 合并到 ≤10Hz；不在 goroutine 触碰 `ui.Element`。

## 7. 状态与数据

- 运行状态单写者：standard 是 GUI，lightweight 是 Agent；见 ADR-0008。
- `core.Snapshot` 不可变；发布用 `atomic.Pointer` 或短暂 mutex，读侧不阻塞渲染。
- `settings`/`profiles` 写入去抖 500ms；退出前 flush。
- Runtime Config 落盘前脱敏（`secret` 等）；真实密钥只在内存。
- 选择节点、模式等运行事实以内核返回为准；UI 不做乐观状态。
- 选中 Profile 只存在 `profiles.yaml`；不在 state 再存一份。

## 8. 错误与日志

- 领域错误带 `Kind`、操作、底层错误、用户提示、下一步动作（见 `03` §13）。
- 错误只在一处翻译成用户文案；不在每层拼句子。
- 日志用 `log/slog`（或 MyGo 日志）结构化字段：`component`、`profile`、`endpoint`、`err`。
- 不记录 secret、订阅 URL 的 token、WebDAV 凭据；诊断包脱敏复用同一函数。
- 后台任务失败：记录日志 + 在对应页面显示错误态；不弹全局模态。

## 9. 测试

- 表驱动；测试名 `TestXxx_condition`；不 sleep 等异步，用 channel/`t.Cleanup`。
- 纯逻辑（合并、注入、格式化、错误映射）必须有单测；`config` 用 golden 文件。
- REST 客户端用 `httptest` 假内核；不访问真实网络。
- UI 用 `ui.NewTester` 测页面三态、键盘路径；不做像素截图断言（外观用 guard/洋红人工截图）。
- Embedded Host 集成测试用临时 HomeDir、随机端口、最小配置；测试间串行（mihomo 全局状态）。
- 三端平台代码：命令构造单测（fake exec）+ 真机手工清单；TUN/系统代理不假装 CI 能自动验证。
- 每个 PR 跑 `go test ./...`、`go vet`；改动外观跑 guard/压力。

## 10. 依赖与资产

- 不引入第三方 UI 组件库（ADR-0010）；图标 Lucide 按需内嵌。
- `internal/ui/acrylic` 逐字保留上游组件，只改 import；业务定制放 `ui/widgets`，不直接改 acrylic。
- 新增图标走生成脚本，记录上游版本与哈希；不手画。
- 新增 Go 依赖需在 PR 说明：用途、许可、二进制增量、替代方案。

## 11. 审查清单

- [ ] 没有循环依赖；`ui` 没 import `app`；`core` 没读设置文件。
- [ ] 新页面只有 `view` + 动作；没有在 view 里做 IO/等待。
- [ ] 状态单写者；快照只读；UI 不持锁。
- [ ] 错误有 Kind/提示/下一步；日志不含密钥。
- [ ] 文件写入原子；权限正确；路径用 filepath。
- [ ] 新增控件先查 `02-ui.md` 对账表；卡片上没有禁用控件/实色层。
- [ ] 测试覆盖失败路径；UI 三态；无真实网络。
- [ ] `gofmt`/`go vet`/`go test` 通过；注释解释为什么。

## 12. 反模式（见到就改）

- `internal/util`、`common`、`helpers`；跨包随便 import。
- 页面直接 `http.Get` 或拼 mihomo URL。
- 在 `view` 里 `sync.Mutex.Lock` 等状态。
- 全局变量缓存 UI/颜色/路径。
- 错误字符串比较；`errors.New` 满天飞没有 Kind。
- 一个文件 1000 行；一个函数 200 行；五层嵌套。
- 把业务逻辑写进 `ui/widgets`。
- 为「以后可能用」提前抽象接口/泛型。
- 复制一份状态给页面（store 一份、app 一份、page 一份）。
- 用 `panic` 处理配置错误。
