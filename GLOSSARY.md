# myclash

桌面代理客户端。用户在这里管理订阅、配置、内核与系统代理。本文件只定义词，不描述实现。

## Language

### 配置

**Subscription（订阅）**:
一个远程来源，返回 mihomo 配置文本。它本身不可运行，同步后产生 Profile。
_Avoid_: 机场、订阅配置、远程配置

**Profile（配置）**:
一个有 uid、可被选为当前运行项的 mihomo 配置工件，来自订阅同步、本地导入或空白创建。
_Avoid_: 订阅、配置文件、节点

**Runtime Config（运行时配置）**:
由 Profile 经覆写链生成、并补入应用设置的最终 YAML；内核只读取它。
_Avoid_: 生效配置、最终订阅

**Profile Chain（覆写链）**:
Base Profile → Merge Override → Script Override → 应用注入 的固定生成顺序。
_Avoid_: 覆写、合并链

**Merge Override（合并覆写）**:
用 YAML 文档对 Profile 做深合并、列表前插/后插与删除的声明式覆写。
_Avoid_: 合并配置、补丁

**Script Override（脚本覆写）**:
用 JavaScript 函数改写配置对象的命令式覆写（里程碑 M2）。
_Avoid_: 脚本、JS 配置

**Last Known Good（最近可用配置）**:
最近一次通过校验并成功加载的 Runtime Config；切换失败时回到它。
_Avoid_: 备份、回滚点

**Subscription Sync（订阅同步）**:
从 Subscription 拉取最新配置并替换 Profile 原始字节的动作。
_Avoid_: 更新、订阅更新

**App Update（应用更新）**:
替换应用自身版本的动作。
_Avoid_: 更新

### 内核

**Core（内核）**:
mihomo 实例，负责代理、规则、DNS 与 TUN 的实际转发；编译进应用，随应用更新（ADR-0012）。
_Avoid_: 核心、代理引擎

**Agent（后台代理进程）**:
轻量模式下的无窗口进程，拥有内核、托盘、系统代理/TUN 与单实例锁。
_Avoid_: 服务、守护进程（易与提权助手混淆）

**Run Mode（运行模式）**:
standard（GUI 内嵌内核，单进程）或 lightweight（Agent 持有内核，GUI 按需附着）。
_Avoid_: 内核模式、代理模式

**Core Host（内核宿主）**:
承载 Core 的进程形态：Embedded Host（与界面同进程）或 Privileged Host（提权后的独立进程）。
_Avoid_: 内核模式、服务

**Privileged Helper（提权助手）**:
持久安装、随开机启动的提权宿主（Windows 服务 / macOS LaunchDaemon / Linux systemd），区别于一次性的 Privileged Host。
_Avoid_: 服务、守护进程

**Controller（外部控制）**:
Core 暴露的本地 REST 接口，界面通过它读状态、改选择、切配置。
_Avoid_: API、管理端口

**Forwarding Mode（转发模式）**:
Core 的转发策略：rule、global、direct。
_Avoid_: 模式、代理模式、规则模式（仅指 rule 时）

**Proxy Group（代理组）**:
Core 内的一组节点与选择策略；只有 select 组接受手动选择。
_Avoid_: 节点组、分组

**Node（节点）**:
一条出站代理定义，属于某个组或提供者。
_Avoid_: 线路、服务器

**Provider（提供者）**:
Core 按 URL 或文件加载的节点集合（proxy provider）或规则集合（rule provider）。
_Avoid_: 资源、订阅源

**Provider File（提供者文件）**:
提供者在磁盘上的缓存文件，位于 Core 的 HomeDir 下；规则内容预览读它，不读 REST。
_Avoid_: 缓存、配置文件

**Rule（规则）**:
决定一条连接交给哪个策略的匹配项。
_Avoid_: 规则项、路由

**Connection（连接）**:
Core 中的一条活动会话，有目标、进程、命中的规则与代理链。
_Avoid_: 会话、链路

**Delay（延迟）**:
对节点或组发起一次 HTTP 探测得到的往返毫秒数。
_Avoid_: 测速（与网速测试混用）、ping

**Speed Test（网速测试）**:
通过本机代理测量一段时间内的吞吐（Mbps），可带 TTFB/抖动；与延迟测试是两件事。
_Avoid_: 测速（单独使用时）、带宽测试

### 网络

**System Proxy（系统代理）**:
修改当前用户的系统代理设置（Windows WinINet、macOS 网络服务、Linux GNOME/KDE），使系统与多数应用走 Core；不接管不遵守系统代理的流量。
_Avoid_: 全局代理、代理开关

**System Proxy Method（系统代理方式）**:
System Proxy 的两种工作方式：manual（固定代理地址）与 PAC。
_Avoid_: 模式、代理模式

**PAC（自动代理脚本）**:
由本机 HTTP 服务提供、按 URL 返回代理决策的脚本；System Proxy 的一种模式。
_Avoid_: 自动配置、脚本代理

**Bypass List（绕过列表）**:
不受 System Proxy 影响的 host / IP 模式列表，语法随平台。
_Avoid_: 例外、白名单

**TUN**:
通过虚拟网卡接管全局流量的模式，需要管理员（Windows）或 root/CAP_NET_ADMIN（macOS/Linux）权限。
_Avoid_: 虚拟网卡模式、全局模式

**Subscription Userinfo（订阅信息）**:
订阅响应头携带的已用流量、总量与到期时间。
_Avoid_: 流量、配额

**Traffic Sample（流量采样）**:
按固定间隔记录的上下行累计值差值，用于历史曲线。
_Avoid_: 流量统计、速率

**Unlock Test（解锁测试）**:
经当前代理访问特定服务，判断地区可用性的诊断。
_Avoid_: 解锁检测、流媒体测试

### 界面

**Preset（预设）**:
一套外观组合：混合、原生、液态玻璃、实色。
_Avoid_: 主题、皮肤

**Material（材质）**:
窗口级系统材质（Mica/Acrylic/NSVisualEffect/Liquid Glass）与窗口内玻璃（`plugins/glass`）的统称；
两者透的东西不同，不可互相替代。
_Avoid_: 主题、透明度

**Native Material（系统材质）**:
由操作系统绘制在窗口背后的磨砂/色调，能透出桌面与其他窗口；macOS 全系、Windows 11 22H2+ 有。
_Avoid_: 玻璃、背景模糊

**In-window Glass（窗口内玻璃）**:
由 `plugins/glass` 绘制、只模糊窗口内已绘制内容的材质；三端一致，但不透桌面。
_Avoid_: 桌面磨砂、系统材质

**Acrylic Discipline（亚克力纪律）**:
Ground/Card/浮层分层、工作卡单一不透明度、控件只用洗色、禁止实心层色的规则集。
_Avoid_: 亚克力材质（它不是一种材质）

**Web Panel（面板）**:
由 Core 托管、在独立窗口打开的外部 Web 管理界面。
_Avoid_: 仪表盘、Web UI

**Floating Window（悬浮窗）**:
无边框、置顶、只显示实时速率的独立小窗。
_Avoid_: 小窗、悬浮球
