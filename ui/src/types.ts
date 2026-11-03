// Mirrors the daemon's JSON API (internal/store, internal/prdiff, internal/api)

export type ChangeKind = 'image' | 'helmChart';

export interface Change {
  file: string;
  kind: ChangeKind;
  name: string;
  repoURL?: string;
  from: string;
  to: string;
}

export interface Workload {
  cluster: string;
  namespace: string;
  kind: string;
  name: string;
  argoApp?: string;
  containers: string[];
  digests: string[];
  source?: Source;
}

// The Argo CD app source (chart) that deploys a workload
export interface Source {
  app: string;
  index: number;
  kind: 'chart' | 'git';
  chart: string;
  repoURL: string;
  path?: string;
  version: string;
  latest?: string;
}

// Whether an image or chart needs an update (internal/api/inventory.go)
export type UpdateStatus = 'pending' | 'current' | 'pinned' | 'behind' | 'unknown';

// The newest version of the same variant as a running tag or deployed chart version
export interface Version {
  kind: 'image' | 'chart';
  repository: string;
  current: string;
  latest?: string;
  error?: string;
  checkedAt: string;
}

// A change with the workloads running its From version; null when unknown
// (no clusters configured, or a chart change not rendered yet)
export interface ChangeView extends Change {
  running: Workload[] | null;
  renders?: ChartRenderView[];
  scan?: ScanDelta;
}

export type Severity = 'CRITICAL' | 'HIGH' | 'MEDIUM' | 'LOW' | 'UNKNOWN';
export type SeverityCounts = Partial<Record<Severity, number>>;
// skipped: not scanned, e.g. an image with no linux/amd64 variant
export type ScanStatus = 'pending' | 'done' | 'error' | 'skipped';

export interface Vulnerability {
  id: string;
  package: string;
  installed: string;
  fixed?: string;
  severity: Severity;
  title?: string;
  url?: string;
}

export interface ScanSide {
  ref: string;
  digest?: string;
  status: ScanStatus;
  error?: string;
  counts?: SeverityCounts;
  scannedAt?: string;
}

// The vulnerabilities an image change fixes and introduces; a side is null for an
// added or removed container. Lists are only included for a single PR.
export interface ScanDelta {
  from: ScanSide | null;
  to: ScanSide | null;
  status: ScanStatus;
  fixed: SeverityCounts;
  introduced: SeverityCounts;
  fixedVulnerabilities?: Vulnerability[];
  introducedVulnerabilities?: Vulnerability[];
}

// A PR's image scans summed up, each vulnerability counted once
export interface PullRequestScan {
  status: ScanStatus;
  images: number;
  pending: number;
  failed: number;
  skipped: number;
  fixed: SeverityCounts;
  introduced: SeverityCounts;
}

// The images a chart change updates in one app
export interface ChartRenderView {
  cluster: string;
  app: string;
  error?: string;
  images: ImageChange[];
}

// A container whose image changes; from is empty when added, to when removed
export interface ImageChange {
  kind: string;
  name: string;
  namespace: string;
  container?: string;
  init?: boolean;
  from: string;
  to: string;
  running: Workload[];
  scan?: ScanDelta;
}

export interface Stage {
  name: string;
  status: string;
  error?: string;
  headSHA: string;
  updatedAt: string;
}

export interface PullRequest {
  repo: string;
  number: number;
  title: string;
  url: string;
  branch: string;
  headSHA: string;
  author: string;
  state: 'open' | 'closed';
  createdAt: string;
  updatedAt: string;
  syncedAt: string;
  changes: ChangeView[];
  stages: Stage[];
  scan?: PullRequestScan;
}

export interface Coverage {
  repo: string;
  number: number;
  title: string;
  from: string;
  to: string;
  matchesRunning: boolean;
  via?: string;
}

export interface RunningImage {
  repository: string;
  tag?: string;
  digests: string[];
  workloads: Workload[];
  coveredBy: Coverage[];
  version?: Version;
  status: UpdateStatus;
  // The scan of the running digest, and how updating to the newest version changes it
  scan?: ScanSide;
  update?: ScanDelta;
  soak?: SoakDecision;
}

// How long the update of a running tag to its newest version soaks (internal/daemon SoakDecision)
export interface SoakDecision {
  repository: string;
  current: string;
  newest: string;
  lane: 'default' | 'major' | 'fast' | 'risky';
  reason: string;
  soak?: string;
  releasedAt?: string;
  releaseSource?: 'build date' | 'first seen';
  eligibleAt?: string;
  // Whether Renovate is told it may propose the update now
  released: boolean;
  scanPending?: boolean;
}

export interface SoakConfig {
  default: string;
  major: string;
  fastLane: string;
  risky: string;
  fastLaneSeverity: string;
  riskySeverity: string;
}

export interface RenderedImage {
  kind: string;
  name: string;
  namespace: string;
  container?: string;
  init?: boolean;
  image: string;
}

export interface Render {
  cluster: string;
  app: string;
  source: number;
  revision: string;
  error?: string;
  images: RenderedImage[];
  renderedAt: string;
}

export interface SourceView {
  index: number;
  kind: 'chart' | 'git' | 'ref' | 'other';
  chart?: string;
  repoURL: string;
  path?: string;
  targetRevision?: string;
  revision?: string;
  render?: Render;
  version?: Version;
  updatePRs?: Coverage[];
  status?: UpdateStatus;
}

export interface App {
  cluster: string;
  name: string;
  project: string;
  destNamespace: string;
  syncStatus: string;
  health: string;
  sources: SourceView[];
  containers: number;
  unattributed: number;
}

export interface Cluster {
  name: string;
  syncedAt: string;
  containers: number;
  error?: string;
}

export interface SyncStep {
  name: string;
  startedAt: string;
  finishedAt: string;
  summary: string;
  error?: string;
}

export interface SyncRun {
  id: number;
  startedAt: string;
  finishedAt?: string;
  pullRequests: number;
  error?: string;
  // Only in ActivityReport.syncRuns
  steps?: SyncStep[];
}

// What the daemon's loops are doing right now (internal/daemon Activity)
export interface Activity {
  sync: {
    running: boolean;
    step?: string;
    stepStartedAt?: string;
    nextRunAt?: string;
  };
  scan: {
    enabled: boolean;
    task?: string;
    image?: string;
    taskStartedAt?: string;
  };
  renovate: {
    enabled: boolean;
    running: boolean;
    runID?: number;
    job?: string;
    repos?: string[];
    startedAt?: string;
    nextRunAt?: string;
  };
}

// One Renovate run as a Kubernetes Job (internal/store RenovateRun)
export interface RenovateRun {
  id: number;
  startedAt: string;
  finishedAt?: string;
  trigger: 'schedule' | 'manual';
  repos: string[];
  job?: string;
  status: 'running' | 'succeeded' | 'failed' | 'interrupted';
  error?: string;
  log?: string;
}

// Open PRs' image references by what they need next
export interface ScanQueue {
  images: number;
  pr: number;
  running: number;
  newest: number;
  resolve: number;
  generate: number;
  check: number;
  failed: number;
  skipped: number;
}

export interface Cache {
  name: string;
  bytes: number;
  files: number;
  updatedAt?: string;
  downloadedAt?: string;
  nextUpdateAt?: string;
}

export interface ActivityReport extends Activity {
  scanQueue?: ScanQueue;
  caches: Cache[];
  tagLists: {
    repositories: number;
    failed: number;
    oldest?: string;
    failures: { kind: 'image' | 'chart'; repository: string; error: string }[];
  };
  syncRuns: SyncRun[];
  renovateRuns: RenovateRun[];
}

// A configured repos entry: a name, or a pattern like owner/* with the repos it matches
export interface RepoPattern {
  name: string;
  exclude?: string[];
  branchPrefix: string;
  pattern: boolean;
  matches: string[];
  // The last listing's error; matches are then from the last good listing
  error?: string;
  listedAt?: string;
}

// The daemon config, without secrets (internal/api Config)
export interface Config {
  syncInterval: string;
  githubAPIURL?: string;
  githubToken: boolean;
  scanning: boolean;
  repos: RepoPattern[];
  clusters: {
    name: string;
    context?: string;
    inCluster?: boolean;
    includeNamespaces?: string[];
    excludeNamespaces?: string[];
  }[];
  renovate?: { image: string; cluster: string; namespace: string; interval?: string; soak?: SoakConfig };
}

export interface Status {
  repos: string[];
  clusters: Cluster[];
  lastRun: SyncRun | null;
  config: Config;
  activity: Activity;
}

// Trends is the history of vulnerabilities running in workload containers (GET /api/v1/trends)
export interface Trends {
  since: string;
  // When findings were first recorded; unset if never
  recordedSince?: string;
  slaDays: Partial<Record<Severity, number>>;
  days: TrendDay[];
  weeks: TrendWeek[];
  timeToFix: TimeToFix[];
  overdue: {
    total: number;
    bySeverity: SeverityCounts;
    findings: OverdueFinding[];
  };
}

export interface TrendDay {
  date: string;
  open: SeverityCounts;
  fixable: number;
}

// Opened excludes findings already running when recording began
export interface TrendWeek {
  start: string;
  opened: number;
  fixed: number;
}

// Days from a fix being available to it running, for findings fixed in the window
export interface TimeToFix {
  severity: Severity;
  fixed: number;
  withinSLA: number;
  median: number;
  p90: number;
}

export interface OverdueFinding {
  id: number;
  cluster: string;
  namespace: string;
  workloadKind: string;
  workloadName: string;
  container: string;
  vulnId: string;
  package: string;
  severity: Severity;
  fixAvailable: string;
  daysOverdue: number;
}
