package kubernetes

import (
	"context"
	"fmt"
	"sync"

	log "github.com/sirupsen/logrus"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type Kubernetes struct {
	*KubernetesConfig
	clientset *kubernetes.Clientset
}

type KubernetesConfig struct {
	IncludeNamespaces []string
	ExcludeNamespaces []string
}

func NewKubernetes(c *KubernetesConfig) (*Kubernetes, error) {

	kubeRestConfig, err := rest.InClusterConfig()
	if err != nil {
		kubeRestConfig, err = clientcmd.BuildConfigFromFlags(
			"",
			clientcmd.NewDefaultClientConfigLoadingRules().GetDefaultFilename(),
		)
		if err != nil {
			return nil, fmt.Errorf("unable to find kube config: %v", err)
		}
	}

	// Create a Kubernetes clientset
	clientset, err := kubernetes.NewForConfig(kubeRestConfig)
	if err != nil {
		return nil, fmt.Errorf("error creating Kubernetes client: %s", err.Error())
	}

	return &Kubernetes{
		KubernetesConfig: c,
		clientset:        clientset,
	}, nil
}

// GetReport retrieves all container images used by resources in all namespaces
func (k *Kubernetes) GetReport() (KubernetesReport, error) {
	report := newKubernetesReportWrapper()

	// List all namespaces
	namespaces, err := k.clientset.CoreV1().Namespaces().List(context.TODO(), v1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("error listing namespaces: %w", err)
	}

	wg := sync.WaitGroup{}
	for _, namespace := range namespaces.Items {

		if len(k.IncludeNamespaces) > 0 {
			included := false
			for _, includedNamespace := range k.IncludeNamespaces {
				if namespace.Name == includedNamespace {
					included = true
					continue
				}
			}
			if !included {
				continue
			}
		}
		if len(k.ExcludeNamespaces) > 0 {
			excluded := false
			for _, excludedNamespace := range k.ExcludeNamespaces {
				if namespace.Name == excludedNamespace {
					excluded = true
					continue
				}
			}
			if excluded {
				continue
			}
		}

		wg.Add(1)

		go func(namespace string, report *kubernetesReportWrapper) {

			log.Debugf("scanning Kubernetes namespace: %s", namespace)

			// List all pods in the current namespace
			pods, err := k.clientset.CoreV1().Pods(namespace).List(context.TODO(), v1.ListOptions{})
			if err != nil {
				log.Errorf("error listing pods in namespace %s: %v", namespace, err)
				return
			}

			// Iterate through all pods
			for _, pod := range pods.Items {

				// We only want to capture pods that are not owned by a regular
				// Kubernetes resource (i.e. deployment, statefulset, etc.)
				ownedByStdResource := false
				owners := pod.GetObjectMeta().GetOwnerReferences()
				for _, owner := range owners {
					if owner.Kind == "ReplicaSet" ||
						owner.Kind == "StatefulSet" ||
						owner.Kind == "DaemonSet" ||
						owner.Kind == "Job" {
						ownedByStdResource = true
						break
					}
				}
				if ownedByStdResource {
					continue
				}

				// Iterate through all init containers
				for _, container := range pod.Spec.InitContainers {
					report.Add(container.Image, "Pod", &pod)
					pod.GetObjectKind()
				}

				// Iterate through all regular containers
				for _, container := range pod.Spec.Containers {
					report.Add(container.Image, "Pod", &pod)
				}
			}

			// List all deployments in the current namespace
			deployments, err := k.clientset.AppsV1().Deployments(namespace).List(context.TODO(), v1.ListOptions{})
			if err != nil {
				log.Errorf("error listing deployments in namespace %s: %v", namespace, err)
				return
			}

			// Iterate through all deployments
			for _, deployment := range deployments.Items {

				// Iterate through all init containers
				for _, container := range deployment.Spec.Template.Spec.InitContainers {
					report.Add(container.Image, "Deployment", &deployment)
				}

				// Iterate through all regular containers
				for _, container := range deployment.Spec.Template.Spec.Containers {
					report.Add(container.Image, "Deployment", &deployment)
				}
			}

			// List all daemonsets in the current namespace
			daemonsets, err := k.clientset.AppsV1().DaemonSets(namespace).List(context.TODO(), v1.ListOptions{})
			if err != nil {
				log.Errorf("error listing daemonsets in namespace %s: %v", namespace, err)
				return
			}

			// Iterate through all daemonsets
			for _, daemonset := range daemonsets.Items {

				// Iterate through all init containers
				for _, container := range daemonset.Spec.Template.Spec.InitContainers {
					report.Add(container.Image, "DaemonSet", &daemonset)
				}

				// Iterate through all regular containers
				for _, container := range daemonset.Spec.Template.Spec.Containers {
					report.Add(container.Image, "DaemonSet", &daemonset)
				}
			}

			// List all statefulsets in the current namespace
			statefulsets, err := k.clientset.AppsV1().StatefulSets(namespace).List(context.TODO(), v1.ListOptions{})
			if err != nil {
				log.Errorf("error listing statefulsets in namespace %s: %v", namespace, err)
				return
			}

			// Iterate through all statefulsets
			for _, statefulset := range statefulsets.Items {

				// Iterate through all init containers
				for _, container := range statefulset.Spec.Template.Spec.InitContainers {
					report.Add(container.Image, "StatefulSet", &statefulset)
				}

				// Iterate through all regular containers
				for _, container := range statefulset.Spec.Template.Spec.Containers {
					report.Add(container.Image, "StatefulSet", &statefulset)
				}
			}

			// List all cronjobs in the current namespace
			cronjobs, err := k.clientset.BatchV1().CronJobs(namespace).List(context.TODO(), v1.ListOptions{})
			if err != nil {
				log.Errorf("error listing cronjobs in namespace %s: %v", namespace, err)
				return
			}

			// Iterate through all cronjobs
			for _, cronjob := range cronjobs.Items {

				// Iterate through all init containers
				for _, container := range cronjob.Spec.JobTemplate.Spec.Template.Spec.InitContainers {
					report.Add(container.Image, "CronJob", &cronjob)
				}

				// Iterate through all regular containers
				for _, container := range cronjob.Spec.JobTemplate.Spec.Template.Spec.Containers {
					report.Add(container.Image, "CronJob", &cronjob)
				}
			}

			// List all jobs in the current namespace
			jobs, err := k.clientset.BatchV1().Jobs(namespace).List(context.TODO(), v1.ListOptions{})
			if err != nil {
				log.Errorf("error listing jobs in namespace %s: %v", namespace, err)
				return
			}

			// Iterate through all jobs
			for _, job := range jobs.Items {

				// We only want to capture jobs that are not owned by a Cronjob
				ownedByCronjob := false
				owners := job.GetObjectMeta().GetOwnerReferences()
				for _, owner := range owners {
					if owner.Kind == "CronJob" {
						ownedByCronjob = true
						break
					}
				}
				if ownedByCronjob {
					continue
				}

				// Iterate through all init containers
				for _, container := range job.Spec.Template.Spec.InitContainers {
					// nsImages = append(nsImages, container.Image)
					report.Add(container.Image, "Job", &job)
				}

				// Iterate through all regular containers
				for _, container := range job.Spec.Template.Spec.Containers {
					// nsImages = append(nsImages, container.Image)
					report.Add(container.Image, "Job", &job)
				}
			}

			log.Debugf("DONE scanning Kubernetes namespace: %s", namespace)
			wg.Done()
		}(namespace.Name, report)
	}

	wg.Wait()

	return report.GetReport(), nil
}
