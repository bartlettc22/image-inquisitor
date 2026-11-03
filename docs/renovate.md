# Renovate and soak policy

Image Inquisitor works with any Renovate setup: it only reads the PRs Renovate opens. Optionally,
the daemon can **run Renovate itself**, which gets you:

- new PRs in the UI as soon as a run finishes (the daemon syncs straight after),
- runs on an interval or on demand from the UI, never overlapping,
- a run history with each run's log tail in the Status dialog,
- a [soak policy](#soak-policy) set from the CVEs each update fixes and introduces.

Renovate still finds and edits the files. Image Inquisitor decides when it runs and with what
settings.

## How runs work

Each run is a Kubernetes Job (`imginq-renovate-*`) in the configured cluster and namespace:

- Renovate's global config is passed in `RENOVATE_CONFIG`, with `repositories` set to the
  daemon's [repos](configuration.md#repos) (patterns expanded) and `autodiscover` off.
  `platform` defaults to `github`.
- One run at a time, also checked against Jobs left running (for example across a daemon restart).
- A run is stopped after `timeout`. Finished Jobs are kept for 24 hours.
- An optional PersistentVolumeClaim is mounted as Renovate's cache so runs don't start from
  scratch.

Trigger a run from the header's Status dialog (all repos) or the Configuration dialog (one repo).

If you already run Renovate as a CronJob, suspend it once you've moved to this, so the two
don't race. A `dryRun: full` in `config` is a safe first test.

## Helm values

```bash
kubectl -n image-inquisitor create secret generic renovate-github --from-literal=token=<token>
```

```yaml
renovate:
  enabled: true
  interval: 6h            # empty to run only from the UI
  github:
    existingSecret: renovate-github
  config:                 # Renovate's global config
    onboarding: false
  soak:                   # optional; see below
    default: 14 days
    major: 21 days
```

Renovate's token needs **write** access to the repos' contents, pull requests and issues. It's
separate from the daemon's read-only token. The chart passes it as `RENOVATE_TOKEN` and
`RENOVATE_GITHUB_COM_TOKEN`.

With `renovate.enabled`, the chart also creates a namespaced Role that lets the daemon create Jobs
and read their pods' logs in the release namespace, the token Secret (from `github.token`), and a
cache PVC.

| Value | Default | Description |
|---|---|---|
| `enabled` | `false` | Run Renovate from the daemon |
| `image.repository` / `tag` | `ghcr.io/renovatebot/renovate` / pinned | Renovate image |
| `interval` | `6h` | Time between scheduled runs (Go duration); empty for UI-only |
| `timeout` | `1h` | Longest a run may take |
| `cluster` | `in-cluster` | Configured cluster the Jobs run in |
| `config` | `{}` | Renovate's global config |
| `soak` | `{}` | [Soak policy](#soak-policy); empty keeps Renovate's own |
| `github.token` / `existingSecret` / `existingSecretKey` | | Renovate's write token |
| `env` | `[]` | More env, e.g. registry credentials for `detectHostRulesFromEnv` |
| `resources`, `dnsConfig`, `nodeSelector` | | Passed to the Renovate pod |
| `cache.enabled` / `size` / `storageClass` | `true` / `5Gi` / | Cache PVC |

## Configuration

Without Helm, configure the `renovate` section of the daemon config directly. `env`,
`resources` and `dnsConfig` use Kubernetes pod spec types, so Secrets work as usual.

| Key | Default | Description |
|---|---|---|
| `cluster` | the only cluster | Configured cluster to run Jobs in; required with several |
| `namespace` | required | Namespace of the Jobs and the Secrets their env refers to |
| `image` | required | Renovate image, e.g. `ghcr.io/renovatebot/renovate:44.181.6` |
| `interval` | unset | Time between scheduled runs; unset for UI-only |
| `timeout` | `1h` | Longest a run may take |
| `config` | `{}` | Renovate's global config |
| `env` | `[]` | Container env; must include `RENOVATE_TOKEN` |
| `resources`, `dnsConfig`, `nodeSelector` | | Passed to the pod |
| `cacheClaim` | unset | PVC in the namespace mounted as Renovate's cache |
| `soak` | unset | [Soak policy](#soak-policy) |

The daemon owns some settings and refuses to start if you set them yourself:

- `config.repositories`, `config.autodiscover`, `config.autodiscoverFilter`, and the matching
  `RENOVATE_*` env vars or `RENOVATE_CONFIG`: the repos come from `repos`.
- `config.minimumReleaseAge` when `soak` is set: it comes from `soak.default`.

The daemon's service account needs `create`, `get` and `list` on `jobs`, `list` on `pods` and `get`
on `pods/log` in that namespace.

## Soak policy

Renovate's `minimumReleaseAge` holds an update until the new version has been out for a while,
so broken releases get caught first. Two problems with using it alone:

- Renovate only knows release dates for Docker Hub images. Images from GHCR, Quay, ECR and others
  are held **indefinitely**.
- Every update waits the same time, even one that fixes a critical CVE you're running.

With `soak` set, the daemon becomes the soak clock for the newest version of each running image:

1. **Dates it** from its build date in the registry, or from when the daemon first saw it for
   reproducible builds (dated 1970).
2. **Picks a lane** from the CVEs the update fixes and introduces in what's running:

   | Lane | When | Soak |
   |---|---|---|
   | **Risky** | It introduces more CVEs at `riskySeverity` or worse than it fixes | `risky` |
   | **Fast** | A minor or patch update that fixes more CVEs at `fastLaneSeverity` or worse than it introduces | `fastLane` |
   | **Major** | A major update (otherwise) | `major` |
   | **Default** | Everything else | `default` |

3. **Tells Renovate** on each run, with `packageRules` matching that exact version: no soak once it
   has soaked, or the risky soak if it's risky. Rules go after any `packageRules` in `config`, so
   they win.

Running images show each update's lane, reason and soak progress in the UI; the Configuration
dialog shows the settings.

| Key | Default | Description |
|---|---|---|
| `default` | required | Soak of every update |
| `major` | `default` | Soak of major updates |
| `fastLane` | `3 days` | Soak of fast-lane updates; short rather than none, so broken releases still get caught |
| `fastLaneSeverity` | `CRITICAL` | `CRITICAL`, `HIGH`, `MEDIUM`, `LOW`, or `NONE` to turn the fast lane off |
| `risky` | twice the longer of `default` and `major` | Soak of risky updates |
| `riskySeverity` | `CRITICAL` | As above; `NONE` turns the risky lane off |

Ages use Renovate's format: `14 days`, `12 hours`, `2 weeks`.

### Limits

- Only the **newest** version of each running image is dated. Other updates Renovate proposes
  (a minor while a major exists, say) keep Renovate's own soak, so they stay held for registries
  without release dates.
- `packageRules` in a repo's own `renovate.json` may override the generated rules.
- Until an update's newest version has been scanned, it soaks in the default or major lane.
