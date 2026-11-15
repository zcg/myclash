# UI 依赖策略：MyGo 原生 + 内嵌 Lucide

Status: accepted

## Context

社区有一批 MyGo 组件库（Onevra、Lobe MyGo、uix、CrychicUI、Mayu 等），能提供大量现成控件与图标。
但我们的设计是「系统原生材质 + 玻璃 + 自有 token」，并且编译在 MyGo v0.9.0；候选库普遍锁在 v0.5–v0.8，
`uix` 还会连带 Onevra 与 CrychicUI。引入它们会把主题、图标、Shell 甚至音效一起带进来。

## Decision

1. M1–M3 不把任何第三方 UI 组件库加入 `go.mod`。主界面使用 MyGo `ui` 原生控件、官方 `plugins/glass`，
   以及从 `mygo-acrylic` 资产复制进 `internal/ui/acrylic` 的自有组件层（带 guard/压力测试）。
2. 图标使用 Lucide SVG，按需内嵌到 `internal/ui/icons`，不引入图标运行时库；保留 ISC/Feather 许可声明。
3. 第三方库作为参考：Onevra（Shell/命令面板/Lucide 组织）、Lobe（token 分层）、uix（组合式 API）、
   CrychicUI（功能覆盖清单）。不复制其视觉与音效。外观的层次与玻璃用法见 ADR-0011。
4. 若未来某个组件确实无法用 MyGo 组合出来，按 `10-ui-libraries.md` §5 的准入清单评估，
   一次只取一个组件，并包在自有接口后；不允许整套主题/Shell 接管。

## Considered Options

- 直接采用 Onevra 作为 Shell 与组件层：开发最快，但会替换我们的设计语言，并把 MyGo 降到 v0.8.3。
- 采用 Lobe/CrychicUI：外观与音效不符合代理工具的定位，组件数量远超需要。
- 只用 MyGo、不引入任何外部资产：图标也要手绘，成本高；Lucide 内嵌是更小的例外。

## Consequences

- 组件与图标都由我们控制，版本跟随 MyGo v0.9.0，不会被迫降级。
- 需要自己实现少量组合组件（统计卡、节点行、空状态、命令面板），已在 UI 文档记录。
- 亚克力组件层需要随仓库维护，并跑 guard 测试与洋红压力截图；上线前确认 Gloss/资产许可与署名。
- 官方 glass 插件只用于窗口内玻璃；不把它当作桌面材质。
- 图标集需要生成脚本与许可清单；新增图标有固定流程。
- 若未来 MyGo 升到 v1 且某个库同步跟进，可重新评估，但仍以「不覆盖设计 token」为前提。
