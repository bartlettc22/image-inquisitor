# Image Inquisitor docs

Image Inquisitor shows which container image updates matter: the CVEs each Renovate PR fixes or
introduces, where the old version runs, and which running images have a newer version no PR covers.

| Doc | Read it to |
|---|---|
| [How it works](how-it-works.md) | Understand what the daemon syncs, scans and reports, and its current limits |
| [Install](install.md) | Deploy with Helm (or Docker), set up tokens, registries and more clusters |
| [Configuration](configuration.md) | Look up every config setting and Helm value |
| [Renovate and soak policy](renovate.md) | Have the daemon run Renovate and set its soak from CVE data |
| [Using the UI](ui.md) | Find your way around the dashboard, PRs, images, apps and status views |
| [Monitoring](monitoring.md) | Scrape `/metrics`, track CVE SLAs and alert on sync health |
| [Roadmap](roadmap.md) | See what's planned |
| [Development](development.md) | Run the local stack, tests and builds |
