# Using the UI

The UI has five views (Dashboard, Running images, Pull requests, Argo CD apps, History), linked from
the header on every page, and two dialogs opened from it (Status and Configuration). It refreshes
itself; there's nothing to reload.
Click an image's name anywhere to open its [page](#image-page).

## Dashboard

The landing page answers "what should I do next?":

- **Running images** by [update status](how-it-works.md#update-statuses), with their CVEs and what
  updating them to the newest version would fix.
- **Open PRs** that fix or introduce critical and high CVEs, and the CVEs fixed by open PRs.
- **Merge first:** open PRs ranked by the critical, then high, then medium CVEs they fix.
- **Not covered by Renovate:** running images with a newer version and no open PR, most-used first.
- **Vulnerability trends** (with clusters configured): open and fixable CVEs, CVEs past their
  [SLA](monitoring.md#cve-slas), CVEs opened vs fixed per week, and days to fix by severity.

## Running images

Every image running in the inventoried clusters, filterable by update status and by image,
workload or Argo CD app. Each row shows how many workloads run the image (the cube icon) and, with
several clusters tracked, which clusters (hover for the list, or open the
[image page](#image-page)), its newest version, the open PR updating it (flagged when the PR
updates a different version than the one running), its CVEs, and, with a
[soak policy](renovate.md#soak-policy), the soak lane and progress of its update.

## Image page

Everything known about one image repository, also for an image that's only in a PR:

- For each version running, three panels show what's **running now**, what an **open PR** updates
  it to and the **latest available**. With a [soak policy](renovate.md#soak-policy) and no PR yet,
  a progress bar shows how long until Renovate may propose the update; the PR itself comes on
  Renovate's next run after that. An image pinned by a chart gets no PR of its own, so it points to
  the chart instead. Below them:
  - **What this means** explains its update status, the size of the update and its soak.
  - **Vulnerabilities** compares its CVEs with the latest version's.
  - **Where it runs** lists its workloads, grouped by what deploys them, with the PR updating each
    group where that's known. A version deployed several ways may need more than one update.
  - **CVE details** lists its CVEs, most severe first, and what updating would fix and introduce.
- **Open pull requests** changing a version that isn't running, directly or through a chart.
- **History:** every time the image has been in a cluster, including versions gone since.

## Pull requests

Every tracked Renovate PR, filterable by change kind (images, charts, or no recognised change) and
by name, title or file. Tick **Show closed** for merged and closed PRs.

A PR's page shows:

- **Changes:** each image and chart bump, from → to, and which workloads run the old version.
- **Vulnerabilities:** CVEs fixed and introduced, per image and in total.
- **Chart image changes:** for chart bumps, the images that change when each app installing the
  chart is rendered at the new version.
- **Pipeline:** each processing stage (extract, render, scan) and its status or error.

## Argo CD apps

Each Argo CD Application's sources, their render status, and running containers labelled as the
app that no rendered source declares. Tick **Needs attention only** to show only apps with a
chart that's behind, a render error, or such containers.

## History

Every container [recorded](how-it-works.md#image-history) in the clusters, kept indefinitely, to
answer questions like "was this digest ever here?" during an incident. Filter by image, digest,
namespace, workload, pod or node, and set **Present at** to see what was there at a point in time.
Click a digest to see everywhere it has run.

The top of the page shows whether each cluster is being recorded, and any gaps. A container that
ran during a gap may be missing, so absence during one proves nothing. `ephemeral` marks a
container added with `kubectl debug`, and **never started** an image that failed to run.

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
