# gitwright — Specification v1.0.0

## 1. Overview

`gitwright` keeps one Docker Compose stack in sync with one path in a git repository. It redeploys the stack whenever new commits touch that path.

## 2. Invocation

```
gitwright [--config <path-to-config.yaml>]
```

`--config` defaults to `/etc/gitwright/config.yml`.

## 3. Configuration

YAML file. All fields are required. Unknown fields are not allowed.

| Field | Description |
|---|---|
| `repo_url` | Git repository URL. No authentication. |
| `watch_path` | Path to watch, relative to the repository root. |
| `data_dir` | Directory the repository is cloned into. |
| `stack_name` | Docker Compose project name. |
| `poll_interval` | Interval between polls (e.g. `30s`, `5m`). |

Example:

```yaml
repo_url: https://github.com/org/infra.git
watch_path: staging
data_dir: /var/lib/gitwright/staging
stack_name: my-staging-stack
poll_interval: 1m
```

## 4. Behavior

### 4.1 Startup

1. Load and validate the config.
2. Verify that `data_dir` exists and is writable. If it is not empty, delete its contents.
3. Clone the repository's default branch into `data_dir`.
4. Run Check and Deploy (§4.3).
5. Start polling (§4.2).

### 4.2 Polling

Every `poll_interval`:

1. Fetch the default branch from the remote. If there are no new commits, do nothing.
2. Determine which files changed between the local copy and the remote.
3. Update the local copy to exactly match the remote, discarding any local differences. This always happens, even when the changes do not touch `watch_path`.
4. If any changed file is under `watch_path`, run Check and Deploy (§4.3).

### 4.3 Check and Deploy

**Check.** Both conditions must hold. If either fails, log an error and do nothing else:
- `watch_path` exists in the local copy.
- `watch_path` contains a compose file named `compose.yaml`, `compose.yml`, `docker-compose.yaml`, or `docker-compose.yml`.

**Deploy.** Run these commands in order, from the `watch_path` directory, using `docker-compose` from `PATH`:

1. `docker-compose -p <stack_name> pull`
2. `docker-compose -p <stack_name> down --remove-orphans`
3. `docker-compose -p <stack_name> up -d --remove-orphans`

If a command fails, log its exit code and output and skip the remaining commands. Named volumes are kept.

## 5. Error handling

| Condition | Behavior |
|---|---|
| Config missing or invalid | Fatal. |
| `data_dir` missing or not writable | Fatal. |
| Initial clone fails | Fatal. |
| Fetch fails | Log error and wait for the next poll. |
| Check fails | Log error and wait for the next poll. |
| Compose command fails | Log error and wait for the next poll. |

A fatal error means the process exits with a non-zero code. Failed deploys are never retried. Only new commits that touch `watch_path` trigger another deploy.

## 6. Shutdown

On SIGINT/SIGTERM, exit. The stack is left running as-is.

