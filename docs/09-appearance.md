# 外观：系统材质、窗口内玻璃与亚克力纪律

> 三个层次，不能混为一谈：
> **系统材质**（`Vibrancy`）在窗口背后，按操作系统给；**窗口内玻璃**（`plugins/glass`）模糊窗口自己画的内容，
> 三端一致但看不到桌面；**亚克力**（`mygo-acrylic`）是透明度与控件的纪律，不绑定平台。
> 三者可以同时存在。任何「Linux 也能透出桌面磨砂」的说法都是假的。

## 1. 三层模型

| 层 | 可选做法 | 透明度 |
| --- | --- | --- |
| **Ground（窗口底）** | ① 系统材质 + `Ground × 0.65` 薄纱；② 无系统材质时实铺/渐变；③ 需要玻璃感时在渐变上叠 `glass.Blur` | 固定薄纱；玻璃另算 |
| **Card（工作卡）** | `Card × 用户不透明度`（0–100%，默认 80%） | **唯一**可调入口 |
| **浮层/模态/代码与数据画布** | `Popover` / `Modal` / `Code` 实色 | 实色 |

- 侧栏不铺底，坐在 Ground 上；卡片里的面板、列表、工具栏不铺底，坐在 Card 上。
- 悬浮窗是 Card 的特例：整窗一张卡，仍然只有一层可调透明度。
- 代码、diff、日志、连接/规则表放在 `Code` 画布；设置块与对话框放在 `Popover`/`Modal` 面。

## 2. 四种外观预设

| 预设 | 窗口底 | 工作卡 | 适用 |
| --- | --- | --- | --- |
| **液态玻璃（默认）** | MyGo `plugins/glass` 的窗口内 Liquid Glass（三端同一 shader） | 半透明色（0–100%） | 毛玻璃、折射与边缘高光；与系统亚克力无关 |
| **透明** | 不铺底、不挂系统材质，直接显示桌面 | 半透明色（0–100%） | 想看穿窗口；0% 时只剩文字 |
| **实色** | `Ground` 实铺 | 实色 | 远程桌面、低端机、提高对比度 |
| **Windows 亚克力** | Windows 11 22H2+ 的 Acrylic（`SetWindowCompositionAttribute` 的 ACCENT_ENABLE_ACRYLICBLURBEHIND） | 半透明色（0–100%） | 仅 Windows 11 22H2+ 显示；Windows 10/macOS/Linux 不出现 |

- 液态玻璃与 Windows 亚克力不是同一个东西：前者是 MyGo 插件的窗口内玻璃，三端一致；后者是 Windows DWM 的系统材质，只在满足版本条件的 Windows 上启用。
- 液态玻璃的窗口底用 `glass.Glass{Style: glass.Regular}`；不挂系统 `Vibrancy`，避免在 Windows 上被 DWM 混成亚克力。
- Windows 亚克力仅在 `platform.AcrylicSupported()`（build >= 22621）时显示；否则设置页隐藏该选项，已保存的值自动回退到液态玻璃。
- 透明预设不挂 `Vibrancy`，窗口根节点也不铺 `Ground`；工作卡不透明度为 0% 时窗口完全透明。
- 用户不透明度只有一个滑杆：0–100%，默认 30%。
- 不设模糊档位：固定使用各材质的最强档，避免「档位看起来没变化」。

## 3. 平台能力矩阵

| 能力 | macOS 12–15 | macOS 26+ | Windows 11 22H2+ | Windows 10 | Linux |
| --- | --- | --- | --- | --- | --- |
| 系统材质（透桌面） | NSVisualEffect | Liquid Glass / NSGlassEffectView | Mica / Acrylic | **无** | **无** |
| 窗口内玻璃（透应用内容） | `glass.Glass` | `glass.Glass` / `Native` | `glass.Glass` | `glass.Glass` | `glass.Glass` |
| 推荐 Ground | 系统材质 | 系统材质 | Mica（柔和）/ Acrylic（强力） | 渐变或实色 | 渐变或实色 |
| 窗口内玻璃 | `glass.Glass` | `glass.Glass` / `Native` | `glass.Glass` | `glass.Glass` | `glass.Glass` |
| Web 面板玻璃 | 普通 | `Native` 可采样页面 | 普通 | 普通 | 普通 |

- `c.Vibrancy()` 为 false 时，不渲染系统材质；液态玻璃只剩半透明卡。
- Windows 10 / Linux 的玻璃只模糊应用自己画的内容；UI 文案写「窗口内玻璃」，不写「桌面磨砂」。透明预设不承诺材质，只承诺看穿。

## 4. 玻璃用在哪里

**适合**：

- 滚动内容上方的工具条（`glass.Glass` + `ScrollEdge` / 渐进 `Blur`）。
- 悬浮窗、命令面板、浮动工具条。
- 工作卡（液态玻璃预设）：卡下有 Ground 渐变或内容，玻璃才有东西可采样。
- 合并窗格（`glass.Group`，≤8 个 pane）。

**不适合**：

- 长文本、表格、日志、代码/ diff、设置表单。
- 透明根节点下没有任何已绘制内容的区域：着色器没有可采样的像素，看起来只会是空的。
- 已有系统材质的整窗大面积：优先用半透明 Card，玻璃留给浮层；两者叠加会互相抢戏。
- 每个控件单独加玻璃；一屏玻璃 pane ≤ 6，group ≤ 8。

## 5. 亚克力纪律（平台无关）

### 5.1 洗色阶梯

`透明 → WashHover → WashChip → WashSelected`，不换几何。

| 角色 | 静息 | 悬停 | 选中/按下 | 组件 |
| --- | --- | --- | --- | --- |
| 面板、栏、列表、工具栏 | 不铺底 | — | — | 卡片内的一切容器 |
| 条带 | `WashChip` | — | — | `HeaderStrip` / `FooterStrip` |
| 幽灵控件/行 | 透明 | `WashHover` | `WashSelected` | `IconButton` / `ToolButton` / `Row` |
| 实心控件 | `WashChip` + Hairline | `WashSelected` + HairlineStrong | 同悬停 | `Button` / `SearchField` / `Checkbox` |
| 芯片/徽章 | `WashChip` + Hairline | — | — | `Chip`（语义用 6px 圆点） |
| 主按钮 | `Accent` | `AccentHover` | `AccentPressed` | 一屏至多 1–2 颗 |

### 5.2 禁用清单（卡片/侧栏上）

| 禁用 | 原因 | 用 |
| --- | --- | --- |
| `ui.Checkbox` / `ui.Radio` | 未选中时铺实心 `t.Background` | `acrylic.Checkbox`；Radio 用 `ui.RadioBase` + 洗色 |
| `ui.Split` | 用 `t.Border` 画线，主题把 Border 设为透明 | `acrylic.Splitter` |
| `ui.Button` | 自带实心底与边 | `acrylic.Button` / `PrimaryButton` / `IconButton` |
| `ui.TextInput` / `ui.SearchField` | 自带不透明底与边框 | `acrylic.SearchField`；输入用 `ui.TextInputBase` + 洗色 |
| `ui.Segmented` | 自带轨道底 | `acrylic.SegmentedText` / `SegmentedIcons` |

例外：实色 `Code` 画布上的数据表/日志/编辑器可以用框架 `Table`/`List`，但要把
`t.SurfaceHover` 设为 `WashHover`；弹层/模态/设置块底是实色，框架控件可用。

## 6. 接线

```go
// 窗口与原生材质
acrylic.Window(&opts, dark, acrylic.NormalizeBlur(blurID)) // Transparent + 首帧底色 + Vibrancy
acrylic.SetAppearance(themeMode)                            // 原生材质跟应用明暗
c.SetTheme(acrylic.Theme(acrylic.For(dark)))                // 必须在建控件之前
acrylic.Ground(c, p)                                        // 有材质铺薄纱，否则实铺

// 工作卡：半透明色
acrylic.Card(c, p, opacity)
// 工作卡：玻璃（液态玻璃预设）—— 圆角/Clip/Raised 与 Card 相同，只换材质
ui.Column(c).Radius(acrylic.RadiusLG).Clip().
    Material(glass.Glass{Clarity: clarity, Tint: p.Card.Alpha(opacity)})

// 工具条浮在滚动内容上
ui.Row(c).Absolute().Top(12).Left(12).Right(12).Padding(6).Radius(20).
    Material(glass.Glass{Clarity: 0.5})
// 内容从下方穿过时的边缘
ui.Box(c).Absolute().Top(0).Left(0).Right(0).Height(74).PassThrough().
    Material(glass.ScrollEdge{})
```

- 运行中切换预设用 `win.SetVibrancy`；改不透明度/玻璃 Tint 只需重画。
- macOS 26 在 WebView 上可用 `glass.Glass{Native: true}` 采样页面；其他场景着色器只采样应用已绘制内容。
- `ReduceTransparency`：系统材质退实色、玻璃退不透明；不要再单独写一套降级。

## 7. 亮暗、强调色与辅助功能

- 明暗默认跟随系统；`acrylic.SetAppearance` 与 `Theme.SetSource` 同步，原生材质不会与界面反色。
- 强调色默认系统强调色；用于主按钮、选中、焦点圈。
- `ReduceTransparency` → 实色/无系统材质；`HighContrast` → 更强发丝线、更实的字色。
- `TextScale` 100–200%：行高、内边距随缩放；不截断。应用内缩放当前 100–125%。
- `ReduceMotion`：去掉位移/缩放，仅保留 120ms 颜色过渡。

## 8. 性能

- 系统材质开销在系统侧；`deep` 在低端机/远程桌面下降级。
- 玻璃 pane 每个都要读下方、模糊、着色；一屏 ≤6，滚动区域用一次工具条玻璃，不给每行加。
- `glass.Blur` 的渐进模糊按步进绘制，CPU 渲染成本更高；动态内容下优先 GPU 渲染。
- 不透明度只重绘；切换预设才碰原生材质。
- 不可见窗口不刷新材质；悬浮窗失焦降频。

## 9. 验证

- **guard 测试**：卡片/侧栏禁止不透明层色与禁用控件；玻璃材质算合法层，但控件仍只能用洗色。
- **洋红压力**：Ground 洋红 + 卡片不透明度 0%，检查是否有实心挡板。
- **采样边界**：Linux/Win10 上截屏确认玻璃只模糊应用内容；不出现「桌面透出」的文案。
- 深/浅色、100%/125%、减少透明度各一张截图，存 `docs/assets/appearance/`。

## 10. 验收矩阵

| 场景 | 期望 |
| --- | --- |
| macOS 12–15 液态玻璃 | NSVisualEffect；半透明卡正常；无实心挡板 |
| macOS 26 液态玻璃 | Liquid Glass；WebView 上 `Native` 生效 |
| Win11 22H2+ 液态玻璃 | Acrylic；半透明卡；明暗切换无闪白 |
| Win10 液态玻璃 | 无系统材质；半透明卡；UI 说明 |
| Linux 液态玻璃 | 无系统材质；半透明卡；无桌面模糊承诺 |
| 减少透明度 | 系统材质与玻璃都退不透明；文字可读 |
| 不透明度 0% / 100% | 0% 完全看穿；100% 实色；无抠穿 |
| 125% 缩放 | 卡片圆角、玻璃圆角、文字不截断 |

## 11. 非目标

- 不用玻璃伪装桌面磨砂（Linux/Win10）。
- 不做每控件独立透明度。
- 不给长文本/数据表铺玻璃。
- 不做自定义着色器或背景图（M3 后再议）。

