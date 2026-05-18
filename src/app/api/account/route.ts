import { NextRequest, NextResponse } from 'next/server';
import { cookies } from 'next/headers';
import { changePassword, enableOtp, startOtpSetup, verifyToken } from '@/lib/auth';
import { audit, clientIp } from '@/lib/security';

async function auth() {
  const cookieStore = await cookies();
  const token = cookieStore.get('panel_token')?.value;
  return token ? verifyToken(token) : null;
}

export async function POST(request: NextRequest) {
  const actor = await auth();
  if (!actor) return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });

  const body = await request.json();
  if (body.action === 'change_password') {
    const result = changePassword(actor.username, body.currentPassword || '', body.newPassword || '');
    audit({
      actor: actor.username,
      action: 'account.change_password',
      status: result.success ? 'success' : 'failure',
      ip: clientIp(request),
    });
    return NextResponse.json(result.success ? { success: true, message: 'Password changed. Sign in again.' } : { error: result.error }, { status: result.success ? 200 : 400 });
  }

  if (body.action === 'start_2fa') {
    const result = startOtpSetup(actor.username);
    audit({
      actor: actor.username,
      action: 'account.2fa_start',
      status: result.success ? 'success' : 'failure',
      ip: clientIp(request),
    });
    return NextResponse.json(result.success ? result : { error: result.error }, { status: result.success ? 200 : 400 });
  }

  if (body.action === 'enable_2fa') {
    const result = enableOtp(actor.username, body.otp || '');
    audit({
      actor: actor.username,
      action: 'account.2fa_enable',
      status: result.success ? 'success' : 'failure',
      ip: clientIp(request),
    });
    return NextResponse.json(result.success ? { success: true, message: '2FA enabled' } : { error: result.error }, { status: result.success ? 200 : 400 });
  }

  return NextResponse.json({ error: 'Unknown account action' }, { status: 400 });
}
