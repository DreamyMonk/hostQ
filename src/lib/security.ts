import crypto from 'crypto';
import fs from 'fs';
import path from 'path';
import type { NextRequest } from 'next/server';

export const CSRF_COOKIE = 'hostq_csrf';
export const CSRF_HEADER = 'x-csrf-token';

export function createCsrfToken(): string {
  return crypto.randomBytes(32).toString('base64url');
}

export function clientIp(request: NextRequest | Request): string {
  const headers = request.headers;
  return (
    headers.get('x-forwarded-for')?.split(',')[0]?.trim() ||
    headers.get('x-real-ip') ||
    'local'
  );
}

function dataDir(): string {
  if (process.env.HOSTQ_DATA_DIR) return process.env.HOSTQ_DATA_DIR;
  if (process.platform === 'linux') return '/etc/hostq';
  return path.join(process.cwd(), '.hostq');
}

export function auditLogPath(): string {
  return path.join(dataDir(), 'audit.log');
}

function rotateAuditIfNeeded() {
  const file = auditLogPath();
  try {
    if (!fs.existsSync(file) || fs.statSync(file).size < 5 * 1024 * 1024) return;
    fs.renameSync(file, `${file}.${new Date().toISOString().replace(/[:.]/g, '-')}`);
  } catch {
    // best effort
  }
}

function lastAuditHash(): string {
  try {
    const lines = fs.readFileSync(auditLogPath(), 'utf8').trim().split('\n');
    const last = lines.at(-1);
    return last ? JSON.parse(last).hash || '' : '';
  } catch {
    return '';
  }
}

export function audit(event: {
  actor?: string;
  action: string;
  target?: string;
  status?: 'success' | 'failure';
  ip?: string;
  details?: Record<string, unknown>;
}) {
  try {
    fs.mkdirSync(dataDir(), { recursive: true, mode: 0o700 });
    rotateAuditIfNeeded();
    const prevHash = lastAuditHash();
    const entryBase = {
      ts: new Date().toISOString(),
      status: event.status || 'success',
      prevHash,
      ...event,
    };
    const hash = crypto.createHash('sha256').update(`${prevHash}:${JSON.stringify(entryBase)}`).digest('hex');
    const entry = { ...entryBase, hash };
    fs.appendFileSync(auditLogPath(), `${JSON.stringify(entry)}\n`, { mode: 0o600 });
  } catch {
    // Audit logging must never break the control plane.
  }
}

function rateLimitPath(): string {
  return path.join(dataDir(), 'rate-limits.json');
}

function readAttempts(): Record<string, { count: number; resetAt: number }> {
  try {
    return JSON.parse(fs.readFileSync(rateLimitPath(), 'utf8')) as Record<string, { count: number; resetAt: number }>;
  } catch {
    return {};
  }
}

function writeAttempts(attempts: Record<string, { count: number; resetAt: number }>) {
  fs.mkdirSync(dataDir(), { recursive: true, mode: 0o700 });
  fs.writeFileSync(rateLimitPath(), JSON.stringify(attempts, null, 2), { mode: 0o600 });
}

export function checkLoginRateLimit(key: string): { allowed: boolean; retryAfter?: number } {
  const now = Date.now();
  const windowMs = 15 * 60 * 1000;
  const limit = 8;
  const attempts = readAttempts();
  const current = attempts[key];
  if (!current || current.resetAt <= now) {
    attempts[key] = { count: 1, resetAt: now + windowMs };
    writeAttempts(attempts);
    return { allowed: true };
  }
  if (current.count >= limit) {
    return { allowed: false, retryAfter: Math.ceil((current.resetAt - now) / 1000) };
  }
  current.count += 1;
  writeAttempts(attempts);
  return { allowed: true };
}

export function clearLoginRateLimit(key: string) {
  const attempts = readAttempts();
  delete attempts[key];
  writeAttempts(attempts);
}

export function safeUsername(value: string): string {
  return value.replace(/[^a-zA-Z0-9_.-]/g, '').slice(0, 32);
}
