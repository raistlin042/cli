# apps 对话（session 生命周期）

> **前置条件：** 先读 [`../lark-shared/SKILL.md`](../../lark-shared/SKILL.md)（认证、全局参数、安全规则）。

在一个妙搭 app 下完成对话生命周期。资源层级 **app > session > message**（session 是 app 子资源，一轮 chat 是 session 子资源）。全部 UAT 用户身份。

## 命令路由

| 命令 | 作用 | 必填参数 | scope |
|------|------|---------|-------|
| `+session-create` | 在已有 app 下新建 session | `--app-id` | `spark:app:write` |
| `+session-list` | 列出 app 下 session（分页 `--page-size`/`--page-token`） | `--app-id` | `spark:app:read` |
| `+session-read` | 查一个 session 的状态/队列/最近一轮（轮询入口） | `--app-id` `--session-id` | `spark:app:read` |
| `+session-stop` | 打断 session 正在执行的那一轮 | `--app-id` `--session-id` `--turn-id` | `spark:app:write` |
| `+chat` | 发一条消息，发起/继续对话 | `--app-id` `--session-id` `--message` | `spark:app:write` |

余下 flag / 默认值见 `--help`；任意命令加 `--dry-run` 预览请求。**附件（`attachment_ids`）本期不支持**（独立模块）。

```bash
lark-cli apps +session-create --app-id app_xxx                                    # → session_id
lark-cli apps +chat --app-id app_xxx --session-id conv_xxx --message "把表头改蓝"   # → turn_id
lark-cli apps +session-read --app-id app_xxx --session-id conv_xxx                 # 轮询
lark-cli apps +session-stop --app-id app_xxx --session-id conv_xxx --turn-id <turn_id>
lark-cli apps +session-list --app-id app_xxx --format table
```

## 选哪个对话（用户提改动时先决定落点）

| 情形 | 动作 |
|------|------|
| 要「全新应用 + 立刻改动」 | `+create --app-type fullstack`（自带首个 session，JSON 响应里有 `data.session_id`；若无则 `+session-create`）→ 直接 `+chat`，**不要**多此一举再建 session |
| 已知 app_id，没指明哪个对话 | 先 `+session-list`；有 `is_active=true` 的对话就**问用户**：现有里继续，还是新开 |
| 「新开一段 / 换个话题」 | `+session-create --app-id <id>` → `+chat` |
| 「接着刚才那段」 | 复用上下文里的 session_id 直接 `+chat`；拿不到就 `+session-list` 让用户选 |
| 拿不到 app_id | **不枚举 / 搜索应用**，向用户要妙搭链接或 app_id |

> 有活跃对话且用户没指明时，先问一句再动手，别替用户拍板。

## 异步与轮询

`+chat` 异步：只返回 `turn_id`，**不返回执行结果**。CLI 是单次读取，**轮询由调用方控制**（无内置 `--wait`），按 `next_poll_after_ms`（本期固定 30000ms）节流。`turn_id` 即 `+session-read` 的 `latest_turn.turnID`，也是 `+session-stop` 的定位句柄。

```
turn_id = +chat(...)
loop:
  s = +session-read(...)
  if not s.is_streaming and s.latest_turn.status == "completed": 本轮完成 → 可发下一条 +chat
  if s.latest_turn.status in ("failed","cancelled"):            转述失败原因 → 问是否重试
  else:                                                          sleep(next_poll_after_ms) 再读
```

## `+session-read` 字段判读

关键字段：`is_active`（能否再 `+chat`）、`is_streaming`（是否有轮在跑）、`latest_turn`（`turnID` / `status`: running/completed/failed/cancelled）、`queued_count`、`next_poll_after_ms`。

- `is_streaming=false` 且 `latest_turn.status=completed` → 本轮结束，可发下一条。
- `is_active=false` → 会话已关闭，引导 `+session-create` 开新会话。

> ⚠️ **字段大小写不一致**：顶层 snake_case（`session_id` / `is_active` / `is_streaming`），嵌套 turn/message 是 **camelCase**（`turnID` / `seqNo` / `messageID` / `finishReason`）。解析嵌套字段按 camelCase 取键，否则取到 `nil`。

## `+session-stop` 语义

返回 `{stopped, state}`。只停**正在跑（RUNNING）**的当前轮；排队中 / 已结束的轮为 no-op（`stopped=false`）。停断**不关闭会话**，停完仍可 `+chat`。

## 错误码

失败返回 `{ok:false, error:{type,code,message,hint}}`，转述 `hint`（优先）或 `message`，别原样吐 JSON。

| 错误码 | 触发 |
|--------|------|
| `INVALID_PARAM` | 必填缺失 / 参数非法 / 归属错配（session 不属于该 app、turn 不属于该 session） |
| `APP_NOT_FOUND` / `SESSION_NOT_FOUND` | app_id / session_id 无效或无权限 |
| `SESSION_CLOSED` | 向 `is_active=false` 的会话写入 |
| `RATE_LIMITED` / `QUOTA_EXCEEDED` | 限流 / 额度耗尽 |
| `PERMISSION_DENIED` | 鉴权失败 / 无权限 |
| `INTERNAL` | 后端 5xx 兜底 |

## 拿不到 app_id / session_id

- **app_id**：向用户要妙搭链接 `https://miaoda.feishu.cn/app/app_xxx`（取 `/app/` 后段）或 `app_xxx`。**不枚举 / 搜索应用。**
- **session_id**：`+session-list --app-id <id>` 让用户选；或 `+session-create` 新建；或全栈 `+create` 一并拿到。

## 参考

[lark-apps](../SKILL.md) · [`lark-apps-create.md`](lark-apps-create.md)（fullstack 返回 session_id）· [lark-shared](../../lark-shared/SKILL.md)
