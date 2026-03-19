import type { Session, Workspace, HistoryEntry, ClincusConfig, FileListResponse, FileContentResponse } from './types';
import { ApiError, showToast } from './errors';

const BASE = '';
const MAX_RETRIES = 3;
const INITIAL_RETRY_DELAY = 1000; // 1s -> 2s -> 4s

interface RequestOptions {
    silent?: boolean;
}

/**
 * Retry wrapper for fetch. Retries on network errors and server 5xx
 * with exponential backoff (1s, 2s, 4s). Non-retryable errors (4xx,
 * auth) are thrown immediately. Toast is shown only on final failure
 * unless silent.
 */
async function fetchWithRetry(
    input: RequestInfo,
    init: RequestInit | undefined,
    opts?: RequestOptions,
): Promise<Response> {
    let lastError: ApiError | undefined;
    let delay = INITIAL_RETRY_DELAY;

    for (let attempt = 0; attempt <= MAX_RETRIES; attempt++) {
        let res: Response;
        try {
            res = await fetch(input, init);
        } catch (_e) {
            // Network error (offline, DNS, CORS, etc.)
            lastError = new ApiError(0, 'Network error', 'network');
            if (attempt < MAX_RETRIES) {
                await new Promise((r) => setTimeout(r, delay));
                delay *= 2;
                continue;
            }
            if (!opts?.silent) showToast(lastError);
            throw lastError;
        }

        if (res.ok) return res;

        // Parse error body
        const body = res.headers.get('content-type')?.includes('application/json')
            ? JSON.stringify(await res.json())
            : await res.text();
        const err = new ApiError(res.status, body);

        // Only retry retryable categories (network, server 5xx)
        if (!err.retryable || attempt >= MAX_RETRIES) {
            if (!opts?.silent) showToast(err);
            throw err;
        }

        lastError = err;
        await new Promise((r) => setTimeout(r, delay));
        delay *= 2;
    }

    // Unreachable, but TypeScript needs it
    throw lastError!;
}

async function get<T>(path: string, opts?: RequestOptions): Promise<T> {
    const res = await fetchWithRetry(`${BASE}${path}`, undefined, opts);
    return res.json();
}

async function post<T>(path: string, body?: unknown, opts?: RequestOptions): Promise<T> {
    const res = await fetchWithRetry(
        `${BASE}${path}`,
        {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: body ? JSON.stringify(body) : undefined,
        },
        opts,
    );
    return res.json();
}

async function del(path: string, opts?: RequestOptions): Promise<void> {
    await fetchWithRetry(`${BASE}${path}`, { method: 'DELETE' }, opts);
}

async function put<T>(path: string, body?: unknown, opts?: RequestOptions): Promise<T> {
    const res = await fetchWithRetry(
        `${BASE}${path}`,
        {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: body ? JSON.stringify(body) : undefined,
        },
        opts,
    );
    return res.json();
}

export const api = {
    listSessions: (opts?: RequestOptions) => get<Session[]>('/api/sessions', opts),
    createSession: (workspace: string, tool: string, opts?: RequestOptions) =>
        post<Session>('/api/sessions', { workspace, tool }, opts),
    stopSession: (id: string, force = false, opts?: RequestOptions) =>
        del(`/api/sessions/${id}${force ? '?force=true' : ''}`, opts),
    resumeSession: (id: string, opts?: RequestOptions) =>
        post<{ id: string }>(`/api/sessions/${id}/resume`, undefined, opts),
    sessionHistory: (opts?: RequestOptions) => get<HistoryEntry[]>('/api/sessions/history', opts),
    listWorkspaces: (opts?: RequestOptions) =>
        get<{ roots: string[]; workspaces: Workspace[] }>('/api/workspaces', opts),
    addWorkspace: (path: string, opts?: RequestOptions) =>
        post('/api/workspaces', { path }, opts),
    removeWorkspace: (path: string, opts?: RequestOptions) =>
        del(`/api/workspaces?path=${encodeURIComponent(path)}`, opts),
    getTools: (opts?: RequestOptions) => get<string[]>('/api/tools', opts),
    getConfig: (opts?: RequestOptions) => get<ClincusConfig>('/api/config', opts),
    listFiles: (sessionId: string, path = '/', opts?: RequestOptions) =>
        get<FileListResponse>(`/api/sessions/${sessionId}/files?path=${encodeURIComponent(path)}`, opts),
    readFile: (sessionId: string, path: string, opts?: RequestOptions) =>
        get<FileContentResponse>(`/api/sessions/${sessionId}/files/content?path=${encodeURIComponent(path)}`, opts),
    writeFile: (sessionId: string, path: string, content: string, opts?: RequestOptions) =>
        put<{ status: string; path: string }>(`/api/sessions/${sessionId}/files/content?path=${encodeURIComponent(path)}`, { content }, opts),
};
