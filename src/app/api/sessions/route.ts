import { NextRequest, NextResponse } from 'next/server';
import { cookies } from 'next/headers';
import { listSessions, revokeSession, revokeUserSessions, verifyToken } from '@/lib/auth';
import { audit, clientIp } from '@/lib/security';
import { canManagePanel } from '@/lib/authz';

async function auth() {
  const cookieStore = await cookies();
  const token = cookieStore.get('panel_token')?.value;
  return token ? verifyToken(token) : null;
}

export async function GET() {
  const actor = await auth();
  if (!actor) return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });
  const sessions = canManagePanel(actor) ? listSessions() : listSessions(actor.username);
  return NextResponse.json({ sessions });
}

export async function DELETE(request: NextRequest) {
  const actor = await auth();
  if (!actor) return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });
  const { sessionId, username } = await request.json();

  if (username) {
    if (!canManagePanel(actor) && actor.username !== username) return NextResponse.json({ error: 'Forbidden' }, { status: 403 });
    revokeUserSessions(username);
    audit({ actor: actor.username, action: 'sessions.revoke_user', target: username, ip: clientIp(request) });
    return NextResponse.json({ success: true, message: `Sessions revoked for ${username}` });
  }

  if (!sessionId) return NextResponse.json({ error: 'sessionId or username required' }, { status: 400 });
  const session = listSessions().find((item) => item.id === sessionId);
  if (!session || (!canManagePanel(actor) && session.username !== actor.username)) {
    return NextResponse.json({ error: 'Forbidden' }, { status: 403 });
  }
  revokeSession(sessionId);
  audit({ actor: actor.username, action: 'sessions.revoke', target: sessionId, ip: clientIp(request) });
  return NextResponse.json({ success: true, message: 'Session revoked' });
}
