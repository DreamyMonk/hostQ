import fs from 'fs';
import path from 'path';

export type PanelConfig = {
  panelDomain: string;
  panelUrl: string;
  allowInsecureHttp: boolean;
  envFile: string;
};

function envFilePath() {
  return process.env.HOSTQ_ENV_FILE || path.join(process.cwd(), '.env.local');
}

function parseEnv(text: string): Record<string, string> {
  const values: Record<string, string> = {};
  for (const line of text.split(/\r?\n/)) {
    const match = line.match(/^\s*([A-Z0-9_]+)\s*=\s*(.*)\s*$/);
    if (!match) continue;
    values[match[1]] = match[2].replace(/^['"]|['"]$/g, '');
  }
  return values;
}

function domainFromUrl(value: string) {
  try {
    return new URL(value).hostname;
  } catch {
    return '';
  }
}

export function readPanelConfig(): PanelConfig {
  const file = envFilePath();
  let values: Record<string, string> = {};
  try {
    values = parseEnv(fs.readFileSync(file, 'utf8'));
  } catch {
    values = {};
  }
  const panelUrl = values.PANEL_URL || process.env.PANEL_URL || '';
  const panelDomain = values.PANEL_DOMAIN || process.env.PANEL_DOMAIN || domainFromUrl(panelUrl);
  const allowValue = values.HOSTQ_ALLOW_INSECURE_HTTP || process.env.HOSTQ_ALLOW_INSECURE_HTTP || 'false';
  return {
    panelDomain,
    panelUrl,
    allowInsecureHttp: allowValue === 'true',
    envFile: file,
  };
}

export function validPanelDomain(value: string) {
  return /^(?:[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$/.test(value);
}

export function upsertEnv(values: Record<string, string>) {
  const file = envFilePath();
  let lines: string[] = [];
  try {
    lines = fs.readFileSync(file, 'utf8').split(/\r?\n/);
  } catch {
    lines = [];
  }
  for (const [key, value] of Object.entries(values)) {
    const line = `${key}=${value}`;
    const index = lines.findIndex((existing) => existing.match(new RegExp(`^\\s*${key}\\s*=`)));
    if (index >= 0) lines[index] = line;
    else lines.push(line);
  }
  fs.mkdirSync(path.dirname(file), { recursive: true });
  fs.writeFileSync(file, `${lines.filter((line, index) => line !== '' || index < lines.length - 1).join('\n')}\n`, { mode: 0o600 });
}
