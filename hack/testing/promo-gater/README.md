# promo-gater

An HTTP server that gates Kargo promotion steps. Use it to pause a promotion
at a specific point — to inspect state, manipulate a remote branch between
`git-clone` and `git-push`, or hold for manual observation.

## Usage

```
promo-gater [flags] [-- command [args...]]
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--port` | `24365` | Port to listen on |
| `--addr` | `0.0.0.0` | Bind address |
| `--once` | `true` | Exit after handling one request |

### Modes

**Command mode** — runs a command when the promotion step hits the gate:

```bash
# Run a script between promotion steps
go run ./hack/testing/promo-gater/ -- bash -c "cd /tmp/repo && git rebase main"

# Block for a fixed duration
go run ./hack/testing/promo-gater/ -- sleep 30

# Simple signal (create a file, return immediately)
go run ./hack/testing/promo-gater/ -- touch /tmp/gate-reached
```

Returns the command's combined output as the response body. HTTP 200 on exit
code 0, HTTP 500 otherwise.

**Interactive mode** — blocks until you press Enter:

```bash
go run ./hack/testing/promo-gater/
```

When a request arrives, the terminal prints a prompt and waits. Press Enter to
release the promotion step with HTTP 200.

### Multiple requests

By default the server exits after one request. To keep it running:

```bash
go run ./hack/testing/promo-gater/ --once=false -- date
```

## Promotion step configuration

Add an `http` step to your promotion (or Stage's `promotionTemplate`) that
calls the gater. Use `host.docker.internal` when the promotion controller runs
in a local cluster (kind, k3d, OrbStack) and the gater runs on the host:

```yaml
- uses: http
  config:
    url: http://host.docker.internal:24365
    method: GET
    timeout: 600s
    successExpression: "response.status == 200"
    failureExpression: "response.status == 500"
```

Set `timeout` high enough for your use case — the HTTP connection stays open
while the gate blocks.

## Environment variables

The command receives request metadata as environment variables:

| Variable | Description |
|----------|-------------|
| `GATE_METHOD` | HTTP method (e.g. `GET`) |
| `GATE_PATH` | Request path (e.g. `/`) |
| `GATE_QUERY` | Raw query string |
| `GATE_BODY` | Request body |

## Example: manipulate a branch between clone and push

Insert the gate between `git-clone` and `git-commit` in your promotion steps:

```yaml
steps:
- uses: git-clone
  config:
    repoURL: https://github.com/example/repo.git
    checkout:
    - branch: main
      path: ./src
    - branch: stage/test
      path: ./out
- uses: http
  config:
    url: http://host.docker.internal:24365
    timeout: 600s
    successExpression: "response.status == 200"
- uses: kustomize-set-image
  config:
    path: ./src/base
    images:
    - image: nginx
- uses: git-commit
  config:
    path: ./out
    message: "promote"
- uses: git-push
  config:
    path: ./out
```

Then start the gater before triggering the promotion:

```bash
go run ./hack/testing/promo-gater/
# ... promotion reaches the http step and blocks ...
# Inspect state, force-push a branch, etc.
# Press Enter to continue
```
