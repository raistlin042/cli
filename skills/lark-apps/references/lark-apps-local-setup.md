# apps 本地开发 setup（规划中 · 命令未上线 · 暂不执行）

> ⚠️ **本文件为设计预览。下列命令（`+git-credential-init` 等）尚未上线，当前请勿执行。** 用户要求"本地开发 / clone 到本地"时，告知该能力规划中、暂未上线。

> **前置条件：** 命令上线后，先阅读 [`../../lark-shared/SKILL.md`](../../lark-shared/SKILL.md)（认证、全局参数）。

进入本地开发只需一次性 setup，两种进入场景最终汇到同一段（git 凭据初始化 → clone → cd）：

## 场景 A：从 0 创建本地开发应用

用户要新建一个全栈 / 需本地改代码的应用：

| 步骤 | 命令（未上线） | 说明 |
|------|---------------|------|
| 1 | `apps +create --app-type fullstack --message "<用户原话>"` | 拿 `app_id`（此命令已上线，见 [lark-apps-create.md](lark-apps-create.md)） |
| 2 | `apps +git-credential-init --app-id <id>` | 初始化 git 凭据 + 注入本地 git 配置；响应返回仓库 repo 地址（命令未上线，repo 地址来源暂定） |
| 3 | `git clone <repo>` → 引导用户 `cd` 进仓库目录 | clone 后进入仓库，换 session 继续本地开发 |

## 场景 B：已有 appId 直接本地开发

用户给了 `app_xxx` / 应用链接并说"帮我本地开发"：跳过创建，直接 setup。

| 步骤 | 命令（未上线） | 说明 |
|------|---------------|------|
| 1 | `apps +git-credential-init --app-id app_xxx` | 同场景 A step 2 |
| 2 | `git clone <repo>` → 引导 `cd` | 同场景 A step 3 |

## 终点

clone 完成、`cd` 进仓库后即进入「本地开发模式」，后续能力见 [lark-apps-local-dev.md](lark-apps-local-dev.md)。理想形态下由仓库内 AGENTS.md bootstrap 自动衔接（AGENTS.md 尚未设计，不在当前范围）。

## 参考

- [lark-apps](../SKILL.md) — 妙搭应用全部命令
- [lark-apps-local-dev](lark-apps-local-dev.md) — 本地开发模式能力
