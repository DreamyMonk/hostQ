import { NextRequest, NextResponse } from 'next/server';
import { cookies } from 'next/headers';
import { verifyToken } from '@/lib/auth';
import { canManagePanel } from '@/lib/authz';
import { audit, clientIp } from '@/lib/security';
import { readPanelConfig, upsertEnv, validPanelDomain } from '@/lib/panel-config';

async function auth() {
  const cookieStore = await cookies();
  const token = cookieStore.get('panel_token')?.value;
  return token ? verifyToken(token) : null;
}

export async function GET() {
  const actor = await auth();
  if (!actor) return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });
  if (!canManagePanel(actor)) return NextResponse.json({ error: 'Forbidden' }, { status: 403 });

  return NextResponse.json(readPanelConfig());
}

export async function PATCH(request: NextRequest) {
  const actor = await auth();
  if (!actor) return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });
  if (!canManagePanel(actor)) return NextResponse.json({ error: 'Forbidden' }, { status: 403 });

  const body = await request.json();
  const panelDomain = String(body.panelDomain || '').trim().toLowerCase();
  const allowInsecureHttp = Boolean(body.allowInsecureHttp);
  const scheme = allowInsecureHttp ? 'http' : 'https';

  if (!validPanelDomain(panelDomain)) {
    return NextResponse.json({ error: 'Enter a valid domain or subdomain, for example panel.example.com' }, { status: 400 });
  }

  const panelUrl = `${scheme}://${panelDomain}`;
  upsertEnv({
    PANEL_DOMAIN: panelDomain,
    PANEL_URL: panelUrl,
    HOSTQ_ALLOW_INSECURE_HTTP: allowInsecureHttp ? 'true' : 'false',
  });

  audit({
    actor: actor.username,
    action: 'panel.domain_update',
    target: panelDomain,
    ip: clientIp(request),
    details: { panelUrl, allowInsecureHttp },
  });

  return NextResponse.json({
    success: true,
    message: 'Panel host saved. Restart hostQ to apply runtime security changes.',
    restartRequired: true,
    config: readPanelConfig(),
  });
}
