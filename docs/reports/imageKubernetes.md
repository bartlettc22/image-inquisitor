# imageKubernetes

```json
{
    "reportGenerated": "2024-09-05T17:22:23.625444979-06:00",
    "reportType": "imageKubernetes",
    "image": "quay.io/prometheus-operator/prometheus-config-reloader:v0.75.2",
    "report": {
        "resources": [
            {
                "kind": "StatefulSet",
                "namespace": "prometheus",
                "name": "prometheus-prometheus-kube-prometheus-prometheus",
                "container": "init-config-reloader",
                "isInit": true
            },
            {
                "kind": "StatefulSet",
                "namespace": "prometheus",
                "name": "prometheus-prometheus-kube-prometheus-prometheus",
                "container": "config-reloader",
                "isInit": false
            }
        ]
    }
}
```
