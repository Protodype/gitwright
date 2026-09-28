# gitwright

Keeps one Docker Compose stack in sync with one path in a git repository. Whenever new commits touch that path, the stack is redeployed.

## Requirements

- `docker-compose` in `PATH`.
- A git repository reachable without authentication.

## Usage

```
gitwright [--config config.yaml]
```

Without `--config`, the config is read from `/etc/gitwright/config.yml`.

## Configuration

All fields are required.

```yaml
repo_url: https://github.com/org/infra.git   # repository to track (default branch)
watch_path: staging                          # path in the repository to watch
data_dir: /var/lib/gitwright/staging         # must exist and be writable; its contents are wiped on start
stack_name: my-staging-stack                 # Docker Compose project name
poll_interval: 1m                            # how often to check for new commits
```

## Running in Docker

Compose resolves bind mount paths inside the gitwright container, but the host daemon uses them. `data_dir` must therefore be mounted at the same path on the host and in the container.

```yaml
services:
  gitwright:
    image: gitwright
    restart: unless-stopped
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - ${DATA_DIR:?DATA_DIR must be set}:${DATA_DIR}
    configs:
      - source: gitwright
        target: /etc/gitwright/config.yml
configs:
  gitwright:
    content: |
      repo_url: https://github.com/org/infra.git
      watch_path: staging
      data_dir: ${DATA_DIR}
      stack_name: my-staging-stack
      poll_interval: 1m
```

With `.env` next to it:

```
DATA_DIR=/srv/gitwright/staging
```

Inline `configs.content` requires Docker Compose 2.23.1 or later.

### Passing secrets and variables to the deployed stack

`docker-compose` inherits the environment of the gitwright process, so variables set on the gitwright container are available for interpolation in the deployed compose file. This keeps secrets out of the repository.

1. Define them in gitwright's `.env`:
   ```
   DATA_DIR=/srv/gitwright/staging
   DB_PASSWORD=changeme
   ```
2. Pass them to the gitwright container:
   ```yaml
   services:
     gitwright:
       # ...
       environment:
         DB_PASSWORD: ${DB_PASSWORD:?DB_PASSWORD must be set}
   ```
3. Use them in the deployed compose file in the repository:
   ```yaml
   services:
     db:
       image: postgres
       environment:
         POSTGRES_PASSWORD: ${DB_PASSWORD}
   ```

Variables from the gitwright environment take precedence over a `.env` file in `watch_path`. Values are visible to anyone who can run `docker inspect` on either container.

## How it works

1. On start, `data_dir` is emptied, the repository is cloned into it and the stack is deployed.
2. Every `poll_interval`, the local copy is updated to match the remote. If any change touches `watch_path`, the stack is deployed again.
3. Before deploying, `watch_path` must exist and contain a compose file (`compose.yaml`, `compose.yml`, `docker-compose.yaml` or `docker-compose.yml`). Otherwise the deploy is skipped.
4. A deploy runs, from `watch_path`:
   ```
   docker-compose -p <stack_name> pull
   docker-compose -p <stack_name> down --remove-orphans
   docker-compose -p <stack_name> up -d --remove-orphans
   ```
   If a command fails, the remaining ones are skipped. Named volumes are kept.

## Failure behavior

- An invalid config, an unusable `data_dir` or a failed initial clone stops the process.
- Any other failure is skipped until the next poll. Failed deploys are not retried; only new commits touching `watch_path` trigger another deploy.
- On SIGINT/SIGTERM the process exits and the stack keeps running.

See [the specification](specs/gitwright-spec-v1.0.0.md) for details.
