# Configuration

The daemon reads one YAML file, `/etc/imginq/config.yaml` in its image. With Helm, you write it under
`config:` in your values and the chart fills in the rest; see [Helm values](#helm-values).
[config.example.yaml](../config.example.yaml) is an annotated example.

The daemon checks the file at startup and exits with an error naming the bad setting.

## Daemon config

### Top level

| Key | Default | Description |
|---|---|---|
| `listen` | `:8080` | Address the API, UI backend and `/metrics` listen on. Set by the chart |
| `dataDir` | `/var/lib/imginq` | SQLite database, SBOMs, render cache. Set by the chart |
| `syncInterval` | `10m` | How often to [sync](how-it-works.md#the-sync-loop) (Go duration) |
| `github` | | [GitHub access](#github) |
| `repos` | | [Repos to track](#repos) |
| `kubernetes` | | [Clusters to inventory](#kubernetes) |
| `scan` | | [Scanning](#scan) |
| `helm` | | [Chart rendering](#helm) |
| `renovate` | unset | [Running Renovate](renovate.md); leave out to run Renovate yourself |

At least one repo or cluster is required.

### github

| Key | Default | Description |
|---|---|---|
| `apiURL` | `https://api.github.com` | GitHub API URL |
| `token` | `$GITHUB_TOKEN` | Read access to the repos' pull requests and contents. Without one, only public repos can be read |

### repos

A list of repos whose Renovate PRs are tracked.

| Key | Default | Description |
|---|---|---|
| `name` | required | `owner/repo`, or a pattern such as `owner/*` |
| `exclude` | `[]` | Names or patterns removed from a pattern's matches |
| `branchPrefix` | `renovate/` | Head branch prefix that marks a PR as Renovate's |

A pattern matches the owner's repos (org or user) that aren't archived or forks, listed with the
GitHub token every hour, so new repos are picked up without a restart. When a repo stops matching,
its PRs are dropped once a listing succeeds. The UI's Configuration dialog shows each pattern's
current matches.

```yaml
repos:
- name: example-org/deploy
- name: example-org/*
  exclude: [example-org/docs, example-org/sandbox-*]
```

### kubernetes

| Key | Default | Description |
|---|---|---|
| `kubeconfig` | `$KUBECONFIG` or `~/.kube/config` | Kubeconfig used for clusters selected by `context` |
| `clusters` | `[]` | Clusters whose running images are inventoried |

Each cluster sets exactly one of `context` or `inCluster`:

| Key | Default | Description |
|---|---|---|
| `context` | | Kubeconfig context |
| `inCluster` | `false` | Use the pod's service account (the chart's default) |
| `name` | the context, or `in-cluster` | Name shown in the UI and metrics; must be unique |
| `includeNamespaces` | all | Only inventory these namespaces |
| `excludeNamespaces` | none | Skip these namespaces |

### scan

| Key | Default | Description |
|---|---|---|
| `disabled` | `false` | Turn off [scanning](how-it-works.md#scanning) |
| `trivy` | `trivy` | Trivy executable (bundled in the image) |

Registry credentials come from `$DOCKER_CONFIG/config.json`, as for Docker.

### helm

| Key | Default | Description |
|---|---|---|
| `binary` | `helm` | Helm executable used to render Argo CD apps (bundled in the image) |

### renovate

See [Renovate and soak policy](renovate.md#configuration).

## Helm values

The chart builds the daemon config from `config`, and sets `listen`, `dataDir`,
`kubernetes.kubeconfig` (when `kubeconfig.existingSecret` is set) and `renovate` (from the
`renovate` values) itself. See [values.yaml](../charts/image-inquisitor/values.yaml) for every value.

| Value | Default | Description |
|---|---|---|
| `config` | one `inCluster` cluster | The [daemon config](#daemon-config) |
| `github.token` | | GitHub token, stored in a Secret the chart creates |
| `github.existingSecret` / `existingSecretKey` | / `token` | Or an existing Secret holding it |
| `registryCredentials.existingSecret` | | A `kubernetes.io/dockerconfigjson` Secret for private registries |
| `kubeconfig.existingSecret` / `existingSecretKey` | / `config` | A kubeconfig for clusters configured by `context` |
| `renovate.*` | disabled | See [Renovate › Helm values](renovate.md#helm-values) |
| `rbac.create` | `true` | ClusterRole to read pods, workloads and Argo CD Applications |
| `daemon.image.tag` | chart `appVersion` | Daemon image tag |
| `daemon.logLevel` / `logFormat` | `info` / `json` | Logging: `error`, `warn`, `info` or `debug`; `json` or `logfmt` |
| `daemon.persistence.enabled` / `size` / `storageClass` | `true` / `10Gi` / | Data volume. Without it, data is lost on restart |
| `daemon.serviceMonitor.enabled` | `false` | Prometheus Operator ServiceMonitor; see [Monitoring](monitoring.md) |
| `daemon.env`, `resources`, `volumes`, `volumeMounts` | | Passed to the daemon container |
| `ui.enabled` | `true` | Deploy the UI |
| `ui.ingress.*` | disabled | Standard Ingress settings (`className`, `annotations`, `hosts`, `tls`) |
| `nodeSelector`, `tolerations`, `affinity` | | Daemon scheduling |

Both pods run as non-root with a read-only root filesystem (daemon) and all capabilities dropped.
