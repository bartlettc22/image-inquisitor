# Install

The Helm chart is the supported way to deploy. It runs:

- the **daemon** as a single-replica StatefulSet (its SQLite database lives on a
  `ReadWriteOnce` volume, so it never scales past one),
- the **UI** as a Deployment, with an optional Ingress.

By default the daemon inventories the cluster it's installed in.

## Requirements

- Kubernetes, and Helm 3.8+ (for OCI charts).
- A GitHub token that can read the tracked repos' pull requests and contents. A fine-grained token
  needs **Pull requests: read** and **Contents: read** on those repos.
- Outbound access from the daemon to the GitHub API, the registries your images come from, the
  Helm and OCI repositories your Argo CD apps use, and Trivy's vulnerability database
  (`mirror.gcr.io` / `ghcr.io`).
- About 10 GB of persistent storage. Trivy's database alone takes about 2 GB.

## Quick start

```bash
kubectl create namespace image-inquisitor
kubectl -n image-inquisitor create secret generic image-inquisitor-github \
  --from-literal=token=<github token>

cat > values.yaml <<'EOF'
config:
  repos:
  - name: example-org/deploy
github:
  existingSecret: image-inquisitor-github
ui:
  ingress:
    enabled: true
    hosts:
    - host: image-inquisitor.example.com
      paths:
      - path: /
        pathType: Prefix
EOF

helm install image-inquisitor oci://ghcr.io/bartlettc22/charts/image-inquisitor \
  -n image-inquisitor -f values.yaml
```

Without an Ingress, reach the UI with
`kubectl -n image-inquisitor port-forward svc/image-inquisitor-ui 8080:80`.

The first sync starts straight away. Scans follow in the background; the first one waits for
Trivy's database to download.

> **The UI and API have no authentication.** Anyone who can reach them can see your repos,
> workloads and CVEs, and trigger syncs and Renovate runs. Put the Ingress behind your own auth
> (for example an OAuth proxy) or keep it on a private network.

## Common setups

### Track every repo of an org or user

```yaml
config:
  repos:
  - name: example-org/*
    exclude: [example-org/docs, example-org/sandbox-*]
```

Patterns match repos that aren't archived or forks, re-listed hourly. See
[Configuration › repos](configuration.md#repos).

### Private registries

Create a `kubernetes.io/dockerconfigjson` Secret and point the chart at it. It's used to list tags
and to scan images.

```bash
kubectl -n image-inquisitor create secret docker-registry imginq-registries \
  --docker-server=registry.example.com --docker-username=<user> --docker-password=<password>
```

```yaml
registryCredentials:
  existingSecret: imginq-registries
```

### More clusters

Store a kubeconfig in a Secret and list each cluster by context. Setting
`config.kubernetes.clusters` replaces the chart's default list, so keep the `inCluster` entry if
you still want the local cluster.

```bash
kubectl -n image-inquisitor create secret generic imginq-kubeconfig --from-file=config=./kubeconfig
```

```yaml
kubeconfig:
  existingSecret: imginq-kubeconfig
config:
  kubernetes:
    clusters:
    - inCluster: true
      name: home
    - context: lab
      excludeNamespaces: [kube-system]
```

The credentials for each context need read access (`get`, `list`) to the resources below, and
`watch` on `pods`, `replicasets` and `jobs` for the [image history](how-it-works.md#image-history):

| API group | Resources |
|---|---|
| core | `pods` |
| `apps` | `replicasets`, `deployments`, `statefulsets`, `daemonsets` |
| `batch` | `jobs`, `cronjobs` |
| `argoproj.io` | `applications` |

The chart creates this ClusterRole for the local cluster (`rbac.create: true`).

### Run Renovate from the daemon

See [Renovate and soak policy](renovate.md).

### Prometheus

```yaml
daemon:
  serviceMonitor:
    enabled: true
    labels:
      release: kube-prometheus-stack   # whatever your Prometheus selects on
```

See [Monitoring](monitoring.md) for the metrics.

## Upgrading

```bash
helm upgrade image-inquisitor oci://ghcr.io/bartlettc22/charts/image-inquisitor \
  -n image-inquisitor -f values.yaml
```

Config changes roll the daemon automatically (the pod template carries a checksum of the config).
Data survives upgrades on the persistent volume.

From v0.4.0 the chart's version matches the app's. Chart 0.4.0 replaces the earlier CronJob chart, which ran the removed `run`/`export`
commands. None of its values carry over, so write new values from the [reference](configuration.md#helm-values).

## Without Kubernetes

The daemon and UI are plain containers. The daemon reads its config from
`/etc/imginq/config.yaml` and keeps its data in `/var/lib/imginq`.

```bash
docker network create imginq
docker run -d --name imginq --network imginq \
  -e GITHUB_TOKEN=<github token> \
  -v ./config.yaml:/etc/imginq/config.yaml:ro \
  -v ~/.kube/config:/kube/config:ro -e KUBECONFIG=/kube/config \
  -v imginq-data:/var/lib/imginq \
  ghcr.io/bartlettc22/image-inquisitor:v0.5.1
docker run -d --name imginq-ui --network imginq -p 8081:8080 \
  -e API_UPSTREAM=http://imginq:8080 \
  ghcr.io/bartlettc22/image-inquisitor-ui:v0.5.1
```

The daemon runs as UID 1001, so mounted files must be readable by it. For private registries,
mount a Docker `config.json` and set `DOCKER_CONFIG` to its directory. The
[development stack](development.md#local-stack) is a working example.
