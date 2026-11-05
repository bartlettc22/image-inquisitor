import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { getActivity, getImage, getPullRequest, runRenovate, getStatus, listApps, listPullRequests, listRunningImages, triggerSync } from './api';
import {
  bump,
  bytes,
  duration,
  imageHref,
  imageName,
  imageTag,
  kindLabel,
  shortRepository,
  shortRevision,
  timeAgo,
  timeUntil,
  version,
  workloadLabel,
} from './format';
import type {
  ActivityReport,
  App as ArgoApp,
  ChangeView,
  ChartRenderView,
  Coverage,
  PullRequest,
  PullRequestScan,
  RunningImage,
  SoakDecision,
  ScanDelta,
  ScanSide,
  SeverityCounts,
  SourceView,
  Stage,
  Status,
  UpdateStatus,
  Version,
  Vulnerability,
  Workload,
} from './types';
import { History, ImageHistory } from './History';
import { Trends } from './Trends';

const POLL_MS = 15_000;

type KindFilter = 'all' | 'image' | 'helmChart' | 'none';

// usePoll runs load now, on every interval, and whenever reload is called
function usePoll<T>(load: () => Promise<T>, deps: unknown[], intervalMs = POLL_MS) {
  const [data, setData] = useState<T | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [tick, setTick] = useState(0);

  useEffect(() => {
    let cancelled = false;
    const run = () =>
      load().then(
        (d) => !cancelled && (setData(d), setError(null)),
        (e: unknown) => !cancelled && setError(e instanceof Error ? e.message : String(e)),
      );
    run();
    const id = setInterval(run, intervalMs);
    return () => {
      cancelled = true;
      clearInterval(id);
    };
  }, [...deps, tick]);

  const reload = useCallback(() => setTick((t) => t + 1), []);
  return { data, error, reload };
}

// useHash returns the hash route without its query, e.g. "#/images", and the query's params
function useHash() {
  const [hash, setHash] = useState(window.location.hash);
  useEffect(() => {
    const onChange = () => setHash(window.location.hash);
    window.addEventListener('hashchange', onChange);
    return () => window.removeEventListener('hashchange', onChange);
  }, []);
  const [route, query = ''] = hash.split('?');
  return { route, params: new URLSearchParams(query) };
}

const prRoute = /^#\/pr\/([^/]+\/[^/]+)\/(\d+)$/;
const prHref = (pr: PullRequest) => `#/pr/${pr.repo}/${pr.number}`;
const imageRoute = /^#\/image\/(.+)$/;

const tabs: Record<string, 'prs' | 'images' | 'apps' | 'history' | undefined> = {
  '#/prs': 'prs',
  '#/images': 'images',
  '#/apps': 'apps',
  '#/history': 'history',
};

export default function App() {
  const { route, params } = useHash();
  // Often enough for the header to show the sync step and scan task as they change
  const status = usePoll(getStatus, [], 5_000);
  const match = prRoute.exec(route);
  const imageMatch = imageRoute.exec(route);
  const tab = tabs[route] ?? 'dashboard';
  const hasClusters = (status.data?.clusters.length ?? 0) > 0;

  return (
    <div className="app">
      <Header
        section={match ? 'prs' : imageMatch ? 'images' : tab}
        status={status.data}
        error={status.error}
        onSynced={status.reload}
      />
      <main>
        {match ? (
          <PullRequestDetail repo={match[1]} number={Number(match[2])} />
        ) : imageMatch ? (
          <ImagePage key={imageMatch[1]} repository={imageMatch[1]} />
        ) : tab === 'images' ? (
          <RunningImages
            key={params.toString()}
            hasClusters={hasClusters}
            multiCluster={(status.data?.clusters.length ?? 0) > 1}
            initialStatus={params.get('status')}
          />
        ) : tab === 'apps' ? (
          <Apps key={params.toString()} hasClusters={hasClusters} initialProblemsOnly={params.has('attention')} />
        ) : tab === 'history' ? (
          <History key={params.toString()} initialQuery={params.get('q')} />
        ) : tab === 'prs' ? (
          <PullRequestList />
        ) : (
          <Dashboard hasClusters={hasClusters} />
        )}
      </main>
      <ThemeToggle />
    </div>
  );
}

type Theme = 'system' | 'light' | 'dark';

const themeLabels: Record<Theme, string> = { system: '◐ Auto', light: '☀ Light', dark: '☾ Dark' };
const nextTheme: Record<Theme, Theme> = { system: 'light', light: 'dark', dark: 'system' };

// ThemeToggle cycles system → light → dark; index.html applies the stored choice before first paint
function ThemeToggle() {
  const [theme, setTheme] = useState<Theme>(() => {
    const t = document.documentElement.dataset.theme;
    return t === 'light' || t === 'dark' ? t : 'system';
  });

  const cycle = () => {
    const t = nextTheme[theme];
    setTheme(t);
    if (t === 'system') {
      delete document.documentElement.dataset.theme;
    } else {
      document.documentElement.dataset.theme = t;
    }
    try {
      if (t === 'system') localStorage.removeItem('theme');
      else localStorage.setItem('theme', t);
    } catch {
      // Storage unavailable (e.g. private mode): the choice lasts until reload
    }
  };

  return (
    <button className="theme-toggle" onClick={cycle} title={`Theme: ${theme}. Click for ${nextTheme[theme]}`}>
      {themeLabels[theme]}
    </button>
  );
}

// The top-level sections in the header
const sections = [
  { id: 'dashboard', href: '#/', label: 'Dashboard' },
  { id: 'images', href: '#/images', label: 'Running images' },
  { id: 'prs', href: '#/prs', label: 'Pull requests' },
  { id: 'apps', href: '#/apps', label: 'Apps' },
  { id: 'history', href: '#/history', label: 'History' },
];

function Header({
  section,
  status,
  error,
  onSynced,
}: {
  section: string;
  status: Status | null;
  error: string | null;
  onSynced: () => void;
}) {
  return (
    <header>
      <div className="header-inner">
        <a className="brand" href="#/">
          <img className="logo" src="/logo.png" alt="" />
          Image Inquisitor
        </a>
        <nav className="tabs">
          {sections.map((s) => (
            <a key={s.id} className={section === s.id ? 'active' : ''} href={s.href}>
              {s.label}
            </a>
          ))}
        </nav>
        <span className="spacer" />
        {error && <span className="badge error" title={error}>API unreachable</span>}
        {status && <StatusButton status={status} onSynced={onSynced} />}
        {status && <ConfigButton status={status} />}
      </div>
    </header>
  );
}

// StatusButton shows what the daemon is doing and opens the Status dialog
function StatusButton({ status, onSynced }: { status: Status; onSynced: () => void }) {
  const dialog = useRef<HTMLDialogElement>(null);
  const [open, setOpen] = useState(false);
  const { activity, lastRun: run } = status;

  let label = 'Not synced yet';
  if (activity.sync.running) {
    label = 'Sync in progress';
  } else if (run?.finishedAt) {
    label = `Synced ${timeAgo(run.finishedAt)}`;
  }

  return (
    <>
      <button
        onClick={() => {
          setOpen(true);
          dialog.current?.showModal();
        }}
        title="Sync and scan status"
      >
        {label}
        {activity.scan.task && <span className="badge kind image">scanning</span>}
        {activity.renovate.running && <span className="badge kind image">renovate</span>}
        {!activity.sync.running && run?.error && <span className="badge error">sync error</span>}
      </button>
      <dialog
        ref={dialog}
        className="config status"
        onClose={() => setOpen(false)}
        onClick={(e) => {
          // A backdrop click, or a link to a page behind the dialog
          if (e.target === dialog.current || (e.target as Element).closest('a[href^="#/"]')) dialog.current?.close();
        }}
      >
        <div className="config-body">
          <div className="config-head">
            <h2>Status</h2>
            <button onClick={() => dialog.current?.close()} aria-label="Close">
              ✕
            </button>
          </div>
          {/* Mounted only while open, so the activity report is only polled then */}
          {open && <StatusDetail onSynced={onSynced} />}
        </div>
      </dialog>
    </>
  );
}

const elapsed = (since?: string) => (since ? duration(Date.now() - Date.parse(since)) : '');

function StatusDetail({ onSynced }: { onSynced: () => void }) {
  const { data: r, error, reload } = usePoll(getActivity, [], 3_000);
  const [syncing, setSyncing] = useState(false);

  const sync = async () => {
    setSyncing(true);
    try {
      await triggerSync();
      // Give the daemon a moment to start the run before refreshing
      setTimeout(() => {
        reload();
        onSynced();
      }, 1500);
    } finally {
      setSyncing(false);
    }
  };

  if (error) return <p className="message error">{error}</p>;
  if (!r) return <p className="message">Loading…</p>;
  const q = r.scanQueue;

  return (
    <>
      <h3>Sync loop</h3>
      <p className="status-line">
        {r.sync.running ? (
          <>
            <span className="badge kind image">running</span> {r.sync.step}{' '}
            <span className="muted">for {elapsed(r.sync.stepStartedAt)}</span>
          </>
        ) : (
          <>
            <span className="badge">idle</span>
            {r.sync.nextRunAt && <span className="muted">next sync {timeUntil(r.sync.nextRunAt)}</span>}
          </>
        )}
        <span className="spacer" />
        <button onClick={sync} disabled={syncing || r.sync.running}>
          Sync now
        </button>
      </p>

      <h3>Scan loop</h3>
      {!r.scan.enabled ? (
        <p className="message">Scanning is disabled.</p>
      ) : (
        <>
          <p className="status-line">
            {r.scan.task ? (
              <>
                <span className="badge kind image">running</span> {r.scan.task}{' '}
                <a href={imageHref(r.scan.image ?? '')}>
                  <code title={r.scan.image}>{shortRepository(r.scan.image ?? '')}</code>
                </a>{' '}
                <span className="muted">for {elapsed(r.scan.taskStartedAt)}</span>
              </>
            ) : (
              <span className="badge">idle</span>
            )}
          </p>
          {q && (
            <dl className="meta">
              <dt>Images</dt>
              <dd>
                {q.images} <span className="muted">({q.pr} in PRs, {q.running} running, {q.newest} newest versions)</span>
              </dd>
              <dt>Waiting</dt>
              <dd>
                {q.resolve + q.generate + q.check === 0
                  ? 'nothing'
                  : [
                      q.resolve && `${q.resolve} to resolve`,
                      q.generate && `${q.generate} SBOMs to generate`,
                      q.check && `${q.check} due for a re-check`,
                    ]
                      .filter(Boolean)
                      .join(', ')}
              </dd>
              {q.failed > 0 && (
                <>
                  <dt>Failed</dt>
                  <dd className="error-text">{q.failed}, retried hourly</dd>
                </>
              )}
              {q.skipped > 0 && (
                <>
                  <dt>Skipped</dt>
                  <dd>{q.skipped} with no linux/amd64 image</dd>
                </>
              )}
            </dl>
          )}
        </>
      )}

      {r.renovate.enabled && <RenovateStatus r={r} onStarted={reload} />}

      <h3>Caches</h3>
      <div className="table-wrap">
        <table className="caches">
          <tbody>
            {r.caches.map((c) => (
              <tr key={c.name}>
                <td>
                  <div className="name">{c.name}</div>
                  {c.updatedAt && (
                    <div className="sub">
                      published {timeAgo(c.updatedAt)}
                      {c.downloadedAt && ` · downloaded ${timeAgo(c.downloadedAt)}`}
                      {c.nextUpdateAt &&
                        (Date.parse(c.nextUpdateAt) > Date.now()
                          ? ` · next update ${timeUntil(c.nextUpdateAt)}`
                          : ' · update due on next scan')}
                    </div>
                  )}
                </td>
                <td className="nowrap num">{bytes(c.bytes)}</td>
                <td className="nowrap num muted">{c.files.toLocaleString()} {c.files === 1 ? 'file' : 'files'}</td>
              </tr>
            ))}
            <tr>
              <td>
                <div className="name">Registry tag lists</div>
                <div className="sub">
                  re-listed every 6h
                  {r.tagLists.oldest && ` · oldest listed ${timeAgo(r.tagLists.oldest)}`}
                </div>
                {r.tagLists.failures.length > 0 && (
                  <ul className="failures">
                    {r.tagLists.failures.map((f) => (
                      <li key={`${f.kind}/${f.repository}`}>
                        {f.kind === 'image' ? (
                          <a href={imageHref(f.repository)}>
                            <code>{shortRepository(f.repository)}</code>
                          </a>
                        ) : (
                          <code>{shortRepository(f.repository)}</code>
                        )}
                        <span className="error-text" title={f.error}>
                          {f.error}
                        </span>
                      </li>
                    ))}
                    {r.tagLists.failed > r.tagLists.failures.length && (
                      <li className="muted">and {r.tagLists.failed - r.tagLists.failures.length} more</li>
                    )}
                  </ul>
                )}
              </td>
              <td className="nowrap num">{r.tagLists.repositories} repos</td>
              <td className="nowrap num">
                {r.tagLists.failed > 0 && <span className="badge error">{r.tagLists.failed} failed</span>}
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <h3>Recent syncs</h3>
      {r.syncRuns.length === 0 ? (
        <p className="message">None yet.</p>
      ) : (
        <div className="runs">
          {r.syncRuns.map((run) => (
            <details key={run.id} className="run">
              <summary>
                <span className="muted nowrap" title={run.startedAt}>
                  {timeAgo(run.startedAt)}
                </span>
                <span className="num">
                  {run.finishedAt ? duration(Date.parse(run.finishedAt) - Date.parse(run.startedAt)) : 'running'}
                </span>
                <span>{run.pullRequests} PRs</span>
                {run.error ? (
                  <span className="badge error" title={run.error}>
                    {run.error.startsWith('interrupted') ? 'interrupted' : 'error'}
                  </span>
                ) : run.finishedAt ? (
                  <span className="badge running">ok</span>
                ) : null}
              </summary>
              <ul className="steps">
                {(run.steps ?? []).map((s, i) => (
                  <li key={i}>
                    <span className="name">{s.name}</span>
                    <span className="num muted">{duration(Date.parse(s.finishedAt) - Date.parse(s.startedAt))}</span>
                    <span className="muted">{s.summary}</span>
                    {s.error && (
                      <span className="badge error" title={s.error}>
                        error
                      </span>
                    )}
                    {s.error && <div className="step-error">{s.error}</div>}
                  </li>
                ))}
                {run.steps?.length === 0 && <li className="muted">No steps recorded</li>}
              </ul>
            </details>
          ))}
        </div>
      )}
    </>
  );
}

// RunRenovateButton starts a Renovate run on every repo, or only on repo
function RunRenovateButton({ repo, small, onStarted }: { repo?: string; small?: boolean; onStarted?: () => void }) {
  const [state, setState] = useState<{ busy?: boolean; message?: string; error?: boolean }>({});

  const run = async () => {
    setState({ busy: true });
    try {
      await runRenovate(repo);
      setState({ message: 'started' });
      // Give the daemon a moment to create the Job before refreshing
      setTimeout(() => onStarted?.(), 1500);
    } catch (e) {
      // e.g. "409 Conflict" while a run is in progress
      setState({ message: e instanceof Error && e.message.includes('409') ? 'a run is in progress' : String(e), error: true });
    }
  };

  return (
    <span className="run-renovate">
      {state.message && <span className={state.error ? 'error-text' : 'muted'}>{state.message}</span>}
      <button className={small ? 'small' : ''} onClick={run} disabled={state.busy}>
        Run Renovate
      </button>
    </span>
  );
}

// RenovateStatus is the Renovate section of the Status dialog: the run in progress and recent runs
function RenovateStatus({ r, onStarted }: { r: ActivityReport; onStarted: () => void }) {
  const live = r.renovate;
  return (
    <>
      <h3>Renovate</h3>
      <div className="status-line">
        {live.running ? (
          <>
            <span className="badge kind image">running</span>
            <span>{live.repos?.length === 1 ? live.repos[0] : `${live.repos?.length ?? 0} repos`}</span>
            {live.job && <code>{live.job}</code>}
            <span className="muted">for {elapsed(live.startedAt)}</span>
          </>
        ) : (
          <>
            <span className="badge">idle</span>
            <span className="muted">{live.nextRunAt ? `next run ${timeUntil(live.nextRunAt)}` : 'runs from the UI only'}</span>
          </>
        )}
        <span className="spacer" />
        {!live.running && <RunRenovateButton onStarted={onStarted} />}
      </div>
      {r.renovateRuns.length > 0 && (
        <div className="runs">
          {r.renovateRuns.map((run) => (
            <details key={run.id} className="run">
              <summary>
                <span className="muted nowrap" title={run.startedAt}>
                  {timeAgo(run.startedAt)}
                </span>
                <span className="num">
                  {run.finishedAt ? duration(Date.parse(run.finishedAt) - Date.parse(run.startedAt)) : 'running'}
                </span>
                <span>{run.repos.length === 1 ? run.repos[0].replace(/^.*\//, '') : `${run.repos.length} repos`}</span>
                <span className={`badge ${{ succeeded: 'running', failed: 'error', running: 'kind image', interrupted: '' }[run.status]}`}>
                  {run.status}
                </span>
              </summary>
              <div className="run-detail">
                <div className="muted">
                  {run.trigger === 'manual' ? 'Started from the UI' : 'Scheduled'}
                  {run.job && (
                    <>
                      {' · Job '}
                      <code>{run.job}</code>
                    </>
                  )}
                  {' · '}
                  {run.repos.join(', ')}
                </div>
                {run.error && <div className="step-error">{run.error}</div>}
                {run.log && <pre className="log">{run.log}</pre>}
              </div>
            </details>
          ))}
        </div>
      )}
    </>
  );
}

// ConfigButton opens the repos, clusters and settings the inventory comes from;
// it's flagged when a cluster failed to sync
function ConfigButton({ status }: { status: Status }) {
  const dialog = useRef<HTMLDialogElement>(null);
  const { config, clusters } = status;
  const failed = clusters.filter((c) => c.error).length;

  return (
    <>
      <button onClick={() => dialog.current?.showModal()} title={failed ? `${failed} cluster sync error` : undefined}>
        Config{failed > 0 && <span className="badge error">{failed}</span>}
      </button>
      <dialog
        ref={dialog}
        className="config"
        // Close on a click on the backdrop, outside the dialog's content
        onClick={(e) => e.target === dialog.current && dialog.current.close()}
      >
        <div className="config-body">
          <div className="config-head">
            <h2>Configuration</h2>
            <button onClick={() => dialog.current?.close()} aria-label="Close">
              ✕
            </button>
          </div>

          <h3>GitHub repositories</h3>
          {config.repos.length === 0 ? (
            <p className="message">None configured</p>
          ) : (
            <ul className="config-list">
              {config.repos.map((r) => (
                <li key={r.name} className={r.pattern ? 'pattern' : ''}>
                  <span className="name">{r.name}</span>
                  <span className="muted">
                    PRs from <code>{r.branchPrefix}*</code>
                  </span>
                  {r.pattern && (
                    <span className="muted">
                      {r.listedAt ? `${r.matches.length} repos, listed ${timeAgo(r.listedAt)}` : 'not listed yet'}
                      {r.exclude?.length ? ` · excluding ${r.exclude.join(', ')}` : ''}
                    </span>
                  )}
                  {r.error && (
                    <span className="badge error" title={r.error}>
                      listing failed
                    </span>
                  )}
                  {!r.pattern && config.renovate && (
                    <>
                      <span className="spacer" />
                      <RunRenovateButton repo={r.name} small />
                    </>
                  )}
                  {r.pattern && r.matches.length > 0 && (
                    <ul className="matches">
                      {r.matches.map((m) => (
                        <li key={m}>
                          <span>{m}</span>
                          {config.renovate && (
                            <>
                              <span className="spacer" />
                              <RunRenovateButton repo={m} small />
                            </>
                          )}
                        </li>
                      ))}
                    </ul>
                  )}
                </li>
              ))}
            </ul>
          )}

          <h3>Kubernetes clusters</h3>
          {config.clusters.length === 0 ? (
            <p className="message">None configured</p>
          ) : (
            <ul className="config-list">
              {config.clusters.map((cc) => {
                const c = clusters.find((s) => s.name === cc.name);
                return (
                  <li key={cc.name}>
                    <span className="name">⎈ {cc.name}</span>
                    <span className="muted">
                      {cc.inCluster ? (
                        'in-cluster service account'
                      ) : (
                        <>
                          context <code>{cc.context}</code>
                        </>
                      )}
                    </span>
                    {cc.includeNamespaces?.length ? (
                      <span className="muted">only {cc.includeNamespaces.join(', ')}</span>
                    ) : null}
                    {cc.excludeNamespaces?.length ? (
                      <span className="muted">excluding {cc.excludeNamespaces.join(', ')}</span>
                    ) : null}
                    {c?.error ? (
                      <span className="badge error" title={c.error}>
                        sync error
                      </span>
                    ) : c?.syncedAt ? (
                      <span className="muted">
                        {c.containers} containers, synced {timeAgo(c.syncedAt)}
                      </span>
                    ) : (
                      <span className="muted">not synced yet</span>
                    )}
                  </li>
                );
              })}
            </ul>
          )}

          <h3>Settings</h3>
          <dl className="meta">
            <dt>Sync interval</dt>
            <dd>{config.syncInterval}</dd>
            <dt>Vulnerability scanning</dt>
            <dd>{config.scanning ? `on (Trivy${config.trivyVersion ? ` ${config.trivyVersion}` : ''})` : 'off'}</dd>
            <dt>GitHub API</dt>
            <dd>
              {config.githubAPIURL || 'api.github.com'}
              {!config.githubToken && <span className="muted"> · no token, public repos only</span>}
            </dd>
            <dt>Renovate</dt>
            <dd>
              {config.renovate ? (
                <>
                  <code>{config.renovate.image.replace(/^.*\//, '')}</code> as Jobs in ⎈ {config.renovate.cluster}/
                  {config.renovate.namespace}, {config.renovate.interval ? `every ${config.renovate.interval}` : 'from the UI only'}
                </>
              ) : (
                <span className="muted">not run by imginq</span>
              )}
            </dd>
            <dt>Soak</dt>
            <dd>
              {config.renovate?.soak ? (
                <>
                  {config.renovate.soak.default}, majors {config.renovate.soak.major}
                  <div className="muted">
                    {config.renovate.soak.fastLaneSeverity === 'NONE'
                      ? 'no fast lane'
                      : `${config.renovate.soak.fastLane} when fixing ${config.renovate.soak.fastLaneSeverity.toLowerCase()} CVEs`}
                    {' · '}
                    {config.renovate.soak.riskySeverity === 'NONE'
                      ? 'no risky soak'
                      : `${config.renovate.soak.risky} when introducing ${config.renovate.soak.riskySeverity.toLowerCase()} CVEs`}
                  </div>
                </>
              ) : (
                <span className="muted">set by Renovate's config</span>
              )}
            </dd>
          </dl>
        </div>
      </dialog>
    </>
  );
}

const TOP_N = 5;

// fixRank orders PRs by the critical, then high, then medium CVEs they fix
const fixRank = (pr: PullRequest) => {
  const f = pr.scan?.fixed ?? {};
  return [f.CRITICAL ?? 0, f.HIGH ?? 0, f.MEDIUM ?? 0];
};

const compareRank = (a: number[], b: number[]) => a.reduce((d, n, i) => d || b[i] - n, 0);

const hasCriticalOrHigh = (counts?: SeverityCounts) => (counts?.CRITICAL ?? 0) + (counts?.HIGH ?? 0) > 0;

function Dashboard({ hasClusters }: { hasClusters: boolean }) {
  const images = usePoll(listRunningImages, []);
  const prs = usePoll(() => listPullRequests('open'), []);
  const error = images.error ?? prs.error;

  const imageStats = useMemo(() => {
    const list = images.data ?? [];
    const byStatus = Object.fromEntries(
      (Object.keys(statusLabels) as UpdateStatus[]).map((s) => [s, list.filter((img) => img.status === s).length]),
    ) as Record<UpdateStatus, number>;
    const workloads = new Set(list.flatMap((img) => img.workloads.map(workloadLabel))).size;
    const behind = list
      .filter((img) => img.status === 'behind')
      .sort((a, b) => b.workloads.length - a.workloads.length);

    // CVEs summed per image, so a CVE in two images counts twice
    const cves: SeverityCounts = {};
    const fixable: SeverityCounts = {};
    let scanned = 0;
    let toScan = 0;
    for (const img of list) {
      if (img.scan?.status === 'done') {
        scanned++;
        for (const s of severityOrder) cves[s] = (cves[s] ?? 0) + (img.scan.counts?.[s] ?? 0);
      } else if (img.scan?.status === 'pending') {
        toScan++;
      }
      if (img.update?.status === 'done') {
        for (const s of severityOrder) fixable[s] = (fixable[s] ?? 0) + (img.update.fixed[s] ?? 0);
      }
    }
    return { total: list.length, workloads, byStatus, behind, cves, fixable, scanned, toScan };
  }, [images.data]);

  const prStats = useMemo(() => {
    const list = prs.data ?? [];
    const fixed: SeverityCounts = {};
    for (const pr of list) {
      for (const s of severityOrder) {
        fixed[s] = (fixed[s] ?? 0) + (pr.scan?.fixed[s] ?? 0);
      }
    }
    const fixers = list
      .filter((pr) => fixRank(pr).some((n) => n > 0))
      .sort((a, b) => compareRank(fixRank(a), fixRank(b)));
    return {
      total: list.length,
      fixed,
      fixers,
      fixCritical: list.filter((pr) => hasCriticalOrHigh(pr.scan?.fixed)).length,
      introduceCritical: list.filter((pr) => hasCriticalOrHigh(pr.scan?.introduced)).length,
      scanning: list.filter((pr) => pr.scan?.status === 'pending').length,
    };
  }, [prs.data]);

  const big = (loaded: unknown, n: number) => <div className="big">{loaded ? n : '…'}</div>;

  return (
    <div className="dashboard">
      {error && <p className="message error">{error}</p>}

      <div className="tiles">
        <section className="tile">
          <h2>
            <a href="#/images">Running images</a>
          </h2>
          {hasClusters ? (
            <>
              {big(images.data, imageStats.total)}
              <div className="sub">in {imageStats.workloads} {imageStats.workloads === 1 ? 'workload' : 'workloads'}</div>
              <ul className="breakdown">
                {(['behind', 'pending', 'pinned', 'current', 'unknown'] as const).map((s) => (
                  <li key={s}>
                    <a href={`#/images?status=${s}`} title={statusTitles[s]}>
                      <StatusBadge status={s} />
                    </a>
                    <span className="n">{imageStats.byStatus[s]}</span>
                  </li>
                ))}
              </ul>
            </>
          ) : (
            <p className="message">No clusters configured</p>
          )}
        </section>

        <section className="tile">
          <h2>
            <a href="#/prs">Open pull requests</a>
          </h2>
          {big(prs.data, prStats.total)}
          <ul className="breakdown">
            <li>
              <span>Fix critical or high CVEs</span>
              <span className="n good">{prStats.fixCritical}</span>
            </li>
            <li>
              <span>Introduce critical or high CVEs</span>
              <span className={`n ${prStats.introduceCritical ? 'bad' : ''}`}>{prStats.introduceCritical}</span>
            </li>
            <li>
              <span>Still scanning</span>
              <span className="n">{prStats.scanning}</span>
            </li>
          </ul>
        </section>

        {hasClusters && (imageStats.scanned > 0 || imageStats.toScan > 0) && (
          <section className="tile">
            <h2 title="Summed per image, so a CVE in two images counts twice">
              <a href="#/images">CVEs in running images</a>
            </h2>
            {big(images.data, total(imageStats.cves))}
            <div className="sub">
              in {imageStats.scanned} scanned images
              {imageStats.toScan > 0 && ` · ${imageStats.toScan} still scanning`}
            </div>
            <ul className="breakdown">
              {(['CRITICAL', 'HIGH', 'MEDIUM', 'LOW'] as const).map((s) => (
                <li key={s}>
                  <span className={`severity-label ${s.toLowerCase()}`}>{s.toLowerCase()}</span>
                  <span className="n">
                    {imageStats.cves[s] ?? 0}
                    {(imageStats.fixable[s] ?? 0) > 0 && (
                      <span className="muted fixable" title="Fixed by updating to the newest version">
                        {' '}
                        −{imageStats.fixable[s]}
                      </span>
                    )}
                  </span>
                </li>
              ))}
            </ul>
          </section>
        )}

        <section className="tile">
          <h2 title="Summed per PR, so a CVE fixed by two PRs counts twice">CVEs fixed by open PRs</h2>
          {big(prs.data, total(prStats.fixed))}
          <ul className="breakdown">
            {severityOrder.map((s) => (
              <li key={s}>
                <span className={`severity-label ${s.toLowerCase()}`}>{s.toLowerCase()}</span>
                <span className="n">{prStats.fixed[s] ?? 0}</span>
              </li>
            ))}
          </ul>
        </section>
      </div>

      <div className="panels">
        <section>
          <h2>Merge first</h2>
          <p className="sub">Open PRs fixing the most critical, high and medium CVEs</p>
          {prs.data && prStats.fixers.length === 0 && <p className="message">No open PR fixes a CVE.</p>}
          {prStats.fixers.length > 0 && (
            <div className="table-wrap">
              <table>
                <tbody>
                  {prStats.fixers.slice(0, TOP_N).map((pr) => (
                    <tr key={`${pr.repo}#${pr.number}`}>
                      <td className="pr">
                        <a href={prHref(pr)}>
                          <span className="number">#{pr.number}</span> {pr.title}
                        </a>
                        <div className="sub">{pr.repo}</div>
                      </td>
                      <td className="nowrap">{pr.scan && <ScanBadges scan={pr.scan} />}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
          {prStats.fixers.length > TOP_N && (
            <a className="more" href="#/prs">
              All {prStats.fixers.length} PRs that fix CVEs →
            </a>
          )}
        </section>

        {hasClusters && (
          <section>
            <h2>Not covered by Renovate</h2>
            <p className="sub">Running images with a newer version and no open PR, most-used first</p>
            {images.data && imageStats.behind.length === 0 && <p className="message">Every outdated image has a PR.</p>}
            {imageStats.behind.length > 0 && (
              <div className="table-wrap">
                <table>
                  <tbody>
                    {imageStats.behind.slice(0, TOP_N).map((img) => (
                      <tr key={`${img.repository}:${img.tag ?? img.digests.join(',')}`}>
                        <td>
                          <div className="name">
                            <a href={imageHref(img.repository)}>{shortRepository(img.repository)}</a>
                          </div>
                          <div className="change">
                            <code>{img.tag}</code>
                            <LatestVersion version={img.version} />
                            {img.soak && <SoakBadge soak={img.soak} />}
                          </div>
                        </td>
                        <td className="nowrap muted">
                          {img.workloads.length} {img.workloads.length === 1 ? 'workload' : 'workloads'}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
            {imageStats.behind.length > TOP_N && (
              <a className="more" href="#/images?status=behind">
                All {imageStats.behind.length} behind images →
              </a>
            )}
          </section>
        )}
      </div>

      {hasClusters && <Trends />}
    </div>
  );
}

function PullRequestList() {
  const [showClosed, setShowClosed] = useState(false);
  const [kind, setKind] = useState<KindFilter>('all');
  const [query, setQuery] = useState('');
  const { data, error } = usePoll(() => listPullRequests(showClosed ? 'all' : 'open'), [showClosed]);

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    return (data ?? []).filter((pr) => {
      if (kind === 'none' && pr.changes.length > 0) return false;
      if (kind !== 'all' && kind !== 'none' && !pr.changes.some((c) => c.kind === kind)) return false;
      if (!q) return true;
      return (
        pr.title.toLowerCase().includes(q) ||
        pr.changes.some((c) => c.name.toLowerCase().includes(q) || c.file.toLowerCase().includes(q))
      );
    });
  }, [data, kind, query]);

  const counts = useMemo(() => {
    const prs = data ?? [];
    return {
      all: prs.length,
      image: prs.filter((p) => p.changes.some((c) => c.kind === 'image')).length,
      helmChart: prs.filter((p) => p.changes.some((c) => c.kind === 'helmChart')).length,
      none: prs.filter((p) => p.changes.length === 0).length,
    };
  }, [data]);

  return (
    <>
      <div className="toolbar">
        <div className="segmented" role="group" aria-label="Filter by change kind">
          {(
            [
              ['all', 'All'],
              ['image', 'Images'],
              ['helmChart', 'Charts'],
              ['none', 'No changes found'],
            ] as const
          ).map(([value, label]) => (
            <button key={value} className={kind === value ? 'active' : ''} onClick={() => setKind(value)}>
              {label} <span className="count">{counts[value]}</span>
            </button>
          ))}
        </div>
        <input
          type="search"
          placeholder="Filter by name, title or file"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
        />
        <label className="checkbox">
          <input type="checkbox" checked={showClosed} onChange={(e) => setShowClosed(e.target.checked)} />
          Show closed
        </label>
      </div>

      {error && <p className="message error">{error}</p>}
      {data && filtered.length === 0 && <p className="message">No pull requests match.</p>}
      {!data && !error && <p className="message">Loading…</p>}

      {filtered.length > 0 && (
        <div className="table-wrap">
          <table>
            <thead>
              <tr>
                <th>Pull request</th>
                <th>Changes</th>
                <th>CVEs</th>
                <th>Pipeline</th>
                <th>Updated</th>
              </tr>
            </thead>
            <tbody>
              {filtered.map((pr) => (
                <tr key={`${pr.repo}#${pr.number}`} className={pr.state === 'closed' ? 'closed' : ''}>
                  <td className="pr">
                    <a href={prHref(pr)}>
                      <span className="number">#{pr.number}</span> {pr.title}
                    </a>
                    <div className="sub">
                      {pr.repo}
                      {pr.state === 'closed' && <span className="badge">closed</span>}
                    </div>
                  </td>
                  <td>
                    {pr.changes.length === 0 ? (
                      <span className="muted">none found</span>
                    ) : (
                      pr.changes.map((c, i) => <ChangeLine key={i} change={c} />)
                    )}
                  </td>
                  <td>
                    <PullRequestScanSummary scan={pr.scan} />
                  </td>
                  <td>
                    <Stages stages={pr.stages} />
                  </td>
                  <td className="nowrap muted" title={pr.updatedAt}>
                    {timeAgo(pr.updatedAt)}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </>
  );
}

function ChangeLine({ change }: { change: ChangeView }) {
  const from = version(change, 'from');
  const to = version(change, 'to');
  const b = bump(from, to);
  return (
    <div className="change">
      <span className={`badge kind ${change.kind}`}>{kindLabel(change.kind)}</span>
      <span className="name" title={change.repoURL ?? change.name}>
        {change.kind === 'image' ? <a href={imageHref(change.name)}>{change.name}</a> : change.name}
      </span>
      <span className="versions">
        {from} → {to}
      </span>
      {b !== 'other' && <span className={`badge bump ${b}`}>{b}</span>}
      <RunningBadge running={change.running} />
      {change.renders && <ChartImagesBadge renders={change.renders} />}
    </div>
  );
}

const severityOrder = ['CRITICAL', 'HIGH', 'MEDIUM', 'LOW', 'UNKNOWN'] as const;
const severityShort = { CRITICAL: 'C', HIGH: 'H', MEDIUM: 'M', LOW: 'L', UNKNOWN: '?' } as const;

const total = (counts: SeverityCounts) => severityOrder.reduce((n, s) => n + (counts[s] ?? 0), 0);

// Severities shows critical, high and medium counts, e.g. "2C 5H 11M", with every count in the title
function Severities({ counts, kind }: { counts: SeverityCounts; kind: 'fixed' | 'introduced' }) {
  const shown = (['CRITICAL', 'HIGH', 'MEDIUM'] as const).filter((s) => counts[s]);
  const title = severityOrder
    .filter((s) => counts[s])
    .map((s) => `${counts[s]} ${s.toLowerCase()}`)
    .join(', ');
  const verb = kind === 'fixed' ? 'fixes' : 'introduces';
  if (shown.length === 0) {
    return total(counts) > 0 ? (
      <span className="muted" title={`${verb} ${title}`}>
        {verb} {total(counts)} low
      </span>
    ) : null;
  }
  return (
    <span className={`badge cve ${kind}`} title={`${verb} ${title}`}>
      {kind === 'fixed' ? '−' : '+'}
      {shown.map((s) => `${counts[s]}${severityShort[s]}`).join(' ')}
    </span>
  );
}

function ScanBadges({ scan }: { scan: ScanDelta | PullRequestScan }) {
  if (scan.status === 'pending' && total(scan.fixed) + total(scan.introduced) === 0) {
    return <span className="muted">scanning…</span>;
  }
  if (scan.status === 'skipped') {
    const side = 'from' in scan ? [scan.from, scan.to].find((s) => s?.status === 'skipped') : undefined;
    return (
      <span className="muted" title={side?.error}>
        not scanned
      </span>
    );
  }
  if (scan.status === 'error' && total(scan.fixed) + total(scan.introduced) === 0) {
    const side = 'from' in scan ? [scan.from, scan.to].find((s) => s?.status === 'error') : undefined;
    return (
      <span className="badge error" title={side?.error}>
        scan failed
      </span>
    );
  }
  if (total(scan.fixed) + total(scan.introduced) === 0) {
    return <span className="muted">no change</span>;
  }
  return (
    <span className="change">
      <Severities counts={scan.fixed} kind="fixed" />
      <Severities counts={scan.introduced} kind="introduced" />
    </span>
  );
}

function PullRequestScanSummary({ scan }: { scan?: PullRequestScan }) {
  if (!scan) {
    return <span className="muted">—</span>;
  }
  return (
    <>
      <ScanBadges scan={scan} />
      {scan.status !== 'done' && (scan.pending > 0 || scan.failed > 0) && total(scan.fixed) + total(scan.introduced) > 0 && (
        <div className="sub">
          {scan.pending > 0 && `${scan.pending} of ${scan.images} still scanning`}
          {scan.failed > 0 && `${scan.failed} of ${scan.images} failed`}
        </div>
      )}
    </>
  );
}

const DAY_MS = 24 * 60 * 60 * 1000;

// SoakBadge shows how far the update to an image's newest version is through its soak
function SoakBadge({ soak }: { soak: SoakDecision }) {
  const released = soak.releasedAt
    ? `released ${timeAgo(soak.releasedAt)} (${soak.releaseSource})`
    : 'release date not known yet';
  const title = [soak.reason, released, soak.scanPending ? 'CVEs not known yet, so no fast lane' : '']
    .filter(Boolean)
    .join(' · ');

  const total = soak.soak ?? '';
  const days = soak.releasedAt ? Math.floor((Date.now() - Date.parse(soak.releasedAt)) / DAY_MS) : null;
  const progress = days === null ? total : `${days} of ${total}`;

  if (soak.lane === 'fast') {
    return (
      <span className="badge running" title={soak.released ? `${title} · Renovate may propose it` : title}>
        {soak.released ? 'fast lane' : `fast lane · ${progress}`}
      </span>
    );
  }
  if (soak.released) {
    return (
      <span className="badge" title={`${title} · Renovate may propose it`}>
        soaked
      </span>
    );
  }
  return (
    <span className={`badge ${soak.lane === 'risky' ? 'error' : 'kind image'}`} title={title}>
      {soak.lane === 'risky' ? `risky · ${progress}` : `soaking ${progress}`}
    </span>
  );
}

// CveCounts shows the critical, high and medium CVEs of an image, e.g. "2C 5H 11M"
function CveCounts({ counts }: { counts: SeverityCounts }) {
  const shown = (['CRITICAL', 'HIGH', 'MEDIUM'] as const).filter((s) => counts[s]);
  const title = severityOrder
    .filter((s) => counts[s])
    .map((s) => `${counts[s]} ${s.toLowerCase()}`)
    .join(', ');
  if (total(counts) === 0) return <span className="muted">none</span>;
  if (shown.length === 0) return <span className="muted" title={title}>{total(counts)} low</span>;
  return (
    <span className="change" title={title}>
      {shown.map((s) => (
        <span key={s} className={`badge severity ${s.toLowerCase()}`}>
          {counts[s]}
          {severityShort[s]}
        </span>
      ))}
    </span>
  );
}

// RunningScan shows a running image's CVEs and what updating to its newest version would change
function RunningScan({ image }: { image: RunningImage }) {
  const s = image.scan;
  if (!s) return <span className="muted" title="No running digest known, or scanning is off">—</span>;
  return (
    <>
      {s.status === 'pending' ? (
        <span className="muted">scanning…</span>
      ) : s.status === 'skipped' ? (
        <span className="muted" title={s.error}>
          not scanned
        </span>
      ) : s.status === 'error' ? (
        <span className="badge error" title={s.error}>
          scan failed
        </span>
      ) : (
        <CveCounts counts={s.counts ?? {}} />
      )}
      {image.update && image.version?.latest && (
        <div className="sub" title={`CVEs updating to ${image.version.latest} fixes and introduces`}>
          update: <ScanBadges scan={image.update} />
        </div>
      )}
    </>
  );
}

// VulnerabilityTable lists CVEs, with a first column when they're marked fixed or introduced
function VulnerabilityTable({ rows }: { rows: { vuln: Vulnerability; kind?: 'fixed' | 'introduced' }[] }) {
  const marked = rows.some((r) => r.kind);
  return (
    <div className="table-wrap">
      <table>
        <thead>
          <tr>
            {marked && <th></th>}
            <th>Severity</th>
            <th>Vulnerability</th>
            <th>Package</th>
            <th>Installed</th>
            <th>Fixed in</th>
          </tr>
        </thead>
        <tbody>
          {rows.map(({ vuln: v, kind }, i) => (
            <tr key={i}>
              {marked && <td>{kind && <span className={`badge cve ${kind}`}>{kind}</span>}</td>}
              <td>
                <span className={`badge severity ${v.severity.toLowerCase()}`}>{v.severity.toLowerCase()}</span>
              </td>
              <td>
                {v.url ? (
                  <a href={v.url} target="_blank" rel="noreferrer">
                    {v.id}
                  </a>
                ) : (
                  v.id
                )}
                {v.title && <div className="sub">{v.title}</div>}
              </td>
              <td>{v.package}</td>
              <td>
                <code>{v.installed}</code>
              </td>
              <td>{v.fixed ? <code>{v.fixed}</code> : <span className="muted">no fix</span>}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

// Vulnerabilities lists the CVEs an image change fixes and introduces
function Vulnerabilities({ label, scan }: { label: string; scan: ScanDelta }) {
  const fixed = scan.fixedVulnerabilities ?? [];
  const introduced = scan.introducedVulnerabilities ?? [];
  if (fixed.length + introduced.length === 0) {
    return null;
  }
  return (
    <details className="vulns">
      <summary>
        {label}: <ScanBadges scan={scan} />
      </summary>
      <VulnerabilityTable
        rows={[...fixed.map((vuln) => ({ vuln, kind: 'fixed' as const })), ...introduced.map((vuln) => ({ vuln, kind: 'introduced' as const }))]}
      />
    </details>
  );
}

// ChartImagesBadge summarizes the images a chart change updates across the apps using it
function ChartImagesBadge({ renders }: { renders: ChartRenderView[] }) {
  const failed = renders.filter((r) => r.error);
  if (failed.length > 0) {
    return (
      <span className="badge error" title={failed.map((r) => `${r.app}: ${r.error}`).join('\n')}>
        render failed
      </span>
    );
  }
  if (renders.length === 0) {
    return null;
  }
  const changes = new Set(renders.flatMap((r) => r.images.map((i) => `${i.from} → ${i.to}`)));
  if (changes.size === 0) {
    return <span className="badge">no image changes</span>;
  }
  return (
    <span className="badge kind image" title={[...changes].join('\n')}>
      {changes.size} image {changes.size === 1 ? 'change' : 'changes'}
    </span>
  );
}

// RunningBadge shows how many workloads run a change's From version
function RunningBadge({ running }: { running: Workload[] | null }) {
  if (running === null) {
    return null;
  }
  if (running.length === 0) {
    return (
      <span className="badge" title="No inventoried workload runs this version">
        not running
      </span>
    );
  }
  return (
    <span className="badge running" title={running.map(workloadLabel).join('\n')}>
      running in {running.length}
    </span>
  );
}

// Beyond this many clusters a running image shows a count instead of their names
const maxClusterNames = 3;

// WorkloadIcon is a cube standing in for the word "workload"
function WorkloadIcon() {
  return (
    <svg className="icon" viewBox="0 0 16 16" aria-hidden="true">
      <path d="M8 1.5 14 5v6l-6 3.5L2 11V5zM2 5l6 3.5L14 5M8 8.5v6" />
    </svg>
  );
}

// WorkloadCounts sums up where an image runs, with its clusters when several are tracked; its page lists the workloads
function WorkloadCounts({ workloads, multiCluster }: { workloads: Workload[]; multiCluster: boolean }) {
  const clusters = [...new Set(workloads.map((w) => w.cluster))].sort();
  const label = `${workloads.length} ${workloads.length === 1 ? 'workload' : 'workloads'}`;
  return (
    <div title={[label, ...workloads.map(workloadLabel)].join('\n')}>
      <span className="workload-count" role="img" aria-label={label}>
        <WorkloadIcon />
        {workloads.length}
      </span>
      {multiCluster && (
        <div className="sub">{clusters.length > maxClusterNames ? `${clusters.length} clusters` : clusters.join(', ')}</div>
      )}
    </div>
  );
}

function WorkloadList({ workloads }: { workloads: Workload[] }) {
  return (
    <ul className="workloads">
      {workloads.map((w) => (
        <li key={workloadLabel(w)}>
          <span className="muted">
            {w.cluster}/{w.namespace}/
          </span>
          {w.kind}/{w.name}
          {w.argoApp && <span className="badge">argo: {w.argoApp}</span>}
          {w.source && (
            <span className="badge kind helmChart" title={`${w.source.repoURL}${w.source.path ? ' ' + w.source.path : ''}`}>
              {w.source.chart} {shortRevision(w.source.version)}
              {w.source.latest && w.source.latest !== w.source.version && ` (latest ${w.source.latest})`}
            </span>
          )}
        </li>
      ))}
    </ul>
  );
}

const statusLabels: Record<UpdateStatus, string> = {
  behind: 'Behind, no PR',
  pending: 'Update pending',
  pinned: 'Pinned by chart',
  current: 'Current',
  unknown: 'Unknown',
};

const statusTitles: Record<UpdateStatus, string> = {
  behind: 'A newer version exists and no open PR updates it',
  pending: 'An open PR updates this version',
  pinned: "A newer image exists, but it's set by a chart from a Helm or OCI repository, so chart updates decide when it changes",
  current: 'No newer version of the same variant',
  unknown: "The tag isn't a version, or its tags couldn't be listed",
};

function StatusBadge({ status, version }: { status: UpdateStatus; version?: Version }) {
  const cls = { behind: 'bump major', pending: 'kind image', pinned: '', current: 'running', unknown: '' }[status];
  return (
    <span className={`badge ${cls}`} title={version?.error || statusTitles[status]}>
      {statusLabels[status].toLowerCase()}
    </span>
  );
}

// LatestVersion shows the newest version when it differs from the current one
function LatestVersion({ version }: { version?: Version }) {
  if (!version?.latest || version.latest === version.current) {
    return null;
  }
  const b = bump(version.current, version.latest);
  return (
    <>
      <span className="versions">→ {version.latest}</span>
      {b !== 'other' && <span className={`badge bump ${b}`}>{b}</span>}
    </>
  );
}

type StatusFilter = 'all' | UpdateStatus;

const isStatus = (s: string | null): s is UpdateStatus => s != null && s in statusLabels;

function RunningImages({
  hasClusters,
  multiCluster,
  initialStatus,
}: {
  hasClusters: boolean;
  multiCluster: boolean;
  initialStatus: string | null;
}) {
  const [status, setStatus] = useState<StatusFilter>(isStatus(initialStatus) ? initialStatus : 'all');
  const [query, setQuery] = useState('');
  const { data, error } = usePoll(listRunningImages, []);

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    return (data ?? []).filter((img) => {
      if (status !== 'all' && img.status !== status) return false;
      if (!q) return true;
      return (
        img.repository.toLowerCase().includes(q) ||
        img.workloads.some((w) => workloadLabel(w).toLowerCase().includes(q) || (w.argoApp ?? '').toLowerCase().includes(q))
      );
    });
  }, [data, query, status]);

  const counts = (s: StatusFilter) => (data ?? []).filter((img) => s === 'all' || img.status === s).length;

  if (!hasClusters) {
    return (
      <p className="message">
        No Kubernetes clusters configured. Add <code>kubernetes.clusters</code> to the daemon config.
      </p>
    );
  }

  return (
    <>
      <div className="toolbar">
        <div className="segmented" role="group" aria-label="Filter by update status">
          {(['all', 'behind', 'pending', 'pinned', 'current', 'unknown'] as const).map((s) => (
            <button
              key={s}
              className={status === s ? 'active' : ''}
              onClick={() => setStatus(s)}
              title={s === 'all' ? undefined : statusTitles[s]}
            >
              {s === 'all' ? 'All' : statusLabels[s]} <span className="count">{counts(s)}</span>
            </button>
          ))}
        </div>
        <input
          type="search"
          placeholder="Filter by image, workload or Argo app"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
        />
      </div>

      {error && <p className="message error">{error}</p>}
      {data && filtered.length === 0 && <p className="message">No running images match.</p>}
      {!data && !error && <p className="message">Loading…</p>}

      {filtered.length > 0 && (
        <div className="table-wrap">
          <table>
            <thead>
              <tr>
                <th>Image</th>
                <th>Status</th>
                <th>CVEs</th>
                <th>Running in</th>
                <th>Open update PR</th>
              </tr>
            </thead>
            <tbody>
              {filtered.map((img) => (
                <tr key={`${img.repository}:${img.tag ?? img.digests.join(',')}`}>
                  <td>
                    <div className="name">
                      <a href={imageHref(img.repository)}>{shortRepository(img.repository)}</a>
                    </div>
                    <div className="sub">
                      <code>{img.tag ?? img.digests[0]?.slice(0, 19)}</code>
                    </div>
                  </td>
                  <td>
                    <div className="change">
                      <StatusBadge status={img.status} version={img.version} />
                      <LatestVersion version={img.version} />
                    </div>
                    {img.soak && img.status !== 'pending' && (
                      <div className="sub">
                        <SoakBadge soak={img.soak} />
                      </div>
                    )}
                    {deploymentGroups(img).length > 1 && (
                      <div className="sub">
                        <span className="badge" title={deployedWaysHelp}>
                          deployed {deploymentGroups(img).length} ways
                        </span>
                      </div>
                    )}
                  </td>
                  <td>
                    <RunningScan image={img} />
                  </td>
                  <td>
                    <WorkloadCounts workloads={img.workloads} multiCluster={multiCluster} />
                  </td>
                  <td>
                    {img.coveredBy.length === 0 ? (
                      <span className="badge">none</span>
                    ) : (
                      img.coveredBy.map((c) => (
                        <div key={`${c.repo}#${c.number}${c.via ?? ''}${c.from}`} className="change">
                          <a href={`#/pr/${c.repo}/${c.number}`}>#{c.number}</a>
                          <span className="versions">{c.to ? `→ ${imageTag(c.to)}` : 'removed'}</span>
                          {c.via && <span className="muted">via chart {c.via}</span>}
                          {!c.matchesRunning && (
                            <span className="badge bump major" title={`PR updates ${c.from}, which isn't the version running`}>
                              updates a different version
                            </span>
                          )}
                        </div>
                      ))
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </>
  );
}

// bySeverity orders CVEs most severe first, those with a fix first within a severity
const bySeverity = (a: Vulnerability, b: Vulnerability) =>
  severityOrder.indexOf(a.severity) - severityOrder.indexOf(b.severity) ||
  Number(!a.fixed) - Number(!b.fixed) ||
  a.id.localeCompare(b.id);

const signed = (n: number) => (n > 0 ? `+${n}` : n < 0 ? `−${-n}` : '0');

const severityList = (counts: SeverityCounts) =>
  severityOrder
    .filter((s) => counts[s])
    .map((s) => `${counts[s]} ${s.toLowerCase()}`)
    .join(', ');

// CveComparison shows an image's CVEs now and, when there's a newer version, after updating to it
function CveComparison({ image }: { image: RunningImage }) {
  const now = image.scan?.counts ?? {};
  const update = image.update;
  const after = update?.status === 'done' ? update.to?.counts ?? {} : null;
  // Low and unknown are only worth a column when there are some
  const columns = severityOrder.filter(
    (s) => s === 'CRITICAL' || s === 'HIGH' || s === 'MEDIUM' || now[s] || after?.[s],
  );
  const fixed = update && severityList(update.fixed);
  const introduced = update && severityList(update.introduced);
  const unavailable = { pending: 'scanning…', error: 'scan failed', skipped: 'not scanned' };

  return (
    <>
      <div className="table-wrap">
        <table className="cve-table">
          <thead>
            <tr>
              <th></th>
              {columns.map((s) => (
                <th key={s}>
                  <span className={`severity-label ${s.toLowerCase()}`}>{s.toLowerCase()}</span>
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            <tr>
              <td>Running {image.tag}</td>
              {columns.map((s) => (
                <td key={s}>{now[s] ?? 0}</td>
              ))}
            </tr>
            {update && after && (
              <>
                <tr>
                  <td>After update</td>
                  {columns.map((s) => (
                    <td key={s}>{after[s] ?? 0}</td>
                  ))}
                </tr>
                <tr>
                  <td>Change</td>
                  {columns.map((s) => {
                    const change = (after[s] ?? 0) - (now[s] ?? 0);
                    return (
                      <td key={s} className={change < 0 ? 'better' : change > 0 ? 'worse' : 'muted'}>
                        {signed(change)}
                      </td>
                    );
                  })}
                </tr>
              </>
            )}
            {update && !after && (
              <tr>
                <td>After update</td>
                <td className="muted" colSpan={columns.length} title={update.to?.error}>
                  {unavailable[update.status as keyof typeof unavailable]}
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
      {after && (fixed || introduced) && (
        <p className="sub">
          {fixed && <>Fixes {fixed}</>}
          {fixed && introduced && ' · '}
          {introduced && <span className="worse">introduces {introduced}</span>}
        </p>
      )}
    </>
  );
}

function CopyButton({ text }: { text: string }) {
  const [copied, setCopied] = useState(false);
  // Browsers only offer the clipboard on HTTPS and localhost
  if (!navigator.clipboard) return null;
  return (
    <button
      className="copy"
      title="Copy digest"
      aria-label="Copy digest"
      onClick={() =>
        navigator.clipboard.writeText(text).then(() => {
          setCopied(true);
          setTimeout(() => setCopied(false), 1500);
        })
      }
    >
      {copied ? '✓' : '⧉'}
    </button>
  );
}

// statusHelp explains an image's update status in terms of this image
function statusHelp(image: RunningImage): string {
  const latest = image.version?.latest;
  switch (image.status) {
    case 'behind':
      return `Version ${latest} is available and no open PR updates this image, so nothing will update it until one does. This usually means Renovate doesn't cover it.`;
    case 'pending': {
      const pr = image.coveredBy.find((c) => c.matchesRunning);
      return `Open PR #${pr?.number} updates this version${pr?.to ? ` to ${imageTag(pr.to)}` : ''}. Merging it is the next step.`;
    }
    case 'pinned': {
      const charts = [...new Set(image.workloads.flatMap((w) => (w.source ? [w.source.chart] : [])))];
      const by = charts.length > 0 ? `the ${charts.join(', ')} chart` : 'a chart';
      return `Version ${latest} is available, but this tag is set by ${by} from a Helm or OCI repository. It only changes when the chart is updated, so look for a newer chart rather than a newer image.`;
    }
    case 'current':
      return "This is the newest version of the same variant, so there's nothing to update.";
    case 'unknown':
      return image.version?.error
        ? `The registry's tags couldn't be listed, so newer versions can't be found: ${image.version.error}`
        : "This tag isn't a version number, so it can't be compared with newer versions.";
  }
}

const bumpHelp: Record<string, string> = {
  major: 'The first number of the version changes. Major versions can include breaking changes, so check the release notes before updating.',
  minor: 'The middle number of the version changes. Minor versions usually add features and stay compatible.',
  patch: 'The last number of the version changes. Patch versions are usually only bug and security fixes.',
};

const soakLaneHelp: Record<SoakDecision['lane'], string> = {
  default: 'This update has the normal soak.',
  major: 'This is a major update, so it has the longer major soak.',
  fast: 'This update fixes more serious CVEs than it introduces, so it has a short soak.',
  risky: 'This update introduces more serious CVEs than it fixes, so it has a longer soak.',
};

// soakHelp explains how long an update waits, and where it has got to
function soakHelp(soak: SoakDecision): string {
  const parts = ['Soak is how long a new version waits before Renovate may propose it, so broken releases get caught first.', soakLaneHelp[soak.lane]];
  if (soak.scanPending) parts.push("Its CVEs aren't known yet, so it can't use the fast lane.");
  if (soak.released) {
    parts.push('It has finished soaking, so Renovate may propose it.');
  } else if (soak.eligibleAt) {
    parts.push(`Renovate may propose it ${timeUntil(soak.eligibleAt)}.`);
  }
  return parts.join(' ');
}

// SOAK_OVERDUE_MS is how long after a soak ends the lack of a PR stops looking like a wait for
// Renovate's next run
const SOAK_OVERDUE_MS = 2 * DAY_MS;

// SoakState names where an update is while there's no PR, in the colours of the soak badges
function SoakState({ soak }: { soak: SoakDecision }) {
  if (soak.lane === 'fast') return <span className="badge running big-badge">Fastlane</span>;
  if (soak.released) return <span className="badge big-badge">Soaked</span>;
  if (soak.lane === 'risky') return <span className="badge error big-badge">Risky</span>;
  return <span className="badge kind image big-badge">Soaking</span>;
}

// SoakProgress shows how far an update has got through its soak, until Renovate may propose it.
// May, not will: the PR only comes on Renovate's next run, and only if Renovate covers the image.
function SoakProgress({ soak }: { soak: SoakDecision }) {
  if (!soak.releasedAt || !soak.eligibleAt) {
    return (
      <p className="sub">
        {soak.newest} soaks for {soak.soak} before Renovate may propose it. Its release date isn't known yet, so the soak hasn't
        started counting.
      </p>
    );
  }
  const released = Date.parse(soak.releasedAt);
  const eligible = Date.parse(soak.eligibleAt);
  const elapsed = Math.max(Date.now() - released, 0);
  const percent = soak.released ? 100 : Math.min(Math.floor((elapsed / (eligible - released)) * 100), 100);
  const overdue = soak.released && Date.now() - eligible > SOAK_OVERDUE_MS;
  return (
    <>
      <div
        className={`soak-bar ${soak.lane}`}
        role="progressbar"
        aria-label={`Soak of ${soak.newest}`}
        aria-valuenow={percent}
        aria-valuemin={0}
        aria-valuemax={100}
        title={`Counted from its ${soak.releaseSource}, ${timeAgo(soak.releasedAt)}`}
      >
        <div style={{ width: `${percent}%` }} />
      </div>
      {overdue ? (
        <p className="sub">
          <span className="worse">Soak finished {timeAgo(soak.eligibleAt)}, but no PR has appeared.</span> Check that Renovate covers
          this image.
        </p>
      ) : soak.released ? (
        <p className="sub">Soak finished. Renovate may open a PR for {soak.newest} on its next run.</p>
      ) : (
        <p className="sub">
          {duration(elapsed)} of {soak.soak}. Renovate may propose {soak.newest} {timeUntil(soak.eligibleAt)}.
        </p>
      )}
    </>
  );
}

const deployedWaysHelp =
  "This version's workloads come from different sources, so an update to one source may not reach all of them.";

interface DeploymentGroup {
  key: string;
  label: string;
  help: string;
  workloads: Workload[];
  // Open PRs known to update this group; a PR changing an image directly can't be tied to workloads
  prs: Coverage[];
}

// deploymentGroups splits an image's workloads by what deploys them
function deploymentGroups(image: RunningImage): DeploymentGroup[] {
  const groups = new Map<string, DeploymentGroup>();
  for (const w of image.workloads) {
    const s = w.source;
    const key = s ? `${s.kind}/${s.chart}` : 'none';
    let group = groups.get(key);
    if (!group) {
      group = {
        key,
        label: !s ? 'No known source' : s.kind === 'chart' ? `Chart: ${s.chart}` : `Git chart: ${s.chart}`,
        help: !s
          ? "No Argo CD source declares these. An operator may create them, or they're deployed some other way, so an update elsewhere may or may not reach them."
          : s.kind === 'chart'
            ? 'From a Helm or OCI repository. These change when the chart is updated.'
            : 'From a chart in a Git repository, so the image is set in that repository or its values.',
        workloads: [],
        prs: s ? image.coveredBy.filter((c) => c.via?.startsWith(`${s.chart} `)) : [],
      };
      groups.set(key, group);
    }
    group.workloads.push(w);
  }
  return [...groups.values()];
}

// DigestScans compares the digests of a tag that runs as several
function DigestScans({ image, scans }: { image: RunningImage; scans: ScanSide[] }) {
  return (
    <>
      <p className="sub">
        This tag runs as {scans.length} different digests, so its workloads aren't all the same build. The table above is for the
        marked one.
      </p>
      <ul className="workloads">
        {scans.map((s) => (
          <li key={s.digest}>
            <a href={`#/history?q=${s.digest}`} title={s.digest}>
              <code>{s.digest?.slice(0, 19)}…</code>
            </a>
            {s.status === 'done' ? (
              <CveCounts counts={s.counts ?? {}} />
            ) : (
              <span className="muted" title={s.error}>
                {s.status === 'pending' ? 'scanning…' : s.status === 'error' ? 'scan failed' : 'not scanned'}
              </span>
            )}
            {s.digest === image.scan?.digest && <span className="muted">shown above</span>}
          </li>
        ))}
      </ul>
    </>
  );
}

// PinnedByChart explains why a chart's image gets no PR of its own: it changes with the chart.
// Soak decisions exist for these images, but Renovate's rules for them match the image and not
// the chart, so they don't affect when the chart updates and aren't shown.
function PinnedByChart({ image }: { image: RunningImage }) {
  const sources = [...new Map(image.workloads.flatMap((w) => (w.source ? [[w.source.chart, w.source] as const] : []))).values()];
  const charts = sources.map((s) => s.chart).join(', ');
  const newer = sources.flatMap((s) => (s.latest && s.latest !== s.version ? [`${s.chart} ${s.latest}`] : []));
  return (
    <>
      <div className="big">
        <span className="badge big-badge">Pinned</span>
      </div>
      <p className="sub">
        Set by {sources.length === 0 ? 'a chart' : `the ${charts} ${sources.length === 1 ? 'chart' : 'charts'}`}, so no image PR will
        come.{' '}
        {newer.length > 0
          ? `A PR updating to ${newer.join(' and ')} would update ${sources.length > 1 ? 'its workloads' : 'this image'}.`
          : "The newest chart still uses this image, so it changes when a newer chart is released."}
      </p>
    </>
  );
}

// PullRequestTile shows the open PRs updating a running version, the one matching it first
function PullRequestTile({ image }: { image: RunningImage }) {
  const prs = [...image.coveredBy].sort((a, b) => Number(b.matchesRunning) - Number(a.matchesRunning));
  const pr = prs[0];
  const soak = image.soak;
  return (
    <section className="tile">
      <h2>Open PR</h2>
      {!pr ? (
        image.status === 'pinned' ? (
          <PinnedByChart image={image} />
        ) : soak ? (
          <>
            <div className="big">
              <SoakState soak={soak} />
            </div>
            <SoakProgress soak={soak} />
          </>
        ) : (
          <>
            <div className="big muted">None</div>
            <p className="sub">No open PR updates this image.</p>
          </>
        )
      ) : (
        <>
          <div className="big">{pr.to ? imageTag(pr.to) : 'removed'}</div>
          {prs.map((c) => (
            <p key={`${c.repo}#${c.number}${c.via ?? ''}${c.from}`} className="sub">
              <a href={`#/pr/${c.repo}/${c.number}`}>
                #{c.number} {c.title}
              </a>
              {c.via && <> via chart {c.via}</>}
              {!c.matchesRunning && <span className="worse"> Updates {imageTag(c.from)}, not the version running.</span>}
            </p>
          ))}
        </>
      )}
    </section>
  );
}

// ImageTag is one running version of an image on its page
function ImageTag({ image, heading }: { image: RunningImage; heading: boolean }) {
  const vulns = useMemo(() => [...(image.vulnerabilities ?? [])].sort(bySeverity), [image.vulnerabilities]);
  const version = image.version;
  const newest = version?.latest;
  const hasUpdate = !!newest && newest !== version?.current;
  const b = hasUpdate ? bump(version!.current, newest) : 'other';
  const tag = image.tag ?? image.digests[0]?.slice(0, 19) ?? 'unknown';
  const prTo = image.coveredBy.find((c) => c.matchesRunning)?.to;
  const clusters = new Set(image.workloads.map((w) => w.cluster)).size;
  const groups = deploymentGroups(image);

  return (
    <section className="image-version">
      {heading && <h2>Version {tag}</h2>}
      <div className="tiles">
        <section className="tile">
          <h2>Running now</h2>
          <div className="big">{tag}</div>
          <p className="sub">
            {image.workloads.length} {image.workloads.length === 1 ? 'workload' : 'workloads'} in {clusters}{' '}
            {clusters === 1 ? 'cluster' : 'clusters'}
          </p>
          {image.digests.map((d) => (
            <p key={d} className="sub">
              <a href={`#/history?q=${d}`} title={`${d}: everywhere this digest has run`}>
                <code>{d.slice(0, 19)}…</code>
              </a>
              <CopyButton text={d} />
            </p>
          ))}
        </section>

        <PullRequestTile image={image} />

        <section className="tile">
          <h2>Latest available</h2>
          {!newest ? (
            <>
              <div className="big muted">Unknown</div>
              <p className="sub">{version?.error ?? 'No newer version could be looked up for this tag.'}</p>
            </>
          ) : (
            <>
              <div className="big">{newest}</div>
              <p className="sub">
                {hasUpdate ? 'The newest tag of the same variant.' : 'Already running the newest tag of the same variant.'}
                {version?.checkedAt && ` Checked ${timeAgo(version.checkedAt)}.`}
              </p>
            </>
          )}
        </section>
      </div>

      <div className="image-panels">
        <section className="tile">
          <h2>What this means</h2>
          <dl className="explain">
            <dt>
              <StatusBadge status={image.status} version={version} />
            </dt>
            <dd>{statusHelp(image)}</dd>
            {b !== 'other' && (
              <>
                <dt>
                  <span className={`badge bump ${b}`}>{b}</span>
                </dt>
                <dd>
                  {bumpHelp[b]}{' '}
                  <span className="muted">
                    That's for {version!.current} → {newest}, assuming this image follows semantic versioning.
                  </span>
                </dd>
              </>
            )}
            {groups.length > 1 && (
              <>
                <dt>
                  <span className="badge">deployed {groups.length} ways</span>
                </dt>
                <dd>{deployedWaysHelp} See where it runs below.</dd>
              </>
            )}
            {image.soak && image.status !== 'pinned' && (
              <>
                <dt>
                  <SoakBadge soak={image.soak} />
                </dt>
                <dd>{soakHelp(image.soak)}</dd>
              </>
            )}
          </dl>
        </section>

        <section className="tile">
          <h2>Vulnerabilities</h2>
          {image.scan?.status === 'done' ? (
            <>
              <CveComparison image={image} />
              <p className="sub">
                Trivy scanned this digest{image.scan.scannedAt && ` ${timeAgo(image.scan.scannedAt)}`}, and re-checks it daily.
                {hasUpdate && ` "After update" is the latest version, ${newest}.`}
                {hasUpdate && prTo && imageTag(prTo) !== newest && ` The open PR updates to ${imageTag(prTo)}, so its numbers may differ.`}
              </p>
              {image.digestScans && <DigestScans image={image} scans={image.digestScans} />}
            </>
          ) : image.scan ? (
            <RunningScan image={image} />
          ) : (
            <p className="message">Not scanned: scanning is off, or no digest is known for this image.</p>
          )}
        </section>
      </div>

      <section className="tile">
        <h2>Where it runs</h2>
        {groups.map((g) => (
          <div key={g.key} className="deployment">
            <h3>
              {g.label} <span className="muted">({g.workloads.length})</span>
            </h3>
            <p className="sub">{g.help}</p>
            <WorkloadList workloads={g.workloads} />
            {g.prs.map((c) => (
              <p key={`${c.repo}#${c.number}`} className="sub">
                Updated by{' '}
                <a href={`#/pr/${c.repo}/${c.number}`}>
                  #{c.number} {c.title}
                </a>
              </p>
            ))}
          </div>
        ))}
      </section>

      {(vulns.length > 0 || (image.update && hasUpdate)) && (
        <section className="tile">
          <h2>CVE details</h2>
          {vulns.length > 0 && (
            <details className="vulns">
              <summary>
                {vulns.length} {vulns.length === 1 ? 'CVE' : 'CVEs'} in this version
              </summary>
              <VulnerabilityTable rows={vulns.map((vuln) => ({ vuln }))} />
            </details>
          )}
          {image.update && hasUpdate && <Vulnerabilities label={`Updating to ${newest}`} scan={image.update} />}
        </section>
      )}
    </section>
  );
}

// ImagePage shows one image repository: the open PRs changing it and each version running
function ImagePage({ repository }: { repository: string }) {
  const { data, error } = usePoll(() => getImage(repository), [repository]);
  // PRs on a running version are shown with it
  const prKey = (c: Coverage) => `${c.repo}#${c.number}/${c.from}/${c.to}/${c.via ?? ''}`;
  const shown = new Set(data?.tags.flatMap((t) => t.coveredBy.map(prKey)));
  const otherPRs = data?.pullRequests.filter((c) => !shown.has(prKey(c))) ?? [];

  return (
    <div className="detail">
      <a className="back" href="#/images">
        ← All running images
      </a>
      {error && <p className="message error">{error}</p>}
      {!data && !error && <p className="message">Loading…</p>}
      {data && (
        <>
          <h1>{shortRepository(data.repository)}</h1>
          {shortRepository(data.repository) !== data.repository && <p className="sub">{data.repository}</p>}
          {data.tags.length === 0 && (
            <p className="message">
              Not running in any inventoried cluster{data.pullRequests.length === 0 && ', and no open PR changes it'}.
            </p>
          )}

          {data.tags.map((image) => (
            <ImageTag key={image.tag ?? image.digests.join(',')} image={image} heading={data.tags.length > 1} />
          ))}

          {otherPRs.length > 0 && (
            <section className="tile">
              <h2>{data.tags.length > 0 ? 'Other open pull requests' : 'Open pull requests'}</h2>
              {data.tags.length > 0 && <p className="sub">These change this image, but not a version that's running.</p>}
              {otherPRs.map((c) => (
                <div key={`${c.repo}#${c.number}${c.via ?? ''}${c.from}${c.to}`} className="change">
                  <a href={`#/pr/${c.repo}/${c.number}`}>
                    <span className="number">#{c.number}</span> {c.title}
                  </a>
                  <span className="versions">
                    {c.from ? imageTag(c.from) : 'added'} → {c.to ? imageTag(c.to) : 'removed'}
                  </span>
                  {c.via && <span className="muted">via chart {c.via}</span>}
                </div>
              ))}
            </section>
          )}

          <ImageHistory repository={repository} />
        </>
      )}
    </div>
  );
}

function Stages({ stages }: { stages: Stage[] }) {
  if (stages.length === 0) {
    return <span className="muted">pending</span>;
  }
  return (
    <div className="stages">
      {stages.map((s) => (
        <span key={s.name} className={`badge stage ${s.status}`} title={s.error || `${s.status} ${timeAgo(s.updatedAt)}`}>
          {s.name}
        </span>
      ))}
    </div>
  );
}

function PullRequestDetail({ repo, number }: { repo: string; number: number }) {
  const { data: pr, error } = usePoll(() => getPullRequest(repo, number), [repo, number]);

  return (
    <div className="detail">
      <a className="back" href="#/prs">
        ← All pull requests
      </a>
      {error && <p className="message error">{error}</p>}
      {pr && (
        <>
          <h1>
            <span className="number">#{pr.number}</span> {pr.title}
          </h1>
          <dl className="meta">
            <dt>Repo</dt>
            <dd>{pr.repo}</dd>
            <dt>Branch</dt>
            <dd>
              <code>{pr.branch}</code>
            </dd>
            <dt>Head</dt>
            <dd>
              <code>{pr.headSHA.slice(0, 12)}</code>
            </dd>
            <dt>State</dt>
            <dd>{pr.state}</dd>
            <dt>Updated</dt>
            <dd title={pr.updatedAt}>{timeAgo(pr.updatedAt)}</dd>
            <dt>GitHub</dt>
            <dd>
              <a href={pr.url} target="_blank" rel="noreferrer">
                Open PR ↗
              </a>
            </dd>
          </dl>

          <h2>Changes</h2>
          {pr.changes.length === 0 ? (
            <p className="message">No image or chart changes found in the diff.</p>
          ) : (
            <div className="table-wrap">
              <table>
                <thead>
                  <tr>
                    <th>Kind</th>
                    <th>Name</th>
                    <th>From</th>
                    <th>To</th>
                    <th>File</th>
                    <th>CVEs</th>
                    <th>From version running in</th>
                  </tr>
                </thead>
                <tbody>
                  {pr.changes.map((c, i) => (
                    <tr key={i}>
                      <td>
                        <span className={`badge kind ${c.kind}`}>{kindLabel(c.kind)}</span>
                      </td>
                      <td>
                        {c.name}
                        {c.repoURL && <div className="sub">{c.repoURL}</div>}
                      </td>
                      <td>
                        <code>{c.from}</code>
                      </td>
                      <td>
                        <code>{c.to}</code>
                      </td>
                      <td>
                        <code>{c.file}</code>
                      </td>
                      <td>{c.scan ? <ScanBadges scan={c.scan} /> : <span className="muted">—</span>}</td>
                      <td>
                        {c.running === null ? (
                          <span className="muted">unknown</span>
                        ) : c.running.length === 0 ? (
                          <span className="muted">not running</span>
                        ) : (
                          <WorkloadList workloads={c.running} />
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}

          {pr.changes.some((c) => c.scan?.fixedVulnerabilities || c.scan?.introducedVulnerabilities) && (
            <>
              <h2>Vulnerabilities</h2>
              {pr.changes.map(
                (c, i) => c.scan && <Vulnerabilities key={i} label={`${c.name} ${version(c, 'from')} → ${version(c, 'to')}`} scan={c.scan} />,
              )}
            </>
          )}

          {pr.changes.some((c) => c.renders && c.renders.length > 0) && (
            <>
              <h2>Chart image changes</h2>
              {pr.changes.map((c, i) =>
                (c.renders ?? []).map((r) => <ChartRender key={`${i}/${r.cluster}/${r.app}`} change={c} render={r} />),
              )}
            </>
          )}

          <h2>Pipeline</h2>
          {pr.stages.length === 0 ? (
            <p className="message">No stages have run yet.</p>
          ) : (
            <div className="table-wrap">
              <table>
                <thead>
                  <tr>
                    <th>Stage</th>
                    <th>Status</th>
                    <th>Head</th>
                    <th>Updated</th>
                    <th>Error</th>
                  </tr>
                </thead>
                <tbody>
                  {pr.stages.map((s) => (
                    <tr key={s.name}>
                      <td>{s.name}</td>
                      <td>
                        <span className={`badge stage ${s.status}`}>{s.status}</span>
                      </td>
                      <td>
                        <code>{s.headSHA.slice(0, 12)}</code>
                      </td>
                      <td title={s.updatedAt}>{timeAgo(s.updatedAt)}</td>
                      <td className="error-text">{s.error}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </>
      )}
    </div>
  );
}

function ChartRender({ change, render }: { change: ChangeView; render: ChartRenderView }) {
  return (
    <div className="render">
      <h3>
        {change.name} {change.from} → {change.to}{' '}
        <span className="muted">
          in {render.cluster}/{render.app}
        </span>
      </h3>
      {render.error ? (
        <p className="message error">{render.error}</p>
      ) : render.images.length === 0 ? (
        <p className="message">No container images change.</p>
      ) : (
        <div className="table-wrap">
          <table>
            <thead>
              <tr>
                <th>Workload</th>
                <th>Image</th>
                <th>From</th>
                <th>To</th>
                <th>CVEs</th>
                <th>Running</th>
              </tr>
            </thead>
            <tbody>
              {render.images.map((img) => (
                <tr key={`${img.kind}/${img.namespace}/${img.name}/${img.container}/${img.init}`}>
                  <td>
                    <span className="muted">{img.namespace}/</span>
                    {img.kind}/{img.name}
                    {img.container && (
                      <div className="sub">
                        {img.init ? 'init container' : 'container'} {img.container}
                      </div>
                    )}
                  </td>
                  <td>
                    <a href={imageHref(img.from || img.to)}>{shortRepository(imageName(img.from || img.to))}</a>
                  </td>
                  <td>{img.from ? <code title={img.from}>{imageTag(img.from)}</code> : <span className="muted">added</span>}</td>
                  <td>{img.to ? <code title={img.to}>{imageTag(img.to)}</code> : <span className="muted">removed</span>}</td>
                  <td>{img.scan ? <ScanBadges scan={img.scan} /> : <span className="muted">—</span>}</td>
                  <td>
                    <RunningBadge running={img.from ? img.running : null} />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
      {render.images.map(
        (img) =>
          img.scan && (
            <Vulnerabilities
              key={`v/${img.kind}/${img.namespace}/${img.name}/${img.container}/${img.init}`}
              label={`${img.kind}/${img.name}${img.container ? ' ' + img.container : ''}`}
              scan={img.scan}
            />
          ),
      )}
    </div>
  );
}

// appNeedsAttention: containers no source declares, a failed render, or a chart that's behind
const appNeedsAttention = (app: ArgoApp) =>
  app.unattributed > 0 || app.sources.some((s) => s.render?.error || s.status === 'behind');

function Apps({ hasClusters, initialProblemsOnly }: { hasClusters: boolean; initialProblemsOnly: boolean }) {
  const [query, setQuery] = useState('');
  const [problemsOnly, setProblemsOnly] = useState(initialProblemsOnly);
  const { data, error } = usePoll(listApps, []);

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    return (data ?? []).filter((app) => {
      if (problemsOnly && !appNeedsAttention(app)) return false;
      if (!q) return true;
      return (
        app.name.toLowerCase().includes(q) ||
        app.destNamespace.toLowerCase().includes(q) ||
        app.sources.some((s) => (s.chart ?? '').toLowerCase().includes(q) || s.repoURL.toLowerCase().includes(q))
      );
    });
  }, [data, query, problemsOnly]);

  if (!hasClusters) {
    return (
      <p className="message">
        No Kubernetes clusters configured. Add <code>kubernetes.clusters</code> to the daemon config.
      </p>
    );
  }

  const problems = (data ?? []).filter(appNeedsAttention).length;

  return (
    <>
      <div className="toolbar">
        <input
          type="search"
          placeholder="Filter by app, namespace or chart"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
        />
        <label className="checkbox">
          <input type="checkbox" checked={problemsOnly} onChange={(e) => setProblemsOnly(e.target.checked)} />
          Needs attention only <span className="count">{problems}</span>
        </label>
        <span className="muted">{data?.length ?? 0} Argo CD apps</span>
      </div>

      {error && <p className="message error">{error}</p>}
      {data && filtered.length === 0 && <p className="message">No apps match.</p>}
      {!data && !error && <p className="message">Loading…</p>}

      {filtered.length > 0 && (
        <div className="table-wrap">
          <table>
            <thead>
              <tr>
                <th>App</th>
                <th>Sources</th>
                <th>Status</th>
                <th>Running containers</th>
              </tr>
            </thead>
            <tbody>
              {filtered.map((app) => (
                <tr key={`${app.cluster}/${app.name}`}>
                  <td>
                    <div className="name">{app.name}</div>
                    <div className="sub">
                      {app.cluster} → {app.destNamespace}
                    </div>
                  </td>
                  <td>
                    {app.sources
                      .filter((s) => s.kind !== 'ref')
                      .map((s) => (
                        <AppSource key={s.index} source={s} />
                      ))}
                  </td>
                  <td>
                    <div className="stages">
                      <span className={`badge ${app.syncStatus === 'Synced' ? 'running' : 'bump major'}`}>
                        {app.syncStatus || 'unknown'}
                      </span>
                      <span className={`badge ${app.health === 'Healthy' ? 'running' : 'bump major'}`}>
                        {app.health || 'unknown'}
                      </span>
                    </div>
                  </td>
                  <td>
                    {app.containers}
                    {app.unattributed > 0 && (
                      <span
                        className="badge bump major"
                        title="Running containers labelled as this app that no rendered source declares"
                      >
                        {app.unattributed} unmatched
                      </span>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </>
  );
}

function AppSource({ source }: { source: SourceView }) {
  const r = source.render;
  return (
    <div className="change">
      <span className={`badge kind ${source.kind === 'chart' ? 'helmChart' : ''}`}>{source.kind}</span>
      <span className="name" title={`${source.repoURL}${source.path ? ' ' + source.path : ''}`}>
        {source.chart ?? source.path ?? source.repoURL}
      </span>
      {source.revision && <span className="versions">{shortRevision(source.revision)}</span>}
      {source.status && <StatusBadge status={source.status} version={source.version} />}
      <LatestVersion version={source.version} />
      {source.updatePRs?.map((pr) => (
        <a key={`${pr.repo}#${pr.number}`} href={`#/pr/${pr.repo}/${pr.number}`} title={pr.title}>
          #{pr.number} → {pr.to}
        </a>
      ))}
      {source.kind === 'other' ? (
        <span className="muted">not rendered</span>
      ) : !r ? (
        <span className="muted">not rendered yet</span>
      ) : r.error ? (
        <span className="badge error" title={r.error}>
          render failed
        </span>
      ) : (
        <span
          className="muted"
          title={r.images.map((i) => `${i.kind}/${i.name}${i.container ? '/' + i.container : ''}: ${i.image}`).join('\n')}
        >
          {r.images.length} {r.images.length === 1 ? 'image' : 'images'}
        </span>
      )}
    </div>
  );
}
