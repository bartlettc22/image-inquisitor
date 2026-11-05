import { useEffect, useState } from 'react';
import { getHistory } from './api';
import { duration, imageHref, imageName, imageTag, shortRepository, timeAgo } from './format';
import type { ClusterCoverage, HistoryEntry, HistoryResponse } from './types';

const POLL_MS = 15_000;

const when = (iso: string) => new Date(iso).toLocaleString();

// useHistory loads matching entries now, on every interval and when the query changes
function useHistory(params: { repository?: string; q?: string; at?: string }) {
  const [data, setData] = useState<HistoryResponse | null>(null);
  const [error, setError] = useState<string | null>(null);
  const { repository, q, at } = params;

  useEffect(() => {
    let cancelled = false;
    const run = () =>
      getHistory({ repository, q, at }).then(
        (d) => !cancelled && (setData(d), setError(null)),
        (e: unknown) => !cancelled && setError(e instanceof Error ? e.message : String(e)),
      );
    // Waits for a pause in typing
    const first = setTimeout(run, q ? 250 : 0);
    const id = setInterval(run, POLL_MS);
    return () => {
      cancelled = true;
      clearTimeout(first);
      clearInterval(id);
    };
  }, [repository, q, at]);

  return { data, error };
}

const endLabels: Record<string, string> = {
  stopped: 'stopped',
  deleted: 'pod deleted',
  replaced: 'replaced',
};

function Ended({ entry }: { entry: HistoryEntry }) {
  if (entry.endReason === 'never_started') {
    return <span className="badge error">never started</span>;
  }
  if (!entry.endReason) {
    return entry.ran ? <span className="badge kind image">running</span> : <span className="badge">pending</span>;
  }
  if (entry.endReason === 'unobserved') {
    return (
      <span title="It ended while the daemon wasn't watching. This is when that was noticed, not when it ended.">
        by {when(entry.endedAt)}
        <div className="sub">not observed</div>
      </span>
    );
  }
  const ran = Date.parse(entry.endedAt) - Date.parse(entry.startedAt);
  return (
    <>
      {when(entry.endedAt)}
      <div className="sub">
        {endLabels[entry.endReason] ?? entry.endReason} after {duration(ran)}
      </div>
    </>
  );
}

function HistoryTable({ entries, showImage }: { entries: HistoryEntry[]; showImage: boolean }) {
  return (
    <div className="table-wrap">
      <table>
        <thead>
          <tr>
            {showImage && <th>Image</th>}
            <th>Tag and digest</th>
            <th>Where</th>
            <th>Node</th>
            <th>Started</th>
            <th>Ended</th>
          </tr>
        </thead>
        <tbody>
          {entries.map((e) => (
            <tr key={e.id}>
              {showImage && (
                <td>
                  <a href={imageHref(e.image)}>{shortRepository(imageName(e.image))}</a>
                </td>
              )}
              <td>
                <code>{imageTag(e.image)}</code>
                <div className="sub">
                  {e.digest ? (
                    <a href={`#/history?q=${e.digest}`} title={e.digest}>
                      <code>{e.digest.slice(0, 19)}</code>
                    </a>
                  ) : (
                    'digest unknown'
                  )}
                </div>
              </td>
              <td>
                <span className="muted">
                  {e.cluster}/{e.namespace}/
                </span>
                {e.workloadKind}/{e.workloadName}
                <div className="sub">
                  {e.containerType && <span className={`badge ${e.containerType === 'ephemeral' ? 'error' : ''}`}>{e.containerType}</span>}{' '}
                  {e.container} in pod {e.pod}
                </div>
              </td>
              <td>{e.node ?? <span className="muted">—</span>}</td>
              <td title={`First seen ${when(e.recordedAt)}`}>
                {when(e.startedAt)}
                <div className="sub">{timeAgo(e.startedAt)}</div>
              </td>
              <td>
                <Ended entry={e} />
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function Coverage({ coverage }: { coverage: ClusterCoverage[] }) {
  return (
    <ul className="coverage">
      {coverage.map((c) => (
        <li key={c.cluster}>
          <span className="name">⎈ {c.cluster}</span>{' '}
          {c.recording ? (
            <span className="muted">recording since {c.since && when(c.since)}</span>
          ) : c.lastSeen ? (
            <span className="error-text">not recording since {when(c.lastSeen)}: containers since then may be missing</span>
          ) : (
            <span className="error-text">never recorded: check the daemon can watch pods, replicasets and jobs</span>
          )}
          {c.gaps.length > 0 && (
            <details>
              <summary>
                {c.gaps.length} {c.gaps.length === 1 ? 'gap' : 'gaps'} in recording
              </summary>
              <p className="sub">Containers may have run during these times without being recorded.</p>
              <ul>
                {[...c.gaps].reverse().map((g) => (
                  <li key={g.from}>
                    {when(g.from)} → {when(g.to)} <span className="muted">({duration(Date.parse(g.to) - Date.parse(g.from))})</span>
                  </li>
                ))}
              </ul>
            </details>
          )}
        </li>
      ))}
    </ul>
  );
}

// History searches every container recorded in the clusters, by text and point in time
export function History({ initialQuery }: { initialQuery: string | null }) {
  const [query, setQuery] = useState(initialQuery ?? '');
  const [at, setAt] = useState('');
  // A datetime-local value is local time, with no zone
  const atISO = at ? new Date(at).toISOString() : undefined;
  const { data, error } = useHistory({ q: query.trim() || undefined, at: atISO });

  return (
    <>
      <div className="toolbar">
        <input
          type="search"
          placeholder="Filter by image, digest, namespace, workload, pod or node"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
        />
        <label className="muted">
          Present at <input type="datetime-local" value={at} onChange={(e) => setAt(e.target.value)} />
        </label>
      </div>

      {error && <p className="message error">{error}</p>}
      {!data && !error && <p className="message">Loading…</p>}
      {data && <Coverage coverage={data.coverage} />}
      {data && data.entries.length === 0 && <p className="message">No containers match.</p>}
      {data && data.entries.length > 0 && <HistoryTable entries={data.entries} showImage />}
      {data?.truncated && <p className="message">Showing the newest {data.entries.length}. Narrow the search to see older ones.</p>}
    </>
  );
}

// ImageHistory is an image's history on its page
export function ImageHistory({ repository }: { repository: string }) {
  const { data, error } = useHistory({ repository });
  return (
    <>
      <h2>History</h2>
      {error && <p className="message error">{error}</p>}
      {data && data.entries.length === 0 && <p className="message">Never recorded running in a cluster.</p>}
      {data && data.entries.length > 0 && <HistoryTable entries={data.entries} showImage={false} />}
      {data?.truncated && (
        <p className="message">
          Showing the newest {data.entries.length}. <a href="#/history">Search the history</a> to see older ones.
        </p>
      )}
    </>
  );
}
