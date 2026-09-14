import { inject } from '@angular/core';
import { CanActivateChildFn, Router } from '@angular/router';

// Only blocks navigation when /auth/status explicitly reports unauthenticated; desktop mode has no such endpoint and passes through.
export const authGuard: CanActivateChildFn = async () => {
  const router = inject(Router);

  try {
    const res = await fetch('/auth/status', { credentials: 'same-origin' });
    if (!res.ok) return true;

    const status = await res.json();
    if (status.authenticated) return true;

    return router.createUrlTree(['/login']);
  } catch {
    return true;
  }
};
