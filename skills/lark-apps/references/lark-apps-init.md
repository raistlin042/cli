# apps +init

> **前置条件：** 先阅读 [`../lark-shared/SKILL.md`](../../lark-shared/SKILL.md) 了解认证、全局参数和安全规则。

初始化妙搭应用的**本地开发仓库**。这是一个编排命令（orchestrator）：先调 `apps +git-credential-init` 签发带凭据的仓库地址，再 `git clone` → 切到 `sprint/default` 分支 →（本版本暂跳过 npx 脚手架步骤）→ 若工作区有改动则 `git add -A` + commit + `git push origin sprint/default`，工作区干净则跳过 commit/push。返回本地克隆路径。

> ⚠️ **依赖未发布：** `+init` 依赖 `apps +git-credential-init` 命令，该命令**当前版本尚未发布**。在它就绪前，`+init` 会以结构化 `credential_init` 错误失败——这是预期行为，不是命令本身坏了。

## 命令

```bash
# 最小调用（克隆到 ./<app-id>）
lark-cli apps +init --app-id app_xxx

# 指定克隆目录（必须是 cwd 内的相对路径）
lark-cli apps +init --app-id app_xxx --dir ./my-app

# 预演（打印计划步骤，不执行任何 git 操作）
lark-cli apps +init --app-id app_xxx --dry-run
```

## 参数

| 参数 | 必填 | 说明 |
|---|---|---|
| `--app-id <id>` | ✅ | 妙搭应用 ID；缺失时 Validate 阶段以结构化 `validation` 错误退出（exit code 2），**不是**纯文本错误 |
| `--dir <path>` | ❌ | 克隆目标目录，默认 `./<app-id>`。**必须是 cwd 内的相对路径**（经 `validate.SafeInputPath` 校验）：绝对路径或越界路径（`../`、`/Users/...`）会被拒绝；目标目录已存在且**非空**也会被拒绝（不存在则由 git clone 创建） |
| `--format <fmt>` | ❌ | 输出格式，默认 `json` |
| `--dry-run` | ❌ | 仅打印计划步骤，不执行 |

## 返回值

**成功：**

```json
{
  "ok": true,
  "data": {
    "app_id": "app_xxx",
    "repository_url": "https://***@example.com/org/app_xxx.git",
    "branch": "sprint/default",
    "clone_path": "/abs/path/to/app_xxx",
    "committed": false,
    "pushed": false,
    "npx_skipped": true,
    "message": "Repository initialized. You can start developing."
  }
}
```

**失败（结构化 envelope）：**

```json
{
  "ok": false,
  "error": {
    "type": "credential_init",
    "message": "apps +git-credential-init failed: ...",
    "hint": "ensure apps +git-credential-init is available and you are logged in"
  }
}
```

## 字段语义

| 字段 | 含义 |
|---|---|
| `app_id` | 透传的应用 ID |
| `repository_url` | 仓库地址，**凭据已脱敏**：URL 里的 userinfo 段被替换为 `***`（如 `https://***@host/...`）。任何回显仓库地址的错误信息也同样脱敏 |
| `branch` | 切出的分支，固定为 `sprint/default` |
| `clone_path` | 本地克隆的**绝对路径** |
| `committed` | 是否产生了 commit（工作区干净时为 `false`） |
| `pushed` | 是否 push 成功（工作区干净时为 `false`；commit 成功但 push 失败时为 `committed=true, pushed=false` 并带 `git_push` 错误） |
| `npx_skipped` | 本版本恒为 `true`（npx 脚手架步骤本版本有意跳过） |
| `message` | 固定的成功提示文案 |

错误 `type` 取值随失败阶段不同：`validation`（参数 / 路径 / repository_url scheme 非 http(s)）、`dependency`（PATH 上找不到 git）、`credential_init`（凭据签发失败或返回不可解析）、`git_clone` / `git_checkout` / `git_status` / `git_add` / `git_commit` / `git_push`（对应 git 步骤失败）。优先转述 `error.hint`，为空时退回 `error.message`。

## --dry-run 行为

预演只打印计划，不执行任何 git 操作，字段包括：`credential_init`（将要跑的 `+git-credential-init` 命令）、`clone`、`checkout`、`commit_push`（条件说明）、`clone_path`、`npx_skipped`。

若 `--dir` 校验失败（绝对路径 / 越界 / 已存在非空），dry-run **不会**直接报错退出，而是把拒绝原因放进 `dir_error` 字段、`clone_path` 退回默认值，并**仍以 exit 0 返回**。所以 dry-run 通过不代表真跑一定通过——要检查输出里有没有 `dir_error`。

## 前置条件与注意事项

- **`git` 必须在 PATH 上**：否则以结构化 `dependency` 错误退出，hint 为 `install git and ensure it is on your PATH`。
- **依赖 `apps +git-credential-init`**：`+init` 通过 shell out 调用同一个 lark-cli 可执行文件去跑 `apps +git-credential-init --app-id <id> --format json`（设置了 `--as` 时会透传），从其 `data.repository_url` 取仓库地址，再用它 `git clone`。运行时若凭据签发失败或远端不可达，`+init` 在此步返回 `credential_init` 结构化错误。
- **commit message 固定**：push 时的 commit 主题是固定常量 `chore: scaffold app via lark-cli apps +init`，绝不拼接用户输入。
- **repository_url 仅接受 http(s)**：从 `+git-credential-init` 拿到的地址若不是 `http://` / `https://`（如 `ssh://`、`ext::`、`file://`）会被直接拒绝（`validation` 错误），以防危险的 git transport 与参数注入。
- **不要**原样把 envelope JSON 复述给用户。

## 协同命令

| 场景 | 命令 |
|---|---|
| 创建应用拿到 app_id | `apps +create` |
| 签发仓库凭据（`+init` 内部依赖，尚未发布） | `apps +git-credential-init` |

## 参考

- [lark-apps](../SKILL.md) — 妙搭应用全部命令
- [lark-shared](../../lark-shared/SKILL.md) — 认证和全局参数
