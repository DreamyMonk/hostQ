import { NextResponse } from 'next/server';
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
  if (!await auth()) return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });

  const { domain, email, webserver = 'nginx', staging = false, mode = 'letsencrypt', certificate, privateKey, chain } = await request.json();
  if (!domain) return NextResponse.json({ error: 'Domain required' }, { status: 400 });
  if (!/^[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$/.test(domain)) {
    return NextResponse.json({ error: 'Invalid domain format' }, { status: 400 });
  }

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
      return NextResponse.json({
        success: true,
        output: `Demo mode\nWould write certificate to ${fullchainPath}\nWould reload ${webserver}`,
        message: `Manual SSL uploaded for ${domain} (demo)`,
      });
    }

    fs.mkdirSync(certDir, { recursive: true, mode: 0o700 });
    fs.writeFileSync(fullchainPath, combinedCert, { mode: 0o644 });
    fs.writeFileSync(keyPath, `${String(privateKey).trim()}\n`, { mode: 0o600 });

    const reload = await runCommand(`${webserver === 'apache' ? 'systemctl reload apache2' : 'nginx -t && systemctl reload nginx'} 2>&1`);
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
  const cmd = `certbot --${plugin} -d ${shellQuote(domain)} --email ${shellQuote(email)} --agree-tos --non-interactive ${stagingFlag} 2>&1`;
  
  const r = await runCommand(cmd, 120000);
  
  return NextResponse.json({
    success: r.success,
    output: r.stdout || r.stderr,
    message: r.success ? `SSL installed for ${domain}` : `Failed: ${r.error}`,
  });
}

// PATCH - renew a certificate
export async function PATCH(request: Request) {
  if (!await auth()) return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });

  const { domain } = await request.json();
  const cmd = domain && /^[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$/.test(domain)
    ? `certbot renew --cert-name ${shellQuote(domain)} --non-interactive 2>&1`
    : 'certbot renew --non-interactive 2>&1';
  
  const r = await runCommand(cmd, 120000);
  return NextResponse.json({
    success: r.success,
    output: r.stdout || r.stderr,
    message: r.success ? 'Certificate renewed' : `Renewal failed: ${r.error}`,
  });
}

// DELETE - revoke/delete certificate
export async function DELETE(request: Request) {
  if (!await auth()) return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });

  const { domain } = await request.json();
  if (!domain) return NextResponse.json({ error: 'Domain required' }, { status: 400 });

  if (!/^[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$/.test(domain)) {
    return NextResponse.json({ error: 'Invalid domain format' }, { status: 400 });
  }

  const r = await runCommand(`certbot delete --cert-name ${shellQuote(domain)} --non-interactive 2>&1`, 30000);
  return NextResponse.json({
    success: r.success,
    output: r.stdout || r.stderr,
    message: r.success ? `Certificate for ${domain} deleted` : `Failed: ${r.error}`,
  });
}
