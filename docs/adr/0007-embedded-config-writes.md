# 内核宿主：embed 模式下配置写入走内部 API

Status: accepted

## Context

最初的架构约定「界面只通过 REST 访问内核」。但 mihomo 的 `route.SetEmbedMode(true)` 会关闭
`PUT/PATCH /configs`、`/restart` 与 `/upgrade`：这正是嵌入方需要的安全边界——否则本地 Web 面板或同用户进程
可以触发 `/restart`（`exec` 应用自身）或 `/upgrade`（用 mihomo 二进制覆盖应用）。若保持 embed 模式关闭，
这些危险路由就暴露给持有 secret 的任何本地面板。

## Decision

1. 每个宿主进程启动时调用 `route.SetEmbedMode(true)`。
2. 配置重载与转发模式不走 REST，改走 mihomo 的导出函数：
   - 重载：`cfg, err := executor.ParseWithBytes(payload)` → `executor.ApplyConfig(cfg, false)`；
   - 模式：`tunnel.SetMode(mode)`；
   - 日志级别：`log.SetLevel(level)`；
   - 面板路径：`route.SetUIPath(dir)`；
   - Geo：`updater` 包的导出函数。
3. REST 只承担读与选择：代理、规则、连接、日志、提供者、缓存、存储。
4. Privileged Host 通过父进程管道接收同样语义的命令（reload/mode/geo/stop），由子进程内部执行。

## Considered Options

- 保持 embed 模式关闭，继续用 `PUT /configs`：配置写入统一，但把 `/restart`、`/upgrade` 暴露给任何持有 secret 的本地页面。
- 给 mihomo 打补丁，只关闭 restart/upgrade：维护 fork，不可接受。
- 子进程跑独立的 mihomo 二进制：需要额外分发内核，失去同进程的内存收益。

## Consequences

- 「只用 REST」不再是绝对约束；允许的内部调用被限定为上面列出的几个函数，集中在一个 `embedded.go` 里，升级时按 §18 的清单检查。
- 配置写入与模式切换不再依赖 controller 端口，端口冲突时更稳。
- 外部 Web 面板无法重启或替换应用，也不能改配置；它只能选择节点与查看状态。
- 需要在升级内核时验证这些导出函数的签名与语义，编译期即可发现大部分变化。
