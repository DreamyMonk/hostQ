// lib/exec.ts - Server-side shell command executor
import { exec } from 'child_process';
import { promisify } from 'util';

const execAsync = promisify(exec);

export interface ExecResult {
  stdout: string;
  stderr: string;
  success: boolean;
  error?: string;
}

export async function runCommand(command: string, timeout = 30000): Promise<ExecResult> {
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
