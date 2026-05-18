import { NextResponse } from 'next/server';
import { createAdminAccount, hasAdminAccount, signToken, validateCredentials } from '@/lib/auth';

export async function GET() {
  return NextResponse.json({ setupRequired: !hasAdminAccount(), product: 'hostQ' });
}

export async function POST(request: Request) {
  try {
    const { username, password } = await request.json();

    if (!username || !password) {
      return NextResponse.json({ error: 'Username and password required' }, { status: 400 });
    }

    if (!validateCredentials(username, password)) {
      return NextResponse.json({ error: 'Invalid username or password' }, { status: 401 });
    }

    const token = signToken(username);

    const response = NextResponse.json({ success: true, message: 'Authenticated' });
    response.cookies.set('panel_token', token, {
      httpOnly: true,
      secure: process.env.NODE_ENV === 'production',
      sameSite: 'lax',
      maxAge: 60 * 60 * 24, // 24 hours
      path: '/',
    });

    return response;
  } catch {
    return NextResponse.json({ error: 'Server error' }, { status: 500 });
  }
}

export async function PUT(request: Request) {
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

    const token = signToken(username);
    const response = NextResponse.json({ success: true, message: 'Admin account created' });
    response.cookies.set('panel_token', token, {
      httpOnly: true,
      secure: process.env.NODE_ENV === 'production',
      sameSite: 'lax',
      maxAge: 60 * 60 * 24,
      path: '/',
    });
    return response;
  } catch {
    return NextResponse.json({ error: 'Server error' }, { status: 500 });
  }
}

export async function DELETE() {
  const response = NextResponse.json({ success: true });
  response.cookies.delete('panel_token');
  return response;
}
