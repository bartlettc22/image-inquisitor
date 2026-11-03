# Image Inquisitor

Image Inquisitor shows which container image updates matter. It follows the Renovate PRs in your
GitOps repos and the images running in your clusters, scans both sides of every update with Trivy,
and shows the CVEs each PR fixes or introduces, where the old version runs, and which running
images have a newer version that no PR covers.

It runs as two pieces:

- **The daemon** syncs PRs and clusters on an interval, scans images in the
  background, and serves a JSON API. Its state is a SQLite database.
- **The UI** is a static web app, served by nginx, that proxies `/api` to the daemon.

## How it works

- **Pull requests:** Renovate PRs are found by their head branch prefix (`renovate/` by default).
  Image tags and Helm chart versions are read from each PR's diff. Chart updates are rendered with
  `helm template` at both versions to find the images they change.
- **Running images:** pods are read from each cluster and attributed to their workloads and Argo CD
  Applications. Each running tag is compared with the newest tag of the same variant in its registry.
- **Renovate (optional):** the daemon runs Renovate itself as Kubernetes Jobs, so it controls when it runs
  and syncs as soon as a run finishes.
- **Scanning:** each image digest gets a Trivy SBOM once; the SBOM is re-checked against Trivy's
  vulnerability database every 24 hours, so re-scans don't pull images.

## Installing with Helm

The chart runs the daemon as a single-replica StatefulSet (one SQLite database on a persistent
volume) and the UI as a Deployment. By default the daemon reads the cluster it's installed in.

```bash
kubectl create secret generic image-inquisitor-github --from-literal=token=<github token>

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

helm install image-inquisitor oci://ghcr.io/bartlettc22/charts/image-inquisitor -f values.yaml
```

The GitHub token needs read access to the repos' pull requests and contents. See
[Install](docs/install.md) for private registries, more clusters, Prometheus and running without
Kubernetes.

## Documentation

- [How it works](docs/how-it-works.md): what gets synced, scanned and reported, and current limits
- [Install](docs/install.md): Helm, tokens, registries, clusters, Docker
- [Configuration](docs/configuration.md): every config setting and Helm value
- [Renovate and soak policy](docs/renovate.md): let the daemon run Renovate and set its soak from CVEs
- [Using the UI](docs/ui.md): dashboard, PRs, images, apps and the status views
- [Monitoring](docs/monitoring.md): metrics, CVE SLAs and alerts
- [Roadmap](docs/roadmap.md): what's planned
- [Development](docs/development.md): local stack, tests and builds
