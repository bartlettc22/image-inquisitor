# Image Inquisitor

<big>Creates a report combining CVE data (using Trivy) with lastest image information.  Meant to be used by Loki to produce a dashboard to make it easier to prioritize upgrades</big>

<img src="image-inquisitor.png " alt="drawing" width="500" />

```
Run tool to get list of images
Pipe into trivy
combine results
```

```yaml
started: 2024-1-2
completed: 2024-1-2
- image: foo/bar
  currentTag: v1.2.3
  currentTagDate: 
  currentTagAge: 
  latestTag: v2.3.4
  usage:
  - namespace: foo
    kind: Deployment
    name: bar
  cves:
    low: 2
    medium: 2
    high: 2
    critical: 2
```
