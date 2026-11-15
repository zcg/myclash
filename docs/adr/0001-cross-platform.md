# 三平台同等支持

Status: accepted

## Context

需求明确 Windows、macOS、Linux 同等支持，不允许某一端缺功能。MyGo 一条命令即可构建
`darwin/universal,windows/amd64,windows/arm64,linux/amd64,linux/arm64`；mihomo 是纯 Go，可交叉编译。
三端真正的差异集中在：系统代理、TUN 授权、开机启动、托盘、打包与签名。

## Decision

支持矩阵：

| 平台 | 架构 | 基线 |
| --- | --- | --- |
| Windows | x64 / arm64 | Windows 10 1809+ |
| macOS | universal（arm64 + x64） | macOS 12+ |
| Linux | x64 / arm64 | glibc 2.28+、GTK 3、systemd + polkit；Web 面板需 WebKitGTK 4.1 |

- 领域层不得出现平台分支；所有系统能力经 `internal/platform` 接口，按 `_windows.go`、`_darwin.go`、`_linux.go` 提供实现。
- 一项功能默认三端都做。确实无法等价的（如 Windows 的 UWP 回环豁免）在界面标注平台，并在其他平台**隐藏**，不显示死按钮。
- CI 使用三个 runner 各自构建与测试；发版资产：Windows `setup.exe` + portable `zip`，macOS `.dmg`（签名+公证），Linux `tar.gz` + `deb` + `rpm` + `PKGBUILD`。
- 每个里程碑的出口包含三端安装与核心路径验收。

## Consequences

- 系统代理、提权助手、DNS 恢复各写三份实现，测试面显著扩大；不接受任何「先 Windows 后补」的临时分支。
- macOS 签名/公证与 Windows 安装器需要对应宿主机工具链，发布流程按平台拆分。
- 平台能力差异必须写进 UI 文档与 PRD 的平台矩阵，而不是留给用户发现。
