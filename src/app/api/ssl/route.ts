import { NextResponse } from 'next/server';
import { cookies } from 'next/headers';
import { verifyToken } from '@/lib/auth';
import { runCommand, runHelper, shellQuote } from '@/lib/exec';
import { audit, clientIp } from '@/lib/security';
import { canManageSite } from '@/lib/authz';
import fs from 'fs';
import path from 'path';

async function auth() {
  const cookieStore = await cookies();
  const token = cookieStore.get('panel_token')?.value;
  return token ? verifyToken(token) : null;
}

const WEB_ROOT = process.env.WEB_ROOT || '/var/www';
const NGINX_AVAILABLE = '/etc/nginx/sites-available';

function nginxVhostTemplate(domain: string, docRoot: string, phpVersion: string, ssl: boolean, cache = false): string {
  const sslBlock = ssl ? `
    listen 443 ssl http2;
    ssl_certificate     /etc/letsencrypt/live/${domain}/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/${domain}/privkey.pem;
    ssl_protocols       TLSv1.2 TLSv1.3;
    ssl_ciphers         HIGH:!aNULL:!MD5;` : '';
  const cachePhp = cache ? `
        fastcgi_cache HOSTQ_FASTCGI;
        fastcgi_cache_valid 200 301 302 10m;
        add_header X-hostQ-Cache $upstream_cache_status always;` : '';

  return `# hostQ managed - ${domain}
# hostQ fastcgi cache: ${cache ? 'on' : 'off'}
server {
    listen 80;${sslBlock}
    server_name ${domain} www.${domain};
    root ${docRoot};
    index index.php index.html index.htm;

    access_log /var/log/nginx/${domain}.access.log;
    error_log  /var/log/nginx/${domain}.error.log;

    location / {
        try_files \\$uri \\$uri/ /index.php?\\$query_string;
    }

    location ~ \\.php$ {
        include snippets/fastcgi-php.conf;
        fastcgi_pass unix:/run/php/php${phpVersion}-fpm.sock;
        fastcgi_param SCRIPT_FILENAME \\$realpath_root\\$fastcgi_script_name;
        include fastcgi_params;${cachePhp}
    }

    location ~ /\\.ht {
        deny all;
    }
}${ssl ? `

server {
    listen 80;
    server_name ${domain} www.${domain};
    return 301 https://\\$host\\$request_uri;
}` : ''}
`;
}

function letsEncryptCertExists(domain: string) {
  return fs.existsSync(`/etc/letsencrypt/live/${domain}/fullchain.pem`) &&
    fs.existsSync(`/etc/letsencrypt/live/${domain}/privkey.pem`);
}

async function removeBrokenNginxSslBlock(domain: string) {
  if (process.platform !== 'linux' || letsEncryptCertExists(domain)) return '';
  const configPath = path.join(NGINX_AVAILABLE, domain);
  if (!fs.existsSync(configPath)) return '';

  const content = fs.readFileSync(configPath, 'utf8');
  if (!content.includes('/etc/letsencrypt/live/')) return '';

  const docRoot = content.match(/root (.+);/)?.[1]?.trim() || `${WEB_ROOT}/${domain}/htdocs`;
  const phpVersion = content.match(/php(\d\.\d)-fpm/)?.[1] || '8.4';
  const cache = content.includes('hostQ fastcgi cache: on');
  const backupPath = `${configPath}.broken-ssl-${Date.now()}.bak`;
  fs.copyFileSync(configPath, backupPath);
  fs.writeFileSync(configPath, nginxVhostTemplate(domain, docRoot, phpVersion, false, cache));
  const reload = await runHelper('web.reload', { server: 'nginx' });
  return reload.success
    ? `Removed stale SSL references from ${configPath}. Backup: ${backupPath}\n`
    : `Tried removing stale SSL references, but Nginx reload still failed: ${reload.stderr || reload.error || reload.stdout}\n`;
}

// GET - list all SSL certs
export async function GET() {
  if (!await auth()) return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });

  // Check if certbot is available
  const certbotCheck = await runCommand('which certbot 2>/dev/null || certbot --version 2>&1');
  const hasCertbot = certbotCheck.success || certbotCheck.stdout.includes('certbot');

  if (!hasCertbot) {
    return NextResponse.json({
      certificates: [
        { domain: 'example.com', expiry: '2025-07-15', daysLeft: 88, status: 'valid', issuer: "Let's Encrypt" },
        { domain: 'shop.example.com', expiry: '2025-06-01', daysLeft: 44, status: 'expiring', issuer: "Let's Encrypt" },
        { domain: 'blog.example.com', expiry: '2025-04-20', daysLeft: 2, status: 'critical', issuer: "Let's Encrypt" },
      ],
      demo: true,
    });
  }

  const r = await runCommand('certbot certificates 2>/dev/null');
  const certs: { domain: string; expiry: string; daysLeft: number; status: string; issuer: string }[] = [];

  if (r.success && r.stdout) {
    const blocks = r.stdout.split('Certificate Name:').slice(1);
    for (const block of blocks) {
      const domain = block.match(/Domains: (.+)/)?.[1]?.split(' ')[0] || '';
      const expiry = block.match(/Expiry Date: (.+?) /)?.[1] || '';
      const daysLeft = parseInt(block.match(/\((\d+) days/)?.[1] || '0');
      const status = daysLeft < 7 ? 'critical' : daysLeft < 30 ? 'expiring' : 'valid';
      if (domain) certs.push({ domain, expiry, daysLeft, status, issuer: "Let's Encrypt" });
    }
  }

  return NextResponse.json({ certificates: certs, demo: false });
}

// POST - install SSL for a domain
export async function POST(request: Request) {
  const actor = await auth();
  if (!actor) return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });

  const { domain, email, webserver = 'nginx', staging = false, mode = 'letsencrypt', certificate, privateKey, chain } = await request.json();
  if (!domain) return NextResponse.json({ error: 'Domain required' }, { status: 400 });
  if (!/^[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$/.test(domain)) {
    return NextResponse.json({ error: 'Invalid domain format' }, { status: 400 });
  }
  if (!canManageSite(actor, domain, 'danger')) return NextResponse.json({ error: 'Forbidden' }, { status: 403 });

  if (mode === 'manual') {
    if (!certificate || !privateKey) {
      return NextResponse.json({ error: 'Certificate and private key required' }, { status: 400 });
    }
    if (!String(certificate).includes('BEGIN CERTIFICATE') || !String(privateKey).includes('BEGIN')) {
      return NextResponse.json({ error: 'Invalid PEM content' }, { status: 400 });
    }

    const certDir = `/etc/ssl/hostq/${domain}`;
    const fullchainPath = path.join(certDir, 'fullchain.pem');
    const keyPath = path.join(certDir, 'privkey.pem');
    const combinedCert = chain ? `${certificate.trim()}\n${String(chain).trim()}\n` : `${certificate.trim()}\n`;

    if (process.platform !== 'linux') {
      audit({ actor: actor.username, action: 'ssl.manual_upload', target: domain, ip: clientIp(request), details: { demo: true } });
      return NextResponse.json({
        success: true,
        output: `Demo mode\nWould write certificate to ${fullchainPath}\nWould reload ${webserver}`,
        message: `Manual SSL uploaded for ${domain} (demo)`,
      });
    }

    fs.mkdirSync(certDir, { recursive: true, mode: 0o700 });
    fs.writeFileSync(fullchainPath, combinedCert, { mode: 0o640 });
    fs.writeFileSync(keyPath, `${String(privateKey).trim()}\n`, { mode: 0o600 });

    const reload = await runHelper('web.reload', { server: webserver === 'apache' ? 'apache' : 'nginx' });
    audit({ actor: actor.username, action: 'ssl.manual_upload', target: domain, status: reload.success ? 'success' : 'failure', ip: clientIp(request) });
    return NextResponse.json({
      success: reload.success,
      output: reload.stdout || reload.stderr || `Certificate stored in ${certDir}`,
      message: reload.success ? `Manual SSL uploaded for ${domain}` : `Certificate stored, reload failed: ${reload.error}`,
      paths: { fullchainPath, keyPath },
    });
  }

  if (!email) return NextResponse.json({ error: "Email required for Let's Encrypt" }, { status: 400 });
  const stagingFlag = staging ? '--staging' : '';
  const plugin = ['nginx', 'apache', 'standalone'].includes(webserver) ? webserver : 'nginx';
  const repairLog = plugin === 'nginx' ? await removeBrokenNginxSslBlock(domain) : '';
  const cmd = `certbot --${plugin} -d ${shellQuote(domain)} --email ${shellQuote(email)} --agree-tos --non-interactive ${stagingFlag} 2>&1`;
  
  const r = await runCommand(cmd, 120000);
  audit({ actor: actor.username, action: 'ssl.letsencrypt_issue', target: domain, status: r.success ? 'success' : 'failure', ip: clientIp(request), details: { webserver, staging } });
  
  return NextResponse.json({
    success: r.success,
    output: `${repairLog}${r.stdout || r.stderr}`,
    message: r.success ? `SSL installed for ${domain}` : `Failed: ${r.error}`,
  });
}

// PATCH - renew a certificate
export async function PATCH(request: Request) {
  const actor = await auth();
  if (!actor) return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });

  const { domain } = await request.json();
  if (domain && !canManageSite(actor, domain, 'danger')) return NextResponse.json({ error: 'Forbidden' }, { status: 403 });
  const cmd = domain && /^[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$/.test(domain)
    ? `certbot renew --cert-name ${shellQuote(domain)} --non-interactive 2>&1`
    : 'certbot renew --non-interactive 2>&1';
  
  const r = await runCommand(cmd, 120000);
  audit({ actor: actor.username, action: 'ssl.renew', target: domain || 'all', status: r.success ? 'success' : 'failure', ip: clientIp(request) });
  return NextResponse.json({
    success: r.success,
    output: r.stdout || r.stderr,
    message: r.success ? 'Certificate renewed' : `Renewal failed: ${r.error}`,
  });
}

// DELETE - revoke/delete certificate
export async function DELETE(request: Request) {
  const actor = await auth();
  if (!actor) return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });

  const { domain } = await request.json();
  if (!domain) return NextResponse.json({ error: 'Domain required' }, { status: 400 });

  if (!/^[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$/.test(domain)) {
    return NextResponse.json({ error: 'Invalid domain format' }, { status: 400 });
  }
  if (!canManageSite(actor, domain, 'danger')) return NextResponse.json({ error: 'Forbidden' }, { status: 403 });

  const r = await runCommand(`certbot delete --cert-name ${shellQuote(domain)} --non-interactive 2>&1`, 30000);
  audit({ actor: actor.username, action: 'ssl.delete', target: domain, status: r.success ? 'success' : 'failure', ip: clientIp(request) });
  return NextResponse.json({
    success: r.success,
    output: r.stdout || r.stderr,
    message: r.success ? `Certificate for ${domain} deleted` : `Failed: ${r.error}`,
  });
}
