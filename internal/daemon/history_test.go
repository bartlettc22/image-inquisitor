package daemon

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/bartlettc22/image-inquisitor/internal/store"
	"github.com/bartlettc22/image-inquisitor/internal/workloads"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
)

func ownedBy(kind, name string) []metav1.OwnerReference {
	controller := true
	return []metav1.OwnerReference{{Kind: kind, Name: name, Controller: &controller}}
}

func testPod(name string, phase corev1.PodPhase) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "web", UID: types.UID("uid-" + name)},
		Spec:       corev1.PodSpec{NodeName: "node-1", Containers: []corev1.Container{{Name: "app", Image: "example.com/app:1"}}},
		Status:     corev1.PodStatus{Phase: phase},
	}
}

func runningStatus(digest string, startedAt time.Time) []corev1.ContainerStatus {
	return []corev1.ContainerStatus{{
		Name: "app", ImageID: "example.com/app@" + digest,
		State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{StartedAt: metav1.NewTime(startedAt)}},
	}}
}

type watchTest struct {
	t      *testing.T
	store  *store.Store
	client *fake.Clientset
}

// startWatch runs the watcher against a fake cluster until the test ends
func startWatch(t *testing.T, s *store.Store, config ClusterConfig, objects ...*corev1.Pod) *watchTest {
	t.Helper()
	client := fake.NewClientset()
	for _, p := range objects {
		_, err := client.CoreV1().Pods(p.Namespace).Create(context.Background(), p, metav1.CreateOptions{})
		require.NoError(t, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	t.Cleanup(func() {
		cancel()
		<-done
	})
	d := &Daemon{store: s}
	go func() {
		defer close(done)
		d.watchCluster(ctx, &cluster{config: config, client: client})
	}()

	// Coverage starts once the pods are listed and the watch is on
	require.Eventually(t, func() bool {
		coverage, err := s.ListCoverage(context.Background())
		return err == nil && len(coverage) > 0
	}, 5*time.Second, 5*time.Millisecond)
	return &watchTest{t, s, client}
}

func historyDB(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	require.NoError(t, err)
	t.Cleanup(func() { s.Close() })
	return s
}

func (w *watchTest) waitFor(what string, check func(entries []store.HistoryEntry) bool) []store.HistoryEntry {
	w.t.Helper()
	var entries []store.HistoryEntry
	require.Eventually(w.t, func() bool {
		var err error
		entries, err = w.store.ListHistory(context.Background(), store.HistoryQuery{})
		return err == nil && check(entries)
	}, 5*time.Second, 5*time.Millisecond, what)
	return entries
}

func TestWatchRecordsShortLivedPod(t *testing.T) {
	ctx := context.Background()
	w := startWatch(t, historyDB(t), ClusterConfig{Name: "home"})
	pods := w.client.CoreV1().Pods("web")

	// Created, ran and was deleted well inside one sync interval
	pod := testPod("quick", corev1.PodPending)
	_, err := pods.Create(ctx, pod, metav1.CreateOptions{})
	require.NoError(t, err)
	started := time.Now().Add(-time.Second).Truncate(time.Second).UTC()
	pod.Status.Phase, pod.Status.ContainerStatuses = corev1.PodRunning, runningStatus("sha256:quick", started)
	_, err = pods.UpdateStatus(ctx, pod, metav1.UpdateOptions{})
	require.NoError(t, err)
	require.NoError(t, pods.Delete(ctx, pod.Name, metav1.DeleteOptions{}))

	entries := w.waitFor("the pod ends", func(e []store.HistoryEntry) bool { return len(e) == 1 && !e[0].EndedAt.IsZero() })
	assert.Equal(t, "sha256:quick", entries[0].Digest)
	assert.Equal(t, "example.com/app", entries[0].Repository)
	assert.Equal(t, "node-1", entries[0].Node)
	assert.Equal(t, started, entries[0].StartedAt)
	assert.Equal(t, store.EndReasonDeleted, entries[0].EndReason)
}

func TestWatchRecordsDebugContainer(t *testing.T) {
	ctx := context.Background()
	pod := testPod("target", corev1.PodRunning)
	pod.Status.ContainerStatuses = runningStatus("sha256:app", time.Now())
	w := startWatch(t, historyDB(t), ClusterConfig{Name: "home"}, pod)
	w.waitFor("the existing pod", func(e []store.HistoryEntry) bool { return len(e) == 1 })

	// kubectl debug adds an ephemeral container to a running pod
	pod.Spec.EphemeralContainers = []corev1.EphemeralContainer{{EphemeralContainerCommon: corev1.EphemeralContainerCommon{Name: "debugger", Image: "evil.example.com/tools:1"}}}
	pod.Status.EphemeralContainerStatuses = []corev1.ContainerStatus{{
		Name: "debugger", ImageID: "evil.example.com/tools@sha256:evil",
		State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{StartedAt: metav1.Now()}},
	}}
	_, err := w.client.CoreV1().Pods("web").Update(ctx, pod, metav1.UpdateOptions{})
	require.NoError(t, err)

	entries := w.waitFor("the debug container", func(e []store.HistoryEntry) bool { return len(e) == 2 })
	assert.Equal(t, "debugger", entries[0].Container)
	assert.Equal(t, workloads.ContainerTypeEphemeral, entries[0].ContainerType)
	assert.Equal(t, "sha256:evil", entries[0].Digest)
}

func TestWatchRespectsNamespaceFilter(t *testing.T) {
	skipped, kept := testPod("skipped", corev1.PodRunning), testPod("kept", corev1.PodRunning)
	skipped.Namespace = "kube-system"
	w := startWatch(t, historyDB(t), ClusterConfig{Name: "home", ExcludeNamespaces: []string{"kube-system"}}, skipped, kept)

	entries := w.waitFor("the allowed pod", func(e []store.HistoryEntry) bool { return len(e) > 0 })
	require.Len(t, entries, 1)
	assert.Equal(t, "kept", entries[0].Pod)
}

func TestWatchEndsPodsGoneDuringDowntime(t *testing.T) {
	ctx := context.Background()
	s := historyDB(t)
	before := time.Now().Add(-time.Hour).UTC()
	require.NoError(t, s.ObservePod(ctx, []store.HistoryEntry{
		{Cluster: "home", PodUID: "uid-gone", Pod: "gone", Container: "app", Image: "example.com/app:1", Digest: "sha256:a", StartedAt: before},
		{Cluster: "home", PodUID: "uid-stays", Pod: "stays", Container: "app", Image: "example.com/app:1", Digest: "sha256:a", StartedAt: before},
		{Cluster: "other", PodUID: "uid-elsewhere", Pod: "elsewhere", Container: "app", Image: "example.com/app:1", Digest: "sha256:a", StartedAt: before},
	}, before))

	stays := testPod("stays", corev1.PodRunning)
	stays.Status.ContainerStatuses = runningStatus("sha256:a", before)
	startWatch(t, s, ClusterConfig{Name: "home"}, stays)

	entries, err := s.ListHistory(ctx, store.HistoryQuery{})
	require.NoError(t, err)
	reasons := map[string]string{}
	for _, e := range entries {
		reasons[e.Pod] = e.EndReason
	}
	assert.Equal(t, map[string]string{"gone": store.EndReasonUnobserved, "stays": "", "elsewhere": ""}, reasons)
}

func TestPodWorkloadThroughOwners(t *testing.T) {
	ctx := context.Background()
	client := fake.NewClientset(&appsv1.ReplicaSet{ObjectMeta: metav1.ObjectMeta{
		Name: "app-5d9", Namespace: "web", OwnerReferences: ownedBy("Deployment", "app"),
	}})
	s := historyDB(t)
	cctx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	defer func() {
		cancel()
		<-done
	}()
	pod := testPod("app-5d9-x", corev1.PodRunning)
	pod.OwnerReferences = ownedBy("ReplicaSet", "app-5d9")
	_, err := client.CoreV1().Pods("web").Create(ctx, pod, metav1.CreateOptions{})
	require.NoError(t, err)

	go func() {
		defer close(done)
		(&Daemon{store: s}).watchCluster(cctx, &cluster{config: ClusterConfig{Name: "home"}, client: client})
	}()
	require.Eventually(t, func() bool {
		entries, err := s.ListHistory(ctx, store.HistoryQuery{})
		return err == nil && len(entries) == 1 && entries[0].WorkloadKind == "Deployment" && entries[0].WorkloadName == "app"
	}, 5*time.Second, 5*time.Millisecond)
}

func TestTrackCoverage(t *testing.T) {
	s := historyDB(t)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	defer func() {
		cancel()
		<-done
	}()
	r := &podRecorder{store: s, cluster: "home"}
	id, err := s.StartCoverage(ctx, "home", time.Now().UTC())
	require.NoError(t, err)
	d := &Daemon{store: s}
	go func() {
		defer close(done)
		d.trackCoverage(ctx, r, id, 20*time.Millisecond)
	}()

	stretches := func() int {
		coverage, err := s.ListCoverage(context.Background())
		require.NoError(t, err)
		return len(coverage)
	}
	time.Sleep(60 * time.Millisecond)
	assert.Equal(t, 1, stretches(), "a healthy watch keeps one stretch")

	// A failed watch ends the stretch; a new one starts once things are clean again
	r.failedAt.Store(time.Now().Unix())
	assert.Eventually(t, func() bool { return stretches() == 2 }, 5*time.Second, 10*time.Millisecond)
}

func TestMissedDeletionCountsAsLostEvents(t *testing.T) {
	ctx := context.Background()
	s := historyDB(t)
	r := &podRecorder{store: s, cluster: "home", lookup: func(string, string, string) *metav1.ObjectMeta { return nil }}

	pod := testPod("missed", corev1.PodRunning)
	pod.Status.ContainerStatuses = runningStatus("sha256:a", time.Now())
	r.observe(ctx, pod)
	assert.Zero(t, r.failedAt.Load())

	r.deleted(ctx, cache.DeletedFinalStateUnknown{Key: "web/missed", Obj: pod})
	assert.NotZero(t, r.failedAt.Load(), "ends the stretch of coverage")

	entries, err := s.ListHistory(ctx, store.HistoryQuery{})
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, store.EndReasonUnobserved, entries[0].EndReason)
}
