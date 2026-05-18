import { NextResponse } from 'next/server';
import { cookies } from 'next/headers';
import fs from 'fs';
import { verifyToken } from '@/lib/auth';
import { auditLogPath } from '@/lib/security';
import { canManagePanel } from '@/lib/authz';

async function auth() {
  const cookieStore = await cookies();
  const token = cookieStore.get('panel_token')?.value;
  return token ? verifyToken(token) : null;
}

function ok(id: string, label: string, detail: string) {
  return { id, label, detail, status: 'pass' as const };
}

function warn(id: string, label: string, detail: string) {
  return { id, label, detail, status: 'warn' as const };
}

function fail(id: string, label: string, detail: string) {
  return { id, label, detail, status: 'fail' as const };
}

export async function GET() {
  const actor = await auth();
  if (!actor) return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });
  if (!canManagePanel(actor)) return NextResponse.json({ error: 'Forbidden' }, { status: 403 });

  const panelUrl = process.env.PANEL_URL || '';
  const jwtSecret = process.env.JWT_SECRET || '';
  const insecureHttp = process.env.HOSTQ_ALLOW_INSECURE_HTTP === 'true';
  const helperRequired = process.env.HOSTQ_REQUIRE_HELPER === 'true';
  const helper = process.env.HOSTQ_HELPER || '/usr/local/sbin/hostq-helper';
  const dataDir = process.env.HOSTQ_DATA_DIR || (process.platform === 'linux' ? '/etc/hostq' : '.hostq');
  const checks = [
    process.env.NODE_ENV === 'production'
      ? ok('node_env', 'Production runtime', 'NODE_ENV is production')
      : warn('node_env', 'Production runtime', 'Run with NODE_ENV=production outside development'),
    panelUrl.startsWith('https://')
      ? ok('panel_url', 'HTTPS panel URL', panelUrl)
      : fail('panel_url', 'HTTPS panel URL', 'Set PANEL_URL=https://panel.yourdomain.com'),
    insecureHttp
      ? warn('insecure_http', 'Plain HTTP setup access', 'HOSTQ_ALLOW_INSECURE_HTTP=true is for first setup only. Disable it after HTTPS works.')
      : ok('insecure_http', 'Plain HTTP setup access', 'Plain HTTP panel access is disabled'),
    jwtSecret.length >= 32 && !jwtSecret.includes('change_this') && !jwtSecret.includes('your_super_secret')
      ? ok('jwt_secret', 'Strong JWT secret', 'JWT secret appears customized')
      : fail('jwt_secret', 'Strong JWT secret', 'Generate a long random JWT_SECRET before production'),
    helperRequired
      ? ok('helper_required', 'Privileged helper enforced', 'Direct privileged shell commands are blocked')
      : warn('helper_required', 'Privileged helper enforced', 'Set HOSTQ_REQUIRE_HELPER=true after helper allowlist coverage is complete'),
    fs.existsSync(helper)
      ? ok('helper_installed', 'Helper installed', helper)
      : warn('helper_installed', 'Helper installed', `Install helper at ${helper}`),
    fs.existsSync(dataDir)
      ? ok('data_dir', 'Protected data directory', dataDir)
      : warn('data_dir', 'Protected data directory', `Create ${dataDir} with chmod 700`),
    fs.existsSync(auditLogPath())
      ? ok('audit_log', 'Audit log active', auditLogPath())
      : warn('audit_log', 'Audit log active', 'Audit log will appear after the first audited action'),
    process.platform === 'linux'
      ? ok('os', 'Linux target', 'Running on a Linux-like production target')
      : warn('os', 'Linux target', 'Windows/macOS are development/demo environments only'),
  ];

  const score = Math.round((checks.filter((check) => check.status === 'pass').length / checks.length) * 100);
  return NextResponse.json({
    score,
    ready: checks.every((check) => check.status !== 'fail') && score >= 80,
    checks,
  });
}
