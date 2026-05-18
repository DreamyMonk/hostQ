import { NextResponse } from 'next/server';
import { cookies } from 'next/headers';
import { verifyToken } from '@/lib/auth';
import { mysqlIdentifier, mysqlString, runCommand, runMysql, shellQuote } from '@/lib/exec';
import { audit, clientIp } from '@/lib/security';
import { canManageSite, domainFromWebPath } from '@/lib/authz';

function validEmail(value: string) {
  return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value);
}

async function auth() {
  const cookieStore = await cookies();
  const token = cookieStore.get('panel_token')?.value;
  return token ? verifyToken(token) : null;
}

export async function GET() {
  const actor = await auth();
  if (!actor) return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });

  const webRoot = process.env.WEB_ROOT || '/var/www/html';
  const wpCli = await runCommand('command -v wp || test -x /usr/local/bin/wp', 5000);
  const r = await runCommand(`find ${shellQuote(webRoot)} -path ${shellQuote(`${webRoot}/.hostq-trash`)} -prune -o -maxdepth 4 -name "wp-config.php" -print 2>/dev/null`);

  if (!wpCli.success) {
    return NextResponse.json({
      installations: [
        { domain: 'myblog.com', path: '/var/www/html/myblog', status: 'running', wpVersion: '6.4.3', db: 'myblog_db' },
        { domain: 'shop.example.com', path: '/var/www/html/shop', status: 'running', wpVersion: '6.4.2', db: 'shop_db' },
      ],
      demo: true,
      message: 'WP-CLI was not detected. Install wp-cli to enable real WordPress management.',
    });
  }

  if (!r.success || !r.stdout) {
    return NextResponse.json({ installations: [], demo: false, wpCliAvailable: true });
  }

  const paths = r.stdout.split('\n').filter(Boolean).map((p) => p.replace('/wp-config.php', ''));
  const installations = paths.filter((p) => {
    const domain = domainFromWebPath(p, webRoot);
    return !domain || canManageSite(actor, domain, 'view');
  }).map((p) => ({
    domain: p.split('/').pop() || p,
    path: p,
    status: 'running',
    wpVersion: 'unknown',
    db: '',
  }));

  return NextResponse.json({ installations, demo: false, wpCliAvailable: true });
}

export async function POST(request: Request) {
  const actor = await auth();
  if (!actor) return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });

  const { domain, dbName, dbUser, dbPassword, adminEmail, siteTitle, adminUser, adminPass } = await request.json();

  if (!domain || !dbName || !dbUser || !dbPassword || !adminEmail) {
    return NextResponse.json({ error: 'All fields required' }, { status: 400 });
  }
  if (!/^[a-zA-Z0-9][a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$/.test(domain)) {
    return NextResponse.json({ error: 'Invalid domain name' }, { status: 400 });
  }
  if (!validEmail(adminEmail)) {
    return NextResponse.json({ error: 'Enter a valid WordPress admin email address' }, { status: 400 });
  }
  if (!canManageSite(actor, domain, 'write')) return NextResponse.json({ error: 'Forbidden' }, { status: 403 });
  if (!/^[a-zA-Z0-9_]+$/.test(dbName) || !/^[a-zA-Z0-9_]+$/.test(dbUser)) {
    return NextResponse.json({ error: 'Database name and user may only contain letters, numbers, and underscores' }, { status: 400 });
  }

  const webRoot = process.env.WEB_ROOT || '/var/www/html';
  const sitePath = `${webRoot}/${domain}`;
  const qSitePath = shellQuote(sitePath);
  const logs: string[] = [];
  let failed = false;

  const steps = [
    { label: 'Create directory', cmd: `mkdir -p ${qSitePath}` },
    { label: 'Download WordPress', cmd: `wp core download --path=${qSitePath} --allow-root 2>&1` },
    { label: 'Create database', sql: `CREATE DATABASE IF NOT EXISTS ${mysqlIdentifier(dbName)};` },
    { label: 'Create DB user', sql: `CREATE USER IF NOT EXISTS ${mysqlString(dbUser)}@'localhost' IDENTIFIED BY ${mysqlString(dbPassword)}; GRANT ALL PRIVILEGES ON ${mysqlIdentifier(dbName)}.* TO ${mysqlString(dbUser)}@'localhost'; FLUSH PRIVILEGES;` },
    {
      label: 'Configure wp-config.php',
      cmd: `wp config create --path=${qSitePath} --dbname=${shellQuote(dbName)} --dbuser=${shellQuote(dbUser)} --dbpass=${shellQuote(dbPassword)} --dbhost=localhost --allow-root 2>&1`,
    },
    {
      label: 'Install WordPress',
      cmd: `wp core install --path=${qSitePath} --url=${shellQuote(`http://${domain}`)} --title=${shellQuote(siteTitle || domain)} --admin_user=${shellQuote(adminUser || 'admin')} --admin_password=${shellQuote(adminPass || 'changeme123')} --admin_email=${shellQuote(adminEmail)} --allow-root 2>&1`,
    },
    { label: 'Set permissions', cmd: `chown -R www-data:www-data ${qSitePath} 2>/dev/null || true` },
  ];

  for (const step of steps) {
    logs.push(`\n> ${step.label}...`);
    if ('sql' in step && step.sql) {
      const r = await runMysql(step.sql);
      if (r.success) logs.push('OK Done');
      else {
        failed = true;
        logs.push(`FAILED ${r.stderr || r.stdout || r.error}`);
        break;
      }
    } else if ('cmd' in step && step.cmd) {
      const r = await runCommand(step.cmd, 120000);
      if (r.success) logs.push(`OK ${r.stdout || 'Done'}`);
      else {
        failed = true;
        logs.push(`FAILED ${r.stderr || r.stdout || r.error}`);
        break;
      }
    }
  }

  audit({ actor: actor.username, action: 'wordpress.install', target: domain, status: failed ? 'failure' : 'success', ip: clientIp(request), details: { dbName } });
  if (failed) {
    return NextResponse.json({
      success: false,
      output: logs.join('\n'),
      error: 'WordPress installation failed. Check the installation log.',
    }, { status: 500 });
  }

  return NextResponse.json({
    success: true,
    output: logs.join('\n'),
    loginUrl: `http://${domain}/wp-admin`,
    message: `WordPress installed at http://${domain}`,
  });
}

export async function DELETE(request: Request) {
  const actor = await auth();
  if (!actor) return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });

  const { path: sitePath, dbName, deleteFiles = false, deleteDatabase = false } = await request.json();
  const webRoot = process.env.WEB_ROOT || '/var/www/html';
  if (!sitePath || !String(sitePath).startsWith(webRoot)) {
    return NextResponse.json({ error: 'Invalid WordPress path' }, { status: 400 });
  }
  const domain = domainFromWebPath(sitePath, webRoot);
  if (domain && !canManageSite(actor, domain, 'danger')) return NextResponse.json({ error: 'Forbidden' }, { status: 403 });

  const logs: string[] = [];
  if (deleteFiles) {
    const trashPath = `${webRoot}/.hostq-trash/${Date.now()}-${String(sitePath).split('/').pop()}`;
    const r = await runCommand(`mkdir -p ${shellQuote(`${webRoot}/.hostq-trash`)} && mv ${shellQuote(sitePath)} ${shellQuote(trashPath)}`, 30000);
    logs.push(r.success ? `Soft-deleted files to ${trashPath}` : `Failed moving files: ${r.stderr || r.stdout || r.error}`);
  }
  if (deleteDatabase && dbName) {
    if (!/^[a-zA-Z0-9_]+$/.test(dbName)) return NextResponse.json({ error: 'Invalid database name' }, { status: 400 });
    const r = await runMysql(`DROP DATABASE IF EXISTS ${mysqlIdentifier(dbName)};`);
    logs.push(r.success ? `Dropped database ${dbName}` : `Failed dropping database: ${r.stderr || r.stdout || r.error}`);
  }

  audit({ actor: actor.username, action: 'wordpress.delete', target: sitePath, ip: clientIp(request), details: { dbName, deleteFiles, deleteDatabase } });
  return NextResponse.json({ success: true, message: 'WordPress site removed', output: logs.join('\n') || 'No destructive action selected' });
}
