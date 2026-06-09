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

`+publish` 部署的是远端 `sprint/default` 上已 push 的代码，不是本地工作区——本地若有未推送的改动，发布前必须先 `git add` + `git commit` 并 `git push` 到 `sprint/default`，否则这些改动不会进入这次发布。门禁是「本次相关改动已提交并推送」，不要求工作区绝对干净（重新发布、刚 `+init` 完等无本地改动的情况可直接发）。发布后若 status 是 `publishing`，用 [`+publish-status`](lark-apps-publish-status.md) 查询。`+publish` 部署上线属高影响动作——作为别的命令的连带前置时，按 SKILL.md「高影响动作：确认与预授权」先征得用户同意再发布。
