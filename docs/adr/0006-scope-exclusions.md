# 明确不做的范围

Status: accepted

## Context

Clash Party 带有一些与代理本身无关、但会显著增加体积与维护面的能力。若不在文档里划掉，它们会以「对齐参考产品」的名义不断被重新提出。

## Decision

以下不做：

- **内置 Sub-Store 后端**：需要 Node 运行时。只支持把外部 Sub-Store 的输出当作普通 Subscription URL。
- **MCP server**：面向 AI 客户端，不是代理工具的职责。
- **Smart Core / 并存的多内核 / 外部内核**：不做多内核同时存在与一键切换，也不做独立 mihomo 二进制
  （ADR-0012）；内核只随应用更新。
- **Tailscale 专属认证流程**：mihomo 的 tailscale 出站仍可用，但不做引导式登录。
- **移动端、浏览器扩展、路由器固件**。

以下做，但不进 M1：

- 流量历史统计与悬浮窗（M3）、WebDAV 同步（M3）、Web 面板（M2）、解锁测试（M3）。

## Consequences

- PRD 的「非目标」与本节一致；新增此类需求需要新 ADR。
- 与 Party 的功能对照表会显示这些缺口，这是有意为之。
