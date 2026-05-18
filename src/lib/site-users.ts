import fs from 'fs';
import path from 'path';

export type SiteRole = 'owner' | 'developer' | 'viewer';

export interface SiteUser {
  id: string;
  domain: string;
  username: string;
  role: SiteRole;
  createdAt: string;
}

function dataDir(): string {
  if (process.env.HOSTQ_DATA_DIR) return process.env.HOSTQ_DATA_DIR;
  if (process.platform === 'linux') return '/etc/hostq';
  return path.join(process.cwd(), '.hostq');
}

function storePath(): string {
  return path.join(dataDir(), 'site-users.json');
}

function readUsers(): SiteUser[] {
  try {
    return JSON.parse(fs.readFileSync(storePath(), 'utf8')) as SiteUser[];
  } catch {
    return [];
  }
}

function writeUsers(users: SiteUser[]) {
  fs.mkdirSync(dataDir(), { recursive: true, mode: 0o700 });
  fs.writeFileSync(storePath(), JSON.stringify(users, null, 2), { mode: 0o600 });
}

export function listSiteUsers(domain?: string): SiteUser[] {
  const users = readUsers();
  return domain ? users.filter((user) => user.domain === domain) : users;
}

export function addSiteUser(domain: string, username: string, role: SiteRole): SiteUser {
  if (!/^[a-zA-Z0-9][a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$/.test(domain)) throw new Error('Invalid domain');
  if (!/^[a-zA-Z0-9_.-]{3,32}$/.test(username)) throw new Error('Invalid username');
  if (!['owner', 'developer', 'viewer'].includes(role)) throw new Error('Invalid role');

  const users = readUsers();
  if (users.some((user) => user.domain === domain && user.username === username)) {
    throw new Error('User already has access to this site');
  }
  const user: SiteUser = {
    id: `${domain}:${username}`,
    domain,
    username,
    role,
    createdAt: new Date().toISOString(),
  };
  writeUsers([...users, user]);
  return user;
}

export function removeSiteUser(domain: string, username: string): boolean {
  const users = readUsers();
  const next = users.filter((user) => !(user.domain === domain && user.username === username));
  writeUsers(next);
  return next.length !== users.length;
}
