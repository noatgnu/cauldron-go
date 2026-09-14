import { TestBed } from '@angular/core/testing';
import { Router, UrlTree, provideRouter } from '@angular/router';
import { vi, describe, it, expect, beforeEach, afterEach } from 'vitest';
import { authGuard } from './auth.guard';

describe('authGuard', () => {
  let fetchMock: ReturnType<typeof vi.fn>;
  const realFetch = globalThis.fetch;

  beforeEach(() => {
    fetchMock = vi.fn();
    globalThis.fetch = fetchMock as unknown as typeof fetch;
    TestBed.configureTestingModule({ providers: [provideRouter([])] });
  });

  afterEach(() => {
    globalThis.fetch = realFetch;
  });

  function runGuard() {
    return TestBed.runInInjectionContext(() => authGuard({} as any, {} as any));
  }

  it('allows navigation when authenticated', async () => {
    fetchMock.mockResolvedValue({ ok: true, json: async () => ({ authenticated: true }) });
    expect(await runGuard()).toBe(true);
  });

  it('redirects to /login when not authenticated', async () => {
    fetchMock.mockResolvedValue({ ok: true, json: async () => ({ authenticated: false }) });
    const result = await runGuard();
    expect(result).toBeInstanceOf(UrlTree);
    const router = TestBed.inject(Router);
    expect(router.serializeUrl(result as UrlTree)).toBe('/login');
  });

  it('allows navigation when the endpoint is not found (desktop mode)', async () => {
    fetchMock.mockResolvedValue({ ok: false, status: 404 });
    expect(await runGuard()).toBe(true);
  });

  it('allows navigation when the fetch fails outright', async () => {
    fetchMock.mockRejectedValue(new Error('network error'));
    expect(await runGuard()).toBe(true);
  });
});
