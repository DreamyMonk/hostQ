// lib/auth.ts
import jwt from 'jsonwebtoken';
import bcrypt from 'bcryptjs';
import fs from 'fs';
import path from 'path';
import crypto from 'crypto';

const JWT_SECRET = process.env.JWT_SECRET || 'fallback-secret';
const JWT_EXPIRY = process.env.JWT_EXPIRY || '24h';
const LEGACY_PASSWORD_PLACEHOLDERS = new Set(['', 'your_secure_password_here', 'changeme', 'changeme123']);
const IDLE_TIMEOUT_MS = Number(process.env.SESSION_IDLE_TIMEOUT_MINUTES || 30) * 60 * 1000;
const BASE32 = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ234567';

function base32Encode(buffer: Buffer): string {
  let bits = '';
  for (const byte of buffer) bits += byte.toString(2).padStart(8, '0');
  return bits.match(/.{1,5}/g)?.map((chunk) => BASE32[parseInt(chunk.padEnd(5, '0'), 2)]).join('') || '';
}

function base32Decode(value: string): Buffer {
  const clean = value.replace(/=+$/g, '').replace(/\s+/g, '').toUpperCase();
  let bits = '';
  for (const char of clean) {
    const index = BASE32.indexOf(char);
    if (index < 0) throw new Error('Invalid base32');
    bits += index.toString(2).padStart(5, '0');
  }
  const bytes = bits.match(/.{8}/g)?.map((chunk) => parseInt(chunk, 2)) || [];
  return Buffer.from(bytes);
}

function totp(secret: string, offset = 0): string {
  const counter = Math.floor(Date.now() / 30000) + offset;
  const msg = Buffer.alloc(8);
  msg.writeBigUInt64BE(BigInt(counter));
  const hmac = crypto.createHmac('sha1', base32Decode(secret)).update(msg).digest();
  const index = hmac[hmac.length - 1] & 0xf;
  const code = ((hmac[index] & 0x7f) << 24) | ((hmac[index + 1] & 0xff) << 16) | ((hmac[index + 2] & 0xff) << 8) | (hmac[index + 3] & 0xff);
  return String(code % 1_000_000).padStart(6, '0');
}

export interface JWTPayload {
  username: string;
  role: 'admin' | 'user';
  sessionId: string;
  iat?: number;
  exp?: number;
}

export interface Account {
  username: string;
  passwordHash: string;
  role: 'admin' | 'user';
  sitePermissions?: Record<string, 'owner' | 'developer' | 'viewer'>;
  otpSecret?: string;
  otpEnabled?: boolean;
  createdAt: string;
}

interface SessionRecord {
  id: string;
  username: string;
  role: 'admin' | 'user';
  createdAt: string;
  lastSeenAt: string;
  userAgent?: string;
  ip?: string;
  revokedAt?: string;
}

export function signToken(username: string, role: 'admin' | 'user', sessionId: string): string {
  return jwt.sign({ username, role, sessionId }, JWT_SECRET, { expiresIn: JWT_EXPIRY } as jwt.SignOptions);
}

export function verifyToken(token: string): JWTPayload | null {
  try {
    const payload = jwt.verify(token, JWT_SECRET) as JWTPayload;
    const session = getSession(payload.sessionId);
    if (!session || session.revokedAt || session.username !== payload.username) return null;
    if (Date.now() - new Date(session.lastSeenAt).getTime() > IDLE_TIMEOUT_MS) {
      revokeSession(payload.sessionId);
      return null;
    }
    touchSession(payload.sessionId);
    return payload;
  } catch {
    return null;
  }
}

function dataDir(): string {
  if (process.env.HOSTQ_DATA_DIR) return process.env.HOSTQ_DATA_DIR;
  if (process.platform === 'linux') return '/etc/hostq';
  return path.join(process.cwd(), '.hostq');
}

function accountPath(): string {
  return path.join(dataDir(), 'admin.json');
}

function usersPath(): string {
  return path.join(dataDir(), 'users.json');
}

function sessionsPath(): string {
  return path.join(dataDir(), 'sessions.json');
}

function readAccount(): Account | null {
  try {
    const account = JSON.parse(fs.readFileSync(accountPath(), 'utf8')) as Account;
    return { ...account, role: 'admin' };
  } catch {
    return null;
  }
}

function writeAccount(account: Account) {
  fs.mkdirSync(dataDir(), { recursive: true, mode: 0o700 });
  fs.writeFileSync(accountPath(), JSON.stringify(account, null, 2), { mode: 0o600 });
}

function readUsers(): Account[] {
  try {
    return JSON.parse(fs.readFileSync(usersPath(), 'utf8')) as Account[];
  } catch {
    return [];
  }
}

function writeUsers(users: Account[]) {
  fs.mkdirSync(dataDir(), { recursive: true, mode: 0o700 });
  fs.writeFileSync(usersPath(), JSON.stringify(users, null, 2), { mode: 0o600 });
}

function legacyEnvAccount(): Account | null {
  const username = process.env.PANEL_USERNAME || '';
  const password = process.env.PANEL_PASSWORD || '';
  if (!username || LEGACY_PASSWORD_PLACEHOLDERS.has(password)) return null;
  return {
    username,
    passwordHash: bcrypt.hashSync(password, 10),
    role: 'admin',
    createdAt: new Date().toISOString(),
  };
}

export function hasAdminAccount(): boolean {
  return Boolean(readAccount() || legacyEnvAccount());
}

export function createAdminAccount(username: string, password: string): { success: boolean; error?: string; otpSecret?: string; otpAuthUrl?: string } {
  if (hasAdminAccount()) return { success: false, error: 'Admin account already exists' };
  if (!/^[a-zA-Z0-9_.-]{3,32}$/.test(username)) {
    return { success: false, error: 'Username must be 3-32 characters: letters, numbers, dot, dash, underscore' };
  }
  if (password.length < 10) {
    return { success: false, error: 'Password must be at least 10 characters' };
  }

  fs.mkdirSync(dataDir(), { recursive: true, mode: 0o700 });
  const account: Account = {
    username,
    passwordHash: bcrypt.hashSync(password, 12),
    role: 'admin',
    otpEnabled: false,
    createdAt: new Date().toISOString(),
  };
  writeAccount(account);
  return { success: true };
}

export function findAccount(username: string): Account | null {
  const admin = readAccount();
  if (admin && username === admin.username) return admin;
  const user = readUsers().find((account) => account.username === username);
  if (user) return user;
  const legacy = legacyEnvAccount();
  if (legacy && username === legacy.username) return legacy;
  return null;
}

export function validateCredentials(username: string, password: string): Account | null {
  const account = findAccount(username);
  if (!account || !bcrypt.compareSync(password, account.passwordHash)) return null;
  return account;
}

export function validateOtp(account: Account, token?: string): boolean {
  if (!account.otpEnabled || !account.otpSecret) return true;
  const clean = String(token || '').replace(/\s+/g, '');
  return clean.length === 6 && [0, -1, 1].some((offset) => totp(account.otpSecret || '', offset) === clean);
}

export function changePassword(username: string, currentPassword: string, newPassword: string): { success: boolean; error?: string } {
  const account = findAccount(username);
  if (!account || !bcrypt.compareSync(currentPassword, account.passwordHash)) return { success: false, error: 'Current password is incorrect' };
  if (newPassword.length < 10) return { success: false, error: 'New password must be at least 10 characters' };
  const updated = { ...account, passwordHash: bcrypt.hashSync(newPassword, 12) };
  if (account.role === 'admin') writeAccount(updated);
  else writeUsers(readUsers().map((user) => user.username === username ? updated : user));
  revokeUserSessions(username);
  return { success: true };
}

export function startOtpSetup(username: string): { success: boolean; error?: string; otpSecret?: string; otpAuthUrl?: string } {
  const account = findAccount(username);
  if (!account) return { success: false, error: 'Account not found' };
  const otpSecret = base32Encode(crypto.randomBytes(20));
  const updated = { ...account, otpSecret, otpEnabled: false };
  if (account.role === 'admin') writeAccount(updated);
  else writeUsers(readUsers().map((user) => user.username === username ? updated : user));
  return {
    success: true,
    otpSecret,
    otpAuthUrl: `otpauth://totp/${encodeURIComponent(`hostQ:${username}`)}?secret=${otpSecret}&issuer=hostQ`,
  };
}

export function enableOtp(username: string, token: string): { success: boolean; error?: string } {
  const account = findAccount(username);
  if (!account?.otpSecret) return { success: false, error: 'Start 2FA setup first' };
  if (!validateOtp({ ...account, otpEnabled: true }, token)) return { success: false, error: 'Invalid 2FA code' };
  const updated = { ...account, otpEnabled: true };
  if (account.role === 'admin') writeAccount(updated);
  else writeUsers(readUsers().map((user) => user.username === username ? updated : user));
  return { success: true };
}

export function createSiteAccount(username: string, password: string, sitePermissions: Account['sitePermissions']): Account {
  if (!/^[a-zA-Z0-9_.-]{3,32}$/.test(username)) throw new Error('Username must be 3-32 safe characters');
  if (password.length < 10) throw new Error('Password must be at least 10 characters');
  if (findAccount(username)) throw new Error('User already exists');
  const account: Account = {
    username,
    passwordHash: bcrypt.hashSync(password, 12),
    role: 'user',
    sitePermissions,
    createdAt: new Date().toISOString(),
  };
  writeUsers([...readUsers(), account]);
  return account;
}

export function updateSiteAccount(username: string, sitePermissions: Account['sitePermissions']) {
  writeUsers(readUsers().map((account) => account.username === username ? { ...account, sitePermissions } : account));
}

function readSessions(): SessionRecord[] {
  try {
    return JSON.parse(fs.readFileSync(sessionsPath(), 'utf8')) as SessionRecord[];
  } catch {
    return [];
  }
}

function writeSessions(sessions: SessionRecord[]) {
  fs.mkdirSync(dataDir(), { recursive: true, mode: 0o700 });
  fs.writeFileSync(sessionsPath(), JSON.stringify(sessions, null, 2), { mode: 0o600 });
}

export function createSession(account: Account, context: { ip?: string; userAgent?: string }): { token: string; session: SessionRecord } {
  const now = new Date().toISOString();
  const session: SessionRecord = {
    id: crypto.randomUUID(),
    username: account.username,
    role: account.role,
    createdAt: now,
    lastSeenAt: now,
    ip: context.ip,
    userAgent: context.userAgent,
  };
  writeSessions([...readSessions().filter((item) => !item.revokedAt), session]);
  return { session, token: signToken(account.username, account.role, session.id) };
}

export function getSession(id: string): SessionRecord | null {
  return readSessions().find((session) => session.id === id) || null;
}

export function listSessions(username?: string): SessionRecord[] {
  return readSessions().filter((session) => !username || session.username === username);
}

export function touchSession(id: string) {
  const now = new Date().toISOString();
  writeSessions(readSessions().map((session) => session.id === id ? { ...session, lastSeenAt: now } : session));
}

export function revokeSession(id: string) {
  const now = new Date().toISOString();
  writeSessions(readSessions().map((session) => session.id === id ? { ...session, revokedAt: now } : session));
}

export function revokeUserSessions(username: string) {
  const now = new Date().toISOString();
  writeSessions(readSessions().map((session) => session.username === username ? { ...session, revokedAt: now } : session));
}
