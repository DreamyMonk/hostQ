import { NextRequest, NextResponse } from 'next/server';
import { cookies } from 'next/headers';
import { verifyToken } from '@/lib/auth';
import { addSiteUser, listSiteUsers, removeSiteUser, type SiteRole } from '@/lib/site-users';
import { audit, clientIp } from '@/lib/security';
import { canManageSite } from '@/lib/authz';

async function auth() {
  const cookieStore = await cookies();
  const token = cookieStore.get('panel_token')?.value;
  return token ? verifyToken(token) : null;
}

export async function GET(request: NextRequest) {
  const actor = await auth();
  if (!actor) return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });
  const domain = request.nextUrl.searchParams.get('domain') || undefined;
  if (domain && !canManageSite(actor, domain, 'admin')) return NextResponse.json({ error: 'Forbidden' }, { status: 403 });
  return NextResponse.json({ users: listSiteUsers(domain) });
}

export async function POST(request: NextRequest) {
  const actor = await auth();
  if (!actor) return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });

  try {
    const { domain, username, role = 'developer', password } = await request.json();
    if (!canManageSite(actor, domain, 'admin')) return NextResponse.json({ error: 'Forbidden' }, { status: 403 });
    const user = addSiteUser(domain, username, role as SiteRole, password);
    audit({
      actor: actor.username,
      action: 'site_user.add',
      target: `${domain}:${username}`,
      ip: clientIp(request),
      details: { role },
    });
    return NextResponse.json({ success: true, user, message: `${username} can now manage ${domain}` });
  } catch (error) {
    return NextResponse.json({ error: error instanceof Error ? error.message : 'Unable to add user' }, { status: 400 });
  }
}

export async function DELETE(request: NextRequest) {
  const actor = await auth();
  if (!actor) return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });

  const { domain, username } = await request.json();
  if (!domain || !username) return NextResponse.json({ error: 'Domain and username required' }, { status: 400 });
  if (!canManageSite(actor, domain, 'admin')) return NextResponse.json({ error: 'Forbidden' }, { status: 403 });
  const removed = removeSiteUser(domain, username);
  audit({
    actor: actor.username,
    action: 'site_user.remove',
    target: `${domain}:${username}`,
    ip: clientIp(request),
    status: removed ? 'success' : 'failure',
  });
  return NextResponse.json({ success: removed, message: removed ? `${username} removed from ${domain}` : 'User was not assigned to this site' }, { status: removed ? 200 : 404 });
}
