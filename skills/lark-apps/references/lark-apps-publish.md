# apps +publish

为妙搭应用创建发布 release。运行时命令事实以 `lark-cli apps +publish --help` 为准。

## 何时用

用于把全栈应用的代码分支推进到发布流程。它不是 HTML 静态发布入口；本地 `index.html` / `dist` 要读 [`lark-apps-html-publish.md`](lark-apps-html-publish.md)。

## 命令骨架

- 必填：`--app-id`。
- 可选：`--branch`；省略时服务端使用默认发布分支。
- 返回 `release_id` 和 `status`，后续用 `+publish-status` 轮询。

## 示例

```bash
lark-cli apps +publish --app-id app_xxx
lark-cli apps +publish --app-id app_xxx --branch sprint/default --dry-run
```

## 输出契约

- 成功读取 `data.release_id` 和 `data.status`；`release_id` 是后续 `+publish-status` / `+publish-error-log` 的入参。
- `status=publishing` 表示发布仍在进行；继续用 `+publish-status` 轮询。

## Agent 规则

发布前通常先确认本地 `git status` 干净且已 push `sprint/default`。发布后若 status 是 `publishing`，用 [`+publish-status`](lark-apps-publish-status.md) 查询。`+publish` 部署上线属高影响动作——作为别的命令的连带前置时，按 SKILL.md「失败与高影响动作」先征得用户同意再发布。

### 发布失败：区分瞬时故障 vs 代码错误

`+publish` / `+publish-status` 为 `failed` 时，先看 [`+publish-error-log`](lark-apps-publish-error-log.md) 的 step/error_log 判别：

- **基础设施瞬时故障**（`EAI_AGAIN` / `ETIMEDOUT` / npm registry 不可达 / DNS 抖动）→ 自动重试上限 **2 次**；仍失败给诚实回执，不要 `ScheduleWakeup` 死等：
  > "平台 npm 源暂不可达，release 已生成但未发布成功。可用 `+publish-history` / `+publish-error-log --release-id <id>` 查询，稍后重试。"
- **应用代码错误**（构建报错、类型错误等）→ 不重试，转述失败步骤与可行动修复，让用户改代码后重发。
