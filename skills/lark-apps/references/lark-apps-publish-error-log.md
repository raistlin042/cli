# apps +publish-error-log

> **前置条件：** 先阅读 [`../lark-shared/SKILL.md`](../../lark-shared/SKILL.md) 了解认证、全局参数和安全规则。

获取指定发布实例的错误日志，列出失败的 job 及其组件名和错误原因。通常在 `apps +publish-status` 返回 `status_name=Failed` 后调用，定位失败根因。

> **⚠️ 过渡期说明：** 这些接口尚未部署到 OpenAPI 网关，当前仅 `--dry-run` 可用；不带 `--dry-run` 的真实调用会返回结构化 "unavailable" 错误（exit 1）。等网关部署后启用。

## 命令

```bash
# 查询发布错误日志
lark-cli apps +publish-error-log --app-id app_xxx --instance-id pipeline_task_yyy

# 表格视图
lark-cli apps +publish-error-log --app-id app_xxx --instance-id pipeline_task_yyy --format table

# 预演（当前可用）
lark-cli apps +publish-error-log --app-id app_xxx --instance-id pipeline_task_yyy --dry-run
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
    "status": 4,
    "status_name": "Failed",
    "error_jobs": [
      {
        "jobID": "job_aaa",
        "componentName": "frontend-build",
        "errorMsg": "dependency conflict: react@18 vs react@17"
      }
    ]
  }
}
```

**`--format table` 视图（网关就绪后）：**

```
status: Failed

jobID     componentName    errorMsg
job_aaa   frontend-build   dependency conflict: react@18 vs react@17
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
| `status` | NodeStatus 整数（0–6，详见 `+publish-status` 文档）|
| `status_name` | NodeStatus 可读名称 |
| `error_jobs` | 失败 job 列表；若无失败 job 则为空数组 `[]` |
| `error_jobs[].jobID` | 失败 job ID |
| `error_jobs[].componentName` | 失败组件名 |
| `error_jobs[].errorMsg` | 错误信息，**转述给用户帮助定位问题** |

## 典型场景

### 场景 1：发布失败后查根因

```bash
# 先确认状态
lark-cli apps +publish-status --app-id app_xxx --instance-id pipeline_task_yyy
# status_name=Failed 时查错误日志
lark-cli apps +publish-error-log --app-id app_xxx --instance-id pipeline_task_yyy --format table
```

把 `error_jobs[].errorMsg` 转述给用户：

> 发布失败，组件 `{componentName}` 报错：{errorMsg}

### 场景 2：error_jobs 为空但 status=Failed

说明失败发生在 job 之前（如参数校验阶段），转述 `status_name` 给用户并建议检查发布配置。

## 协同命令

| 场景 | 命令 |
|---|---|
| 触发新发布 | `apps +publish` |
| 查单个发布状态 | `apps +publish-status` |
| 查发布历史 | `apps +publish-history` |

## 参考

- [lark-apps](../SKILL.md)
- [lark-shared](../../lark-shared/SKILL.md)
