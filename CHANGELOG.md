# v0.5.1
Fixed:
* Renovate failing on every run with `EACCES` on `/cache/containerbase`: it couldn't write to its cache volume

# v0.5.0
Adds a page for every image, and a permanent record of every image that runs in your clusters. See [Using the UI](docs/ui.md).

Upgrading:
* The daemon now needs `watch` on `pods`, `replicasets` and `jobs`. The chart's ClusterRole includes it; clusters reached by kubeconfig need it [added](docs/install.md). Without it the History view shows the cluster as never recorded, and everything else keeps working
* History adds to the database for as long as it runs, with no retention limit yet, so keep an eye on the volume's size

Added:
* Image history: watches pods live and records every container with its image, digest, pod, node and workload, and when it started and ended. Includes init containers, `kubectl debug` ephemeral containers, pods that live for seconds and images that never started. Kept indefinitely
* History view: search by image, digest, namespace, workload, pod or node, or see what was present at a point in time, with each cluster's recording status and gaps
* Image page: click an image's name anywhere for what's running, the open PR and the latest version, plain explanations of its status, update size and soak, a before and after CVE comparison, where it runs and its history
* Soak progress on the image page, showing how long until Renovate may propose an update, and a warning when a soak finished long ago with no PR
* Workloads grouped by what deploys them on the image page, with a "deployed N ways" note when a version comes from several sources
* CVEs per digest when one tag runs as several digests
* Workload counts in Running images, and which clusters run each image when there are several
* The Trivy version in the Configuration dialog
* `GET /api/v1/images/{repository}` and `GET /api/v1/history`

Changed:
* The views are linked from the header on every page, with Running images before Pull requests
* The landing tab is called Dashboard, and no longer has an Argo CD apps panel
* The UI uses one sans-serif font throughout, dropping the bundled serif

# v0.4.0
A rewrite: image-inquisitor is now a long-running daemon with a web UI that follows your Renovate PRs and running images. See the [docs](docs/README.md).

Breaking:
* Removed the `run` and `export` commands, their reports (including Loki and `dashboard.json`) and the Docker Hub mirroring flags
* Helm chart 0.4.0 (its version now matches the app's) replaces the CronJob chart: a daemon StatefulSet (SQLite on a persistent volume) and a UI Deployment. No values carry over

Added:
* Tracks open Renovate PRs (by `renovate/` branch prefix) in configured repos, or every repo of an owner with `owner/*` patterns, and reads the image and Helm chart bumps from their diffs
* Inventories running containers per cluster (in-cluster or by kubeconfig context), attributed to their workload and Argo CD Application
* Renders Argo CD apps' Helm sources with `helm template` to attribute images to charts and to find the images a chart bump changes
* Finds the newest version of the same variant for every running image and chart, and flags images that are behind with no PR
* Scans with Trivy by digest (an SBOM once, re-checked daily): CVEs each PR fixes and introduces, CVEs in running images, and what updating them would fix
* Web UI: dashboard, PRs, running images, Argo CD apps, vulnerability trends, and sync/scan status
* Prometheus `/metrics`: open CVEs, CVEs past SLA, time to fix, update status, sync, scan and Renovate health; optional ServiceMonitor
* Optional Renovate runs as Kubernetes Jobs from the daemon, with a soak policy set from each update's CVEs

# v0.3.0
* Added Helm chart
* Added argument `--enable-dockerhub-mirror` to enable docker.io mirroring (default is false)
* Added argument `--dockerhub-mirror` to specify `docker.io` mirror to use (default is `mirror.gcr.io`)
* Added `identifier` field to image report(s).  This is digest if it was referenced by digest, otherwise it will be the tag.
* Removed report format `simplified-json` (was not actually simplified at all)
* Fixed error message when no latest semver digest is found (now only logs a debug message)
* Formatting Kubernetes client-side throttling logs more nicely

# v0.2.0
* Major refactor of code and command line arguments

# v0.1.0
* Initial release
