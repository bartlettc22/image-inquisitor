import type { ActivityReport, App, PullRequest, RunningImage, Status, Trends } from './types';

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const resp = await fetch(path, init);
  if (!resp.ok) {
    throw new Error(`${init?.method ?? 'GET'} ${path}: ${resp.status} ${resp.statusText}`);
  }
  return resp.status === 202 ? (undefined as T) : resp.json();
}

export const getStatus = () => request<Status>('/api/v1/status');

export const getActivity = () => request<ActivityReport>('/api/v1/activity');

export const listPullRequests = (state: 'open' | 'all') =>
  request<PullRequest[]>(`/api/v1/prs?state=${state}`);

export const getPullRequest = (repo: string, number: number) =>
  request<PullRequest>(`/api/v1/prs/${repo}/${number}`);

export const listRunningImages = () => request<RunningImage[]>('/api/v1/images');

export const listApps = () => request<App[]>('/api/v1/apps');

export const getTrends = (days: number) => request<Trends>(`/api/v1/trends?days=${days}`);

export const triggerSync = () => request<void>('/api/v1/sync', { method: 'POST' });

// runRenovate starts a Renovate run on every repo, or only on repo
export const runRenovate = (repo?: string) =>
  request<void>(`/api/v1/renovate${repo ? `?repo=${encodeURIComponent(repo)}` : ''}`, { method: 'POST' });
