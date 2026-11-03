// Package workloads lists the container images running in a Kubernetes cluster,
// attributed to their top-level workload (Deployment, StatefulSet, CronJob, ...)
package workloads

import (
	"context"
	"fmt"
	"slices"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

const (
	argoTrackingIDAnnotation = "argocd.argoproj.io/tracking-id"
	argoInstanceLabel        = "app.kubernetes.io/instance"
)

// Container is one container of a workload, as running in a pod or,
// for CronJobs with no running job, as declared in the job template
type Container struct {
	Cluster      string `json:"cluster"`
	Namespace    string `json:"namespace"`
	WorkloadKind string `json:"workloadKind"`
	WorkloadName string `json:"workloadName"`

	// Pod is empty for containers taken from a CronJob template
	Pod       string `json:"pod,omitempty"`
	Container string `json:"container"`
	Init      bool   `json:"init,omitempty"`

	// Image is the image reference from the pod spec
	Image string `json:"image"`

	// Digest is the digest of the running image, from the container status, if known
	Digest string `json:"digest,omitempty"`

	// ArgoApp is the Argo CD application that manages the workload, if any
	ArgoApp string `json:"argoApp,omitempty"`
}

type Filter struct {
	IncludeNamespaces []string
	ExcludeNamespaces []string
}

func (f Filter) allows(namespace string) bool {
	if len(f.IncludeNamespaces) > 0 && !slices.Contains(f.IncludeNamespaces, namespace) {
		return false
	}
	return !slices.Contains(f.ExcludeNamespaces, namespace)
}

// objects is everything Collect lists, kept separate so containers can be built without a cluster
type objects struct {
	pods         []corev1.Pod
	replicaSets  []appsv1.ReplicaSet
	deployments  []appsv1.Deployment
	statefulSets []appsv1.StatefulSet
	daemonSets   []appsv1.DaemonSet
	jobs         []batchv1.Job
	cronJobs     []batchv1.CronJob
}

// Collect lists every container in the cluster that the filter allows
func Collect(ctx context.Context, client kubernetes.Interface, cluster string, filter Filter) ([]Container, error) {
	o := &objects{}
	all := metav1.ListOptions{}

	pods, err := client.CoreV1().Pods("").List(ctx, all)
	if err != nil {
		return nil, fmt.Errorf("error listing pods: %w", err)
	}
	o.pods = pods.Items

	replicaSets, err := client.AppsV1().ReplicaSets("").List(ctx, all)
	if err != nil {
		return nil, fmt.Errorf("error listing replicasets: %w", err)
	}
	o.replicaSets = replicaSets.Items

	deployments, err := client.AppsV1().Deployments("").List(ctx, all)
	if err != nil {
		return nil, fmt.Errorf("error listing deployments: %w", err)
	}
	o.deployments = deployments.Items

	statefulSets, err := client.AppsV1().StatefulSets("").List(ctx, all)
	if err != nil {
		return nil, fmt.Errorf("error listing statefulsets: %w", err)
	}
	o.statefulSets = statefulSets.Items

	daemonSets, err := client.AppsV1().DaemonSets("").List(ctx, all)
	if err != nil {
		return nil, fmt.Errorf("error listing daemonsets: %w", err)
	}
	o.daemonSets = daemonSets.Items

	jobs, err := client.BatchV1().Jobs("").List(ctx, all)
	if err != nil {
		return nil, fmt.Errorf("error listing jobs: %w", err)
	}
	o.jobs = jobs.Items

	cronJobs, err := client.BatchV1().CronJobs("").List(ctx, all)
	if err != nil {
		return nil, fmt.Errorf("error listing cronjobs: %w", err)
	}
	o.cronJobs = cronJobs.Items

	return o.containers(cluster, filter), nil
}

type workload struct {
	kind    string
	name    string
	argoApp string
}

func (o *objects) containers(cluster string, filter Filter) []Container {
	// Top-level workloads and the intermediate owners (ReplicaSet, Job) that point at them, by UID.
	// A workload controlled by a custom resource (an operator's Prometheus, say) is attributed
	// to that resource, which is what Argo CD deploys.
	top := map[types.UID]workload{}
	for _, d := range o.deployments {
		top[d.UID] = customOwnerOr(d.ObjectMeta, "Deployment")
	}
	for _, s := range o.statefulSets {
		top[s.UID] = customOwnerOr(s.ObjectMeta, "StatefulSet")
	}
	for _, d := range o.daemonSets {
		top[d.UID] = customOwnerOr(d.ObjectMeta, "DaemonSet")
	}
	for _, c := range o.cronJobs {
		top[c.UID] = customOwnerOr(c.ObjectMeta, "CronJob")
	}

	intermediate := map[types.UID]workload{}
	for _, rs := range o.replicaSets {
		intermediate[rs.UID] = ownerOrSelf(rs.ObjectMeta, "ReplicaSet", top)
	}
	for _, j := range o.jobs {
		intermediate[j.UID] = ownerOrSelf(j.ObjectMeta, "Job", top)
	}

	containers := []Container{}
	runningCronJobs := map[string]bool{}

	for _, pod := range o.pods {
		if !filter.allows(pod.Namespace) || pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
			continue
		}

		w := workload{"Pod", pod.Name, argoApp(pod.ObjectMeta)}
		if owner := metav1.GetControllerOf(&pod); owner != nil {
			if iw, ok := intermediate[owner.UID]; ok {
				w = iw
			} else if tw, ok := top[owner.UID]; ok {
				w = tw
			} else {
				w = workload{owner.Kind, owner.Name, w.argoApp}
			}
		}
		if w.kind == "CronJob" {
			runningCronJobs[pod.Namespace+"/"+w.name] = true
		}

		digests := map[string]string{}
		for _, s := range append(slices.Clone(pod.Status.InitContainerStatuses), pod.Status.ContainerStatuses...) {
			digests[s.Name] = digest(s.ImageID)
		}

		add := func(c corev1.Container, init bool) {
			containers = append(containers, Container{
				Cluster:      cluster,
				Namespace:    pod.Namespace,
				WorkloadKind: w.kind,
				WorkloadName: w.name,
				Pod:          pod.Name,
				Container:    c.Name,
				Init:         init,
				Image:        c.Image,
				Digest:       digests[c.Name],
				ArgoApp:      w.argoApp,
			})
		}
		for _, c := range pod.Spec.InitContainers {
			add(c, true)
		}
		for _, c := range pod.Spec.Containers {
			add(c, false)
		}
	}

	// CronJobs without a running job still count, from their template
	for _, cj := range o.cronJobs {
		if !filter.allows(cj.Namespace) || runningCronJobs[cj.Namespace+"/"+cj.Name] {
			continue
		}
		spec := cj.Spec.JobTemplate.Spec.Template.Spec
		add := func(c corev1.Container, init bool) {
			containers = append(containers, Container{
				Cluster:      cluster,
				Namespace:    cj.Namespace,
				WorkloadKind: "CronJob",
				WorkloadName: cj.Name,
				Container:    c.Name,
				Init:         init,
				Image:        c.Image,
				ArgoApp:      argoApp(cj.ObjectMeta),
			})
		}
		for _, c := range spec.InitContainers {
			add(c, true)
		}
		for _, c := range spec.Containers {
			add(c, false)
		}
	}

	return containers
}

// customOwnerOr returns the custom resource controlling a workload, or the workload itself
func customOwnerOr(meta metav1.ObjectMeta, kind string) workload {
	if ref := metav1.GetControllerOfNoCopy(&meta); ref != nil {
		if group, _, _ := strings.Cut(ref.APIVersion, "/"); strings.Contains(group, ".") {
			return workload{ref.Kind, ref.Name, argoApp(meta)}
		}
	}
	return workload{kind, meta.Name, argoApp(meta)}
}

// ownerOrSelf returns the top-level workload controlling an object, or the object itself
func ownerOrSelf(meta metav1.ObjectMeta, kind string, top map[types.UID]workload) workload {
	for _, ref := range meta.OwnerReferences {
		if ref.Controller != nil && *ref.Controller {
			if w, ok := top[ref.UID]; ok {
				return w
			}
		}
	}
	return workload{kind, meta.Name, argoApp(meta)}
}

// argoApp returns the Argo CD application from the tracking annotation
// ("<app>:<group>/<kind>:<namespace>/<name>") or the instance label
func argoApp(meta metav1.ObjectMeta) string {
	if id := meta.Annotations[argoTrackingIDAnnotation]; id != "" {
		app, _, _ := strings.Cut(id, ":")
		return app
	}
	return meta.Labels[argoInstanceLabel]
}

// digest extracts the manifest digest from a container status imageID,
// e.g. "docker.io/library/nginx@sha256:..." or "docker-pullable://nginx@sha256:..."
func digest(imageID string) string {
	if _, d, ok := strings.Cut(imageID, "@"); ok {
		return d
	}
	return ""
}
