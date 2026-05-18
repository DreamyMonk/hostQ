// proxy.ts - Protect all /dashboard routes (Next.js 16+ convention)
import { NextResponse } from 'next/server';
import type { NextRequest } from 'next/server';

export function proxy(request: NextRequest) {
  const { pathname } = request.nextUrl;

  // Only protect dashboard routes
  if (pathname.startsWith('/dashboard')) {
    const token = request.cookies.get('panel_token')?.value;

    if (!token) {
      return NextResponse.redirect(new URL('/', request.url));
    }

    // Basic token presence check (full verify happens in API routes)
    if (token.split('.').length !== 3) {
      const response = NextResponse.redirect(new URL('/', request.url));
      response.cookies.delete('panel_token');
      return response;
    }
  }

  return NextResponse.next();
}

export const config = {
  matcher: ['/dashboard/:path*'],
};
