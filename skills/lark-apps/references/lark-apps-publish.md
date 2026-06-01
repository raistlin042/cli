# apps +publish

> **前置条件：** 先阅读 [`../lark-shared/SKILL.md`](../../lark-shared/SKILL.md) 了解认证、全局参数和安全规则。

为指定妙搭应用创建一个发布单（release），触发应用发布流水线。

> **⚠️ 过渡期说明：** 这些接口尚未部署到 OpenAPI 网关，当前仅 `--dry-run` 可用；不带 `--dry-run` 的真实调用会返回结构化 "unavailable" 错误（exit 1）。等网关部署后启用。

> **行动指引：** 用户明确要求"发版 / 上线 / 发布现有应用"时：直接执行 `apps +publish --app-id <id> --dry-run` 展示将要发起的发布（dry-run 只读、无副作用，当前端点未上网关时这是可用的预演），不要只在文字里"建议"命令而不执行。现有应用发版 → `+publish`；上传本地 HTML/静态产物 → `+html-publish`；新建应用 → `+create`。

## 命令

```bash
# 使用服务端默认分支发布
lark-cli apps +publish --app-id app_xxx

# 指定发布分支
lark-cli apps +publish --app-id app_xxx --branch main

# 预演（打印意图请求，不执行，当前可用）
lark-cli apps +publish --app-id app_xxx --dry-run
```

## 参数

| 参数 | 必填 | 说明 |
|---|---|---|
| `--app-id <id>` | ✅ | 应用 ID（如 `app_xxx`）|
| `--branch <branch>` | ❌ | 发布分支；不传则服务端使用默认分支 |

## 返回值

**成功（网关就绪后）：**

```json
{
  "ok": true,
  "data": {
    "release_id": "release_yyy"
  }
}
```

`release_id` 即发布ID；后续查状态 / 错误日志时把该值传给 `--release-id`。

**接口未上线（当前行为）：**

```json
{
  "ok": false,
  "error": {
    "type": "unavailable",
    "message": "apps publish endpoints are not yet deployed to the OpenAPI gateway",
    "hint": "fill the gateway paths in apps_publish_common.go and set publishAPIWired=true once the gateway exposes lark.apaas.devops / lark.apaas.devops_platform"
  }
}
```

**Validate 失败（如缺 --app-id）：**

```json
{
  "ok": false,
  "error": { "type": "validation", "message": "--app-id is required" }
}
```

## --dry-run 示例输出

```json
{
  "ok": true,
  "data": {
    "dry_run": true,
    "psm": "lark.apaas.devops",
    "rpc_method": "OpenAPICreateRelease",
    "request": { "appID": "app_xxx" },
    "gateway_status": "not_deployed",
    "note": "upstream PSM reference path (lark.apaas.devops); gateway path TBD"
  }
}
```

## 字段语义

| 字段 | 含义 |
|---|---|
| `data.release_id` | 发布ID，用于 `+publish-status` / `+publish-error-log` 的 `--release-id` |
| `error.type=unavailable` | 接口尚未上网关，等部署后自动启用 |
| `error.type=validation` | 本地参数错，修正 flag 后重试 |

## 典型场景

### 场景 1：触发发布并轮询状态

```bash
# 当前只能 dry-run
lark-cli apps +publish --app-id app_xxx --dry-run

# 网关就绪后：
lark-cli apps +publish --app-id app_xxx
# 拿到 release_id 后查状态：
lark-cli apps +publish-status --app-id app_xxx --release-id release_yyy
```

### 场景 2：接口未上线时的处理

当前调用会返回 `unavailable` 错误（exit 1）。转述给用户：

> 发布接口尚未部署到 OpenAPI 网关，暂无法触发真实发布。可用 `--dry-run` 预演请求内容。

## 协同命令

| 场景 | 命令 |
|---|---|
| 查发布历史 | `apps +publish-history` |
| 查单个发布状态 | `apps +publish-status` |
| 查发布错误日志 | `apps +publish-error-log` |

## 参考

- [lark-apps](../SKILL.md)
- [lark-shared](../../lark-shared/SKILL.md)
