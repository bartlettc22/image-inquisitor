package daemon

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/bartlettc22/image-inquisitor/internal/store"
	yaml "github.com/goccy/go-yaml"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	defaultRenovateTimeout = time.Hour

	// renovateJobTTL is how long finished Jobs (and their pods' logs) are kept
	renovateJobTTL = 24 * time.Hour

	// renovatePollInterval is how often a running Job's status is checked
	renovatePollInterval = 10 * time.Second

	// renovateLogLines is how much of a run's log is kept
	renovateLogLines = 40

	renovateCacheDir = "/cache"

	managedByLabel     = "app.kubernetes.io/managed-by"
	managedByValue     = "image-inquisitor"
	renovateRunLabel   = "image-inquisitor.io/renovate-run"
	renovateJobPrefix  = "imginq-renovate-"
	renovateContainer  = "renovate"
	renovateConfigEnv  = "RENOVATE_CONFIG"
	renovateCacheEnv   = "RENOVATE_CACHE_DIR"
	renovateCacheMount = "cache"
)

// renovateReservedEnv would override the repositories the daemon passes in RENOVATE_CONFIG
var renovateReservedEnv = map[string]bool{
	"RENOVATE_REPOSITORIES":        true,
	"RENOVATE_AUTODISCOVER":        true,
	"RENOVATE_AUTODISCOVER_FILTER": true,
	renovateConfigEnv:              true,
}

// RenovateConfig runs Renovate on the configured repos as Kubernetes Jobs the daemon creates.
// It is read as JSON (converted from YAML) so it can use Kubernetes types.
type RenovateConfig struct {
	// Cluster is the configured cluster the Jobs run in; defaults to the only one
	Cluster string `json:"cluster"`

	// Namespace the Jobs run in; it holds the Secrets their env refers to
	Namespace string `json:"namespace"`

	// Image is the Renovate image, e.g. ghcr.io/renovatebot/renovate:44.181.6
	Image string `json:"image"`

	// Interval between scheduled runs, e.g. "6h"; unset to only run from the UI
	Interval string `json:"interval"`

	// Timeout bounds one run (default 1h)
	Timeout string `json:"timeout"`

	// Config is Renovate's global config. The daemon sets repositories (from repos) and
	// turns autodiscover off; platform defaults to github.
	Config map[string]any `json:"config"`

	// Env is added to the Renovate container, e.g. RENOVATE_TOKEN from a Secret
	Env []corev1.EnvVar `json:"env"`

	Resources    corev1.ResourceRequirements `json:"resources"`
	DNSConfig    *corev1.PodDNSConfig        `json:"dnsConfig"`
	NodeSelector map[string]string           `json:"nodeSelector"`

	// CacheClaim is a PersistentVolumeClaim mounted as Renovate's cache dir; runs never overlap
	CacheClaim string `json:"cacheClaim"`

	// Soak, when set, has the daemon set Renovate's minimumReleaseAge from the CVEs each update
	// fixes and introduces, dating images itself
	Soak *SoakConfig `json:"soak"`

	interval time.Duration
	timeout  time.Duration
}

func (r *RenovateConfig) UnmarshalYAML(b []byte) error {
	j, err := yaml.YAMLToJSON(b)
	if err != nil {
		return err
	}
	type plain RenovateConfig
	dec := json.NewDecoder(bytes.NewReader(j))
	dec.DisallowUnknownFields()
	return dec.Decode((*plain)(r))
}

// RunEvery is the interval between scheduled runs; zero when Renovate only runs from the UI
func (r *RenovateConfig) RunEvery() time.Duration {
	return r.interval
}

func (r *RenovateConfig) validate(c *Config) error {
	if r.Image == "" {
		return errors.New("image is required")
	}
	if r.Namespace == "" {
		return errors.New("namespace is required")
	}
	if len(c.Repos) == 0 {
		return errors.New("no repos to run on")
	}

	if r.Cluster == "" {
		if len(c.Kubernetes.Clusters) != 1 {
			return errors.New("cluster is required when more than one kubernetes cluster is configured")
		}
		r.Cluster = c.Kubernetes.Clusters[0].Name
	}
	found := false
	for _, cl := range c.Kubernetes.Clusters {
		found = found || cl.Name == r.Cluster
	}
	if !found {
		return fmt.Errorf("cluster %q isn't one of the configured kubernetes clusters", r.Cluster)
	}

	var err error
	if r.Interval != "" {
		if r.interval, err = time.ParseDuration(r.Interval); err != nil {
			return fmt.Errorf("invalid interval %q: %w", r.Interval, err)
		}
	}
	r.timeout = defaultRenovateTimeout
	if r.Timeout != "" {
		if r.timeout, err = time.ParseDuration(r.Timeout); err != nil {
			return fmt.Errorf("invalid timeout %q: %w", r.Timeout, err)
		}
	}

	for _, e := range r.Env {
		if renovateReservedEnv[e.Name] {
			return fmt.Errorf("env %s would override the repos the daemon runs Renovate on; set Renovate options in config instead", e.Name)
		}
	}
	for _, key := range []string{"repositories", "autodiscover", "autodiscoverFilter"} {
		if _, ok := r.Config[key]; ok {
			return fmt.Errorf("config.%s is set by the daemon from repos", key)
		}
	}
	if r.Soak != nil {
		if _, ok := r.Config["minimumReleaseAge"]; ok {
			return errors.New("config.minimumReleaseAge is set by the daemon from soak.default")
		}
		if err := r.Soak.validate(); err != nil {
			return err
		}
	}
	return nil
}

// globalConfig is the Renovate config of one run on repos. With a soak, it sets the default
// minimumReleaseAge and appends the soak's packageRules after the configured ones, so they win.
func (r *RenovateConfig) globalConfig(repos []string, soakRules []map[string]any) map[string]any {
	global := map[string]any{"platform": "github"}
	for k, v := range r.Config {
		global[k] = v
	}
	global["repositories"] = repos
	global["autodiscover"] = false
	if r.Soak != nil {
		global["minimumReleaseAge"] = r.Soak.Default
		rules := []any{}
		if configured, ok := r.Config["packageRules"].([]any); ok {
			rules = append(rules, configured...)
		}
		for _, rule := range soakRules {
			rules = append(rules, rule)
		}
		global["packageRules"] = rules
	}
	return global
}

// job is the Job of one Renovate run on repos
func (r *RenovateConfig) job(runID int64, repos []string, soakRules []map[string]any) (*batchv1.Job, error) {
	config, err := json.Marshal(r.globalConfig(repos, soakRules))
	if err != nil {
		return nil, err
	}

	labels := map[string]string{managedByLabel: managedByValue, renovateRunLabel: strconv.FormatInt(runID, 10)}
	env := []corev1.EnvVar{{Name: renovateConfigEnv, Value: string(config)}}
	var mounts []corev1.VolumeMount
	var volumes []corev1.Volume
	if r.CacheClaim != "" {
		env = append(env, corev1.EnvVar{Name: renovateCacheEnv, Value: renovateCacheDir})
		mounts = append(mounts, corev1.VolumeMount{Name: renovateCacheMount, MountPath: renovateCacheDir})
		volumes = append(volumes, corev1.Volume{Name: renovateCacheMount, VolumeSource: corev1.VolumeSource{
			PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: r.CacheClaim},
		}})
	}
	env = append(env, r.Env...)

	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{GenerateName: renovateJobPrefix, Namespace: r.Namespace, Labels: labels},
		Spec: batchv1.JobSpec{
			BackoffLimit:            ptr(int32(0)),
			ActiveDeadlineSeconds:   ptr(int64(r.timeout.Seconds())),
			TTLSecondsAfterFinished: ptr(int32(renovateJobTTL.Seconds())),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{
					RestartPolicy:                corev1.RestartPolicyNever,
					AutomountServiceAccountToken: ptr(false),
					DNSConfig:                    r.DNSConfig,
					NodeSelector:                 r.NodeSelector,
					SecurityContext: &corev1.PodSecurityContext{
						SeccompProfile: &corev1.SeccompProfile{Type: corev1.SeccompProfileTypeRuntimeDefault},
					},
					Containers: []corev1.Container{{
						Name:      renovateContainer,
						Image:     r.Image,
						Env:       env,
						Resources: r.Resources,
						SecurityContext: &corev1.SecurityContext{
							AllowPrivilegeEscalation: ptr(false),
							Capabilities:             &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
						},
						VolumeMounts: mounts,
					}},
					Volumes: volumes,
				},
			},
		},
	}, nil
}

func ptr[T any](v T) *T {
	return &v
}

// jobClient is the part of the Kubernetes API Renovate runs need, in one namespace
type jobClient interface {
	CreateJob(ctx context.Context, job *batchv1.Job) (*batchv1.Job, error)
	GetJob(ctx context.Context, name string) (*batchv1.Job, error)

	// ActiveJobs are the daemon's Renovate Jobs that haven't finished
	ActiveJobs(ctx context.Context) ([]string, error)

	// Logs returns the end of the log of a Job's pod
	Logs(ctx context.Context, job string, lines int64) (string, error)
}

type kubeJobClient struct {
	client    kubernetes.Interface
	namespace string
}

func (k *kubeJobClient) CreateJob(ctx context.Context, job *batchv1.Job) (*batchv1.Job, error) {
	return k.client.BatchV1().Jobs(k.namespace).Create(ctx, job, metav1.CreateOptions{})
}

func (k *kubeJobClient) GetJob(ctx context.Context, name string) (*batchv1.Job, error) {
	return k.client.BatchV1().Jobs(k.namespace).Get(ctx, name, metav1.GetOptions{})
}

func (k *kubeJobClient) ActiveJobs(ctx context.Context) ([]string, error) {
	list, err := k.client.BatchV1().Jobs(k.namespace).List(ctx, metav1.ListOptions{
		LabelSelector: managedByLabel + "=" + managedByValue + "," + renovateRunLabel,
	})
	if err != nil {
		return nil, err
	}
	active := []string{}
	for _, job := range list.Items {
		if _, done := jobResult(&job); !done {
			active = append(active, job.Name)
		}
	}
	return active, nil
}

func (k *kubeJobClient) Logs(ctx context.Context, job string, lines int64) (string, error) {
	pods, err := k.client.CoreV1().Pods(k.namespace).List(ctx, metav1.ListOptions{LabelSelector: "job-name=" + job})
	if err != nil {
		return "", err
	}
	if len(pods.Items) == 0 {
		return "", errors.New("no pod found")
	}
	b, err := k.client.CoreV1().Pods(k.namespace).GetLogs(pods.Items[0].Name, &corev1.PodLogOptions{
		Container: renovateContainer, TailLines: &lines,
	}).DoRaw(ctx)
	return string(b), err
}

// jobResult reports whether a Job finished and, if it failed, why
func jobResult(job *batchv1.Job) (errMsg string, done bool) {
	for _, c := range job.Status.Conditions {
		if c.Status != corev1.ConditionTrue {
			continue
		}
		switch c.Type {
		case batchv1.JobComplete:
			return "", true
		case batchv1.JobFailed:
			msg := c.Reason
			if c.Message != "" {
				msg += ": " + c.Message
			}
			return msg, true
		}
	}
	return "", false
}

// RenovateRequest asks for a Renovate run; Repo limits it to one repo
type RenovateRequest struct {
	Repo string
}

var (
	// ErrRenovateDisabled is returned by RunRenovate when no renovate section is configured
	ErrRenovateDisabled = errors.New("renovate isn't configured")

	// ErrRenovateBusy is returned by RunRenovate while a run is in progress or queued
	ErrRenovateBusy = errors.New("a renovate run is already in progress")

	// ErrUnknownRepo is returned by RunRenovate for a repo that isn't configured
	ErrUnknownRepo = errors.New("repo isn't configured")
)

// RunRenovate requests a Renovate run as soon as possible
func (d *Daemon) RunRenovate(req RenovateRequest) error {
	if d.config.Renovate == nil {
		return ErrRenovateDisabled
	}
	if req.Repo != "" {
		found := false
		for _, r := range d.Repos() {
			found = found || strings.EqualFold(r.Name, req.Repo)
		}
		if !found {
			return ErrUnknownRepo
		}
	}
	if d.Activity().Renovate.Running {
		return ErrRenovateBusy
	}
	select {
	case d.renovateTrigger <- req:
		return nil
	default:
		return ErrRenovateBusy
	}
}

// renovateLoop runs Renovate on every interval and on request, one run at a time,
// syncing after each run so its PRs show up straight away
func (d *Daemon) renovateLoop(ctx context.Context) {
	cfg := d.config.Renovate
	if err := d.store.InterruptRenovateRuns(ctx, time.Now().UTC()); err != nil {
		svcLog.WithError(err).Error("error recording interrupted renovate runs")
	}

	var tick <-chan time.Time
	if cfg.interval > 0 {
		ticker := time.NewTicker(cfg.interval)
		defer ticker.Stop()
		tick = ticker.C
	}

	for {
		if cfg.interval > 0 {
			// Approximate: the ticker keeps its own schedule
			next := time.Now().Add(cfg.interval).UTC()
			d.activity.update(func(a *Activity) { a.Renovate.NextRunAt = &next })
		}

		trigger, repos := "", []string{}
		select {
		case <-ctx.Done():
			return
		case <-tick:
			trigger = store.RenovateTriggerSchedule
		case req := <-d.renovateTrigger:
			trigger = store.RenovateTriggerManual
			if req.Repo != "" {
				repos = []string{req.Repo}
			}
		}
		if len(repos) == 0 {
			for _, r := range d.Repos() {
				repos = append(repos, r.Name)
			}
		}

		if err := d.runRenovate(ctx, trigger, repos); err != nil {
			svcLog.WithError(err).Error("error running renovate")
		}
		d.Trigger()
	}
}

// runRenovate creates a Renovate Job for repos and waits for it to finish, recording the run.
// Only storage errors are returned; a failed run is recorded on the run.
func (d *Daemon) runRenovate(ctx context.Context, trigger string, repos []string) error {
	run := &store.RenovateRun{StartedAt: time.Now().UTC(), Trigger: trigger, Repos: repos, Status: store.RenovateStatusRunning}
	if err := d.store.StartRenovateRun(ctx, run); err != nil {
		return err
	}
	d.activity.update(func(a *Activity) {
		a.Renovate.Running, a.Renovate.RunID, a.Renovate.Repos, a.Renovate.StartedAt = true, run.ID, repos, &run.StartedAt
		a.Renovate.NextRunAt = nil
	})
	defer d.activity.update(func(a *Activity) {
		a.Renovate.Running, a.Renovate.RunID, a.Renovate.Repos, a.Renovate.StartedAt, a.Renovate.Job = false, 0, nil, nil, ""
	})

	logger := svcLog.WithField("renovateRun", run.ID)
	finish := func(errMsg string) error {
		run.FinishedAt = ptr(time.Now().UTC())
		run.Status = store.RenovateStatusSucceeded
		if errMsg != "" {
			run.Status = store.RenovateStatusFailed
			run.Error = errMsg
			logger.WithField("error", errMsg).Warn("renovate run failed")
		} else {
			logger.WithField("duration", run.FinishedAt.Sub(run.StartedAt).Round(time.Second).String()).Info("renovate run finished")
		}
		return d.store.FinishRenovateRun(ctx, run)
	}

	// Jobs left by an earlier daemon (or created by hand with our labels) still use the cache
	active, err := d.renovateJobs.ActiveJobs(ctx)
	if err != nil {
		return finish("error listing renovate jobs: " + err.Error())
	}
	if len(active) > 0 {
		return finish("an earlier renovate job is still running: " + strings.Join(active, ", "))
	}

	var soakRules []map[string]any
	if soak := d.config.Renovate.Soak; soak != nil {
		soakRules = soak.renovateRules(d.SoakDecisions())
	}
	spec, err := d.config.Renovate.job(run.ID, repos, soakRules)
	if err != nil {
		return finish(err.Error())
	}
	job, err := d.renovateJobs.CreateJob(ctx, spec)
	if err != nil {
		return finish("error creating job: " + err.Error())
	}
	run.Job = job.Name
	if err := d.store.SetRenovateRunJob(ctx, run.ID, job.Name); err != nil {
		return err
	}
	d.activity.update(func(a *Activity) { a.Renovate.Job = job.Name })
	logger.WithField("job", job.Name).WithField("repos", len(repos)).Info("started renovate run")

	errMsg := ""
	for {
		select {
		case <-ctx.Done():
			// Left running; the next daemon records the run as interrupted
			return nil
		case <-time.After(d.renovatePoll):
		}
		job, err = d.renovateJobs.GetJob(ctx, job.Name)
		if err != nil {
			logger.WithError(err).Warn("error checking renovate job")
			continue
		}
		var done bool
		if errMsg, done = jobResult(job); done {
			break
		}
	}

	logs, err := d.renovateJobs.Logs(ctx, run.Job, renovateLogLines)
	if err != nil {
		logs = "error reading logs: " + err.Error()
	}
	run.Log = stripANSI(logs)
	return finish(errMsg)
}

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string {
	return ansiRe.ReplaceAllString(s, "")
}
