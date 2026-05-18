import { NextRequest, NextResponse } from 'next/server';
import { cookies } from 'next/headers';
import { verifyToken } from '@/lib/auth';
import { canManagePanel } from '@/lib/authz';
import { runHelper } from '@/lib/exec';
import { audit, clientIp } from '@/lib/security';
import pkg from '../../../../package.json';

const REPO = 'DreamyMonk/hostQ';
const RELEASES_URL = `https://api.github.com/repos/${REPO}/releases/latest`;

async function auth() {
  const cookieStore = await cookies();
  const token = cookieStore.get('panel_token')?.value;
  return token ? verifyToken(token) : null;
}

function normalize(version: string) {
  return version.replace(/^v/, '');
}

function newerThan(latest: string, current: string) {
  const l = normalize(latest).split(/[.-]/).slice(0, 3).map(Number);
  const c = normalize(current).split(/[.-]/).slice(0, 3).map(Number);
  for (let i = 0; i < 3; i += 1) {
    if ((l[i] || 0) > (c[i] || 0)) return true;
    if ((l[i] || 0) < (c[i] || 0)) return false;
  }
  return false;
}

async function latestRelease() {
  const response = await fetch(RELEASES_URL, {
    headers: { Accept: 'application/vnd.github+json', 'User-Agent': 'hostQ-updater' },
    cache: 'no-store',
  });
  if (!response.ok) return null;
  const release = await response.json();
  return {
    tag: release.tag_name as string,
    name: release.name as string,
    notes: release.body as string,
    url: release.html_url as string,
    publishedAt: release.published_at as string,
    prerelease: Boolean(release.prerelease),
  };
}

export async function GET() {
  const actor = await auth();
  if (!actor) return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });
  if (!canManagePanel(actor)) return NextResponse.json({ error: 'Forbidden' }, { status: 403 });

  const latest = await latestRelease();
  return NextResponse.json({
    current: `v${pkg.version}`,
    repo: REPO,
    latest,
    updateAvailable: latest ? newerThan(latest.tag, pkg.version) : false,
  });
}

export async function POST(request: NextRequest) {
  const actor = await auth();
  if (!actor) return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });
  if (!canManagePanel(actor)) return NextResponse.json({ error: 'Forbidden' }, { status: 403 });

  const { tag, confirm } = await request.json();
  if (!tag || !/^v\d+\.\d+\.\d+([.-][A-Za-z0-9]+)?$/.test(tag)) {
    return NextResponse.json({ error: 'Invalid release tag' }, { status: 400 });
  }
  if (confirm !== tag) {
    return NextResponse.json({ error: `Type/submit confirm="${tag}" to update` }, { status: 400 });
  }

  const r = await runHelper('panel.update', { tag }, 600000);
  audit({
    actor: actor.username,
    action: 'panel.update',
    target: tag,
    status: r.success ? 'success' : 'failure',
    ip: clientIp(request),
  });

  return NextResponse.json({
    success: r.success,
    message: r.success ? `hostQ updated to ${tag}` : 'Update failed',
    output: r.stdout || r.stderr || r.error,
  }, { status: r.success ? 200 : 500 });
}
