# GPL-3.0 与分发形态

Status: accepted

## Context

同进程嵌入 mihomo（GPL-3.0）使整个应用成为衍生作品；参考产品 Verge 与 Party 也都是 GPL-3.0。
三端各有惯用的安装形态。

## Decision

项目以 GPL-3.0 发布，源码公开。使用 `mygo build -platform darwin/universal,windows/amd64,windows/arm64,linux/amd64,linux/arm64` 产出：

| 平台 | 资产 | 备注 |
| --- | --- | --- |
| Windows | `setup.exe`（NSIS）、portable `zip` | 含 `wintun.dll`；安装器可按需注册开机启动 |
| macOS | `.app` 装入 `.dmg` | 在 macOS 上签名与公证；提权助手为 LaunchDaemon |
| Linux | `tar.gz`（含 `install.sh`）、`.deb`、`.rpm`、`PKGBUILD` | 依赖 GTK 3；Web 面板需 WebKitGTK 4.1 |

更新走 GitHub Releases，MyGo updater 插件下载并以 `mygo keygen` 生成的密钥对验签；同版本保留上一份安装包用于回滚。
代码签名与 macOS 公证在首次公开发布前接入。

- 内核随应用更新（ADR-0012）；「检查内核更新」只引导更新应用，不下载外部二进制。

Linux 分两种：以 `tar.gz` 安装的允许应用内替换；以 `deb`/`rpm`/`PKGBUILD` 安装的检测到包管理来源后**禁用**应用内替换，
只提示用包管理器更新，避免绕过包数据库。

## Consequences

- 不提供闭源分发或应用商店独占版本。
- 界面内需要「关于 → 许可」页，列出 mihomo、MyGo 与其他依赖的许可。
- 三端发布资产与校验和必须同批产出；任一平台缺包视为发布失败。
- 更新前停止系统代理/TUN 与内核；轻量模式先停 Agent；更新后按原 `run_mode` 重启并恢复上次 Profile。
- 若设置了 `updates.tagPrefix`，Git tag 必须与之一致；未设置时任何 `vX.Y.Z` 发布都可能被当作更新源。
- 更新检查频率限制为每天一次（手动检查除外），避免 GitHub API 限额。
