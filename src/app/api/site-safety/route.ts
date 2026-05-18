import { NextRequest, NextResponse } from 'next/server';
import { cookies } from 'next/headers';
import fs from 'fs';
import path from 'path';
import { verifyToken } from '@/lib/auth';
import { canManageSite } from '@/lib/authz';
import { mysqlString, runCommand, runMysql, shellQuote } from '@/lib/exec';
import { audit, clientIp } from '@/lib/security';

const WEB_ROOT = path.resolve(process.env.WEB_ROOT || '/var/www');
const SECRET_NAMES = new Set(['.env', '.env.local', '.env.production', 'id_rsa', 'id_dsa', 'authorized_keys']);
const SECRET_EXTS = new Set(['.key', '.pem', '.p12', '.pfx', '.crt', '.csr']);
const MAX_SCAN_ITEMS = 2000;

type Finding = {
  path: string;
  type: 'secret' | 'permission' | 'ownership' | 'database' | 'path';
  severity: 'low' | 'medium' | 'high';
  message: string;
};

async function auth() {
  const cookieStore = await cookies();
  const token = cookieStore.get('panel_token')?.value;
  return token ? verifyToken(token) : null;
}

function validDomain(domain: string) {
  return /^[a-zA-Z0-9][a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$/.test(domain);
}

function safeDocRoot(docRoot: string) {
  const full = path.resolve(docRoot || '');
  if (process.platform !== 'linux') return full;
  if (full !== WEB_ROOT && !full.startsWith(WEB_ROOT + path.sep)) return null;
  return full;
}

function isSecret(fullPath: string) {
  const base = path.basename(fullPath).toLowerCase();
  return SECRET_NAMES.has(base) || SECRET_EXTS.has(path.extname(base));
}

function relative(fullPath: string, root: string) {
  return '/' + path.relative(root, fullPath).replace(/\\/g, '/');
}

function scanFiles(root: string): Finding[] {
  const findings: Finding[] = [];
  let scanned = 0;

  function walk(dir: string) {
    if (scanned >= MAX_SCAN_ITEMS) return;
    let entries: fs.Dirent[] = [];
    try {
      entries = fs.readdirSync(dir, { withFileTypes: true });
    } catch {
      return;
    }

    for (const entry of entries) {
      if (scanned >= MAX_SCAN_ITEMS) return;
      const full = path.join(dir, entry.name);
      scanned += 1;
      if (entry.name === '.hostq-trash' || entry.name === 'node_modules') continue;
      if (isSecret(full)) {
        findings.push({
          path: relative(full, root),
          type: 'secret',
          severity: 'high',
          message: 'Secret-like file is present inside the public site tree',
        });
      }
      try {
        const stat = fs.statSync(full);
        if (entry.isFile() && (stat.mode & 0o002)) {
          findings.push({ path: relative(full, root), type: 'permission', severity: 'high', message: 'File is world-writable' });
        }
        if (entry.isDirectory() && (stat.mode & 0o002)) {
          findings.push({ path: relative(full, root), type: 'permission', severity: 'medium', message: 'Directory is world-writable' });
        }
      } catch {
        // skip unreadable entries
      }
      if (entry.isDirectory()) walk(full);
    }
  }

  walk(root);
  if (scanned >= MAX_SCAN_ITEMS) {
    findings.push({ path: '/', type: 'path', severity: 'low', message: `Scan stopped after ${MAX_SCAN_ITEMS} entries` });
  }
  return findings;
}

function readWordPressDb(root: string) {
  const wpConfig = path.join(root, 'wp-config.php');
  if (!fs.existsSync(wpConfig)) return null;
  const content = fs.readFileSync(wpConfig, 'utf8');
  const dbName = content.match(/define\s*\(\s*['"]DB_NAME['"]\s*,\s*['"]([^'"]+)['"]\s*\)/)?.[1] || '';
  const dbUser = content.match(/define\s*\(\s*['"]DB_USER['"]\s*,\s*['"]([^'"]+)['"]\s*\)/)?.[1] || '';
  return dbName ? { dbName, dbUser } : null;
}

async function databaseStatus(root: string): Promise<{ detected: boolean; dbName?: string; dbUser?: string; exists?: boolean; message: string }> {
  const wp = readWordPressDb(root);
  if (!wp) return { detected: false, message: 'No WordPress database config detected' };
  const r = await runMysql(`SELECT SCHEMA_NAME FROM INFORMATION_SCHEMA.SCHEMATA WHERE SCHEMA_NAME = ${mysqlString(wp.dbName)};`);
  const exists = r.success && r.stdout.split('\n').slice(1).some((line) => line.trim() === wp.dbName);
  return {
    detected: true,
    dbName: wp.dbName,
    dbUser: wp.dbUser,
    exists,
    message: exists ? `Database ${wp.dbName} exists` : `Database ${wp.dbName} is missing or not readable`,
  };
}

async function inspect(domain: string, docRoot: string) {
  const root = safeDocRoot(docRoot);
  if (!root) return { error: 'Site root must stay inside the configured web root' };
  if (process.platform !== 'linux') {
    return {
      root,
      findings: [
        { path: '/', type: 'path', severity: 'low', message: 'Demo mode: real file permissions are checked on Linux deployments' },
      ] as Finding[],
      database: { detected: false, message: 'Demo mode database check' },
      mode: 'demo',
    };
  }
  if (!fs.existsSync(root)) {
    return {
      root,
      findings: [{ path: '/', type: 'path', severity: 'high', message: 'Site document root does not exist' }] as Finding[],
      database: { detected: false, message: 'Site root missing' },
      mode: 'live',
    };
  }
  const findings = scanFiles(root);
  const database = await databaseStatus(root);
  if (database.detected && !database.exists) {
    findings.push({ path: '/wp-config.php', type: 'database', severity: 'high', message: database.message });
  }
  return { root, findings, database, mode: 'live', domain };
}

async function sanitize(root: string) {
  const quarantineRoot = `/var/backups/hostq/quarantine/${Date.now()}`;
  const findings = scanFiles(root).filter((finding) => finding.type === 'secret');
  for (const finding of findings) {
    const source = path.resolve(root, finding.path.replace(/^\/+/, ''));
    if (source !== root && source.startsWith(root + path.sep)) {
      const destination = path.join(quarantineRoot, finding.path.replace(/^\/+/, ''));
      await runCommand(`mkdir -p ${shellQuote(path.dirname(destination))} && mv ${shellQuote(source)} ${shellQuote(destination)} 2>/dev/null || true`);
    }
  }
  await runCommand(`find ${shellQuote(root)} -type d -exec chmod 755 {} \\; 2>/dev/null || true`);
  await runCommand(`find ${shellQuote(root)} -type f -exec chmod 644 {} \\; 2>/dev/null || true`);
  await runCommand(`chown -R www-data:www-data ${shellQuote(root)} 2>/dev/null || true`);
  return { quarantined: findings.length, quarantineRoot };
}

export async function GET(request: NextRequest) {
  const actor = await auth();
  if (!actor) return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });
  const domain = request.nextUrl.searchParams.get('domain') || '';
  const docRoot = request.nextUrl.searchParams.get('docRoot') || '';
  if (!validDomain(domain)) return NextResponse.json({ error: 'Invalid domain' }, { status: 400 });
  if (!canManageSite(actor, domain, 'view')) return NextResponse.json({ error: 'Forbidden' }, { status: 403 });
  const result = await inspect(domain, docRoot);
  if ('error' in result) return NextResponse.json({ error: result.error }, { status: 400 });
  return NextResponse.json(result);
}

export async function POST(request: NextRequest) {
  const actor = await auth();
  if (!actor) return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });
  const { domain, docRoot, action = 'sanitize' } = await request.json();
  if (!validDomain(domain)) return NextResponse.json({ error: 'Invalid domain' }, { status: 400 });
  if (!canManageSite(actor, domain, 'danger')) return NextResponse.json({ error: 'Forbidden' }, { status: 403 });
  const root = safeDocRoot(docRoot);
  if (!root) return NextResponse.json({ error: 'Site root must stay inside the configured web root' }, { status: 400 });

  if (process.platform !== 'linux') {
    audit({ actor: actor.username, action: `site_safety.${action}`, target: domain, ip: clientIp(request), details: { demo: true } });
    return NextResponse.json({ success: true, message: `${domain} safety action simulated`, demo: true });
  }
  if (!fs.existsSync(root)) return NextResponse.json({ error: 'Site root missing' }, { status: 404 });

  const result = await sanitize(root);
  audit({ actor: actor.username, action: `site_safety.${action}`, target: domain, ip: clientIp(request), details: result });
  return NextResponse.json({ success: true, message: `${domain} files sanitized and permissions repaired`, ...result });
}
