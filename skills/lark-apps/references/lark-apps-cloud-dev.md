# apps 云端开发模式（规划中 · 命令未上线 · 暂不执行）

> ⚠️ **本文件为设计预览。下列命令（`+session-*`、`+chat`、`+publish`）尚未上线，当前请勿执行。** 用户要求云端 session 对话式开发时，告知该能力规划中、暂未上线。

> **前置条件：** 命令上线后，先阅读 [`../../lark-shared/SKILL.md`](../../lark-shared/SKILL.md)。

## 定位

用户给了 app_id / 应用链接 / 应用名，且在非代码目录时的云端 chat 式开发。

## 可用能力（命令未上线）

| 命令 | 职责 |
|------|------|
| `apps +session-create --app-id <id>` | 创建一个云端开发 session |
| `apps +session-read --session-id <id>` | 读取 session 状态 / 历史消息 |
| `apps +session-list --app-id <id>` | 列举当前应用的活跃 session |
| `apps +session-stop --session-id <id>` | 停止指定 session |
| `apps +chat --session-id <id> --message "<内容>" [--attach ...]` | 向云端 session 发消息（多轮） |
| `apps +publish` | 发布 |

## 典型串联组合（命令上线后）

- `apps +session-create` → 多轮 `apps +chat` → `apps +publish`
- 或 `apps +session-list` 查活跃 session → `apps +chat` → `apps +publish`

## 参考

- [lark-apps](../SKILL.md)
