# apps +env-pull

> **Prerequisite:** Read [`../../lark-shared/SKILL.md`](../../lark-shared/SKILL.md) first.

Pull local startup environment variables for an app and sync them into `<project-path>/.env.local`.

## Command

```bash
lark-cli apps +env-pull --app-id app_xxx
lark-cli apps +env-pull --app-id app_xxx --project-path /workspace/demo
```

## Flags

| Flag | Required | Description |
|---|---|---|
| `--app-id <id>` | ✅ | App ID |
| `--project-path <path>` | ❌ | Local project root path. Defaults to the current directory. |

The command always reads and writes:

```text
<project-path>/.env.local
```

## Behavior

- Sends `POST /open-apis/spark/v1/apps/:app_id/env_vars`
- Creates `.env.local` if missing
- Updates active `KEY = "value"` lines for keys returned by the backend
- Appends missing returned keys
- Preserves comments, blank lines, malformed lines, and unrelated keys
- Treats `SUDA_DATABASE_URL` as the signal for `Development database detected` in pretty output

## Output

Success output is in English.

The CLI never prints env values.

Structured output exposes only safe summary fields such as:
- `app_id`
- `project_path`
- `env_file`
- `database_detected`
- `updated_count`
- `created_count`

## Typical usage

### Refresh the current project

```bash
lark-cli apps +env-pull --app-id app_xxx
```

### Refresh a different local project directory

```bash
lark-cli apps +env-pull --app-id app_xxx --project-path /workspace/demo
```

## Failure behavior

- Missing `--app-id` fails in validation with `--app-id is required`
- Invalid project path fails in validation
- Symlink / non-regular-file targets are rejected
- Malformed backend payload fails

## Auth behavior

- The command requires user identity (`--as user`)
- Dry-run still performs the command's scope pre-check, so missing `spark:app:read` blocks both dry-run and live execution

## Related commands

| Scenario | Command |
|---|---|
| Create an app | `apps +create` |
| Update app metadata | `apps +update` |
| List apps | `apps +list` |

## References

- [lark-apps](../SKILL.md)
- [lark-shared](../../lark-shared/SKILL.md)
