# myclash

三端原生桌面代理客户端：Go + [MyGo](https://github.com/egoist/mygo) 原生 UI + [mihomo](https://github.com/MetaCubeX/mihomo) 内核。
界面与功能对齐 Clash Verge Rev 与 Clash Party 的并集；系统材质、窗口内玻璃与亚克力纪律三者并存。

**状态：文档已收口，准备实现。** 开工前清单见 `docs/12-doc-audit.md`；本文不罗列未实现的特性。

## 支持

| 平台 | 基线 |
| --- | --- |
| Windows | 10 1809+，x64 / arm64 |
| macOS | 12+，universal（arm64 + x64） |
| Linux | glibc 2.28+、GTK 3、systemd/polkit，x64 / arm64 |

三端同等支持：同一功能在三个平台都要成立；无法等价的平台专属项在界面标注并在其他平台隐藏。

## 设计要点

- **默认单进程**：把 mihomo 嵌入应用进程（`hub.Parse`），界面用 loopback REST 读状态、选节点；配置切换与转发模式走进程内调用（见 ADR-0007）。TUN 时切换到提权内核；轻量模式按需增加 Agent 进程（ADR-0008/0012）。
- **配置三分**：Subscription（来源）→ Profile（可运行单元）→ Runtime Config（内核唯一读取的生成物）。见 ADR-0003。
- **原生界面**：MyGo 控件构建；系统材质（macOS/Win11）与 `plugins/glass` 窗口内玻璃是两个层次，
  亚克力纪律约束透明度与控件；预设为液态玻璃 / 透明 / 实色；TUN 默认 mips 栈。见 ADR-0011。
- **GPL-3.0**：嵌入 mihomo 的必然结果；三端安装包与源码同批发布。见 ADR-0005。

## 文档

| 文档 | 内容 |
| --- | --- |
| [GLOSSARY.md](./GLOSSARY.md) | 项目术语的唯一含义 |
| [docs/00-research.md](./docs/00-research.md) | 参考产品、mihomo API、MyGo 能力与风险 |
| [docs/01-prd.md](./docs/01-prd.md) | 功能全集、平台矩阵、非功能要求、验收 |
| [docs/02-ui.md](./docs/02-ui.md) | 信息架构、亚克力/玻璃规范、控件对账、逐页规格 |
| [docs/03-architecture.md](./docs/03-architecture.md) | 进程模型、包结构、配置管线、三端系统集成 |
| [docs/04-roadmap.md](./docs/04-roadmap.md) | 里程碑、依赖、风险闸门、完成定义 |
| [docs/05-config-pipeline.md](./docs/05-config-pipeline.md) | 覆写语法、注入优先级、脚本契约、回滚与测试 |
| [docs/06-parity.md](./docs/06-parity.md) | 与 Verge / Party 的功能对齐矩阵 |
| [docs/07-security.md](./docs/07-security.md) | 威胁模型、安全默认值与验收 |
| [docs/08-quality.md](./docs/08-quality.md) | 内存、性能、体验准则与评审清单 |
| [docs/09-appearance.md](./docs/09-appearance.md) | 系统材质、窗口内玻璃、明暗与辅助功能 |
| [docs/10-ui-libraries.md](./docs/10-ui-libraries.md) | 组件库与图标评估、采纳规则 |
| [docs/11-references.md](./docs/11-references.md) | 官方文档、插件、mihomo、平台与设计参考 |
| [docs/12-doc-audit.md](./docs/12-doc-audit.md) | 文档审计、一致性检查、动工前清单 |
| [docs/13-code-style.md](./docs/13-code-style.md) | 包结构、依赖方向、代码/UI/测试规范 |
| [docs/adr/](./docs/adr/) | 难以回退的决策记录 |

## 开发

要求 Go 1.27+。

```sh
go tool mygo dev                   # 开发运行
mygo dev                          # 若已安装 MyGo CLI
go test ./...
go tool mygo build \
  -platform darwin/universal,windows/amd64,windows/arm64,linux/amd64,linux/arm64
```

## 许可

GPL-3.0，见 [LICENSE](./LICENSE)；第三方声明见 [NOTICE](./NOTICE)。Copyright (C) 2026 myclash contributors。
第三方许可见应用内「关于 → 许可」。
