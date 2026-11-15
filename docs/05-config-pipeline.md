# 配置管线规格

> 本文是 `docs/03-architecture.md` 第 7 节的细化，可被黄金测试直接编码。术语见 `GLOSSARY.md`。

## 1. 不变量

1. 内核只读取 Runtime Config；任何设置、覆写、切换都必须经过本管线。
2. 一次切换要么完整生效，要么回到 Last Known Good；不存在「部分生效」。
3. 原始 Profile 字节不被就地修改；覆写是独立文件。
4. 应用设置对约定字段拥有最终决定权（见 §4）。
5. 生成过程可重放：相同输入必得相同输出（脚本覆写除外，需记录其副作用范围）。

## 2. 输入

| 输入 | 位置 | 可变性 |
| --- | --- | --- |
| Base | `profiles/<uid>.yaml` | 订阅更新或用户编辑时整体替换 |
| Merge | `overrides/<uid>.yaml` | 用户编辑 |
| Script | `profiles/<uid>.js` | 用户编辑（M2） |
| Settings | `settings.yaml` | 设置在界面修改 |
| Last Known Good | `runtime/last-good.yaml` | 仅成功应用后写入；secret 脱敏 |

### 2.1 Core HomeDir 与 provider 路径

- 内核 HomeDir 固定为 `<userData>/core`，在 `hub.Parse` 之前通过 `constant.SetHomeDir` 设置。
  `cache.db`、GeoIP/GeoSite、规则的本地缓存都在这里，且**不能**落到用户主目录。
- Base/Merge 中的 `proxy-providers.*.path` 与 `rule-providers.*.path` 在注入阶段重写为
  `providers/<profileUID>/<basename>`（相对 HomeDir）。绝对路径、`..`、盘符、UNC 一律拒绝。
- 由此：同一 provider 在不同 Profile 不互相覆盖；删除 Profile 时其 provider 目录可一并清理。
- 重写结果在 Runtime 预览中可见，不静默改用户文件。
- 没有可用 Profile 时，内核不启动；若需临时恢复，使用内置最小配置（`mixed-port` + `MATCH,DIRECT`），不写入 profiles。

## 3. 合并覆写语法

Merge 是一份 YAML 文档，对 Base 的映射做处理：

```yaml
# 1) 普通键：深合并，覆写方优先
dns:
  enhanced-mode: fake-ip
  nameserver:
    - https://1.1.1.1/dns-query

# 2) 列表：默认整体替换
rules:
  - MATCH,DIRECT

# 3) 列表前插/后插
prepend-rules:
  - DOMAIN-SUFFIX,corp.example,DIRECT
append-rules:
  - MATCH,PROXY

prepend-proxies: []
append-proxies: []

prepend-proxy-groups: []
append-proxy-groups: []

prepend-listeners: []
append-listeners: []

# 4) 删除：点路径，删除映射键或整个列表
delete:
  - dns.fallback
  - hosts
```

规则：

- 映射递归合并；数组与非数组冲突时覆写方胜。
- `prepend-*` / `append-*` 的目标不存在时创建；目标不是数组时报错。
- `delete` 的路径不存在时忽略，并在预览中提示。
- 未知键原样保留；预览中标记为「未识别」但不阻断。
- 合并结果必须仍是合法 YAML 映射。

## 4. 应用注入

注入发生在 Merge 与 Script 之后，优先级最高：

| 字段 | 来源 | 规则 |
| --- | --- | --- |
| `external-controller` | 设置 | `127.0.0.1:<port>`；端口占用时回退临时端口 |
| `secret` | 每次启动生成 | 用户固定端口并显式开启外部访问时才持久化 |
| `mixed-port` | 设置 | 默认 7890；与 `port`/`socks-port` 冲突时以 mixed 为准 |
| `mode` | 设置 / 托盘 | `rule` / `global` / `direct` |
| `log-level` | 设置 | 默认 `info` |
| `allow-lan` | 设置 | 默认 false；开启需二次确认 |
| `bind-address` | 设置 | 默认 `*`，仅 allow-lan 时生效 |
| `ipv6` | 设置 | 默认 false |
| `profile.store-selected` | 固定 | true |
| `profile.store-fake-ip` | 固定 | true |
| `tun.*` | TUN 页 | 仅在启用 TUN 时注入；`dns-hijack` 固定 `any:53` |
| `dns.*` | DNS 页 | 仅当用户保存过 DNS 设置；TUN 开启时强制 `enable: true` |
| `proxy-providers.*.path` / `rule-providers.*.path` | 注入 | 重写到 `providers/<profileUID>/`；拒绝绝对路径与 `..` |

未列出的字段一律不注入，保留 Profile 的写法。

作用域：TUN、DNS、嗅探、端口、`allow-lan`、`find-process-mode` 与高级内核项都是**应用级设置**，
对所有 Profile 生效。DNS 页与嗅探页必须在界面上明示「应用于所有配置」，并提供「恢复为配置值」清空注入；
需要单配置差异时，写进该 Profile 的 Merge 覆写。

## 5. 脚本覆写契约（M2）

```js
// profiles/<uid>.js
function main(config, env) {
  // config: 合并后的配置对象（可变）
  // env: { profileName, now, appVersion }，只读
  config.dns = config.dns || {}
  config.dns.enable = true
  return config
}
```

- 运行环境：goja；无 `require`、无文件/网络/进程/时间旅行；只暴露 `console.log` 到覆写日志。
- 超时 2s；抛错或超时视为本次生成失败，不写盘、不切换。
- 返回值必须是对象；返回 `undefined` 时使用原地修改后的 `config`。
- 脚本只能改配置，不能改应用设置；注入阶段仍会覆盖 §4 字段。
- 脚本内容不做持久化执行缓存；每次生成都重新运行。

## 6. 校验与错误定位

1. YAML 解析：使用 `yaml.v3`，保留行号；语法错误映射为 `KindConfigInvalid`，带行列。
2. 语义校验：调用 mihomo 的 `executor.ParseWithBytes`（不启动内核）。
3. 端口校验：监听测试（绑定后立即释放），返回 `KindPortInUse` 与建议端口。
4. 错误对象包含：阶段（base/merge/script/inject/parse）、行号（可得时）、原始错误、用户提示。

## 7. 应用状态机

落盘规则：`runtime/candidate.yaml` 与 `runtime/last-good.yaml` 中的 `secret`、订阅 Authorization 一律替换为
`<redacted>`；真实 secret 只存在内存与内核进程中，重启时重新生成。

```
Idle ──Build──► Built ──Validate──► Valid ──Reload──► Applied
                  │                   │                 │
                  └─error─────────────┴─error───────────┘
                                        │
                                        ▼
                                  Reverted（last-good 继续运行）
```

- `Build`：读三份输入 + 设置，生成 `runtime/candidate.yaml`。
- `Validate`：解析 + 端口检查。
- `Reload`：`Host.Reload(candidate)`；成功后 `last-good := candidate`，更新 `state.selected_profile`。
- 失败：若 Host 仍运行，保持旧配置；若 Host 已停，用 `last-good` 重启。
- 连续失败 2 次：停止自动重试，概览显示错误条与「查看诊断」。

## 8. 预览与 diff

- Base：原始 Profile。
- Merged：Merge 之后、Script 之前（M1 可看）。
- Runtime：全部阶段之后（只读）。
- 默认展示 Base ↔ Runtime 的 diff；可切换 Merged ↔ Runtime。
- diff 按行，忽略键顺序；大文件只渲染可视区。

## 9. 黄金测试清单

| 用例 | 断言 |
| --- | --- |
| 纯 Base | 输出等于 Base + 注入字段 |
| 深合并 | 嵌套映射保留未覆盖键 |
| 列表替换 / 前插 / 后插 | 顺序与数量正确 |
| `delete` 存在 / 不存在 | 删除成功 / 忽略并提示 |
| 目标非数组的前插 | 报错并指向行号 |
| 注入覆盖 Profile 的端口与 mode | 注入值胜出 |
| TUN 开启 | `dns.enable=true` 且含 `dns-hijack` |
| 脚本抛错 / 超时 | 不写盘、不切换、错误可读 |
| 校验失败 | last-good 不变，内核继续运行 |
| 连续两次失败 | 不再自动重试 |
| provider 路径 `../../etc/x` | 拒绝并报告路径非法 |
| 落盘 runtime 文件 | 搜索不到真实 secret |

## 10. 保存、订阅与回退语义

- **订阅同步**：下载 → 解析/校验 → 才替换 Base；任一步失败保留旧 Base，并把错误显示在对应 Profile 上。
- **编辑当前启用 Profile**：保存时先写草稿并校验；校验通过则提示「已保存，是否立即应用」，
  M1 采用「自动应用 + 失败回退」；失败时保留磁盘草稿但运行配置不变。
- **编辑非当前 Profile**：只校验与保存，不影响运行中的内核。
- **删除当前 Profile**：确认后停核、清理 provider 目录、回到空状态；不自动切到别的 Profile。
- **删除订阅来源**：只解除关联，已同步的 Profile 保留并转为 local；需二次确认。
- **节点选择**：由 mihomo 的 `cache.db`（`profile.store-selected`）持久化；不同 Profile 的同名组可能沿用旧选择，
  目标节点不存在时由内核回退到组策略。切换 Profile 后 UI 以 `/proxies` 的真实返回为准。
- **无可用配置**：内核不启动；概览与配置页只显示导入引导。

## 11. 非目标

- 不做任意 Go 插件或 Lua 脚本。
- 不支持按字段来源追踪到具体覆写行的可视化编辑（只在预览中标注未识别键）。
- 不自动修复用户配置；只报告并给出最小建议。
