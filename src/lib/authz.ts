import path from 'path';
import type { JWTPayload } from './auth';
import { findAccount } from './auth';

export type SiteAction = 'view' | 'files' | 'backup' | 'write' | 'danger' | 'admin';

const ROLE_LEVEL: Record<string, number> = {
  viewer: 1,
  developer: 2,
  owner: 3,
};

const ACTION_LEVEL: Record<SiteAction, number> = {
  view: 1,
  files: 2,
  backup: 2,
  write: 2,
  danger: 3,
  admin: 3,
};

export function canManagePanel(actor: JWTPayload | null): boolean {
  return actor?.role === 'admin';
}

export function canManageSite(actor: JWTPayload | null, domain: string, action: SiteAction): boolean {
  if (!actor) return false;
  if (actor.role === 'admin') return true;
  const account = findAccount(actor.username);
  const role = account?.sitePermissions?.[domain];
  if (!role) return false;
  return ROLE_LEVEL[role] >= ACTION_LEVEL[action];
}

export function domainFromWebPath(filePath: string, webRoot = process.env.WEB_ROOT || '/var/www'): string | null {
  const normalized = path.resolve(filePath).replace(/\\/g, '/');
  const root = path.resolve(webRoot).replace(/\\/g, '/').replace(/\/html$/, '');
  const rel = normalized.startsWith(root) ? normalized.slice(root.length).replace(/^\/+/, '') : '';
  const first = rel.split('/')[0];
  return first && first.includes('.') ? first : null;
}

export function forbidden() {
  return Response.json({ error: 'Forbidden' }, { status: 403 });
}
