import { NextRequest, NextResponse } from 'next/server';
import { cookies } from 'next/headers';
import { verifyToken } from '@/lib/auth';
import { findAccount } from '@/lib/auth';
import { audit, clientIp } from '@/lib/security';
import { canManageSite, domainFromWebPath } from '@/lib/authz';
import fs from 'fs';
import path from 'path';
import { promisify } from 'util';

export const runtime = 'nodejs';
export const dynamic = 'force-dynamic';

const stat   = promisify(fs.stat);
const readdir = promisify(fs.readdir);
const mkdir  = promisify(fs.mkdir);
const rename = promisify(fs.rename);
const readFile  = promisify(fs.readFile);
const writeFile = promisify(fs.writeFile);

async function auth() {
  const cookieStore = await cookies();
  const token = cookieStore.get('panel_token')?.value;
  return token ? verifyToken(token) : null;
}

const ROOT = path.resolve(/*turbopackIgnore: true*/ process.platform === 'linux' ? '/var/www' : (process.env.FILE_MANAGER_ROOT || path.join(/*turbopackIgnore: true*/ process.cwd(), '.hostq-www')));
const TRASH = path.join(ROOT, '.hostq-trash');
const BLOCKED_NAMES = new Set(['.env', '.env.local', '.env.production', 'id_rsa', 'id_dsa', 'authorized_keys']);
const BLOCKED_EXTS = new Set(['.key', '.pem', '.p12', '.pfx', '.crt', '.csr']);

function safePath(reqPath: string): string {
  const normalized = path.normalize(reqPath || '/').replace(/^(\.\.(\/|\\|$))+/, '');
  const full = path.resolve(/*turbopackIgnore: true*/ ROOT, normalized.replace(/^[/\\]+/, ''));
  if (full !== ROOT && !full.startsWith(ROOT + path.sep)) return ROOT;
  return full;
}

function safeChildName(name: string): string | null {
  if (!name || name.includes('/') || name.includes('\\') || name === '.' || name === '..') return null;
  const sanitized = name
    .replace(/[^\w.\- ]+/g, '-')
    .replace(/\s+/g, '-')
    .replace(/-+/g, '-')
    .slice(0, 180);
  if (!sanitized || sanitized === '.' || sanitized === '..' || sanitized.startsWith('.env')) return null;
  return sanitized;
}

function blockedPath(full: string): boolean {
  if (full === ROOT) return false;
  const rel = path.relative(ROOT, full);
  const parts = rel.split(path.sep).filter(Boolean);
  if (parts.includes('.hostq-trash')) return true;
  const base = path.basename(full).toLowerCase();
  return BLOCKED_NAMES.has(base) || BLOCKED_EXTS.has(path.extname(base));
}

function blockedResponse() {
  return NextResponse.json({ error: 'Protected file type is blocked by default' }, { status: 403 });
}

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / 1024 / 1024).toFixed(1)} MB`;
  return `${(bytes / 1024 / 1024 / 1024).toFixed(2)} GB`;
}

// GET - list directory or read file
export async function GET(request: NextRequest) {
  const actor = await auth();
  if (!actor) return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });

  if (!fs.existsSync(ROOT)) await mkdir(ROOT, { recursive: true, mode: 0o755 });
  const { searchParams } = request.nextUrl;
  const reqPath = searchParams.get('path') || '/';
  const action  = searchParams.get('action') || 'list';
  const full    = safePath(reqPath);
  if (blockedPath(full)) return blockedResponse();
  const domain = domainFromWebPath(full, ROOT);
  if (domain && !canManageSite(actor, domain, 'view')) return NextResponse.json({ error: 'Forbidden' }, { status: 403 });

  try {
    const s = await stat(full);

    if (action === 'read' && s.isFile()) {
      const ext = path.extname(full).toLowerCase();
      const textExts = ['.txt','.html','.css','.js','.ts','.tsx','.jsx','.json','.xml','.php','.py','.sh','.md','.conf','.nginx','.htaccess','.log','.sql','.yaml','.yml','.ini','.cfg'];
      if (textExts.includes(ext) || ext === '') {
        const content = await readFile(full, 'utf8');
        return NextResponse.json({ content, path: reqPath, name: path.basename(full) });
      }
      return NextResponse.json({ error: 'Binary file - cannot read as text' }, { status: 415 });
    }

    if (s.isDirectory()) {
      const entries = await readdir(full, { withFileTypes: true });
      const items = await Promise.all(entries.map(async (entry) => {
        try {
          const entryFull = path.join(full, entry.name);
          if (blockedPath(entryFull)) return null;
          if (full === ROOT && actor.role !== 'admin') {
            const account = findAccount(actor.username);
            if (!account?.sitePermissions?.[entry.name]) return null;
          }
          const entrystat = await stat(entryFull);
          return {
            name: entry.name,
            type: entry.isDirectory() ? 'dir' : 'file',
            size: entry.isFile() ? formatSize(entrystat.size) : '',
            sizeBytes: entry.isFile() ? entrystat.size : 0,
            modified: entrystat.mtime.toISOString(),
            ext: path.extname(entry.name).toLowerCase().replace('.', ''),
          };
        } catch { return null; }
      }));

      const sorted = items.filter((item): item is NonNullable<typeof item> => item !== null).sort((a, b) => {
        if (a.type !== b.type) return a.type === 'dir' ? -1 : 1;
        return a.name.localeCompare(b.name);
      });

      // Build breadcrumb
      const relPath     = path.relative(ROOT, full).replace(/\\/g, '/');
      const parts       = relPath ? relPath.split('/') : [];
      const breadcrumbs = [{ name: 'Root', path: '/' }];
      let acc           = '';
      for (const part of parts) {
        acc += '/' + part;
        breadcrumbs.push({ name: part, path: acc });
      }

      return NextResponse.json({ items: sorted, path: reqPath, breadcrumbs });
    }

    return NextResponse.json({ error: 'Not found' }, { status: 404 });
  } catch (err) {
    return NextResponse.json({ error: String(err) }, { status: 500 });
  }
}

// POST - create dir, write file
export async function POST(request: NextRequest) {
  const actor = await auth();
  if (!actor) return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });

  const contentType = request.headers.get('content-type') || '';
  if (contentType.includes('multipart/form-data')) {
    const form = await request.formData();
    const reqPath = String(form.get('path') || '/');
    const full = safePath(reqPath);
    const domain = domainFromWebPath(full, ROOT);
    if (domain && !canManageSite(actor, domain, 'files')) return NextResponse.json({ error: 'Forbidden' }, { status: 403 });
    const file = form.get('file');
    if (!(file instanceof File)) return NextResponse.json({ error: 'No file uploaded' }, { status: 400 });
    const name = safeChildName(file.name);
    if (!name) return NextResponse.json({ error: 'Invalid file name' }, { status: 400 });
    const target = path.join(full, name);
    if (blockedPath(full) || blockedPath(target)) return blockedResponse();
    const bytes = Buffer.from(await file.arrayBuffer());
    await writeFile(target, bytes);
    audit({ actor: actor.username, action: 'file.upload', target, ip: clientIp(request) });
    return NextResponse.json({ success: true, message: `Uploaded '${name}'` });
  }

  const body = await request.json();
  const { action, path: reqPath, name, content } = body;
  const full  = safePath(reqPath);
  if (blockedPath(full)) return blockedResponse();
  const domain = domainFromWebPath(full, ROOT);
  if (domain && !canManageSite(actor, domain, 'files')) return NextResponse.json({ error: 'Forbidden' }, { status: 403 });

  try {
    if (action === 'mkdir') {
      const childName = safeChildName(name);
      if (!childName) return NextResponse.json({ error: 'Invalid folder name' }, { status: 400 });
      const target = path.join(full, childName);
      if (blockedPath(target)) return blockedResponse();
      await mkdir(target, { recursive: true });
      audit({ actor: actor.username, action: 'file.mkdir', target, ip: clientIp(request) });
      return NextResponse.json({ success: true, message: `Folder '${childName}' created` });
    }

    if (action === 'create_file') {
      const childName = safeChildName(name);
      if (!childName) return NextResponse.json({ error: 'Invalid file name' }, { status: 400 });
      const target = path.join(full, childName);
      if (blockedPath(target)) return blockedResponse();
      await writeFile(target, content || '', 'utf8');
      audit({ actor: actor.username, action: 'file.create', target, ip: clientIp(request) });
      return NextResponse.json({ success: true, message: `File '${childName}' created` });
    }

    if (action === 'save') {
      await writeFile(full, content, 'utf8');
      audit({ actor: actor.username, action: 'file.save', target: full, ip: clientIp(request) });
      return NextResponse.json({ success: true, message: 'File saved' });
    }

    return NextResponse.json({ error: 'Unknown action' }, { status: 400 });
  } catch (err) {
    return NextResponse.json({ error: String(err) }, { status: 500 });
  }
}

// PATCH - rename
export async function PATCH(request: NextRequest) {
  const actor = await auth();
  if (!actor) return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });

  const { path: reqPath, newName } = await request.json();
  const full    = safePath(reqPath);
  const domain = domainFromWebPath(full, ROOT);
  if (domain && !canManageSite(actor, domain, 'files')) return NextResponse.json({ error: 'Forbidden' }, { status: 403 });
  const childName = safeChildName(newName);
  if (!childName) return NextResponse.json({ error: 'Invalid file name' }, { status: 400 });
  const newFull = path.join(path.dirname(full), childName);
  if (blockedPath(full) || blockedPath(newFull)) return blockedResponse();

  try {
    await rename(full, newFull);
    audit({ actor: actor.username, action: 'file.rename', target: full, ip: clientIp(request), details: { newFull } });
    return NextResponse.json({ success: true, message: `Renamed to '${newName}'` });
  } catch (err) {
    return NextResponse.json({ error: String(err) }, { status: 500 });
  }
}

// DELETE - delete file or folder
export async function DELETE(request: NextRequest) {
  const actor = await auth();
  if (!actor) return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });

  const { path: reqPath } = await request.json();
  const full = safePath(reqPath);
  if (full === ROOT) return NextResponse.json({ error: 'Cannot delete root' }, { status: 400 });
  if (blockedPath(full)) return blockedResponse();
  const domain = domainFromWebPath(full, ROOT);
  if (domain && !canManageSite(actor, domain, 'danger')) return NextResponse.json({ error: 'Forbidden' }, { status: 403 });

  try {
    await mkdir(TRASH, { recursive: true, mode: 0o700 });
    const destination = path.join(TRASH, `${Date.now()}-${path.basename(full)}`);
    await rename(full, destination);
    audit({ actor: actor.username, action: 'file.soft_delete', target: full, ip: clientIp(request), details: { destination } });
    return NextResponse.json({ success: true, message: 'Moved to hostQ trash' });
  } catch (err) {
    return NextResponse.json({ error: String(err) }, { status: 500 });
  }
}
