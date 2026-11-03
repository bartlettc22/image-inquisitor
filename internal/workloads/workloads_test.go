package workloads

import (
	"testing"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func meta(namespace, name, uid string, owner *metav1.OwnerReference) metav1.ObjectMeta {
	m := metav1.ObjectMeta{Namespace: namespace, Name: name, UID: types.UID(uid)}
	if owner != nil {
		m.OwnerReferences = []metav1.OwnerReference{*owner}
	}
	return m
}

func controller(kind, name, uid string) *metav1.OwnerReference {
	isController := true
	return &metav1.OwnerReference{Kind: kind, Name: name, UID: types.UID(uid), Controller: &isController}
}

func pod(m metav1.ObjectMeta, image, imageID string) corev1.Pod {
	return corev1.Pod{
		ObjectMeta: m,
		Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "main", Image: image}}},
		Status: corev1.PodStatus{
			Phase:             corev1.PodRunning,
			ContainerStatuses: []corev1.ContainerStatus{{Name: "main", ImageID: imageID}},
		},
	}
}

func TestContainers(t *testing.T) {
	deploy := appsv1.Deployment{ObjectMeta: meta("media", "romm", "d1", nil)}
	deploy.Annotations = map[string]string{argoTrackingIDAnnotation: "romm:apps/Deployment:media/romm"}

	cron := batchv1.CronJob{ObjectMeta: meta("ops", "backup", "c1", nil)}
	cron.Spec.JobTemplate.Spec.Template.Spec.Containers = []corev1.Container{{Name: "backup", Image: "restic/restic:0.17"}}

	idle := batchv1.CronJob{ObjectMeta: meta("ops", "cleanup", "c2", nil)}
	idle.Labels = map[string]string{argoInstanceLabel: "ops"}
	idle.Spec.JobTemplate.Spec.Template.Spec.Containers = []corev1.Container{{Name: "cleanup", Image: "busybox:1.36"}}

	// An operator's StatefulSet is attributed to the custom resource that Argo CD deploys
	promOwner := controller("Prometheus", "main", "cr1")
	promOwner.APIVersion = "monitoring.coreos.com/v1"
	prom := appsv1.StatefulSet{ObjectMeta: meta("mon", "prometheus-main", "s1", promOwner)}
	prom.Labels = map[string]string{argoInstanceLabel: "prometheus"}

	finished := pod(meta("media", "old", "p9", nil), "busybox:1.36", "")
	finished.Status.Phase = corev1.PodSucceeded

	o := &objects{
		deployments:  []appsv1.Deployment{deploy},
		statefulSets: []appsv1.StatefulSet{prom},
		cronJobs:     []batchv1.CronJob{cron, idle},
		replicaSets:  []appsv1.ReplicaSet{{ObjectMeta: meta("media", "romm-abc", "rs1", controller("Deployment", "romm", "d1"))}},
		jobs:         []batchv1.Job{{ObjectMeta: meta("ops", "backup-123", "j1", controller("CronJob", "backup", "c1"))}},
		pods: []corev1.Pod{
			pod(meta("media", "romm-abc-x", "p1", controller("ReplicaSet", "romm-abc", "rs1")), "rommapp/romm:3.9.0", "docker.io/rommapp/romm@sha256:aaa"),
			pod(meta("ops", "backup-123-y", "p2", controller("Job", "backup-123", "j1")), "restic/restic:0.17", "docker-pullable://restic/restic@sha256:bbb"),
			pod(meta("default", "debug", "p3", nil), "alpine:3.20", "sha256:ccc"),
			pod(meta("mon", "prometheus-main-0", "p5", controller("StatefulSet", "prometheus-main", "s1")), "quay.io/prometheus/prometheus:v3.0.0", ""),
			pod(meta("kube-system", "coredns-x", "p4", nil), "coredns/coredns:1.13.1", ""),
			finished,
		},
	}

	containers := o.containers("bernes", Filter{ExcludeNamespaces: []string{"kube-system"}})

	assert.Equal(t, []Container{
		{Cluster: "bernes", Namespace: "media", WorkloadKind: "Deployment", WorkloadName: "romm", Pod: "romm-abc-x", Container: "main", Image: "rommapp/romm:3.9.0", Digest: "sha256:aaa", ArgoApp: "romm"},
		{Cluster: "bernes", Namespace: "ops", WorkloadKind: "CronJob", WorkloadName: "backup", Pod: "backup-123-y", Container: "main", Image: "restic/restic:0.17", Digest: "sha256:bbb"},
		{Cluster: "bernes", Namespace: "default", WorkloadKind: "Pod", WorkloadName: "debug", Pod: "debug", Container: "main", Image: "alpine:3.20"},
		{Cluster: "bernes", Namespace: "mon", WorkloadKind: "Prometheus", WorkloadName: "main", Pod: "prometheus-main-0", Container: "main", Image: "quay.io/prometheus/prometheus:v3.0.0", ArgoApp: "prometheus"},
		// backup is running, so only the idle CronJob comes from its template
		{Cluster: "bernes", Namespace: "ops", WorkloadKind: "CronJob", WorkloadName: "cleanup", Container: "cleanup", Image: "busybox:1.36", ArgoApp: "ops"},
	}, containers)
}
