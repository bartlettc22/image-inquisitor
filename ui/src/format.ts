import type { Change } from './types';

export type Bump = 'major' | 'minor' | 'patch' | 'other';

// version returns the tag (images, without digest) or chart version
export function version(change: Change, side: 'from' | 'to'): string {
  const ref = change[side];
  if (change.kind !== 'image') {
    return ref;
  }
  const withoutDigest = ref.split('@')[0];
  return withoutDigest.startsWith(change.name + ':')
    ? withoutDigest.slice(change.name.length + 1)
    : withoutDigest;
}

const versionRe = /^v?(\d+)(?:\.(\d+))?(?:\.(\d+))?/;

// bump compares the leading numeric parts of two versions
export function bump(from: string, to: string): Bump {
  const a = versionRe.exec(from);
  const b = versionRe.exec(to);
  if (!a || !b) {
    return 'other';
  }
  if (a[1] !== b[1]) return 'major';
  if ((a[2] ?? '0') !== (b[2] ?? '0')) return 'minor';
  if ((a[3] ?? '0') !== (b[3] ?? '0')) return 'patch';
  return 'other';
}

export function timeAgo(iso: string, now: number = Date.now()): string {
  const seconds = Math.round((now - Date.parse(iso)) / 1000);
  if (seconds < 60) return `${Math.max(seconds, 0)}s ago`;
  const minutes = Math.round(seconds / 60);
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.round(minutes / 60);
  if (hours < 48) return `${hours}h ago`;
  return `${Math.round(hours / 24)}d ago`;
}

// shortRepository drops the Docker Hub registry and library/ prefix added by normalization
export function shortRepository(repository: string): string {
  return repository.replace(/^index\.docker\.io\/(library\/)?/, '');
}

export function workloadLabel(w: { cluster: string; namespace: string; kind: string; name: string }): string {
  return `${w.cluster}/${w.namespace}/${w.kind}/${w.name}`;
}

// shortRevision abbreviates commit SHAs; chart versions are returned as is
export function shortRevision(revision: string): string {
  return /^[0-9a-f]{40}$/.test(revision) ? revision.slice(0, 7) : revision;
}

// imageTag returns the tag of an image reference, or a short digest when it has none
export function imageTag(ref: string): string {
  const [base, digest] = ref.split('@');
  const lastSlash = base.lastIndexOf('/');
  const colon = base.indexOf(':', lastSlash + 1);
  if (colon >= 0) return base.slice(colon + 1);
  return digest ? digest.slice(0, 19) : 'latest';
}

// imageName returns an image reference without its tag or digest
export function imageName(ref: string): string {
  const base = ref.split('@')[0];
  const lastSlash = base.lastIndexOf('/');
  const colon = base.indexOf(':', lastSlash + 1);
  return colon >= 0 ? base.slice(0, colon) : base;
}

// imageHref links to an image's page, from any reference or repository
export function imageHref(ref: string): string {
  return `#/image/${shortRepository(imageName(ref))}`;
}

export const kindLabel = (kind: Change['kind']) => (kind === 'image' ? 'image' : 'chart');

// timeUntil is the future counterpart of timeAgo, e.g. "in 6m"
export function timeUntil(iso: string, now: number = Date.now()): string {
  return 'in ' + duration(Date.parse(iso) - now);
}

// duration formats milliseconds briefly, e.g. "850ms", "12s", "4m 10s", "3h", "2d"
export function duration(ms: number): string {
  if (ms < 1000) return `${Math.max(Math.round(ms), 0)}ms`;
  const seconds = Math.round(ms / 1000);
  if (seconds < 60) return `${seconds}s`;
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return seconds % 60 ? `${minutes}m ${seconds % 60}s` : `${minutes}m`;
  const hours = Math.round(minutes / 60);
  if (hours < 48) return `${hours}h`;
  return `${Math.round(hours / 24)}d`;
}

export function bytes(n: number): string {
  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  let i = 0;
  while (n >= 1024 && i < units.length - 1) {
    n /= 1024;
    i++;
  }
  return `${n < 10 && i > 0 ? n.toFixed(1) : Math.round(n)} ${units[i]}`;
}
