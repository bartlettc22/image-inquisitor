package daemon

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bartlettc22/image-inquisitor/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

const renovateConfigYAML = `
repos:
- name: example/deploy
- name: example/webapp
kubernetes:
  clusters:
  - context: lab
renovate:
  namespace: renovate
  image: ghcr.io/renovatebot/renovate:44.181.6
  interval: 6h
  config:
    minimumReleaseAge: 14 days
    packageRules:
    - matchUpdateTypes: [major]
      minimumReleaseAge: 21 days
  env:
  - name: RENOVATE_TOKEN
    valueFrom:
      secretKeyRef:
        name: renovate-github
        key: token
  resources:
    requests:
      memory: 1Gi
  dnsConfig:
    options:
    - name: ndots
      value: "0"
  cacheClaim: renovate-cache
`

func TestLoadRenovateConfig(t *testing.T) {
	c, err := loadConfig(t, renovateConfigYAML)
	require.NoError(t, err)
	r := c.Renovate
	require.NotNil(t, r)
	assert.Equal(t, "lab", r.Cluster, "defaults to the only cluster")
	assert.Equal(t, 6*time.Hour, r.RunEvery())
	assert.Equal(t, time.Hour, r.timeout)
	assert.Equal(t, "renovate-github", r.Env[0].ValueFrom.SecretKeyRef.Name)
	assert.Equal(t, resource.MustParse("1Gi"), r.Resources.Requests[corev1.ResourceMemory])
	assert.Equal(t, "ndots", r.DNSConfig.Options[0].Name)

	tests := []struct {
		name, replace, with, wantErr string
	}{
		{"typo", "cacheClaim:", "cacheClam:", "unknown field"},
		{"reserved env", "name: RENOVATE_TOKEN", "name: RENOVATE_AUTODISCOVER", "would override the repos"},
		{"reserved config", "minimumReleaseAge: 14 days", "autodiscover: true", "config.autodiscover is set by the daemon"},
		{"unknown cluster", "namespace: renovate", "namespace: renovate\n  cluster: elsewhere", `cluster "elsewhere" isn't one`},
		{"no image", "image: ghcr.io/renovatebot/renovate:44.181.6", "", "image is required"},
		{"bad interval", "interval: 6h", "interval: daily", "invalid interval"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := loadConfig(t, strings.Replace(renovateConfigYAML, tt.replace, tt.with, 1))
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestRenovateJob(t *testing.T) {
	c, err := loadConfig(t, renovateConfigYAML)
	require.NoError(t, err)

	job, err := c.Renovate.job(7, []string{"example/deploy"}, nil)
	require.NoError(t, err)
	assert.Equal(t, "renovate", job.Namespace)
	assert.Equal(t, "7", job.Labels[renovateRunLabel])
	assert.Equal(t, int64(3600), *job.Spec.ActiveDeadlineSeconds)
	assert.Equal(t, int32(0), *job.Spec.BackoffLimit)

	pod := job.Spec.Template.Spec
	assert.Equal(t, "7", job.Spec.Template.Labels[renovateRunLabel])
	assert.False(t, *pod.AutomountServiceAccountToken)
	assert.Equal(t, "renovate-cache", pod.Volumes[0].PersistentVolumeClaim.ClaimName)
	assert.Equal(t, renovateFSGroup, *pod.SecurityContext.FSGroup, "a new cache volume is root's, and Renovate isn't")

	ctr := pod.Containers[0]
	assert.Equal(t, "ghcr.io/renovatebot/renovate:44.181.6", ctr.Image)
	names := []string{}
	for _, e := range ctr.Env {
		names = append(names, e.Name)
	}
	assert.Equal(t, []string{"RENOVATE_CONFIG", "RENOVATE_CACHE_DIR", "RENOVATE_TOKEN"}, names, "configured env last, so it wins")

	global := map[string]any{}
	require.NoError(t, json.Unmarshal([]byte(ctr.Env[0].Value), &global))
	assert.Equal(t, "github", global["platform"])
	assert.Equal(t, []any{"example/deploy"}, global["repositories"])
	assert.Equal(t, false, global["autodiscover"])
	assert.Equal(t, "14 days", global["minimumReleaseAge"])
}

// fakeJobs completes or fails each Job on its first status check
type fakeJobs struct {
	created []*batchv1.Job
	active  []string
	fail    string
}

func (f *fakeJobs) CreateJob(_ context.Context, job *batchv1.Job) (*batchv1.Job, error) {
	job = job.DeepCopy()
	job.Name = job.GenerateName + "abcde"
	f.created = append(f.created, job)
	return job, nil
}

func (f *fakeJobs) GetJob(_ context.Context, name string) (*batchv1.Job, error) {
	job := f.created[len(f.created)-1].DeepCopy()
	cond := batchv1.JobCondition{Type: batchv1.JobComplete, Status: corev1.ConditionTrue}
	if f.fail != "" {
		cond = batchv1.JobCondition{Type: batchv1.JobFailed, Status: corev1.ConditionTrue, Reason: f.fail}
	}
	job.Status.Conditions = []batchv1.JobCondition{cond}
	return job, nil
}

func (f *fakeJobs) ActiveJobs(context.Context) ([]string, error) {
	return f.active, nil
}

func (f *fakeJobs) Logs(context.Context, string, int64) (string, error) {
	return "\x1b[32mINFO\x1b[39m: Renovate is exiting with success", nil
}

func TestRunRenovate(t *testing.T) {
	ctx := context.Background()
	c, err := loadConfig(t, renovateConfigYAML)
	require.NoError(t, err)
	s, err := store.Open(filepath.Join(t.TempDir(), "imginq.db"))
	require.NoError(t, err)
	defer s.Close()

	jobs := &fakeJobs{}
	d := &Daemon{config: c, store: s, repos: newRepoSet(c.Repos), renovateJobs: jobs, renovatePoll: time.Millisecond}

	require.NoError(t, d.runRenovate(ctx, store.RenovateTriggerManual, []string{"example/deploy"}))
	jobs.fail = "DeadlineExceeded"
	require.NoError(t, d.runRenovate(ctx, store.RenovateTriggerSchedule, []string{"example/deploy", "example/webapp"}))
	jobs.active = []string{"imginq-renovate-old"}
	require.NoError(t, d.runRenovate(ctx, store.RenovateTriggerSchedule, []string{"example/deploy"}))

	runs, err := s.ListRenovateRuns(ctx, 10)
	require.NoError(t, err)
	require.Len(t, runs, 3)

	assert.Equal(t, store.RenovateStatusFailed, runs[0].Status)
	assert.Contains(t, runs[0].Error, "still running: imginq-renovate-old")
	assert.Len(t, jobs.created, 2, "no job while another is running")

	assert.Equal(t, store.RenovateStatusFailed, runs[1].Status)
	assert.Equal(t, "DeadlineExceeded", runs[1].Error)
	assert.Equal(t, []string{"example/deploy", "example/webapp"}, runs[1].Repos)

	assert.Equal(t, store.RenovateStatusSucceeded, runs[2].Status)
	assert.Equal(t, store.RenovateTriggerManual, runs[2].Trigger)
	assert.Equal(t, "imginq-renovate-abcde", runs[2].Job)
	assert.Equal(t, "INFO: Renovate is exiting with success", runs[2].Log)
	assert.NotNil(t, runs[2].FinishedAt)

	assert.False(t, d.Activity().Renovate.Running)
}

func TestRunRenovateRequests(t *testing.T) {
	c, err := loadConfig(t, renovateConfigYAML)
	require.NoError(t, err)
	d := &Daemon{config: c, repos: newRepoSet(c.Repos), renovateTrigger: make(chan RenovateRequest, 1)}

	assert.ErrorIs(t, d.RunRenovate(RenovateRequest{Repo: "example/unknown"}), ErrUnknownRepo)
	require.NoError(t, d.RunRenovate(RenovateRequest{Repo: "example/deploy"}))
	assert.ErrorIs(t, d.RunRenovate(RenovateRequest{}), ErrRenovateBusy, "one queued at a time")

	d.config.Renovate = nil
	assert.ErrorIs(t, d.RunRenovate(RenovateRequest{}), ErrRenovateDisabled)
}
