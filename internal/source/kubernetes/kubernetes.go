package kubernetes

import (
	"context"
	"errors"
	"fmt"

	"github.com/bartlettc22/image-inquisitor/internal/worker"
	sourcesapi "github.com/bartlettc22/image-inquisitor/pkg/api/v1alpha1/sources"
	log "github.com/sirupsen/logrus"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var svcLog = log.WithField("service", "source-kubernetes")

const (
	DefaultNumWorkers = 6
)

type KubernetesSourceGenerator struct {
	*KubernetesSourceGeneratorConfig
	clientset  *kubernetes.Clientset
	workerPool *worker.EphemeralWorkerPool
}

type KubernetesSourceGeneratorConfig struct {
	SourceID          string
	NumWorkers        int
	IncludeNamespaces []string
	ExcludeNamespaces []string
}

func NewKubernetesSourceGenerator(c *KubernetesSourceGeneratorConfig) (*KubernetesSourceGenerator, error) {

	if c == nil {
		return nil, fmt.Errorf("kubernetes source generator config must be specified")
	}

	if c.SourceID == "" {
		return nil, fmt.Errorf("sourceID must be specified")
	}

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

	if c.NumWorkers == 0 {
		c.NumWorkers = DefaultNumWorkers
	}

	workerPool := worker.NewEphemeralWorkerPool(worker.EphemeralWorkerPoolConfig{
		NumWorkers: c.NumWorkers,
	})

	return &KubernetesSourceGenerator{
		KubernetesSourceGeneratorConfig: c,
		clientset:                       clientset,
		workerPool:                      workerPool,
	}, nil
}

// GetReport retrieves all container images used by resources in all namespaces
func (k *KubernetesSourceGenerator) Generate() (sourcesapi.SourceList, error) {

	svcLog.Info("generating Kubernetes sources")

	// List all namespaces
	svcLog.Debug("listing Kubernetes namespaces")
	namespaces, err := k.clientset.CoreV1().Namespaces().List(context.TODO(), v1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("error listing namespaces: %w", err)
	}

	for _, namespace := range namespaces.Items {
		if !isNamespaceEligible(namespace.Name, k.IncludeNamespaces, k.ExcludeNamespaces) {
			svcLog.Debugf("skipping Kubernetes namespace: %s", namespace.Name)
			continue
		}

		k.workerPool.AddTask(newKubernetesTask(k.clientset, k.SourceID, namespace.Name, "Pods"))
		k.workerPool.AddTask(newKubernetesTask(k.clientset, k.SourceID, namespace.Name, "Deployment"))
		k.workerPool.AddTask(newKubernetesTask(k.clientset, k.SourceID, namespace.Name, "StatefulSet"))
		k.workerPool.AddTask(newKubernetesTask(k.clientset, k.SourceID, namespace.Name, "DaemonSet"))
		k.workerPool.AddTask(newKubernetesTask(k.clientset, k.SourceID, namespace.Name, "CronJob"))
		k.workerPool.AddTask(newKubernetesTask(k.clientset, k.SourceID, namespace.Name, "Job"))
	}
	k.workerPool.Done()

	sources := sourcesapi.SourceList{}
	errs := []error{}

	// Blocks until all results are in
	for result := range k.workerPool.ResultChan() {
		if resultErrs := result.Errors(); len(errs) != 0 {
			errs = append(errs, resultErrs...)
			continue
		}
		sources = append(sources, result.Result().(sourcesapi.SourceList)...)
	}

	return sources, errors.Join(errs...)
}

func isNamespaceEligible(namespace string, includeNamespaces []string, excludeNamespaces []string) bool {
	if len(includeNamespaces) > 0 {
		included := false
		for _, includedNamespace := range includeNamespaces {
			if namespace == includedNamespace {
				included = true
				break
			}
		}
		if !included {
			return false
		}
	}
	if len(excludeNamespaces) > 0 {
		excluded := false
		for _, excludedNamespace := range excludeNamespaces {
			if namespace == excludedNamespace {
				excluded = true
				break
			}
		}
		if excluded {
			return false
		}
	}

	return true
}
