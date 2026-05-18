import { NextRequest, NextResponse } from 'next/server';
import { cookies } from 'next/headers';
import { verifyToken } from '@/lib/auth';
import { runCommand, shellQuote } from '@/lib/exec';
import fs from 'fs';
import path from 'path';

async function auth() {
  const cookieStore = await cookies();
  const token = cookieStore.get('panel_token')?.value;
  return token ? verifyToken(token) : null;
}

const WEB_ROOT = process.env.WEB_ROOT || '/var/www';
const NGINX_AVAILABLE = '/etc/nginx/sites-available';
const NGINX_ENABLED   = '/etc/nginx/sites-enabled';
const APACHE_AVAILABLE = '/etc/apache2/sites-available';
const SUPPORTED_SITE_TYPES = ['html', 'php', 'wordpress'] as const;

// ──────────────────────────────────────────────
// Nginx vhost template for a domain
// ──────────────────────────────────────────────
function nginxVhostTemplate(domain: string, docRoot: string, phpVersion: string, ssl: boolean): string {
  const sslBlock = ssl ? `
    listen 443 ssl http2;
    ssl_certificate     /etc/letsencrypt/live/${domain}/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/${domain}/privkey.pem;
    ssl_protocols       TLSv1.2 TLSv1.3;
    ssl_ciphers         HIGH:!aNULL:!MD5;` : '';

  return `# HostPanel managed - ${domain}
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
        include fastcgi_params;
    }

    location ~ /\\.ht {
        deny all;
    }

    location ~* \\.(jpg|jpeg|png|gif|ico|css|js|woff2?)$ {
        expires 30d;
        add_header Cache-Control "public, no-transform";
    }
}${ssl ? `

server {
    listen 80;
    server_name ${domain} www.${domain};
    return 301 https://\\$host\\$request_uri;
}` : ''}
`;
}

// ──────────────────────────────────────────────
// Apache vhost template
// ──────────────────────────────────────────────
function apacheVhostTemplate(domain: string, docRoot: string): string {
  return `# HostPanel managed - ${domain}
<VirtualHost *:80>
    ServerName ${domain}
    ServerAlias www.${domain}
    DocumentRoot ${docRoot}

    <Directory ${docRoot}>
        Options -Indexes +FollowSymLinks
        AllowOverride All
        Require all granted
    </Directory>

    ErrorLog \${APACHE_LOG_DIR}/${domain}.error.log
    CustomLog \${APACHE_LOG_DIR}/${domain}.access.log combined

    <FilesMatch \\.php$>
        SetHandler "proxy:unix:/run/php/php8.2-fpm.sock|fcgi://localhost"
    </FilesMatch>
</VirtualHost>
`;
}

// ──────────────────────────────────────────────
// Detect active web server
// ──────────────────────────────────────────────
async function detectWebServer(): Promise<'nginx' | 'apache' | 'none'> {
  const ng = await runCommand('systemctl is-active nginx 2>/dev/null');
  if (ng.stdout.trim() === 'active') return 'nginx';
  const ap = await runCommand('systemctl is-active apache2 2>/dev/null');
  if (ap.stdout.trim() === 'active') return 'apache';
  return 'none';
}

// ──────────────────────────────────────────────
// List all managed domains from config files
// ──────────────────────────────────────────────
async function listDomains(): Promise<{ domain: string; type: 'domain'|'subdomain'; docRoot: string; enabled: boolean; server: string; ssl: boolean }[]> {
  const domains: { domain: string; type: 'domain'|'subdomain'; docRoot: string; enabled: boolean; server: string; ssl: boolean }[] = [];

  // Check if on Linux (no real configs on Windows dev)
  if (process.platform !== 'linux') {
    return [
      { domain: 'example.com',      type: 'domain',    docRoot: '/var/www/example.com/public_html', enabled: true,  server: 'nginx', ssl: true  },
      { domain: 'shop.example.com', type: 'subdomain', docRoot: '/var/www/example.com/public_html/shop', enabled: true, server: 'nginx', ssl: true },
      { domain: 'blog.mysite.com',  type: 'subdomain', docRoot: '/var/www/mysite.com/public_html/blog', enabled: false, server: 'nginx', ssl: false },
      { domain: 'mysite.com',       type: 'domain',    docRoot: '/var/www/mysite.com/public_html', enabled: true, server: 'nginx', ssl: false },
    ];
  }

  // Read Nginx sites
  try {
    const available = fs.readdirSync(NGINX_AVAILABLE);
    for (const file of available) {
      try {
        const content = fs.readFileSync(path.join(NGINX_AVAILABLE, file), 'utf8');
        if (!content.includes('HostPanel managed')) continue;
        const domainMatch = content.match(/# HostPanel managed - (.+)/);
        const rootMatch   = content.match(/root (.+);/);
        const sslMatch    = content.includes('ssl_certificate');
        const enabled     = fs.existsSync(path.join(NGINX_ENABLED, file));
        if (domainMatch && rootMatch) {
          const domainName = domainMatch[1].trim();
          const isSubdomain = (domainName.match(/\./g) || []).length >= 2;
          domains.push({
            domain: domainName,
            type: isSubdomain ? 'subdomain' : 'domain',
            docRoot: rootMatch[1].trim(),
            enabled,
            server: 'nginx',
            ssl: sslMatch,
          });
        }
      } catch { /* skip */ }
    }
  } catch { /* nginx not configured */ }

  return domains;
}

// ──────────────────────────────────────────────
// GET - list domains
// ──────────────────────────────────────────────
export async function GET() {
  if (!await auth()) return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });

  const domains  = await listDomains();
  const webserver = await detectWebServer();
  const demo     = process.platform !== 'linux';

  return NextResponse.json({ domains, webserver, demo });
}

// ──────────────────────────────────────────────
// POST - add domain or subdomain
// ──────────────────────────────────────────────
export async function POST(request: NextRequest) {
  if (!await auth()) return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });

  const { domain, type = 'domain', phpVersion = '8.4', server = 'nginx', parentDomain = '', siteType = 'php' } = await request.json();

  if (!domain || !/^[a-zA-Z0-9][a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$/.test(domain)) {
    return NextResponse.json({ error: 'Invalid domain name' }, { status: 400 });
  }
  if (!/^(8\.2|8\.3|8\.4|8\.5)$/.test(phpVersion)) {
    return NextResponse.json({ error: 'Unsupported PHP version' }, { status: 400 });
  }
  if (!SUPPORTED_SITE_TYPES.includes(siteType)) {
    return NextResponse.json({ error: 'Unsupported site type' }, { status: 400 });
  }

  const docRoot = type === 'subdomain' && parentDomain
    ? `${WEB_ROOT}/${parentDomain}/public_html/${domain.split('.')[0]}`
    : `${WEB_ROOT}/${domain}/public_html`;

  const logs: string[] = [];

  if (process.platform !== 'linux') {
    return NextResponse.json({
      success: true,
      message: `Domain ${domain} configured (demo mode - no actual files created on Windows)`,
      docRoot,
      output: `▶ Demo mode\n✓ Would create: ${docRoot}\n✓ Would write vhost to: ${NGINX_AVAILABLE}/${domain}\n✓ Would enable site and reload Nginx\n✓ Done!`,
    });
  }

  // 1. Create document root
  logs.push(`▶ Creating document root: ${docRoot}`);
  const mkdirR = await runCommand(`mkdir -p ${shellQuote(docRoot)}`);
  logs.push(mkdirR.success ? `✓ Directory created` : `✗ ${mkdirR.error}`);

  // 2. Create starter file
  const indexHtml = siteType === 'php' ? `<?php
?><!DOCTYPE html>
<html><head><title>Welcome to ${domain}</title>
<style>body{font-family:system-ui,sans-serif;text-align:center;padding:60px;background:#0a0c10;color:#e6edf3}</style>
</head><body>
<h1>${domain}</h1>
<p>Your PHP website is ready. Upload your files to ${docRoot}</p>
<p style="color:#8b949e;font-size:13px">PHP <?php echo PHP_VERSION; ?> - Managed by HostPanel</p>
</body></html>` : `<!DOCTYPE html>
<html><head><title>Welcome to ${domain}</title>
<style>body{font-family:system-ui,sans-serif;text-align:center;padding:60px;background:#0a0c10;color:#e6edf3}</style>
</head><body>
<h1>${domain}</h1>
<p>Your website is ready. Upload your files to ${docRoot}</p>
<p style="color:#8b949e;font-size:13px">Managed by HostPanel</p>
</body></html>`;
  const starterFile = siteType === 'php' ? 'index.php' : 'index.html';
  fs.writeFileSync(path.join(docRoot, starterFile), indexHtml);
  logs.push(`✓ Default ${starterFile} created`);

  // 3. Set permissions
  await runCommand(`chown -R www-data:www-data ${shellQuote(`${WEB_ROOT}/${domain.split('.').slice(-2).join('.')}`)} 2>/dev/null || true`);
  await runCommand(`chmod -R 755 ${shellQuote(docRoot)}`);
  logs.push(`✓ Permissions set`);

  if (server === 'nginx') {
    // 4. Write Nginx vhost
    const configPath = path.join(NGINX_AVAILABLE, domain);
    const vhostContent = nginxVhostTemplate(domain, docRoot, phpVersion, false);
    fs.writeFileSync(configPath, vhostContent);
    logs.push(`✓ Nginx vhost created: ${configPath}`);

    // 5. Enable site
    const enableR = await runCommand(`ln -sf ${shellQuote(configPath)} ${shellQuote(path.join(NGINX_ENABLED, domain))}`);
    logs.push(enableR.success ? `✓ Site enabled` : `✗ ${enableR.error}`);

    // 6. Test & reload
    const testR = await runCommand('nginx -t 2>&1');
    logs.push(testR.success ? `✓ Nginx config valid` : `✗ Config test failed: ${testR.stderr}`);
    if (testR.success) {
      await runCommand('systemctl reload nginx');
      logs.push(`✓ Nginx reloaded`);
    }
  } else if (server === 'apache') {
    const configPath = path.join(APACHE_AVAILABLE, `${domain}.conf`);
    const vhostContent = apacheVhostTemplate(domain, docRoot);
    fs.writeFileSync(configPath, vhostContent);
    logs.push(`✓ Apache vhost created: ${configPath}`);
    await runCommand(`a2ensite ${shellQuote(`${domain}.conf`)} 2>&1`);
    await runCommand('systemctl reload apache2 2>&1');
    logs.push(`✓ Apache reloaded`);
  }

  return NextResponse.json({
    success: true,
    message: `Domain ${domain} added successfully`,
    docRoot,
    output: logs.join('\n'),
  });
}

// ──────────────────────────────────────────────
// PATCH - enable/disable domain
// ──────────────────────────────────────────────
export async function PATCH(request: NextRequest) {
  if (!await auth()) return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });

  const { domain, action, server = 'nginx' } = await request.json();
  if (!domain) return NextResponse.json({ error: 'Domain required' }, { status: 400 });
  if (!/^[a-zA-Z0-9][a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$/.test(domain)) {
    return NextResponse.json({ error: 'Invalid domain name' }, { status: 400 });
  }

  if (process.platform !== 'linux') {
    return NextResponse.json({ success: true, message: `${domain} ${action}d (demo)` });
  }

  if (server === 'nginx') {
    if (action === 'enable') {
      await runCommand(`ln -sf ${shellQuote(path.join(NGINX_AVAILABLE, domain))} ${shellQuote(path.join(NGINX_ENABLED, domain))}`);
      await runCommand('systemctl reload nginx');
    } else {
      await runCommand(`rm -f ${shellQuote(path.join(NGINX_ENABLED, domain))}`);
      await runCommand('systemctl reload nginx');
    }
  } else {
    if (action === 'enable') {
      await runCommand(`a2ensite ${shellQuote(`${domain}.conf`)}`);
    } else {
      await runCommand(`a2dissite ${shellQuote(`${domain}.conf`)}`);
    }
    await runCommand('systemctl reload apache2');
  }

  return NextResponse.json({ success: true, message: `${domain} ${action}d` });
}

// ──────────────────────────────────────────────
// DELETE - remove domain
// ──────────────────────────────────────────────
export async function DELETE(request: NextRequest) {
  if (!await auth()) return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });

  const { domain, deleteFiles = false, server = 'nginx' } = await request.json();
  if (!domain) return NextResponse.json({ error: 'Domain required' }, { status: 400 });
  if (!/^[a-zA-Z0-9][a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$/.test(domain)) {
    return NextResponse.json({ error: 'Invalid domain name' }, { status: 400 });
  }

  if (process.platform !== 'linux') {
    return NextResponse.json({ success: true, message: `${domain} deleted (demo)` });
  }

  const logs: string[] = [];

  if (server === 'nginx') {
    await runCommand(`rm -f ${shellQuote(path.join(NGINX_ENABLED, domain))}`);
    await runCommand(`rm -f ${shellQuote(path.join(NGINX_AVAILABLE, domain))}`);
    await runCommand('systemctl reload nginx');
    logs.push(`✓ Nginx vhost removed`);
  } else {
    await runCommand(`a2dissite ${shellQuote(`${domain}.conf`)}`);
    await runCommand(`rm -f ${shellQuote(path.join(APACHE_AVAILABLE, domain + '.conf'))}`);
    await runCommand('systemctl reload apache2');
    logs.push(`✓ Apache vhost removed`);
  }

  if (deleteFiles) {
    const domainRoot = `${WEB_ROOT}/${domain}`;
    await runCommand(`rm -rf ${shellQuote(domainRoot)}`);
    logs.push(`✓ Files deleted: ${domainRoot}`);
  }

  return NextResponse.json({ success: true, message: `Domain ${domain} removed`, output: logs.join('\n') });
}
