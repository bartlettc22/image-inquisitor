# Using the UI

The UI has four views (Dashboard, Pull requests, Running images, Argo CD apps) and two dialogs
opened from the header (Status and Configuration). It refreshes itself; there's nothing to reload.

## Dashboard

The landing page answers "what should I do next?":

- **Running images** by [update status](how-it-works.md#update-statuses), with their CVEs and what
  updating them to the newest version would fix.
- **Open PRs** that fix or introduce critical and high CVEs, and the CVEs fixed by open PRs.
- **Argo CD apps:** out of sync, not healthy, and needing attention (a chart that's behind, a
  render error, or running containers no source declares).
- **Merge first:** open PRs ranked by the critical, then high, then medium CVEs they fix.
- **Not covered by Renovate:** running images with a newer version and no open PR, most-used first.
- **Vulnerability trends** (with clusters configured): open and fixable CVEs, CVEs past their
  [SLA](monitoring.md#cve-slas), CVEs opened vs fixed per week, and days to fix by severity.

## Pull requests

Every tracked Renovate PR, filterable by change kind (images, charts, or no recognised change) and
by name, title or file. Tick **Show closed** for merged and closed PRs.

A PR's page shows:

- **Changes:** each image and chart bump, from → to, and which workloads run the old version.
- **Vulnerabilities:** CVEs fixed and introduced, per image and in total.
- **Chart image changes:** for chart bumps, the images that change when each app installing the
  chart is rendered at the new version.
- **Pipeline:** each processing stage (extract, render, scan) and its status or error.

## Running images

Every image running in the inventoried clusters, filterable by update status and by image,
workload or Argo CD app. Each row shows where the image runs, its newest version, the open PR
updating it (flagged when the PR updates a different version than the one running), its CVEs,
and, with a [soak policy](renovate.md#soak-policy), the soak lane and progress of its update.

## Argo CD apps

Each Argo CD Application's sources, their render status, and running containers labelled as the
app that no rendered source declares. Tick **Needs attention only** to show only apps with a
chart that's behind, a render error, or such containers.

## Status dialog

Click the sync status in the header. It shows:

- **Sync loop:** what's running now, the next sync, and a **Sync now** button.
- **Scan loop:** the current scan and the queue, split by kind (PR images, running digests, newest
  versions).
- **Caches:** the size of each cache, and the Trivy database's publish, download and next-update
  times.
- **Registry tag lists** and any failures listing them.
- **Recent syncs:** the last 50, with each step's duration, summary and error. Start here when
  something looks missing.
- **Renovate** (when the daemon runs it): the current run, recent runs with their log tails, and
  **Run Renovate**.

## Configuration dialog

A read-only view of the running config: the GitHub repos (with each pattern's current matches and
a per-repo **Run Renovate** button), the Kubernetes clusters, and settings such as the sync
interval, scanning and the soak policy.
