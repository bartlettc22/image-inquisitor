import { useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react';
import { getTrends } from './api';
import { timeAgo } from './format';
import type { Severity, TimeToFix, TrendDay, Trends as TrendsData, TrendWeek } from './types';
import './trends.css';

const REFRESH_MS = 60_000;

const windows = [
  [30, '30 days'],
  [90, '90 days'],
  [365, '1 year'],
] as const;

// Severities with an SLA; unknown findings count in totals only
const slaSeverities = ['CRITICAL', 'HIGH', 'MEDIUM', 'LOW'] as const;

const severityName = (s: Severity) => s.charAt(0) + s.slice(1).toLowerCase();

const shortDate = (date: string) =>
  new Date(date).toLocaleDateString(undefined, { month: 'short', day: 'numeric', timeZone: 'UTC' });

const days = (n: number) => (n < 10 ? `${Math.round(n * 10) / 10}d` : `${Math.round(n)}d`);

const sum = (counts: Partial<Record<Severity, number>>) => Object.values(counts).reduce((a, b) => a + (b ?? 0), 0);

// niceMax rounds a maximum up to a 1, 2 or 5 step, so the axis reads in clean numbers
function niceScale(max: number, ticks = 4): { max: number; step: number } {
  if (max <= 0) return { max: 1, step: 1 };
  const raw = max / ticks;
  const mag = 10 ** Math.floor(Math.log10(raw));
  const step = [1, 2, 5, 10].map((m) => m * mag).find((s) => s >= raw) ?? 10 * mag;
  return { max: Math.max(step, Math.ceil(max / step) * step), step: Math.max(1, step) };
}

// useWidth tracks an element's width, so charts draw at their real size and text stays undistorted
function useWidth<T extends HTMLElement>() {
  const ref = useRef<T>(null);
  const [width, setWidth] = useState(0);
  useLayoutEffect(() => {
    if (!ref.current) return;
    const observer = new ResizeObserver(([entry]) => setWidth(Math.floor(entry.contentRect.width)));
    observer.observe(ref.current);
    return () => observer.disconnect();
  }, []);
  return [ref, width] as const;
}

export function Trends() {
  const [windowDays, setWindowDays] = useState<number>(90);
  const [data, setData] = useState<TrendsData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    const load = () => {
      setLoading(true);
      getTrends(windowDays)
        .then((d) => !cancelled && (setData(d), setError(null)))
        .catch((e) => !cancelled && setError(String(e)))
        .finally(() => !cancelled && setLoading(false));
    };
    load();
    const timer = setInterval(load, REFRESH_MS);
    return () => {
      cancelled = true;
      clearInterval(timer);
    };
  }, [windowDays]);

  // Days and weeks before findings were first recorded would read as zero exposure, so they're left out
  const view = useMemo(() => {
    if (!data?.recordedSince) return null;
    const recorded = data.recordedSince.slice(0, 10);
    const dayList = data.days.filter((d) => d.date >= recorded);
    const weekList = data.weeks.filter((w) => addDays(w.start, 6) >= recorded);
    return { recorded, days: dayList, weeks: weekList };
  }, [data]);

  return (
    <section className="trends">
      <div className="trends-head">
        <div>
          <h2>Vulnerability trends</h2>
          <p className="sub">
            CVEs running in workload containers, counted once per container
            {view && <> · recorded since {shortDate(view.recorded)}</>}
          </p>
        </div>
        <span className="spacer" />
        <div className="segmented" role="group" aria-label="Time range">
          {windows.map(([n, label]) => (
            <button key={n} className={windowDays === n ? 'active' : ''} onClick={() => setWindowDays(n)}>
              {label}
            </button>
          ))}
        </div>
      </div>

      {error && <p className="message error">{error}</p>}
      {data && !view && (
        <p className="message">
          No history yet. Each sync records the CVEs in running images once they've been scanned, so trends appear after
          the first scans finish.
        </p>
      )}

      {data && view && view.days.length > 0 && (
        <div className={loading ? 'trends-body refreshing' : 'trends-body'}>
          <Summary data={data} days={view.days} />
          <div className="trend-multiples">
            {slaSeverities.map((s) => (
              <SeverityTrend key={s} severity={s} days={view.days} />
            ))}
          </div>
          <div className="trend-panels">
            <div className="chart-card">
              <h3>Opened vs fixed per week</h3>
              <p className="sub">New CVEs in workloads, and those an image update removed</p>
              <WeeklyChart weeks={view.weeks} />
            </div>
            <div className="chart-card">
              <h3>Days to fix</h3>
              <p className="sub">From a fix being available to it running, fixes in the last {windowLabel(windowDays)}</p>
              <TimeToFixChart rows={data.timeToFix} sla={data.slaDays} />
            </div>
          </div>
          <Overdue data={data} />
          <DataTable days={view.days} weeks={view.weeks} />
        </div>
      )}
    </section>
  );
}

const windowLabel = (n: number) => windows.find(([w]) => w === n)?.[1] ?? `${n} days`;

const addDays = (date: string, n: number) => {
  const d = new Date(date + 'T00:00:00Z');
  d.setUTCDate(d.getUTCDate() + n);
  return d.toISOString().slice(0, 10);
};

function Delta({ from, to, since }: { from: number; to: number; since: string }) {
  const d = to - from;
  if (d === 0) return <span className="delta">no change since {shortDate(since)}</span>;
  // Fewer open CVEs is good
  return (
    <span className={`delta ${d < 0 ? 'good' : 'bad'}`}>
      <span aria-hidden>{d < 0 ? '▼' : '▲'}</span> {Math.abs(d)} since {shortDate(since)}
    </span>
  );
}

function Summary({ data, days: dayList }: { data: TrendsData; days: TrendDay[] }) {
  const first = dayList[0];
  const last = dayList[dayList.length - 1];
  const fixes = data.timeToFix.filter((t) => t.severity in data.slaDays);
  const fixed = fixes.reduce((n, t) => n + t.fixed, 0);
  const within = fixes.reduce((n, t) => n + t.withinSLA, 0);

  return (
    <div className="tiles trend-tiles">
      <div className="tile">
        <h2>Open CVEs</h2>
        <div className="big">{sum(last.open)}</div>
        <div className="sub">
          <Delta from={sum(first.open)} to={sum(last.open)} since={first.date} />
        </div>
      </div>
      <div className="tile">
        <h2 title="Open CVEs with a fixed version available">Fixable now</h2>
        <div className="big">{last.fixable}</div>
        <div className="sub">
          {sum(last.open) ? Math.round((100 * last.fixable) / sum(last.open)) : 0}% of open CVEs have a fix
        </div>
      </div>
      <div className="tile">
        <h2 title="Fixable CVEs running longer than their severity's SLA allows">Past SLA</h2>
        <div className={`big ${data.overdue.total ? 'bad' : ''}`}>{data.overdue.total}</div>
        <div className="sub">
          {data.overdue.total === 0
            ? 'Every fixable CVE is within SLA'
            : slaSeverities
                .filter((s) => data.overdue.bySeverity[s])
                .map((s) => `${data.overdue.bySeverity[s]} ${s.toLowerCase()}`)
                .join(' · ')}
        </div>
      </div>
      <div className="tile">
        <h2>Fixed within SLA</h2>
        <div className="big">{fixed ? `${Math.round((100 * within) / fixed)}%` : '–'}</div>
        <div className="sub">{fixed ? `${within} of ${fixed} fixes` : 'No fixes in this range yet'}</div>
      </div>
    </div>
  );
}

const CHART_PAD_X = 4;

// SeverityTrend is one small multiple: the open CVEs of a severity per day, on its own scale
function SeverityTrend({ severity, days: dayList }: { severity: Severity; days: TrendDay[] }) {
  const [ref, width] = useWidth<HTMLDivElement>();
  const [hover, setHover] = useState<number | null>(null);
  const values = dayList.map((d) => d.open[severity] ?? 0);
  const current = values[values.length - 1];
  const height = 64;
  // The max label sits above the plot, so a line at the max never runs through it
  const top = 14;
  const { max } = niceScale(Math.max(...values), 5);

  const n = values.length;
  const plotW = Math.max(width - 2 * CHART_PAD_X, 1);
  const x = (i: number) => CHART_PAD_X + (n === 1 ? plotW / 2 : (i / (n - 1)) * plotW);
  const y = (v: number) => top + (1 - v / max) * (height - top);
  const line = values.map((v, i) => `${i ? 'L' : 'M'}${x(i)},${y(v)}`).join('');
  const area = `${line}L${x(n - 1)},${height}L${x(0)},${height}Z`;

  const pick = (clientX: number, el: Element) => {
    const left = el.getBoundingClientRect().left;
    const i = n === 1 ? 0 : Math.round(((clientX - left - CHART_PAD_X) / plotW) * (n - 1));
    setHover(Math.min(n - 1, Math.max(0, i)));
  };
  const shown = hover ?? n - 1;

  return (
    <div className="chart-card multiple">
      <div className="multiple-head">
        <span className={`severity-label ${severity.toLowerCase()}`}>{severityName(severity)}</span>
        <span className="multiple-value">{hover == null ? current : values[hover]}</span>
      </div>
      <div className="sub multiple-sub">
        {hover == null ? (
          <Delta from={values[0]} to={current} since={dayList[0].date} />
        ) : (
          <span>open on {shortDate(dayList[hover].date)}</span>
        )}
      </div>
      <div ref={ref} className="plot">
        {width > 0 && (
          <svg
            width={width}
            height={height + 1}
            role="img"
            aria-label={`Open ${severity.toLowerCase()} CVEs per day, ${values[0]} on ${shortDate(dayList[0].date)} to ${current} today`}
            tabIndex={0}
            onPointerMove={(e) => pick(e.clientX, e.currentTarget)}
            onPointerLeave={() => setHover(null)}
            onBlur={() => setHover(null)}
            onKeyDown={(e) => {
              if (e.key === 'ArrowLeft') setHover(Math.max(0, shown - 1));
              if (e.key === 'ArrowRight') setHover(Math.min(n - 1, shown + 1));
            }}
          >
            <line className="baseline" x1={0} x2={width} y1={height + 0.5} y2={height + 0.5} />
            <line className="grid" x1={0} x2={width} y1={top + 0.5} y2={top + 0.5} />
            <text className="tick" x={0} y={top - 4}>
              {max}
            </text>
            <path className="area opened" d={area} />
            <path className="line opened" d={line} />
            {hover != null && <line className="crosshair" x1={x(hover)} x2={x(hover)} y1={0} y2={height} />}
            <circle className="dot opened" cx={x(shown)} cy={y(values[shown])} r={4} />
          </svg>
        )}
      </div>
    </div>
  );
}

const BAR_MAX = 24;
const AXIS_W = 32;

function WeeklyChart({ weeks }: { weeks: TrendWeek[] }) {
  // The current week is still filling up, so it's drawn faded rather than read as a drop
  const today = new Date().toISOString().slice(0, 10);
  const partial = (w: TrendWeek) => addDays(w.start, 6) >= today;
  const [ref, width] = useWidth<HTMLDivElement>();
  const [hover, setHover] = useState<number | null>(null);
  const height = 180;
  const bottom = 20;
  const plotH = height - bottom;
  const { max, step } = niceScale(Math.max(1, ...weeks.flatMap((w) => [w.opened, w.fixed])));

  const plotW = Math.max(width - AXIS_W, 1);
  const group = plotW / Math.max(weeks.length, 1);
  const bar = Math.max(2, Math.min(BAR_MAX, (group * 0.7 - 2) / 2));
  const y = (v: number) => (1 - v / max) * plotH;
  const labelEvery = Math.max(1, Math.ceil(56 / group));
  const ticks = Array.from({ length: Math.round(max / step) + 1 }, (_, i) => i * step);

  // A column with a 4px rounded top and a square base
  const column = (x: number, v: number) => {
    const h = plotH - y(v);
    if (h <= 0) return '';
    const r = Math.min(4, h, bar / 2);
    return `M${x},${plotH}V${plotH - h + r}Q${x},${plotH - h} ${x + r},${plotH - h}H${x + bar - r}Q${x + bar},${plotH - h} ${x + bar},${plotH - h + r}V${plotH}Z`;
  };

  const tip = hover != null ? weeks[hover] : null;
  const tipLeft = hover != null ? AXIS_W + group * (hover + 0.5) : 0;

  return (
    <>
      <div className="legend">
        <span>
          <i className="swatch opened" /> Opened
        </span>
        <span>
          <i className="swatch fixed" /> Fixed
        </span>
      </div>
      <div ref={ref} className="plot">
        {width > 0 && (
          <svg width={width} height={height} role="img" aria-label="CVEs opened and fixed per week">
            {ticks.map((t) => (
              <g key={t}>
                <line className={t === 0 ? 'baseline' : 'grid'} x1={AXIS_W} x2={width} y1={y(t) + 0.5} y2={y(t) + 0.5} />
                <text className="tick" x={AXIS_W - 6} y={y(t)} textAnchor="end" dominantBaseline="middle">
                  {t}
                </text>
              </g>
            ))}
            {weeks.map((w, i) => {
              const left = AXIS_W + group * i + (group - (2 * bar + 2)) / 2;
              return (
                <g key={w.start} className={hover != null && hover !== i ? 'dimmed' : partial(w) ? 'partial' : ''}>
                  <path className="bar opened" d={column(left, w.opened)} />
                  <path className="bar fixed" d={column(left + bar + 2, w.fixed)} />
                  {i % labelEvery === 0 && (
                    <text className="tick" x={AXIS_W + group * (i + 0.5)} y={height - 4} textAnchor="middle">
                      {shortDate(w.start)}
                    </text>
                  )}
                  <rect
                    className="hit"
                    x={AXIS_W + group * i}
                    y={0}
                    width={group}
                    height={plotH}
                    tabIndex={0}
                    aria-label={`Week of ${shortDate(w.start)}${partial(w) ? ' so far' : ''}: ${w.opened} opened, ${w.fixed} fixed`}
                    onPointerEnter={() => setHover(i)}
                    onPointerLeave={() => setHover(null)}
                    onFocus={() => setHover(i)}
                    onBlur={() => setHover(null)}
                  />
                </g>
              );
            })}
          </svg>
        )}
        {tip && (
          <div className="tooltip" style={{ left: tipLeft, top: 0 }} role="presentation">
            <div className="tooltip-title">
              Week of {shortDate(tip.start)}
              {partial(tip) && ' (so far)'}
            </div>
            <div className="tooltip-row">
              <i className="key opened" />
              <strong>{tip.opened}</strong> opened
            </div>
            <div className="tooltip-row">
              <i className="key fixed" />
              <strong>{tip.fixed}</strong> fixed
            </div>
          </div>
        )}
      </div>
      <p className="sub footnote">Opened leaves out CVEs that were already running when recording began.</p>
    </>
  );
}

// TimeToFixChart puts each severity on its own SLA, so a 7-day and a 180-day target share one axis:
// the SLA sits at the middle, twice the SLA at the end
function TimeToFixChart({ rows, sla }: { rows: TimeToFix[]; sla: Partial<Record<Severity, number>> }) {
  const bySeverity = Object.fromEntries(rows.map((r) => [r.severity, r]));
  return (
    <div className="ttf">
      <div className="ttf-axis" aria-hidden>
        <span />
        <div className="ttf-scale">
          <span style={{ left: 0 }}>0</span>
          <span style={{ left: '50%' }}>SLA</span>
          <span style={{ left: '100%' }}>2× SLA</span>
        </div>
        <span />
      </div>
      {slaSeverities.map((s) => {
        const r = bySeverity[s] as TimeToFix | undefined;
        const target = sla[s] ?? 0;
        const share = r && target ? r.median / (2 * target) : 0;
        const over = r != null && r.median > target;
        return (
          <div className="ttf-row" key={s}>
            <span className={`severity-label ${s.toLowerCase()}`}>
              {severityName(s)}
              <span className="muted"> · {target}d</span>
            </span>
            <div
              className="ttf-track"
              role="img"
              aria-label={
                r
                  ? `${severityName(s)}: median ${days(r.median)}, p90 ${days(r.p90)}, SLA ${target} days`
                  : `${severityName(s)}: no fixes`
              }
            >
              <div className="ttf-sla" />
              {r && <div className="ttf-bar" style={{ width: `${Math.min(share, 1) * 100}%` }} />}
              {share > 1 && <span className="ttf-overflow">›</span>}
            </div>
            <span className="ttf-value">
              {r ? (
                <>
                  <strong>{days(r.median)}</strong> median
                  <span className="muted">
                    {' '}
                    · p90 {days(r.p90)} · {r.withinSLA}/{r.fixed} in SLA
                  </span>
                  {over && (
                    <span className="badge error over-sla">
                      <span aria-hidden>▲</span> over SLA
                    </span>
                  )}
                </>
              ) : (
                <span className="muted">no fixes</span>
              )}
            </span>
          </div>
        );
      })}
    </div>
  );
}

const OVERDUE_SHOWN = 8;

function Overdue({ data }: { data: TrendsData }) {
  const { findings, total } = data.overdue;
  const [expanded, setExpanded] = useState(false);
  if (total === 0) return null;
  const shown = expanded ? findings : findings.slice(0, OVERDUE_SHOWN);
  return (
    <div className="overdue">
      <h3>Past SLA</h3>
      <p className="sub">Fixable CVEs running longer than their SLA, most severe and most overdue first</p>
      <div className="table-wrap">
        <table>
          <thead>
            <tr>
              <th>Severity</th>
              <th>CVE</th>
              <th>Workload</th>
              <th>Fix available</th>
              <th className="num">Overdue</th>
            </tr>
          </thead>
          <tbody>
            {shown.map((f) => (
              <tr key={f.id}>
                <td>
                  <span className={`severity-label ${f.severity.toLowerCase()}`}>{f.severity.toLowerCase()}</span>
                </td>
                <td>
                  <div className="name">{f.vulnId}</div>
                  <div className="sub">{f.package}</div>
                </td>
                <td>
                  <div>
                    {f.namespace}/{f.workloadName}
                  </div>
                  <div className="sub">
                    {f.cluster} · {f.workloadKind} · {f.container}
                  </div>
                </td>
                <td className="nowrap muted">{timeAgo(f.fixAvailable)}</td>
                <td className="num nowrap">{days(f.daysOverdue)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      {shown.length < findings.length && (
        <button className="more-button" onClick={() => setExpanded(true)}>
          Show {findings.length - shown.length} more
        </button>
      )}
      {expanded && total > findings.length && <p className="sub">and {total - findings.length} more</p>}
    </div>
  );
}

// DataTable is the charts' numbers by week, for reading without hovering
function DataTable({ days: dayList, weeks }: { days: TrendDay[]; weeks: TrendWeek[] }) {
  const openAt = (weekStart: string) => {
    const end = addDays(weekStart, 6);
    return [...dayList].reverse().find((d) => d.date <= end);
  };
  return (
    <details className="trend-data">
      <summary>Data by week</summary>
      <div className="table-wrap">
        <table>
          <thead>
            <tr>
              <th>Week of</th>
              <th className="num">Opened</th>
              <th className="num">Fixed</th>
              {slaSeverities.map((s) => (
                <th className="num" key={s}>
                  Open {s.toLowerCase()}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {weeks.map((w) => {
              const d = openAt(w.start);
              return (
                <tr key={w.start}>
                  <td>{shortDate(w.start)}</td>
                  <td className="num">{w.opened}</td>
                  <td className="num">{w.fixed}</td>
                  {slaSeverities.map((s) => (
                    <td className="num" key={s}>
                      {d?.open[s] ?? 0}
                    </td>
                  ))}
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    </details>
  );
}
