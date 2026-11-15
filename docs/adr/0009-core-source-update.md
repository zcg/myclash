# 内核来源与独立更新

Status: superseded by ADR-0012（内核只做 bundled，外部内核推迟）

## Context

mihomo 默认编译进应用，内存最低，但 Go 无法在进程内热替换内核代码；需求又明确要求「应用内可更新内核」。
直接把 mihomo 作为独立二进制运行可以独立更新，但多一个进程、需要新的生命周期管理。两种需求都必须满足。

## Decision

引入 `core_source`：

- **bundled（默认）**：内嵌 mihomo。支持重启（`executor.Shutdown` + 重新 `hub.Parse`），
  但版本随应用发布；「检查内核更新」只说明需要更新应用。UI 不得假装可以单独升级。
- **external**：下载官方 mihomo 二进制，独立版本、独立更新：
  1. 从 GitHub Releases 选择匹配 `GOOS/GOARCH` 的 `.gz` 资产（amd64 优先 `compatible` 变体）；
  2. 用 Release API 的 `digest`（sha256）校验，解压到 `<userData>/core/versions/<version>/`；
  3. 用 `mihomo -t -f -` 以 stdin 喂配置做校验；
  4. 切换：停旧进程 → 启动新版本 → 等就绪 → 失败则回滚到上一版本；
  5. 保留当前与上一版本，其余可清理。

external 的运行方式：

```sh
mihomo -d <userData>/core -f -        # 配置与 secret 从 stdin 写入，不进 argv/临时文件
```

- 外部内核是独立进程，由 GUI 或 Agent 监督；日志落 `<logs>/core.log`。
- 外部内核不支持 embed 模式，因此 `PUT /configs`、`PATCH /configs` 可用；`/restart` 因 stdin 已关闭而不可用，
  由监督者重启进程。`/upgrade` 不由应用调用。
- TUN 在 external 下需要提权监督进程（M3）：监督者以管理员/root 启动，再 exec 外部二进制并转发配置。
  M2 的 external 只承诺系统代理；UI 在 external + TUN 时明确提示「M3 支持」。
- 切换来源会重启内核并短暂断流；TUN 开启时先停 TUN、恢复 DNS，再切换。

## Considered Options

- 只用 bundled：无法独立更新内核。
- 只用 external：日常多一个进程，内存更高，且默认体验更复杂。
- Go plugin / 动态库加载：平台受限、Go 插件不支持卸载，不可行。

## Consequences

- `core.Host` 增加 external 实现；监督、日志、重启、回滚都要测试。
- 外部二进制的校验只有 HTTPS + GitHub digest，没有签名；在安全文档中记录为残余风险。
- 外部内核版本的配置字段可能变化，切换前必须用目标版本自身 `-t` 校验。
- 默认仍是 bundled；「内核更新」是给需要独立版本的用户的可选项，不损害默认低内存。
- TUN + external 的完整支持排到 M3；M2 的 UI 必须显示限制，而不是静默失效。
