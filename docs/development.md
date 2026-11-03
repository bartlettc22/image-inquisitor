# Development

You need Docker and [Task](https://taskfile.dev).

## Local stack

```bash
cp config.example.yaml config.local.yaml   # then edit it
task dev-serve                             # daemon on :8080, UI on http://localhost:8081
task ui-dev                                # UI with hot reload on http://localhost:5173
```

`task dev-serve` builds both images and runs them with Docker Compose ([compose.yaml](../compose.yaml)).
It uses your `GITHUB_TOKEN`, or the `gh` CLI's token, and mounts your kubeconfig and Docker
credentials read-only. `config.local.yaml` is gitignored; set `IMGINQ_CONFIG` to use another file.

| Task | Does |
|---|---|
| `task dev-logs` | Follow the stack's logs |
| `task dev-down` | Stop the stack, keeping its data in `.data/dev` |
| `task dev-reset` | Stop the stack and delete its data |

## Checks

| Task | Does |
|---|---|
| `task test` | Unit tests |
| `task lint` | golangci-lint |
| `task fmt-check` | Fail on Go files that aren't gofmt-formatted |
| `task tidy` | Tidy Go modules |
| `task ui-lock` | Regenerate `ui/package-lock.json` |

CI runs `fmt-check`, `lint` and `test` on pull requests and pushes to `main`.

## Images and chart

| Task | Does |
|---|---|
| `task build-image` / `publish-image` | Daemon image (`VERSION=dev` by default) |
| `task build-ui-image` / `publish-ui-image` | UI image |
| `task publish-helm` | Package and push the Helm chart (needs a release `VERSION`) |

Run `task --list` for everything.

## Releasing

The git tag is the version. Add the release to [CHANGELOG.md](../CHANGELOG.md), merge it, then tag
`main`:

```bash
git tag v1.2.3 && git push origin v1.2.3
```

The [release workflow](../.github/workflows/release.yaml) publishes both images as `v1.2.3` and the
chart as `1.2.3` with `appVersion: v1.2.3`, so the chart deploys the images from the same tag.
Nothing is published from `main`.
