# apps 本地开发模式（规划中 · 命令未上线 · 暂不执行）

> ⚠️ **本文件为设计预览。下列命令（`+db-*`、`+env-pull`、`+publish*`）尚未上线，当前请勿执行。** 用户要求本地数据库操作 / 拉环境 / 发布本地开发应用时，告知该能力规划中、暂未上线。

> **前置条件：** 命令上线后，先阅读 [`../../lark-shared/SKILL.md`](../../lark-shared/SKILL.md)。

## 定位

cwd 在已 clone 的本地开发仓库内时的开发指导。通过本地 setup（[local-setup](lark-apps-local-setup.md)）进入。

## 可用能力（命令未上线）

| 命令 | 职责 |
|------|------|
| `apps +db-table-list` | 列出应用数据库的所有表 |
| `apps +db-table-schema` | 查看指定表的 schema |
| `apps +db-sql` | 执行 SQL 查询 / 操作 |
| `apps +db-multi-env-init` | 多环境数据库初始化 |
| `apps +env-pull --app-id <id>` | 拉取应用 env 到本地 |
| `apps +publish` | 本地开发应用发布 |
| `apps +publish-status` | 查询发布状态 |
| `apps +publish-history` | 查询发布历史 |

## 典型串联组合（命令上线后）

- 编辑迭代 → `apps +db-sql` / `apps +db-table-schema` 改 / 查数据 → `apps +publish`
- 或 `apps +env-pull` 拉新环境配置 → 继续开发

## 参考

- [lark-apps-local-setup](lark-apps-local-setup.md) — 进入本地开发的一次性 setup
- [lark-apps](../SKILL.md)
