# Core Host：默认同进程，TUN 时提权独立进程

Status: accepted

## Context

mihomo 可以用 `hub.Parse` 嵌入同一进程，也可以作为独立二进制运行。TUN 需要提升权限创建虚拟网卡与改路由，
而界面不应整体提权。三端提权机制不同，但「界面普通权限 + 内核提权」的结构一致。

## Decision

界面永远以普通权限运行。内核通过 `core.Host` 接口访问，两种实现：

- **Embedded Host**（默认，也是唯一内核来源）：同进程 `hub.Parse`，REST 只绑 `127.0.0.1`，secret 每次启动随机生成。
- **Privileged Host**（TUN）：同一个可执行文件以 `--core --parent <pid>` 提权启动，
  通过 stdin 接收运行时配置与一次性 token。

| 平台 | 提权方式（M2） | 虚拟网卡 | 后续形态 |
| --- | --- | --- | --- |
| Windows | `ShellExecuteEx` + `runas`（UAC） | wintun.dll | 可选 `myclash-core` Windows 服务（SYSTEM） |
| macOS | 授权对话框启动一次提权进程 | utun | LaunchDaemon 助手（`SMAppService`，macOS 13+） |
| Linux | `pkexec` 请求 polkit 授权 | tun（CAP_NET_ADMIN） | systemd system service + polkit 规则 |

切换 Host 会重启内核，接受数秒断流。

## Considered Options

- 永远双进程 sidecar（Verge 式）：内核可热替换，但常驻两套 Go 运行时，与内存目标冲突，且 IPC/生命周期复杂。
- 永远同进程、放弃 TUN：缺刚需功能。
- 各平台使用不同的内核形态：破坏「一套 REST 契约」，否决。

## Consequences

- 各 Host、三个平台共用一套 REST 读契约与控制语义；配置写入走内部 API（ADR-0007）。
- 内核随应用发布（ADR-0012）；不支持单独更新内核或多内核并存。外部内核方案已评估并推迟。
- TUN 关闭时回到 Embedded，进程数降回一。
- 提权进程必须校验父进程 PID + 可执行文件路径 + 一次性 token，避免成为任意配置提权执行器；
  配置与命令只走 stdin，不写临时文件、不进 argv。
- Linux 的持久化助手需要 polkit 规则；macOS 需要签名后的 LaunchDaemon。两者都排到 M2 末/M3。
