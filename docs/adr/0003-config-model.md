# 配置模型：Subscription → Profile → Runtime Config

Status: accepted

## Context

用户口语里的「订阅」「配置」混用：订阅是 URL，配置是文件，而内核实际读的是生成后的 YAML。若三者同名，切换、回滚、覆写与分享都会含糊。

## Decision

三个独立概念，固定单向关系：

1. `Subscription` 只描述来源（URL、更新间隔、请求头、订阅信息）。
2. `Profile` 是可运行单元，有 uid、来源类型、原始字节与覆写链。
3. `Runtime Config` 由 Profile Chain 生成，是内核唯一读取的配置，不可直接编辑。

界面只用一个「配置」页面呈现 Profile，用来源筛选与徽标表达它是否来自订阅。订阅的新版本同步为 Profile 的新原始字节，而不是新建 Profile。

## Consequences

- 导入任意 YAML 都先成为一个 Profile；不存在「裸配置直接跑」的旁路。
- 分享与导出以 Profile 为单位；Runtime Config 仅供诊断与预览。
- 订阅删除不自动删除其 Profile，反之亦然；两者生命周期分开，需在界面明示。
