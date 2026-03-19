import { describe, it, expect } from 'vitest';
import { ApiError } from './errors';

describe('ApiError', () => {
    it('categorizes 401 as auth', () => {
        const err = new ApiError(401, 'Unauthorized');
        expect(err.category).toBe('auth');
        expect(err.status).toBe(401);
        expect(err.body).toBe('Unauthorized');
        expect(err.name).toBe('ApiError');
    });

    it('categorizes 403 as auth', () => {
        expect(new ApiError(403, '').category).toBe('auth');
    });

    it('categorizes 400 as validation', () => {
        expect(new ApiError(400, 'bad request').category).toBe('validation');
    });

    it('categorizes 422 as validation', () => {
        expect(new ApiError(422, 'invalid').category).toBe('validation');
    });

    it('categorizes 500 as server', () => {
        expect(new ApiError(500, 'internal').category).toBe('server');
    });

    it('categorizes 502 as server', () => {
        expect(new ApiError(502, 'bad gateway').category).toBe('server');
    });

    it('categorizes 0 as network when explicitly set', () => {
        expect(new ApiError(0, 'Network error', 'network').category).toBe('network');
    });

    it('categorizes unknown status as unknown', () => {
        expect(new ApiError(418, 'teapot').category).toBe('unknown');
    });

    it('allows explicit category override', () => {
        const err = new ApiError(500, 'oops', 'network');
        expect(err.category).toBe('network');
    });

    it('extends Error with correct message', () => {
        const err = new ApiError(404, 'not found');
        expect(err.message).toBe('API error 404: not found');
        expect(err instanceof Error).toBe(true);
    });

    it('marks network errors as retryable', () => {
        expect(new ApiError(0, 'fail', 'network').retryable).toBe(true);
    });

    it('marks server errors as retryable', () => {
        expect(new ApiError(500, 'fail').retryable).toBe(true);
        expect(new ApiError(502, 'fail').retryable).toBe(true);
    });

    it('marks auth errors as not retryable', () => {
        expect(new ApiError(401, 'fail').retryable).toBe(false);
    });

    it('marks validation errors as not retryable', () => {
        expect(new ApiError(400, 'fail').retryable).toBe(false);
    });
});
