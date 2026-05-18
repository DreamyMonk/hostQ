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
    const entry = {
      ts: new Date().toISOString(),
      status: event.status || 'success',
      ...event,
    };
    fs.appendFileSync(auditLogPath(), `${JSON.stringify(entry)}\n`, { mode: 0o600 });
  } catch {
    // Audit logging must never break the control plane.
  }
}

const attempts = new Map<string, { count: number; resetAt: number }>();

export function checkLoginRateLimit(key: string): { allowed: boolean; retryAfter?: number } {
  const now = Date.now();
  const windowMs = 15 * 60 * 1000;
  const limit = 8;
  const current = attempts.get(key);
  if (!current || current.resetAt <= now) {
    attempts.set(key, { count: 1, resetAt: now + windowMs });
    return { allowed: true };
  }
  if (current.count >= limit) {
    return { allowed: false, retryAfter: Math.ceil((current.resetAt - now) / 1000) };
  }
  current.count += 1;
  return { allowed: true };
}

export function clearLoginRateLimit(key: string) {
  attempts.delete(key);
}

export function safeUsername(value: string): string {
  return value.replace(/[^a-zA-Z0-9_.-]/g, '').slice(0, 32);
}
