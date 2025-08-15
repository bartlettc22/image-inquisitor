package kubernetes

import (
	"context"
	"fmt"

	"github.com/bartlettc22/image-inquisitor/internal/worker"
	"github.com/bartlettc22/image-inquisitor/pkg/api/v1alpha1/sources"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type kubernetesTask struct {
	clientset *kubernetes.Clientset
	sourceID  string
	namespace string
	kind      string
	errors    []error
	result    sources.SourceList
}

func newKubernetesTask(clientset *kubernetes.Clientset, sourceID, namespace, kind string) *kubernetesTask {
	return &kubernetesTask{
		clientset: clientset,
		sourceID:  sourceID,
		namespace: namespace,
		kind:      kind,
	}
}

func (t *kubernetesTask) Run(workerID int) worker.Result {
	sources := sources.SourceList{}
	runLog := log.WithField("worker", workerID)

	switch t.kind {
	case "Pods":
		runLog.WithField("namespace", t.namespace).Debug("listing pods")
		resources, err := t.clientset.CoreV1().Pods(t.namespace).List(context.TODO(), v1.ListOptions{})
		if err != nil {
			t.errors = append(t.errors, fmt.Errorf("error listing resources in namespace %s, kind %s: %v", t.namespace, t.kind, err))
			return t
		}

		for _, resource := range resources.Items {
			// We only want to capture pods that are not owned by a regular
			// Kubernetes resource (i.e. deployment, statefulset, etc.)
			ownedByStdResource := false
			owners := resource.GetObjectMeta().GetOwnerReferences()
			for _, owner := range owners {
				if owner.Kind == "ReplicaSet" ||
					owner.Kind == "StatefulSet" ||
					owner.Kind == "DaemonSet" ||
					owner.Kind == "Job" ||
					owner.Kind == "CronJob" {
					ownedByStdResource = true
					break
				}
			}
			if ownedByStdResource {
				continue
			}

			sources = append(sources, sourceFromPodSpec(t.sourceID, t.namespace, t.kind, resource.Name, &resource.Spec, false)...)
		}
	case "Deployment":
		runLog.WithField("namespace", t.namespace).Debug("listing deployments")
		resources, err := t.clientset.AppsV1().Deployments(t.namespace).List(context.TODO(), v1.ListOptions{})
		if err != nil {
			t.errors = append(t.errors, fmt.Errorf("error listing resources in namespace %s, kind %s: %v", t.namespace, t.kind, err))
			return t
		}
		for _, resource := range resources.Items {
			sources = append(sources, sourceFromPodSpec(t.sourceID, t.namespace, t.kind, resource.Name, &resource.Spec.Template.Spec, false)...)
		}
	case "StatefulSet":
		runLog.WithField("namespace", t.namespace).Debug("listing statefulsets")
		resources, err := t.clientset.AppsV1().StatefulSets(t.namespace).List(context.TODO(), v1.ListOptions{})
		if err != nil {
			t.errors = append(t.errors, fmt.Errorf("error listing resources in namespace %s, kind %s: %v", t.namespace, t.kind, err))
			return t
		}
		for _, resource := range resources.Items {
			sources = append(sources, sourceFromPodSpec(t.sourceID, t.namespace, t.kind, resource.Name, &resource.Spec.Template.Spec, false)...)
		}
	case "DaemonSet":
		runLog.WithField("namespace", t.namespace).Debug("listing daemonsets")
		resources, err := t.clientset.AppsV1().DaemonSets(t.namespace).List(context.TODO(), v1.ListOptions{})
		if err != nil {
			t.errors = append(t.errors, fmt.Errorf("error listing resources in namespace %s, kind %s: %v", t.namespace, t.kind, err))
			return t
		}
		for _, resource := range resources.Items {
			sources = append(sources, sourceFromPodSpec(t.sourceID, t.namespace, t.kind, resource.Name, &resource.Spec.Template.Spec, false)...)
		}
	case "CronJob":
		runLog.WithField("namespace", t.namespace).Debug("listing cronjobs")
		resources, err := t.clientset.BatchV1().CronJobs(t.namespace).List(context.TODO(), v1.ListOptions{})
		if err != nil {
			t.errors = append(t.errors, fmt.Errorf("error listing resources in namespace %s, kind %s: %v", t.namespace, t.kind, err))
			return t
		}
		for _, resource := range resources.Items {
			sources = append(sources, sourceFromPodSpec(t.sourceID, t.namespace, t.kind, resource.Name, &resource.Spec.JobTemplate.Spec.Template.Spec, false)...)
		}
	case "Job":
		runLog.WithField("namespace", t.namespace).Debug("listing jobs")
		resources, err := t.clientset.BatchV1().Jobs(t.namespace).List(context.TODO(), v1.ListOptions{})
		if err != nil {
			t.errors = append(t.errors, fmt.Errorf("error listing resources in namespace %s, kind %s: %v", t.namespace, t.kind, err))
			return t
		}
		for _, resource := range resources.Items {
			sources = append(sources, sourceFromPodSpec(t.sourceID, t.namespace, t.kind, resource.Name, &resource.Spec.Template.Spec, false)...)
		}
	default:
		t.errors = append(t.errors, fmt.Errorf("unknown kind: %s", t.kind))
	}

	t.result = sources

	return t
}

func (t *kubernetesTask) Errors() []error {
	return t.errors
}

func (t *kubernetesTask) Result() interface{} {
	return t.result
}

func sourceFromPodSpec(sourceID, namespace, kind, name string, podSpec *corev1.PodSpec, isInit bool) []*sources.Source {
	sources := []*sources.Source{}
	for _, container := range podSpec.Containers {
		sources = append(sources, sourceFromContainer(sourceID, namespace, kind, name, &container, false))
	}
	for _, container := range podSpec.InitContainers {
		sources = append(sources, sourceFromContainer(sourceID, namespace, kind, name, &container, true))
	}
	return sources
}

func sourceFromContainer(sourceID, namespace, kind, name string, container *corev1.Container, isInit bool) *sources.Source {
	return &sources.Source{
		Type:           sources.KubernetesSourceType,
		ImageReference: container.Image,
		SourceID:       sourceID,
		SourceDetails: &sources.KubernetesSource{
			Namespace: namespace,
			Kind:      kind,
			Name:      name,
			Container: container.Name,
			IsInit:    isInit,
		},
	}
}
