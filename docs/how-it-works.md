# How it works

Image Inquisitor runs as two pieces:

- **The daemon** syncs on an interval, scans images in the background and serves
  a JSON API and Prometheus metrics. Its state is one SQLite database.
- **The UI** is a static web app served by nginx, which proxies `/api` to the daemon.

## The sync loop

Every `syncInterval` (10 minutes by default) the daemon runs these steps. Each step's duration,
summary and error show in the UI's [Status dialog](ui.md#status-dialog).

1. **Repos.** Expands repo patterns (`owner/*`) to the owner's active repos, re-listed hourly.
2. **Pull requests.** Reads each repo's open PRs and keeps those whose head branch starts with
   `renovate/` (configurable). Renovate PRs are matched by branch, not author, so a self-hosted
   Renovate posting as any account is found. The old → new image tags and Helm chart versions
   are read from each PR's diff (skipped while the head commit is unchanged).
3. **Clusters.** Reads every pod in each configured cluster: the digest each container runs,
   the top-level workload that owns it, and its Argo CD Application. Idle CronJob templates are
   included. Workloads run by an operator are attributed to the custom resource Argo CD deploys
   (for example a `Prometheus`).
4. **Argo CD apps.** Renders each Application's Helm sources with `helm template` the way Argo CD
   does (value files from `$ref` sources, inline values, parameters, release name, namespace, kube
   version, API versions) at the revisions Argo CD last synced. Renders are cached by their inputs.
   Running containers are matched to the chart that declares them.
5. **Chart PRs.** For each PR that bumps a chart, every app installing the old version is
   rendered at the new version with the PR's values, so the PR shows the images it changes.
6. **Latest versions.** Lists the tags of every running image and the versions of every chart
   from a Helm or OCI repository (re-listed every 6 hours) and finds the newest of the same
   variant. See [update statuses](#update-statuses).
7. **Soak policy.** Only with [Renovate soak](renovate.md#soak-policy) on: dates each newest
   version and picks its soak lane.
8. **Findings.** Records which CVEs run in each container, opening and closing findings. This is
   the history behind the trends, time-to-fix and SLA numbers.

## Recognised PR diffs

| Shape | Example |
|---|---|
| Image in a values file | `image: ghcr.io/example/app:1.2.3` |
| Helm repo chart in an Argo CD Application | `repoURL` + `chart` + `targetRevision` |
| OCI chart in an Argo CD Application | `repoURL: oci://…` + `targetRevision` |
| Dockerfile base image | `FROM node:22-alpine` |

Renovate reports OCI chart bumps as Docker tag updates; Image Inquisitor treats them as charts.

## Update statuses

Each running image is compared with the newest tag of the **same variant**: the same `v` prefix,
precision and suffix, so `16.2-alpine` is only compared with other `<major>.<minor>-alpine` tags.

| Status | Meaning |
|---|---|
| **Pending** | An open PR updates it |
| **Current** | It runs the newest version |
| **Pinned** | A newer image exists, but a chart from a Helm or OCI repository sets the tag |
| **Behind** | A newer version exists and no PR updates it: a Renovate coverage gap |
| **Unknown** | The tag isn't a version, or the registry's tags couldn't be listed |

## Scanning

A background scan loop, separate from sync, works through:

1. both sides of every PR image change (including images a chart update changes),
2. the digest each running container runs,
3. the newest version of each running tag that has a newer one.

Each digest gets a CycloneDX SBOM once (`trivy image`, pulled straight from the registry, no Docker
daemon needed). The SBOM is then checked against Trivy's current vulnerability database with
`trivy sbom`, and again every 24 hours, so re-checks don't pull images. Scans of the same digest
are shared between PRs and running images.

The results show as CVEs **fixed** and **introduced** by severity for each image change and PR
(a CVE shared by several images in one PR counts once), CVEs in each running image, and what
updating it to the newest version would fix and introduce.

Set `scan.disabled: true` to turn scanning off.

## Image history

Separately from the sync loop, the daemon watches each cluster's pods live and records every
container it sees: the image and digest, the pod, node and workload, and when it started and ended.
Init containers and `kubectl debug` ephemeral containers are included, and so are pods that only
live for seconds, or that never start. History is kept indefinitely, in the same SQLite database,
and follows the same `includeNamespaces` and `excludeNamespaces` as the inventory.

- **Entries:** one per container and digest in each pod. A restart with the same digest stays the
  same entry; a different digest in the same pod, such as a mutable tag re-pulled, starts a new one.
- **Times** come from Kubernetes where it reports them, so pods already running when the daemon
  starts keep their real start time. The time the daemon first saw each container is kept too.
- **Coverage:** the daemon records when it was watching each cluster. A daemon restart, a failed
  watch or a failed write ends a stretch of coverage, and the UI lists the gaps.

What it can't tell you:

- A pod that came and went during a gap leaves no trace, so absence during a gap proves nothing.
  One that outlived the gap is recorded, and if it ended during one it's marked **not observed**.
- If a watch expires or drops mid-stream, Kubernetes clients re-list without reporting an error.
  That's usually over in seconds, but a pod that appeared and vanished in that moment isn't
  recorded, and shows as a gap only if the re-list also found another pod missing.
- History starts when the daemon first runs with this feature. It's a record for investigating,
  not tamper-proof evidence: anyone who can reach the daemon's volume can change it.
- Containers with no registry digest (a locally built image, say) are recorded without one.

## Limits

- Only `linux/amd64` images are scanned, and the daemon image is built only for `linux/amd64`.
- Values files that split an image into `repository:` and `tag:`, and Kustomize `images:`, aren't
  read from PR diffs yet.
- Only Helm sources of Argo CD apps are rendered; Kustomize and plain-manifest sources aren't.
  Git charts and value files are read from GitHub only. Apps are assumed to deploy to the cluster
  they're defined in.
- Images an operator sets by default (rather than from its custom resource) have no chart source.
- CronJobs that haven't run have no running digest, so they aren't scanned.
- Images a newer chart version would bring in aren't scanned until a PR bumps the chart.
- Private registries need Docker credentials; there's no other registry auth.
- Results aren't posted back to the PRs yet; see the [roadmap](roadmap.md).
