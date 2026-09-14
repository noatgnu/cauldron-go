import { ComponentFixture, TestBed } from '@angular/core/testing';
import { Router, provideRouter } from '@angular/router';
import { vi, describe, it, expect, beforeEach, afterEach } from 'vitest';
import { Login } from './login';

describe('Login', () => {
  let component: Login;
  let fixture: ComponentFixture<Login>;
  let router: Router;
  let fetchMock: ReturnType<typeof vi.fn>;
  const realFetch = globalThis.fetch;

  beforeEach(async () => {
    fetchMock = vi.fn();
    globalThis.fetch = fetchMock as unknown as typeof fetch;

    await TestBed.configureTestingModule({
      imports: [Login],
      providers: [provideRouter([])]
    }).compileComponents();

    fixture = TestBed.createComponent(Login);
    component = fixture.componentInstance;
    router = TestBed.inject(Router);
    vi.spyOn(router, 'navigateByUrl').mockResolvedValue(true);
    fixture.detectChanges();
  });

  afterEach(() => {
    globalThis.fetch = realFetch;
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('does nothing when the token field is empty', async () => {
    await component.submit();
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it('posts the token and navigates home on success', async () => {
    fetchMock.mockResolvedValue({ ok: true, json: async () => ({ ok: true }) });
    (component as any).token.set('secret-token');

    await component.submit();

    expect(fetchMock).toHaveBeenCalledWith('/auth/login', expect.objectContaining({
      method: 'POST',
      credentials: 'same-origin',
      body: JSON.stringify({ token: 'secret-token' })
    }));
    expect(router.navigateByUrl).toHaveBeenCalledWith('/');
  });

  it('shows the server error message on an invalid token', async () => {
    fetchMock.mockResolvedValue({ ok: false, json: async () => ({ error: 'invalid token' }) });
    (component as any).token.set('wrong-token');

    await component.submit();

    expect((component as any).error()).toBe('invalid token');
    expect(router.navigateByUrl).not.toHaveBeenCalled();
  });

  it('shows a generic error when the request fails outright', async () => {
    fetchMock.mockRejectedValue(new Error('network down'));
    (component as any).token.set('secret-token');

    await component.submit();

    expect((component as any).error()).toBe('Failed to reach the server');
  });
});
