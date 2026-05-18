// lib/exec.ts - Server-side shell command executor
import { exec } from 'child_process';
import path from 'path';
import { promisify } from 'util';

const execAsync = promisify(exec);

export interface ExecResult {
  stdout: string;
  stderr: string;
  success: boolean;
  error?: string;
}

export async function runCommand(command: string, timeout = 30000): Promise<ExecResult> {
  const dangerous = /\b(apt-get|systemctl|pm2|rsync|curl|npm\s+(ci|run|prune)|rm\s+-|mv\s+|chmod|chown|ln\s+-|certbot|wp\s|mysql\s|tar\s+-|a2en|a2dis|update-alternatives)\b/.test(command);
  if (process.env.HOSTQ_REQUIRE_HELPER === 'true' && dangerous) {
    return {
      stdout: '',
      stderr: 'Direct privileged shell execution is disabled. Add this action to hostq-helper allowlist.',
      success: false,
      error: 'Direct privileged shell execution disabled',
    };
  }
  try {
    const { stdout, stderr } = await execAsync(command, {
      timeout,
      maxBuffer: 1024 * 1024 * 10, // 10MB
    });
    return { stdout: stdout.trim(), stderr: stderr.trim(), success: true };
  } catch (err: unknown) {
    const error = err as { message?: string; stdout?: string; stderr?: string };
    return {
      stdout: error.stdout || '',
      stderr: error.stderr || '',
      success: false,
      error: error.message || 'Unknown error',
    };
  }
}

export async function runHelper(task: string, payload: Record<string, unknown> = {}, timeout = 180000): Promise<ExecResult> {
  const helper = process.env.HOSTQ_HELPER || path.join(process.cwd(), 'scripts', 'hostq-helper.mjs');
  return runCommand(`node ${shellQuote(helper)} ${shellQuote(JSON.stringify({ task, payload }))}`, timeout);
}

export function shellQuote(value: string): string {
  return `'${String(value).replace(/'/g, `'\\''`)}'`;
}

export function mysqlString(value: string): string {
  return `'${String(value).replace(/\\/g, '\\\\').replace(/'/g, "''")}'`;
}

export function mysqlIdentifier(value: string): string {
  if (!/^[A-Za-z0-9_]+$/.test(value)) {
    throw new Error('Invalid MySQL identifier');
  }
  return `\`${value}\``;
}

export async function runMysql(sql: string): Promise<ExecResult> {
  const host = process.env.DB_HOST || 'localhost';
  const user = process.env.DB_ROOT_USER || 'root';
  const pass = process.env.DB_ROOT_PASSWORD || '';
  const cmd = [
    'mysql',
    '-h', shellQuote(host),
    '-u', shellQuote(user),
    pass ? `-p${shellQuote(pass)}` : '',
    '-e', shellQuote(sql),
  ].filter(Boolean).join(' ');
  return runCommand(cmd);
}
