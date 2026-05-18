'use client';
import { useEffect } from 'react';

function csrfToken() {
  return document.cookie
    .split('; ')
    .find((part) => part.startsWith('hostq_csrf='))
    ?.split('=')
    .slice(1)
    .join('=');
}

function isSameOrigin(input: RequestInfo | URL) {
  const url = typeof input === 'string'
    ? new URL(input, window.location.origin)
    : input instanceof URL
      ? input
      : new URL(input.url, window.location.origin);
  return url.origin === window.location.origin;
}

export default function SecurityFetch() {
  useEffect(() => {
    if ((window as typeof window & { __hostqFetchPatched?: boolean }).__hostqFetchPatched) return;
    (window as typeof window & { __hostqFetchPatched?: boolean }).__hostqFetchPatched = true;

    const originalFetch = window.fetch.bind(window);
    window.fetch = (input, init = {}) => {
      const method = (init.method || (input instanceof Request ? input.method : 'GET')).toUpperCase();
      if (['POST', 'PUT', 'PATCH', 'DELETE'].includes(method) && isSameOrigin(input)) {
        const headers = new Headers(init.headers || (input instanceof Request ? input.headers : undefined));
        const token = csrfToken();
        if (token && !headers.has('x-csrf-token')) headers.set('x-csrf-token', decodeURIComponent(token));
        return originalFetch(input, { ...init, headers });
      }
      return originalFetch(input, init);
    };
  }, []);

  return null;
}
