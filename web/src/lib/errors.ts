import { toast } from 'svelte-sonner';

export type ErrorCategory = 'auth' | 'server' | 'network' | 'validation' | 'unknown';

export class ApiError extends Error {
    readonly status: number;
    readonly category: ErrorCategory;
    readonly body: string;

    constructor(status: number, body: string, category?: ErrorCategory) {
        super(`API error ${status}: ${body}`);
        this.name = 'ApiError';
        this.status = status;
        this.body = body;
        this.category = category ?? categorize(status);
    }

    get retryable(): boolean {
        return this.category === 'network' || this.category === 'server';
    }
}

function categorize(status: number): ErrorCategory {
    if (status === 401 || status === 403) return 'auth';
    if (status === 422 || status === 400) return 'validation';
    if (status >= 500) return 'server';
    return 'unknown';
}

const COPY: Record<ErrorCategory, string> = {
    network: 'Connection lost -- retrying automatically',
    server: 'Server error -- please try again',
    auth: 'Session expired -- reload the page',
    validation: '',  // uses server-provided body
    unknown: 'Something went wrong -- try refreshing the page',
};

export function showToast(err: ApiError): void {
    const message = err.category === 'validation' && err.body
        ? err.body
        : COPY[err.category] || COPY.unknown;

    if (err.category === 'auth') {
        toast.error(message, { duration: Infinity });
    } else {
        toast.error(message, { duration: 5000 });
    }
}
