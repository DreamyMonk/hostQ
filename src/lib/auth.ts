// lib/auth.ts
import jwt from 'jsonwebtoken';
import bcrypt from 'bcryptjs';
import fs from 'fs';
import path from 'path';

const JWT_SECRET = process.env.JWT_SECRET || 'fallback-secret';
const JWT_EXPIRY = process.env.JWT_EXPIRY || '24h';
const LEGACY_PASSWORD_PLACEHOLDERS = new Set(['', 'your_secure_password_here', 'changeme', 'changeme123']);

export interface JWTPayload {
  username: string;
  iat?: number;
  exp?: number;
}

export function signToken(username: string): string {
  return jwt.sign({ username }, JWT_SECRET, { expiresIn: JWT_EXPIRY } as jwt.SignOptions);
}

export function verifyToken(token: string): JWTPayload | null {
  try {
    return jwt.verify(token, JWT_SECRET) as JWTPayload;
  } catch {
    return null;
  }
}

interface AdminAccount {
  username: string;
  passwordHash: string;
  createdAt: string;
}

function dataDir(): string {
  if (process.env.HOSTQ_DATA_DIR) return process.env.HOSTQ_DATA_DIR;
  if (process.platform === 'linux') return '/etc/hostq';
  return path.join(process.cwd(), '.hostq');
}

function accountPath(): string {
  return path.join(dataDir(), 'admin.json');
}

function readAccount(): AdminAccount | null {
  try {
    return JSON.parse(fs.readFileSync(accountPath(), 'utf8')) as AdminAccount;
  } catch {
    return null;
  }
}

function legacyEnvAccount(): AdminAccount | null {
  const username = process.env.PANEL_USERNAME || '';
  const password = process.env.PANEL_PASSWORD || '';
  if (!username || LEGACY_PASSWORD_PLACEHOLDERS.has(password)) return null;
  return {
    username,
    passwordHash: bcrypt.hashSync(password, 10),
    createdAt: new Date().toISOString(),
  };
}

export function hasAdminAccount(): boolean {
  return Boolean(readAccount() || legacyEnvAccount());
}

export function createAdminAccount(username: string, password: string): { success: boolean; error?: string } {
  if (hasAdminAccount()) return { success: false, error: 'Admin account already exists' };
  if (!/^[a-zA-Z0-9_.-]{3,32}$/.test(username)) {
    return { success: false, error: 'Username must be 3-32 characters: letters, numbers, dot, dash, underscore' };
  }
  if (password.length < 10) {
    return { success: false, error: 'Password must be at least 10 characters' };
  }

  fs.mkdirSync(dataDir(), { recursive: true, mode: 0o700 });
  const account: AdminAccount = {
    username,
    passwordHash: bcrypt.hashSync(password, 12),
    createdAt: new Date().toISOString(),
  };
  fs.writeFileSync(accountPath(), JSON.stringify(account, null, 2), { mode: 0o600 });
  return { success: true };
}

export function validateCredentials(username: string, password: string): boolean {
  const account = readAccount();
  if (account) {
    return username === account.username && bcrypt.compareSync(password, account.passwordHash);
  }

  const legacy = legacyEnvAccount();
  if (!legacy) return false;
  return username === legacy.username && bcrypt.compareSync(password, legacy.passwordHash);
}
