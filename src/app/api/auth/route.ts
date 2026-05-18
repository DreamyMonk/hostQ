import { NextResponse } from 'next/server';
import { createAdminAccount, createSession, hasAdminAccount, revokeSession, validateCredentials, validateOtp, verifyToken } from '@/lib/auth';
import {
  CSRF_COOKIE,
  audit,
  checkLoginRateLimit,
  clearLoginRateLimit,
  clientIp,
  createCsrfToken,
  safeUsername,
} from '@/lib/security';

function setSessionCookies(response: NextResponse, token: string) {
  const secure = process.env.NODE_ENV === 'production';
  response.cookies.set('panel_token', token, {
    httpOnly: true,
    secure,
    sameSite: 'strict',
    maxAge: 60 * 60 * 24,
    path: '/',
  });
  response.cookies.set(CSRF_COOKIE, createCsrfToken(), {
    httpOnly: false,
    secure,
    sameSite: 'strict',
    maxAge: 60 * 60 * 24,
    path: '/',
  });
}

export async function GET() {
  const response = NextResponse.json({ setupRequired: !hasAdminAccount(), product: 'hostQ' });
  response.cookies.set(CSRF_COOKIE, createCsrfToken(), {
    httpOnly: false,
    secure: process.env.NODE_ENV === 'production',
    sameSite: 'strict',
    maxAge: 60 * 60,
    path: '/',
  });
  return response;
}

export async function POST(request: Request) {
  const ip = clientIp(request);
  try {
    const { username, password, otp } = await request.json();
    const actor = safeUsername(String(username || 'unknown')) || 'unknown';
    const limit = checkLoginRateLimit(`${ip}:${actor}`);
    if (!limit.allowed) {
      audit({ actor, action: 'auth.login.rate_limited', status: 'failure', ip });
      return NextResponse.json(
        { error: 'Too many login attempts. Try again later.' },
        { status: 429, headers: { 'Retry-After': String(limit.retryAfter || 900) } },
      );
    }

    if (!username || !password) {
      return NextResponse.json({ error: 'Username and password required' }, { status: 400 });
    }

    const account = validateCredentials(username, password);
    if (!account) {
      audit({ actor, action: 'auth.login', status: 'failure', ip });
      return NextResponse.json({ error: 'Invalid username or password' }, { status: 401 });
    }
    if (!validateOtp(account, otp)) {
      audit({ actor, action: 'auth.login.otp_failed', status: 'failure', ip });
      return NextResponse.json({ error: 'Valid two-factor code required' }, { status: 401 });
    }

    const { token, session } = createSession(account, {
      ip,
      userAgent: request.headers.get('user-agent') || undefined,
    });
    clearLoginRateLimit(`${ip}:${actor}`);
    audit({ actor, action: 'auth.login', status: 'success', ip, details: { sessionId: session.id, role: account.role } });

    const response = NextResponse.json({ success: true, message: 'Authenticated' });
    setSessionCookies(response, token);

    return response;
  } catch {
    return NextResponse.json({ error: 'Server error' }, { status: 500 });
  }
}

export async function PUT(request: Request) {
  const ip = clientIp(request);
  try {
    if (hasAdminAccount()) {
      return NextResponse.json({ error: 'Admin account already exists' }, { status: 409 });
    }

    const { username, password, confirmPassword } = await request.json();
    if (!username || !password) {
      return NextResponse.json({ error: 'Username and password required' }, { status: 400 });
    }
    if (password !== confirmPassword) {
      return NextResponse.json({ error: 'Passwords do not match' }, { status: 400 });
    }

    const created = createAdminAccount(username, password);
    if (!created.success) {
      return NextResponse.json({ error: created.error || 'Account setup failed' }, { status: 400 });
    }

    audit({ actor: username, action: 'auth.first_admin_created', status: 'success', ip });
    return NextResponse.json({
      success: true,
      requiresLogin: true,
      message: 'Admin account created. Sign in and enable 2FA from Security settings.',
    });
  } catch {
    return NextResponse.json({ error: 'Server error' }, { status: 500 });
  }
}

export async function DELETE(request: Request) {
  const raw = request.headers.get('cookie')?.match(/panel_token=([^;]+)/)?.[1];
  if (raw) {
    const payload = verifyToken(decodeURIComponent(raw));
    if (payload?.sessionId) revokeSession(payload.sessionId);
  }
  const response = NextResponse.json({ success: true });
  response.cookies.delete('panel_token');
  response.cookies.delete(CSRF_COOKIE);
  return response;
}
