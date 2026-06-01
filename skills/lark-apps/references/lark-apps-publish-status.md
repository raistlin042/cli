# apps +publish-status

> **前置条件：** 先阅读 [`../lark-shared/SKILL.md`](../../lark-shared/SKILL.md) 了解认证、全局参数和安全规则。

查询指定发布实例的状态和详情。通常在触发 `apps +publish` 后，用返回的 `instance_id` 轮询进度。

> **⚠️ 过渡期说明：** 这些接口尚未部署到 OpenAPI 网关，当前仅 `--dry-run` 可用；不带 `--dry-run` 的真实调用会返回结构化 "unavailable" 错误（exit 1）。等网关部署后启用。

> **⚠️ 注意：** 这里的「实例号 / 流水线实例号」是妙搭**发布实例** ID（`apps +publish` 返回的 `instance_id`），**不是飞书审批实例号**。查发布进度用 `apps +publish-status`、查失败原因用 `apps +publish-error-log`；不要路由到 lark-approval / 审批相关命令。

## 命令

```bash
# 查询发布实例状态
lark-cli apps +publish-status --app-id app_xxx --instance-id pipeline_task_yyy

# 预演（当前可用）
lark-cli apps +publish-status --app-id app_xxx --instance-id pipeline_task_yyy --dry-run
```

## 参数

| 参数 | 必填 | 说明 |
|---|---|---|
| `--app-id <id>` | ✅ | 应用 ID |
| `--instance-id <id>` | ✅ | 发布实例 ID（即 `apps +publish` 响应的 `instance_id`）|

## 返回值

**成功（网关就绪后）：**

```json
{
  "ok": true,
  "data": {
    "ID": "pipeline_task_yyy",
    "status": 3,
    "status_name": "Success",
    "appID": "app_xxx",
    "creator": "user_zzz",
    "createdAt": 1748000000,
    "updatedAt": 1748000120,
    "description": ""
  }
}
```

**接口未上线（当前行为）：**

```json
{
  "ok": false,
  "error": {
    "type": "unavailable",
    "message": "apps publish endpoints are not yet deployed to the OpenAPI gateway",
    "hint": "..."
  }
}
```

**Validate 失败：**

```json
{
  "ok": false,
  "error": { "type": "validation", "message": "--instance-id is required" }
}
```

## 字段语义

| 字段 | 含义 |
|---|---|
| `status` | NodeStatus 整数（见下表）|
| `status_name` | NodeStatus 可读名称，**优先向用户展示此字段** |
| `createdAt` / `updatedAt` | Unix 秒时间戳（不做时区格式化）|

### NodeStatus 枚举

| 值 | 名称 | 含义 |
|---|---|---|
| 0 | Unspecified | 未知/初始化 |
| 1 | ToDo | 待执行 |
| 2 | Running | 执行中 |
| 3 | Success | 成功 |
| 4 | Failed | 失败 |
| 5 | Canceled | 已取消 |
| 6 | HoldOn | 等待中 |

## 典型场景

### 场景 1：轮询发布进度（网关就绪后）

```bash
# 触发发布
lark-cli apps +publish --app-id app_xxx
# 拿到 instance_id，查状态
lark-cli apps +publish-status --app-id app_xxx --instance-id pipeline_task_yyy
```

- `status_name=Running`：发布仍在进行，稍后重试
- `status_name=Success`：发布成功
- `status_name=Failed`：发布失败，用 `apps +publish-error-log` 获取错误详情

### 场景 2：发布失败后跳转错误日志

```bash
lark-cli apps +publish-error-log --app-id app_xxx --instance-id pipeline_task_yyy
```

## 协同命令

| 场景 | 命令 |
|---|---|
| 触发新发布 | `apps +publish` |
| 查发布历史（获取 instance_id） | `apps +publish-history` |
| 查发布错误详情 | `apps +publish-error-log` |

## 参考

- [lark-apps](../SKILL.md)
- [lark-shared](../../lark-shared/SKILL.md)
