import { NextResponse } from 'next/server';
import { cookies } from 'next/headers';
import { verifyToken } from '@/lib/auth';
import { runCommand } from '@/lib/exec';
import { audit, clientIp } from '@/lib/security';
import { canManagePanel } from '@/lib/authz';

async function auth() {
  const cookieStore = await cookies();
  const token = cookieStore.get('panel_token')?.value;
  return token ? verifyToken(token) : null;
}

const SUPPORTED_PHP_VERSIONS = ['8.2', '8.3', '8.4', '8.5'];

export async function GET() {
  const actor = await auth();
  if (!actor) return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });
  if (!canManagePanel(actor)) return NextResponse.json({ error: 'Forbidden' }, { status: 403 });
  
  const results = await Promise.all([
    runCommand('php --version'),
    runCommand('update-alternatives --list php 2>/dev/null || ls /usr/bin/php* 2>/dev/null || echo "php"'),
    runCommand('php -r "echo PHP_VERSION;"'),
  ]);

  // Parse available versions
  const rawList = results[1].stdout;
  const versions: string[] = [];
  
  // Try to extract version numbers from update-alternatives output
  const vMatches = rawList.match(/php(\d+\.\d+)/g);
  if (vMatches && vMatches.length > 0) {
    vMatches.forEach(v => {
      const ver = v.replace('php', '');
      if (!versions.includes(ver)) versions.push(ver);
    });
  }
  
  // Fallback: manually check common PHP binaries
  if (versions.length === 0) {
    const checks = await Promise.all(
      SUPPORTED_PHP_VERSIONS.map(v =>
        runCommand(`which php${v} 2>/dev/null || ls /usr/bin/php${v} 2>/dev/null`)
      )
    );
    SUPPORTED_PHP_VERSIONS.forEach((v, i) => {
      if (checks[i].success && checks[i].stdout) versions.push(v);
    });
  }

  const currentVersion = results[2].stdout || 'unknown';
  const activeMatch = currentVersion.match(/^(\d+\.\d+)/);
  const active = activeMatch ? activeMatch[1] : '';

  // If no PHP installed at all, return demo data
  if (versions.length === 0 && !results[0].success) {
    return NextResponse.json({
      active: 'N/A',
      versions: SUPPORTED_PHP_VERSIONS,
      currentOutput: 'PHP not installed or not in PATH',
      demo: true,
    });
  }

  if (active && !versions.includes(active)) versions.unshift(active);

  return NextResponse.json({ active, versions: versions.sort(), currentOutput: results[0].stdout, demo: false });
}

export async function POST(request: Request) {
  const actor = await auth();
  if (!actor) return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });
  if (!canManagePanel(actor)) return NextResponse.json({ error: 'Forbidden' }, { status: 403 });
  
  const { version } = await request.json();
  if (!version || !/^\d+\.\d+$/.test(version)) {
    return NextResponse.json({ error: 'Invalid version format' }, { status: 400 });
  }
  if (!SUPPORTED_PHP_VERSIONS.includes(version)) {
    return NextResponse.json({ error: 'Only currently supported PHP versions are allowed' }, { status: 400 });
  }

  const cmds = [
    `update-alternatives --set php /usr/bin/php${version}`,
    `update-alternatives --set php-fpm.sock /run/php/php${version}-fpm.sock 2>/dev/null || true`,
    `systemctl restart php${version}-fpm 2>/dev/null || true`,
    `systemctl reload nginx 2>/dev/null || systemctl reload apache2 2>/dev/null || true`,
  ];

  const outputs: string[] = [];
  for (const cmd of cmds) {
    const r = await runCommand(cmd, 15000);
    outputs.push(`$ ${cmd}\n${r.stdout || r.stderr || '(no output)'}`);
  }

  const verify = await runCommand('php --version');
  const newVersion = verify.stdout.match(/PHP (\d+\.\d+)/)?.[1] || version;
  audit({ actor: actor.username, action: 'php.switch', target: version, ip: clientIp(request), details: { newVersion } });

  return NextResponse.json({
    success: true,
    newVersion,
    output: outputs.join('\n\n'),
    message: `PHP switched to ${version}`,
  });
}
