# 数据目录与密钥存放

Status: accepted

## Context

配置、日志、缓存、流量历史与密钥混在一起会让备份、便携模式与故障诊断都变难；三端对「应用数据放哪」
有各自的惯例，自造路径会与系统备份、清理工具和用户预期冲突。

## Decision

使用 MyGo 的路径服务（`mygo.App.Path`），不硬编码平台路径：

| 用途 | 调用 | Windows | macOS | Linux |
| --- | --- | --- | --- | --- |
| 配置与状态 | `PathUserData` | `%AppData%\myclash` | `~/Library/Application Support/myclash` | `$XDG_CONFIG_HOME/myclash` |
| 缓存 | `PathCache` | `%LocalAppData%\myclash` | `~/Library/Caches/myclash` | `$XDG_CACHE_HOME/myclash` |
| 日志 | `PathLogs` | `%AppData%\myclash\logs` | `~/Library/Logs/myclash` | `$XDG_CONFIG_HOME/myclash/logs` |
| 随包资源 | `PathResources` | 安装目录 | `.app/Contents/Resources` | 安装目录 |

上表是 MyGo v0.9.0 的实际返回值，不在应用层另造路径。Linux 的日志落在配置目录是 MyGo 的既有行为；若后续要改到
`$XDG_STATE_HOME`，用 `SetPath` 覆盖并同步本文，不散落在调用点。

若可执行文件同级存在 `portable` 标记文件，则用 `SetPath` 把上述三类目录全部改到 `<exe>/data` 下。

用户数据目录布局：

```
<userData>/
  settings.yaml          应用设置
  state.yaml             上次使用的 Profile、退出时的开关状态
  profiles.yaml          Profile 索引
  profiles/<uid>.yaml    原始配置
  profiles/<uid>.js      脚本覆写（可选）
  overrides/<uid>.yaml   合并覆写（可选）
  runtime/
    candidate.yaml       本次生成的运行时配置
    last-good.yaml       最近可用配置
  traffic/YYYY-MM.jsonl  流量采样
  backup/                本地备份 zip
  agent.sock             Agent 控制通道（Unix；Windows 用命名管道）
<cache>/
  geo/                   下载的 Geo 数据
  panel/                 Web 面板静态资源
<logs>/
  app-YYYYMMDD.log
  core.log
```

密钥（订阅 Authorization、WebDAV 凭据）存系统凭据库：Windows Credential Manager、macOS Keychain、
Linux Secret Service（经 MyGo `Secrets`）。Controller secret 每次启动随机生成，仅在内存与内核参数中出现；
用户显式固定端口并对外暴露时才持久化。

## Consequences

- 备份 = 复制 `profiles*`、`overrides/`、`settings.yaml`、`state.yaml`，不含缓存与日志。
- Linux 无 Secret Service 时（无桌面会话/无 keyring）降级为权限 0600 的本地文件，并在界面提示。
- 目录结构带 `schema-version`，升级时迁移，不静默改写用户原始 Profile。
- 便携模式下所有路径由根目录推导，不写注册表/LaunchAgents/systemd。
