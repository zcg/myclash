# 运行模式：标准单进程 / 轻量 Agent

Status: accepted

## Context

需求是「关闭主窗口后释放界面内存，只保留内核与托盘；点托盘可重新拉起界面」。
内嵌内核活在同一进程里，GUI 退出内核就死；只隐藏窗口则不释放 Skia/UI 的内存。要真正释放，
内核必须活在另一个没有 UI 的进程里。MyGo 支持无窗口运行、托盘与单实例，具备实现条件。

## Decision

引入 `run_mode` 设置。**两种模式都实现，但运行时互斥**：同一时刻只运行一种，不能同时存在两个内核/两个托盘。
内核只做 bundled（ADR-0012）。

- **standard（默认）**：GUI 进程内嵌内核，同时拥有托盘与单实例锁。一个进程，内存最低。
- **lightweight**：`myclash --agent` 作为后台进程，拥有内嵌的 bundled 内核、托盘、系统代理/TUN、全局快捷键与单实例锁；
  GUI 作为前端连接 Agent，关闭窗口即退出 GUI，Agent 继续代理。

规则：

1. 切换模式需要停止内核并按新模式重启应用；接受数秒断流，不做热交接。TUN 开启时先停 TUN、恢复 DNS，再切换。
2. Agent 与 GUI 通过本地控制通道通信：Windows 命名管道、Unix domain socket；路径在 `<userData>`，
   权限 0600，带每次启动生成的 token。
3. 控制协议（JSON Lines）：`status`、`reload`、`set-mode`、`restart-core`、
   `set-sysproxy`、`set-tun`、`speedtest`、`open-window`、`quit`。
4. 单实例：同一时刻只有 Agent 或 GUI 之一持有锁与托盘。启动 `myclash` 时先探测 Agent，
   存在则附着为前端；不存在则按模式启动。
5. 状态归属：**内核所有者写 `state.yaml` 与流量采样**；GUI 只写 `settings.yaml` 与 Profile 文件，
   运行期操作经控制通道发给 Agent。避免两个进程同时写运行状态。
6. 轻量模式下窗口打开时，状态条显示「内核在后台」；不重复创建托盘。
7. Agent 崩溃：内核随之中断；GUI 若在运行则提示并可重启 Agent；无 GUI 时下次启动修复。
   v1 不做自动守护进程，M3 再评估。

## Considered Options

- 关闭窗口只隐藏（同一进程）：实现最简单，但不释放 UI 内存，不满足需求。
- 始终双进程（GUI + Agent）：一致，但日常也常驻两套运行时，与低内存目标冲突。
- 把内核做成系统服务：权限与安装成本更高，且标准模式不需要。

## Consequences

| 模式 | 窗口打开 | 窗口关闭 | 说明 |
| --- | --- | --- | --- |
| standard | 1 进程 | 1 进程（托盘）或退出 | 内存最低；关闭后若进程保留，UI 内存不释放 |
| lightweight | 2 进程（GUI + Agent） | 1 进程（Agent） | 释放 UI 内存；窗口打开时反而更高 |

- 轻量模式是 opt-in；设置页必须写清两种模式的进程数与代价。
- 应用更新前停止 Agent/内核与系统代理；更新后按原 `run_mode` 重启并恢复上次 Profile。
- 模式切换会重启内核；再次打开 GUI 附着 Agent，不重启内核。
- 托盘交互在两种模式下必须一致；Agent 负责拉起 GUI。
- 轻量模式不依赖 TUN；TUN 时 Agent 仍按 ADR-0002 启动提权内核。
- 需要一套本地控制协议、状态归属规则与「附着/脱离」流程，测试面增加。
- Linux 无托盘（无 AppIndicator）时，Agent 仍可运行；用户通过再次启动应用打开窗口。
