package daemon

import (
	"context"
	"strings"
	"sync/atomic"
	"time"

	"github.com/bartlettc22/image-inquisitor/internal/store"
	"github.com/bartlettc22/image-inquisitor/internal/workloads"
	"github.com/google/go-containerregistry/pkg/name"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
)

// historyHeartbeat is how often the watch's coverage is extended, so it's also how far the end
// of a stretch of coverage can trail the moment the daemon stopped
const historyHeartbeat = time.Minute

// podRecorder writes the pods of one cluster to the image history
type podRecorder struct {
	store   *store.Store
	cluster string
	filter  workloads.Filter
	lookup  func(kind, namespace, name string) *metav1.ObjectMeta

	// failedAt is the Unix time of the last failed watch or write: events may have been missed
	failedAt atomic.Int64
}

func (r *podRecorder) failed(err error, what string) {
	r.failedAt.Store(time.Now().Unix())
	svcLog.WithField("cluster", r.cluster).WithError(err).Error(what)
}

func (r *podRecorder) entries(pod *corev1.Pod) []store.HistoryEntry {
	kind, workload := workloads.PodWorkload(pod, r.lookup)
	containers := workloads.PodContainers(pod)
	entries := make([]store.HistoryEntry, 0, len(containers))
	for _, c := range containers {
		entries = append(entries, store.HistoryEntry{
			Cluster: r.cluster, Namespace: pod.Namespace, WorkloadKind: kind, WorkloadName: workload,
			Pod: pod.Name, PodUID: string(pod.UID), Node: pod.Spec.NodeName,
			Container: c.Name, ContainerType: c.Type, Image: c.Image, Repository: imageRepository(c.Image), Digest: c.Digest,
			ArgoApp: workloads.PodArgoApp(pod), StartedAt: c.StartedAt, EndedAt: c.FinishedAt,
		})
	}
	return entries
}

// imageRepository returns a reference's normalized repository, or "" if it isn't a valid one
func imageRepository(image string) string {
	base, _, _ := strings.Cut(image, "@")
	ref, err := name.ParseReference(base, name.WeakValidation)
	if err != nil {
		return ""
	}
	return ref.Context().Name()
}

// observe records a pod's current state, ending what's left of it once it has finished
func (r *podRecorder) observe(ctx context.Context, pod *corev1.Pod) {
	if !r.filter.Allows(pod.Namespace) {
		return
	}
	now := time.Now().UTC()
	if err := r.store.ObservePod(ctx, r.entries(pod), now); err != nil {
		r.failed(err, "error recording pod in image history")
		return
	}
	if workloads.PodFinished(pod) {
		if err := r.store.EndPod(ctx, r.cluster, string(pod.UID), now, store.EndReasonStopped); err != nil {
			r.failed(err, "error ending pod in image history")
		}
	}
}

// deleted records a pod's last known state and ends it
func (r *podRecorder) deleted(ctx context.Context, obj any) {
	reason := store.EndReasonDeleted
	// The watch missed the deletion and a relist found the pod gone. A watch that expires doesn't
	// reach the watch error handler, so this is the only sign that events were lost.
	if tombstone, ok := obj.(cache.DeletedFinalStateUnknown); ok {
		obj, reason = tombstone.Obj, store.EndReasonUnobserved
		r.failedAt.Store(time.Now().Unix())
	}
	pod, ok := obj.(*corev1.Pod)
	if !ok || !r.filter.Allows(pod.Namespace) {
		return
	}
	r.observe(ctx, pod)
	if err := r.store.EndPod(ctx, r.cluster, string(pod.UID), time.Now().UTC(), reason); err != nil {
		r.failed(err, "error ending pod in image history")
	}
}

// watchCluster records every pod of a cluster in the image history as it changes, until ctx is
// cancelled, and keeps track of the time it covered
func (d *Daemon) watchCluster(ctx context.Context, cl *cluster) {
	logger := svcLog.WithField("cluster", cl.config.Name)
	factory := informers.NewSharedInformerFactory(cl.client, 0)
	replicaSets := factory.Apps().V1().ReplicaSets()
	jobs := factory.Batch().V1().Jobs()

	r := &podRecorder{
		store:   d.store,
		cluster: cl.config.Name,
		filter:  workloads.Filter{IncludeNamespaces: cl.config.IncludeNamespaces, ExcludeNamespaces: cl.config.ExcludeNamespaces},
		lookup: func(kind, namespace, name string) *metav1.ObjectMeta {
			switch kind {
			case "ReplicaSet":
				if rs, err := replicaSets.Lister().ReplicaSets(namespace).Get(name); err == nil {
					return &rs.ObjectMeta
				}
			case "Job":
				if job, err := jobs.Lister().Jobs(namespace).Get(name); err == nil {
					return &job.ObjectMeta
				}
			}
			return nil
		},
	}
	onWatchError := func(_ *cache.Reflector, err error) { r.failed(err, "error watching cluster") }
	for _, informer := range []cache.SharedIndexInformer{replicaSets.Informer(), jobs.Informer()} {
		if err := informer.SetWatchErrorHandler(onWatchError); err != nil {
			logger.WithError(err).Error("error setting watch error handler")
		}
	}

	// Owners first, so the pods listed at startup are attributed to their workloads
	factory.Start(ctx.Done())
	if !cache.WaitForCacheSync(ctx.Done(), replicaSets.Informer().HasSynced, jobs.Informer().HasSynced) {
		return
	}

	pods := factory.Core().V1().Pods()
	if err := pods.Informer().SetWatchErrorHandler(onWatchError); err != nil {
		logger.WithError(err).Error("error setting watch error handler")
	}
	registration, err := pods.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj any) {
			if pod, ok := obj.(*corev1.Pod); ok {
				r.observe(ctx, pod)
			}
		},
		UpdateFunc: func(_, obj any) {
			if pod, ok := obj.(*corev1.Pod); ok {
				r.observe(ctx, pod)
			}
		},
		DeleteFunc: func(obj any) { r.deleted(ctx, obj) },
	})
	if err != nil {
		logger.WithError(err).Error("error watching pods, image history is off")
		return
	}
	factory.Start(ctx.Done())
	if !cache.WaitForCacheSync(ctx.Done(), registration.HasSynced) {
		return
	}

	// Pods that went away while the daemon was down never sent a delete
	listedAt := time.Now().UTC()
	live := map[string]bool{}
	for _, obj := range pods.Informer().GetStore().List() {
		if pod, ok := obj.(*corev1.Pod); ok {
			live[string(pod.UID)] = true
		}
	}
	if n, err := d.store.EndMissingPods(ctx, r.cluster, live, listedAt, listedAt); err != nil {
		r.failed(err, "error ending missing pods in image history")
	} else if n > 0 {
		logger.WithField("pods", n).Info("ended pods that went away while the daemon was down")
	}

	coverage, err := d.store.StartCoverage(ctx, r.cluster, listedAt)
	if err != nil {
		r.failed(err, "error recording image history coverage")
	}
	logger.Info("watching pods for image history")
	d.trackCoverage(ctx, r, coverage, historyHeartbeat)
}

// trackCoverage extends a stretch of coverage until ctx is cancelled. After a failed watch or
// write events may be lost, so the stretch stops there and a new one starts after a clean interval.
func (d *Daemon) trackCoverage(ctx context.Context, r *podRecorder, coverage int64, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	var err error
	last := time.Now().Unix()
	down := coverage == 0
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		now := time.Now().UTC()
		if r.failedAt.Load() >= last {
			down = true
		} else if down {
			if coverage, err = d.store.StartCoverage(ctx, r.cluster, now); err == nil {
				down = false
			}
		} else {
			err = d.store.TouchCoverage(ctx, coverage, now)
		}
		if err != nil {
			r.failed(err, "error recording image history coverage")
			err = nil
		}
		last = now.Unix()
	}
}
