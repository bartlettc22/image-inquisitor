# Monitoring

## Endpoints

| Path | Description |
|---|---|
| `/healthz` | Liveness and readiness (daemon and UI) |
| `/metrics` | Prometheus metrics (daemon) |

With Helm, set `daemon.serviceMonitor.enabled: true` to create a Prometheus Operator
ServiceMonitor (default interval 60s, timeout 30s). Add the label your Prometheus selects
ServiceMonitors by under `daemon.serviceMonitor.labels`.

Metrics are read from the database at scrape time, so they survive restarts and match the UI.
Prometheus keeps the history.

## Metrics

### Vulnerabilities

| Metric | Labels | Description |
|---|---|---|
| `imginq_open_findings` | `severity`, `fixable` | CVEs running in workload containers, counted once per container |
| `imginq_findings_past_sla` | `severity` | Open CVEs with a fix available for longer than their severity's [SLA](#cve-slas) |
| `imginq_sla_seconds` | `severity` | Each severity's SLA |
| `imginq_findings_fixed_total` | `severity` | CVEs removed from a container by an image update |
| `imginq_time_to_fix_seconds` | `severity` | Histogram: from a fix being available (or the CVE first seen) to an update removing it. Buckets from 1 to 365 days |

### Inventory and PRs

| Metric | Labels | Description |
|---|---|---|
| `imginq_running_images` | `status` | Running images by [update status](how-it-works.md#update-statuses): `pending`, `current`, `pinned`, `behind`, `unknown` |
| `imginq_open_prs` | `repo` | Open Renovate PRs |
| `imginq_cluster_containers` | `cluster` | Containers in the cluster's inventory |

### Health

| Metric | Labels | Description |
|---|---|---|
| `imginq_last_sync_timestamp_seconds` | | Start of the last finished sync |
| `imginq_last_sync_duration_seconds` | | Duration of the last finished sync |
| `imginq_last_sync_success` | | 1 if every step of the last sync succeeded |
| `imginq_cluster_last_sync_timestamp_seconds` | `cluster` | Last inventory of the cluster |
| `imginq_cluster_last_sync_success` | `cluster` | 1 if the cluster's last inventory succeeded |
| `imginq_scan_queue` | `stage` | References waiting to `resolve`, `generate` (an SBOM) or `check` |
| `imginq_renovate_last_run_timestamp_seconds` | | Start of the last finished Renovate run |
| `imginq_renovate_last_run_success` | | 1 if the last Renovate run succeeded |

Go runtime and process metrics are also exported.

## CVE SLAs

A finding is a CVE running in one container. It opens when a scan finds the CVE in what the
container runs and closes when an update removes it. A finding is **past SLA** when a fixed version
has been available for longer than its severity allows:

| Severity | SLA |
|---|---|
| Critical | 7 days |
| High | 30 days |
| Medium | 90 days |
| Low | 180 days |

These are fixed for now. Clusters whose inventory fails are skipped, so a failed sync isn't
mistaken for everything being fixed.

## Example alerts

```yaml
groups:
- name: image-inquisitor
  rules:
  - alert: ImginqSyncFailing
    expr: imginq_last_sync_success == 0
    for: 1h
  - alert: ImginqSyncStale
    expr: time() - imginq_last_sync_timestamp_seconds > 3600
  - alert: ImginqClusterInventoryFailing
    expr: imginq_cluster_last_sync_success == 0
    for: 1h
  - alert: ImginqCriticalPastSLA
    expr: imginq_findings_past_sla{severity="CRITICAL"} > 0
  - alert: ImginqRenovateFailing
    expr: imginq_renovate_last_run_success == 0
    for: 12h
```

## Logs

The daemon logs in logfmt by default, JSON with the chart (`daemon.logFormat`). Set
`daemon.logLevel: debug` for detail. Each sync step's result is also
kept in the [Status dialog](ui.md#status-dialog).
