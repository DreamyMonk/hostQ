import { NextResponse } from 'next/server';
import { cookies } from 'next/headers';
import { findAccount, verifyToken } from '@/lib/auth';

export async function GET() {
  const cookieStore = await cookies();
  const token = cookieStore.get('panel_token')?.value;
  const actor = token ? verifyToken(token) : null;
  if (!actor) return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });

  const account = findAccount(actor.username);
  return NextResponse.json({
    username: actor.username,
    role: actor.role,
    sessionId: actor.sessionId,
    sites: account?.sitePermissions || {},
  });
}
