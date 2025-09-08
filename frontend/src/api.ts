import { safeRemoveLocal } from './runtimeConfig';

export async function apiFetch(input: RequestInfo | URL, init?: RequestInit): Promise<Response> {
  // Default to sending credentials (cookies) so protected API endpoints
  // that expect the HttpOnly session cookie work across dev/prod.
  const mergedInit: RequestInit = { credentials: 'include', ...init };
  const res = await fetch(input, mergedInit);
  if (res.status === 401) {
    // Best-effort local cleanup; ignore storage errors.
    safeRemoveLocal('game_id');
    safeRemoveLocal('player_email');
    safeRemoveLocal('session_ok');
    safeRemoveLocal('user');
    if (window.location.pathname === '/') {
      // Already on home; force a reload so UI shows the login prompt
      window.location.reload();
    } else {
      window.location.href = '/';
    }
    // Return a rejected promise to stop caller logic after redirect
    throw new Error('Unauthorized');
  }
  return res;
}

export async function apiJson<T = any>(input: RequestInfo | URL, init?: RequestInit): Promise<T> {
  const res = await apiFetch(input, init);
  return res.json() as Promise<T>;
}
