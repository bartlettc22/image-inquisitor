package kubernetes

import (
	"sync"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type KubernetesReport map[string]*KubernetesImageReport

type kubernetesReportWrapper struct {
	mu     *sync.Mutex
	report KubernetesReport
}

func newKubernetesReportWrapper() *kubernetesReportWrapper {

	return &kubernetesReportWrapper{
		mu:     &sync.Mutex{},
		report: make(KubernetesReport),
	}
}

func (k *kubernetesReportWrapper) Add(image string, kind string, o KubernetesResourceObject) {
	k.mu.Lock()
	defer k.mu.Unlock()
	if _, ok := k.report[image]; !ok {
		k.report[image] = &KubernetesImageReport{}
	}

	k.report[image].Resources = append(k.report[image].Resources, &KubernetesResource{
		Kind:      kind,
		Namespace: o.GetObjectMeta().GetNamespace(),
		Name:      o.GetObjectMeta().GetName(),
	})
}

func (k *kubernetesReportWrapper) GetReport() KubernetesReport {
	return k.report
}

type KubernetesImageReport struct {
	Resources []*KubernetesResource
}

type KubernetesResourceObject interface {
	GetObjectMeta() metav1.Object
}

type KubernetesResource struct {
	Kind      string
	Namespace string
	Name      string
}
