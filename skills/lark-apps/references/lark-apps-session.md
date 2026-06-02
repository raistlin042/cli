# apps 对话（session 生命周期）

> **前置条件：** 先阅读 [`../lark-shared/SKILL.md`](../../lark-shared/SKILL.md) 了解认证、全局参数和安全规则。

在一个妙搭 app 下完成完整对话生命周期：**创建 session → 发消息 → 查询/轮询状态 → 停止 → 列出对话**。

资源层级：**app > session > message**。session 是 app 的子资源，message（一次对话 = chat）是 session 的子资源。

涉及命令：`+session-create`、`+session-list`、`+session-read`、`+session-stop`、`+chat`。全部需登录态（UAT，用户身份）。

| 命令 | 作用 | scope |
|------|------|-------|
| `+session-create` | 在已有 app 下新建一个 session | `spark:app:write` |
| `+session-list` | 列出 app 下的 session（分页） | `spark:app:read` |
| `+session-read` | 查询一个 session 的状态 + 队列 + 最近一轮（轮询入口） | `spark:app:read` |
| `+session-stop` | 打断 session 上正在执行的那一轮 | `spark:app:write` |
| `+chat` | 在 session 下发一条消息，发起/继续对话 | `spark:app:write` |

---

## 核心概念

- **session_id**：会话 ID。`apps +create --app-type fullstack` 或 `+session-create` 返回。
- **turn_id**：一次对话（一轮 chat）的处理句柄。`+chat` 返回；`+session-stop` 用它定位要停的轮；`+session-read` 的 `latest_turn.turnID` 也是它。
- **异步 + 轮询**：`+chat` 只发起对话并返回 `turn_id`，**不返回执行结果**。云端异步处理，需用 `+session-read` 查询/轮询最新状态。CLI 是单次读取——**轮询循环由你（调用方）控制**，没有内置 `--wait`。
- **轮询节奏**：`+chat` / `+session-read` 返回 `next_poll_after_ms`（本期固定 30000，即 30 秒）。两次 `+session-read` 之间按它节流，不要忙等。

---

## 命令用法

```bash
# 新建 session
lark-cli apps +session-create --app-id app_xxx

# 列出 app 下的 session（分页）
lark-cli apps +session-list --app-id app_xxx
lark-cli apps +session-list --app-id app_xxx --page-size 50 --page-token <上一页的 next_page_token>
lark-cli apps +session-list --app-id app_xxx --format table     # 表格只显示 session_id/name/is_active/updated_at

# 发消息（发起/继续对话），返回 turn_id
lark-cli apps +chat --app-id app_xxx --session-id conv_xxx --message "把首页表头改成蓝色"

# 查询/轮询状态
lark-cli apps +session-read --app-id app_xxx --session-id conv_xxx

# 停止当前正在跑的那一轮
lark-cli apps +session-stop --app-id app_xxx --session-id conv_xxx --turn-id <turn_id>

# 任意命令加 --dry-run 只打印请求不执行
lark-cli apps +chat --app-id app_xxx --session-id conv_xxx --message "hi" --dry-run
```

## 参数

| 命令 | 参数 | 必填 | 说明 |
|------|------|------|------|
| `+session-create` | `--app-id` | ✅ | app ID |
| `+session-list` | `--app-id` | ✅ | app ID |
| | `--page-size` | ❌ | 每页条数，默认 20，最大 50 |
| | `--page-token` | ❌ | 上一页返回的 `next_page_token`，首页不传 |
| `+session-read` | `--app-id` / `--session-id` | ✅ | app ID / 会话 ID |
| `+session-stop` | `--app-id` / `--session-id` / `--turn-id` | ✅ | turn-id 来自 `+chat` 返回或 `+session-read` 的 `latest_turn.turnID` |
| `+chat` | `--app-id` / `--session-id` / `--message` | ✅ | message 是用户消息文本 |

> **附件不在本期范围**：`+chat` 暂不支持上传附件（`attachment_ids`）。附件上传是独立模块，后续单独提供。

---

## 返回值与字段语义

### `+session-create` / `+chat`

```json
{ "ok": true, "data": { "session_id": "conv_new" } }                              // +session-create
{ "ok": true, "data": { "turn_id": "8421374925", "next_poll_after_ms": 30000 } }  // +chat
```

### `+session-list`

`data.sessions[]` 元素：`session_id` / `name`（会话/任务名，可能为空）/ `is_active`（bool，是否可写）/ `created_at` / `updated_at`（ISO 8601 UTC）。还有 `next_page_token` / `has_more` 用于翻页。list 是轻量索引，"某 session 进行到哪了"要用 `+session-read`。

### `+session-read`（判读重点）

`data` 关键字段：

- `is_active`（bool）：会话是否可写（能否 `+chat`）。`false` = 已关闭，不能再发消息。
- `is_streaming`（bool）：是否有 turn 正在执行。轮询循环靠它判断"还要不要再轮"。
- `latest_turn`：最近一轮。`turnID` / `status`（`running` / `completed` / `failed` / `cancelled`）/ `messages[]` / `sender`。
- `queued_count` / `queued_turns[]`：排队中等待执行的轮。
- `summary`：会话摘要（可能为空）。
- `next_poll_after_ms`：建议下次轮询间隔（毫秒）。

> ⚠️ **字段大小写**：顶层字段是 snake_case（`session_id` / `is_active` / `is_streaming`），但嵌套的 turn/message 字段是 **camelCase**（`turnID` / `seqNo` / `submittedAt` / `messageID` / `finishReason`）。解析嵌套字段时注意。

**判读规则：**
- `is_streaming=false` 且 `latest_turn.status=completed` → 本轮结束，可发下一条 `+chat`。
- `is_streaming=true` → 还在跑，等 `next_poll_after_ms` 后再 `+session-read`。
- `is_active=false` → 会话已关闭，不能再写（需 `+session-create` 开新会话）。

### `+session-stop`

```json
{ "ok": true, "data": { "stopped": true,  "state": "running" } }    // 确实停了一个 RUNNING turn
{ "ok": true, "data": { "stopped": false, "state": "completed" } }  // no-op：turn 不在运行中
```

只停**正在跑（RUNNING）**的当前轮；排队中或已结束的 turn 为 no-op（`stopped=false`）。停断**不关闭会话**，停完仍可继续 `+chat`。

---

## 典型编排

### 1. 创建全栈应用并发起对话（最常见）

`apps +create --app-type fullstack` 会一并返回 `session_id`，可直接 `+chat`，**无需** `+session-create`：

```bash
lark-cli apps +create --app-type fullstack --name "素材管理后台" --description "..."   # → app_id + session_id
lark-cli apps +chat --app-id <app_id> --session-id <session_id> --message "创建一个素材管理后台"   # → turn_id
# 然后轮询：
lark-cli apps +session-read --app-id <app_id> --session-id <session_id>               # 直到 is_streaming=false
```

### 2. 在已有 app 上新开一个对话

```bash
lark-cli apps +session-create --app-id <app_id>                                       # → session_id
lark-cli apps +chat --app-id <app_id> --session-id <session_id> --message "..."        # → turn_id
lark-cli apps +session-read --app-id <app_id> --session-id <session_id>
```

### 3. 打断当前轮

```bash
lark-cli apps +session-read --app-id <app_id> --session-id <session_id>                # 取 latest_turn.turnID
lark-cli apps +session-stop --app-id <app_id> --session-id <session_id> --turn-id <turn_id>
```

### 4. 重新进入 agent，列出对话再继续

```bash
lark-cli apps +session-list --app-id <app_id>                                         # 挑一个 is_active=true 的
lark-cli apps +chat --app-id <app_id> --session-id <选中的 session_id> --message "..."
```

### 轮询循环（伪代码）

```
turn_id = chat(...)            # +chat
loop:
  s = session_read(...)        # +session-read
  if not s.is_streaming and s.latest_turn.status == "completed": break   # 本轮完成
  if s.latest_turn.status in ("failed","cancelled"): handle; break
  sleep(s.next_poll_after_ms ms)
```

---

## 错误码

失败返回 `{ "ok": false, "error": { "type", "code", "message", "hint" } }`。常见：

| 错误码 | 触发条件 |
|--------|----------|
| `INVALID_PARAM` | 必填缺失 / 参数非法（turn_id 非数字、session 不属于该 app、turn 不属于该 session 等归属错配） |
| `APP_NOT_FOUND` | app_id 无效或无权限 |
| `SESSION_NOT_FOUND` | session_id 无效 |
| `SESSION_CLOSED` | 向已关闭/归档的 session 写入（`is_active=false` 后还 `+chat`） |
| `RATE_LIMITED` / `QUOTA_EXCEEDED` | 限流 / 额度耗尽 |
| `PERMISSION_DENIED` | 鉴权失败 / 无权限 |

转述 `error.hint`（优先）或 `error.message`，不要原样把 envelope JSON 复述给用户。

---

## 拿不到 app_id / session_id 时

- **app_id**：让用户提供妙搭应用链接 `https://miaoda.feishu.cn/app/app_xxx`（取 `/app/` 后的段）或直接给 `app_xxx`。**不要枚举/搜索应用。**
- **session_id**：用 `+session-list --app-id <app_id>` 列出该 app 下的会话让用户选；或 `+session-create` 新建；或全栈 `+create` 时一并拿到。

## 参考

- [lark-apps](../SKILL.md) — 妙搭应用全部命令
- [`lark-apps-create.md`](lark-apps-create.md) — 创建应用（fullstack 返回 session_id）
- [lark-shared](../../lark-shared/SKILL.md) — 认证和全局参数
