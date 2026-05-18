// proxy.ts - Protect all /dashboard routes (Next.js 16+ convention)
import { NextResponse } from 'next/server';
import type { NextRequest } from 'next/server';

const CSRF_COOKIE = 'hostq_csrf';
const CSRF_HEADER = 'x-csrf-token';
const MUTATING = new Set(['POST', 'PUT', 'PATCH', 'DELETE']);

function addSecurityHeaders(response: NextResponse, production: boolean) {
  response.headers.set('X-Frame-Options', 'DENY');
  response.headers.set('X-Content-Type-Options', 'nosniff');
  response.headers.set('Referrer-Policy', 'same-origin');
  response.headers.set('Permissions-Policy', 'camera=(), microphone=(), geolocation=(), payment=()');
  response.headers.set(
    'Content-Security-Policy',
    "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data: blob:; connect-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'",
  );
  if (production) {
    response.headers.set('Strict-Transport-Security', 'max-age=31536000; includeSubDomains');
  }
  return response;
}

export function proxy(request: NextRequest) {
  const { pathname } = request.nextUrl;
  const production = process.env.NODE_ENV === 'production';
  const host = request.headers.get('host') || '';
  const forwardedProto = request.headers.get('x-forwarded-proto') || request.nextUrl.protocol.replace(':', '');
  const localHost = host.startsWith('localhost') || host.startsWith('127.0.0.1') || host.startsWith('[::1]');

  if (production && !localHost && forwardedProto !== 'https') {
    const url = request.nextUrl.clone();
    url.protocol = 'https:';
    return addSecurityHeaders(NextResponse.redirect(url, 308), production);
  }

  if (pathname.startsWith('/dashboard') || (pathname.startsWith('/api') && pathname !== '/api/auth')) {
    const token = request.cookies.get('panel_token')?.value;

    if (!token) {
      const response = pathname.startsWith('/api')
        ? NextResponse.json({ error: 'Unauthorized' }, { status: 401 })
        : NextResponse.redirect(new URL('/', request.url));
      return addSecurityHeaders(response, production);
    }

    if (token.split('.').length !== 3) {
      const response = NextResponse.redirect(new URL('/', request.url));
      response.cookies.delete('panel_token');
      return addSecurityHeaders(response, production);
    }
  }

  if (pathname.startsWith('/api') && MUTATING.has(request.method)) {
    const skipAuthBootstrap = pathname === '/api/auth' && (request.method === 'POST' || request.method === 'PUT');
    if (!skipAuthBootstrap) {
      const cookieToken = request.cookies.get(CSRF_COOKIE)?.value || '';
      const headerToken = request.headers.get(CSRF_HEADER) || '';
      if (!cookieToken || !headerToken || cookieToken !== headerToken) {
        return addSecurityHeaders(NextResponse.json({ error: 'Invalid CSRF token' }, { status: 403 }), production);
      }
    }
  }

  return addSecurityHeaders(NextResponse.next(), production);
}

export const config = {
  matcher: ['/dashboard/:path*', '/api/:path*'],
};
