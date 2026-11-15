# UI 组件库与图标：评估与采纳规则

> 结论先说：**主界面不引入第三方组件库**，用 MyGo v0.9.0 原生控件 + 官方 `plugins/glass` +
> 从 `mygo-acrylic` 复制进仓库的自有组件层；图标用 Lucide SVG 按需内嵌。
> 评估时间 2026-11-14；星数与版本会变，采纳前重新核对。

## 1. 评估标准

| 标准 | 为什么重要 |
| --- | --- |
| MyGo 版本 | 我们编译在 v0.9.0；库若锁在旧版本，会降级/冲突，`ui.Element` 生命周期与主题 API 可能已变 |
| 依赖重量 | 组件库若连带主题、图标、Markdown、二维码、音频，二进制与供应链都会膨胀 |
| 设计契合 | 我们的视觉是「系统原生材质 + 玻璃 + 自有 token」；Lobe/Material/像素/哥特会覆盖这套语言 |
| 可拆分 | 能否只取一个控件，而不引入它的 Shell、主题、路由与全局初始化 |
| 维护与许可 | 提交活跃度、MIT/ISC 许可、是否要求署名 |
| 无障碍 | 是否用真实控件语义；图标是否可命名 |

## 2. 候选库事实

| 库 | 约星数 | 许可 | 依赖的 MyGo | 特点 | 结论 |
| --- | --- | --- | --- | --- | --- |
| [Onevra](https://github.com/oneo-me/onevra) | 288 | MIT | **v0.8.3** | 主题控件、窗口 Shell、Lucide 图标、聊天/Markdown/代码高亮/图表/命令面板；模块在 `source/` | **参考**：Shell、命令面板、图标组织方式 |
| [Lobe MyGo](https://github.com/lobehub/lobe-mygo) | 245 | MIT | **v0.8.4 伪版本** | Lobe 设计语言，70+ 组件、2300+ Lucide | **参考**：token/主题架构；不采用外观 |
| [uix](https://github.com/smallnest/uix) | 25 | MIT | **v0.6.4** | shadcn 风格；依赖 Onevra + CrychicUI + Chroma + goldmark | **不采用**：依赖网太深、版本最旧 |
| [CrychicUI](https://github.com/ELY-Open-Source/CrychicUI) | 55 | MIT | **v0.8.3** | 1500 组件、动效、音效 | **参考覆盖清单**；不采用音效与规模 |
| [Mayu](https://github.com/harukizmoe/mayu) | 2 | MIT | **v0.5.1** | Material 3 | 不做 Material 预设，暂不采用 |
| [8bitMyGoUI](https://github.com/ELY-Open-Source/8bitMyGoUI) | 15 | MIT | 未核实 | 16-bit 像素风 | 不做 |
| [MujicaUI](https://github.com/ELY-Open-Source/MujicaUI) | 26 | MIT | 未核实 | 宫廷哥特 | 不做 |
| Breeze UI | — | — | — | 模拟系统原生控件 | 未找到可核实仓库，暂不引用 |
| [mygo-native-ui](https://github.com/xingkaixin/mygo-native-ui) | — | — | 未核实 | 原生 UI 组件集 | 未评估，仅参考 |
| [mygo-platinum](https://github.com/firede/mygo-platinum) | — | — | 未核实 | 设计系统/组件 | 未评估，仅参考 |
| [Sunbeam](https://github.com/AstraFoundry/design) | — | — | 未核实 | 设计语言 | 未评估，仅参考 |
| [NuxtUI Go](https://github.com/josexy/nuxt-ui-go) | — | — | 未核实 | Nuxt UI 风格 | 未评估，仅参考 |
| [Kubai Icons](https://github.com/yldm-tech/mygo-kubai-icons) | — | — | — | MyGo 图标集 | 图标候选；默认 Lucide |
| [AI Logo](https://github.com/yldm-tech/ai-logo) | — | — | — | 生成式 logo | 品牌占位参考 |

### 2.1 版本冲突是硬约束

- 我们的 `go.mod` 是 `github.com/egoist/mygo v0.9.0`。
- 候选库全部要求更旧的 MyGo（v0.5.1–v0.8.4）。`go get` 会把 MyGo 降级或要求伪版本，
  直接破坏我们依赖的新 API。即使强行 `replace`，也无法保证编译与行为。
- `uix` 进一步依赖 Onevra 与 CrychicUI，等于把三套设计系统与主题一起拉进来。
- 结论：**在 M1–M3 不把任何组件库加入 `go.mod`**。

## 3. 决策

1. 主界面 = MyGo `ui` 原生控件 + 官方 `plugins/glass`（窗口内玻璃）+ `internal/ui/acrylic`
   （从 `mygo-acrylic` 资产复制，含 Ground/Card/洗色/控件/guard/压力测试）。
   **系统材质**（Vibrancy）与玻璃是两个层次，都保留；平台差异见 ADR-0011。

`mygo-acrylic/assets/acrylic` 没有独立 `go.mod`；已用 MyGo **v0.9.0** 在临时模块编译并通过其自带测试（2026-11-14）。
落地时放进本模块，保留来源说明；发布前确认许可与 Gloss 署名。
2. 需要 MyGo 没有的语义时，先用 `ui` 的 base/part 组合，并在 `02-ui.md` 记录判定；不引入整套库。
3. 图标 = Lucide SVG 按需内嵌，不引入图标运行时库。
4. 第三方库只作为**参考**：
   - Onevra：窗口 Shell 的层级、命令面板交互、Lucide 懒解析与许可组织。
   - Lobe：设计 token 的分层与组件 API 命名。
   - uix：shadcn 式「组合而非配置」的组件写法。
   - CrychicUI：功能覆盖清单（检查我们是否漏了某类控件），不采用其视觉与音效。
5. 未来若要采用某个库，必须先过 §5 的准入清单；一次只取一个组件，包在我们自己的组件接口后。

## 4. 图标方案

### 4.1 选择 Lucide

- 许可：ISC（Feather 衍生），允许商用与修改，保留版权声明即可。
- 风格：1.5–2px 线性、24×24 网格，与桌面工具的密度一致。
- 覆盖：导航、操作、状态、连接、规则、日志、网络等所需图标齐全。
- MyGo `ui.Icon` 明确支持 Lucide/Heroicons/Tabler/Material Symbols 的 SVG，直接 `ui.MustParseSVG`。

备选：Tabler（MIT）、Heroicons（MIT）、Material Symbols（Apache-2.0）。**只选一套**，不混用线性与填充风格。

### 4.2 内嵌方式

```go
// internal/ui/icons/lucide.go（生成文件，仅包含实际使用的图标）
//go:embed svg/arrow-down.svg
var arrowDownSVG []byte

var ArrowDown = ui.MustParseSVG(arrowDownSVG)
```

- 只提交用到的 SVG；新增图标走一个生成脚本，从上游固定版本导出，记录 `upstream.json`（版本 + 文件哈希）。
- 懒解析：首次使用时 `MustParseSVG`，之后共享指针；不在启动时解析全部图标。
- 命名：上游 kebab-case → PascalCase（`arrow-up-right` → `ArrowUpRight`），与 Onevra 的约定一致，便于替换。
- 许可：`NOTICE`/`关于 → 许可` 列出 Lucide 与 Feather 版权；SVG 文件头不裁剪许可。
- 尺寸：默认随字号（16/20）；工具栏 18；导航 18；状态点不用图标。
- 无障碍：纯图标按钮必须 `Label`；装饰性图标不命名。

### 4.3 不做什么

- 不用图标字体（字形对齐、可访问性、缩放都更差）。
- 不在运行时下载图标。
- 不为了一个图标引入整套图标库依赖。
- 不混用两套图标的线宽与圆角。

## 5. 未来采纳清单（任何组件库）

- [ ] 目标 MyGo 版本与我们的 `v0.9.0` 一致或更新；`go list -m all` 无降级。
- [ ] 只引入需要的包；`go mod graph` 不拉入第二套主题/图标/路由。
- [ ] 用 `ui.Tester` 写一个最小示例，验证焦点、键盘、主题跟随。
- [ ] 包一层 `internal/ui` 接口，业务代码不直接依赖第三方 API。
- [ ] 许可与 NOTICE 更新；二进制增量可接受（记录前后大小）。
- [ ] 视觉不覆盖我们的 token；若带主题，必须能关闭。
- [ ] 有维护者响应记录；最近 6 个月有提交。

## 6. 需要作者提供什么

不是必需，以下会提高贴合度：

1. 品牌 logo / wordmark 的 SVG（没有就用占位，后续替换）。
2. 图标风格偏好：Lucide（默认）/ Tabler / Heroicons；只选一套。
3. 喜欢的强调色或品牌色（默认跟随系统强调色）。
4. 参考截图：Verge/Party 中你觉得「就该这样」的具体页面状态。
5. 是否要命令面板（`Ctrl/Cmd+K`）作为主入口之一。默认做，见 PRD F-UI-01。

不需要提供组件库代码；若你希望某种库的**外观**，告诉我们库名与页面，我们按「主题预设」评估，而不是直接换整套控件。

## 7. 与现有文档的关系

- 控件选择与判定：`02-ui.md` 的对账表。
- 材质与明暗：`09-appearance.md`。
- 内存/性能/体验：`08-quality.md`。
- 决策记录：`docs/adr/0010-ui-dependencies.md`。
