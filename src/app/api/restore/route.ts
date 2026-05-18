import { NextRequest, NextResponse } from 'next/server';
import { cookies } from 'next/headers';
import path from 'path';
import fs from 'fs';
import { verifyToken } from '@/lib/auth';
import { runCommand, shellQuote } from '@/lib/exec';
import { audit, clientIp } from '@/lib/security';
import { canManageSite } from '@/lib/authz';

async function auth() {
  const cookieStore = await cookies();
  const token = cookieStore.get('panel_token')?.value;
  return token ? verifyToken(token) : null;
}

const BACKUP_ROOT = '/var/backups/hostq';
const WEB_ROOT = process.env.WEB_ROOT || '/var/www';

function validBackup(file: string) {
  const full = path.resolve(BACKUP_ROOT, file);
  return full.startsWith(path.resolve(BACKUP_ROOT) + path.sep) && /\.tar\.gz$/.test(full);
}

function unsafeArchiveList(output: string) {
  return output.split('\n').some((line) => line.startsWith('/') || line.includes('../') || /\.(env|key|pem|p12|pfx)$/i.test(line));
}

export async function POST(request: NextRequest) {
  const actor = await auth();
  if (!actor) return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });

  const { domain, backupFile, dryRun = true, confirm } = await request.json();
  if (!domain || !/^[a-zA-Z0-9][a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$/.test(domain)) {
    return NextResponse.json({ error: 'Invalid domain' }, { status: 400 });
  }
  if (!canManageSite(actor, domain, 'danger')) return NextResponse.json({ error: 'Forbidden' }, { status: 403 });
  if (!backupFile || !validBackup(backupFile)) return NextResponse.json({ error: 'Invalid backup file' }, { status: 400 });

  const backupPath = path.resolve(BACKUP_ROOT, backupFile);
  if (process.platform === 'linux' && !fs.existsSync(backupPath)) return NextResponse.json({ error: 'Backup not found' }, { status: 404 });

  const list = process.platform === 'linux'
    ? await runCommand(`tar -tzf ${shellQuote(backupPath)} 2>&1`, 60000)
    : { success: true, stdout: 'index.php\nwp-content/', stderr: '' };
  if (!list.success) return NextResponse.json({ error: list.stderr || 'Unable to inspect archive' }, { status: 400 });
  if (unsafeArchiveList(list.stdout)) return NextResponse.json({ error: 'Backup contains unsafe paths or secret files' }, { status: 400 });

  if (dryRun || confirm !== domain) {
    return NextResponse.json({
      success: true,
      dryRun: true,
      message: `Dry run complete. To restore, send confirm="${domain}".`,
      files: list.stdout.split('\n').filter(Boolean).slice(0, 200),
    });
  }

  const target = path.join(WEB_ROOT, domain, 'public_html');
  const restorePoint = `/var/backups/hostq/pre-restore-${domain}-${Date.now()}.tar.gz`;
  const backupCurrent = await runCommand(`mkdir -p ${shellQuote(path.dirname(restorePoint))} && tar -czf ${shellQuote(restorePoint)} -C ${shellQuote(target)} . 2>&1`, 120000);
  const restore = backupCurrent.success
    ? await runCommand(`tar -xzf ${shellQuote(backupPath)} -C ${shellQuote(target)} --no-same-owner --no-same-permissions 2>&1`, 120000)
    : backupCurrent;

  audit({
    actor: actor.username,
    action: 'site.restore',
    target: domain,
    status: restore.success ? 'success' : 'failure',
    ip: clientIp(request),
    details: { backupFile, restorePoint },
  });

  return NextResponse.json({
    success: restore.success,
    message: restore.success ? `${domain} restored` : 'Restore failed',
    output: restore.stdout || restore.stderr || restore.error,
    restorePoint,
  }, { status: restore.success ? 200 : 500 });
}
