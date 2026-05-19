#!/usr/bin/env node
import { execFile } from 'node:child_process';

const SERVICES = new Map([
  ['nginx', 'nginx'],
  ['apache', 'apache2'],
  ['mysql', 'mariadb'],
  ['php85fpm', 'php8.5-fpm'],
  ['php84fpm', 'php8.4-fpm'],
  ['php83fpm', 'php8.3-fpm'],
  ['php82fpm', 'php8.2-fpm'],
  ['pureftpd', 'pure-ftpd'],
  ['redis', 'redis-server'],
]);

const INSTALLS = new Map([
  ['nginx', ['apt-get install -y nginx']],
  ['apache', ['apt-get install -y apache2']],
  ['mysql', ['apt-get install -y mariadb-server mariadb-client']],
  ['php85fpm', ['apt-get install -y php8.5-fpm php8.5-cli php8.5-common php8.5-mysql php8.5-curl php8.5-gd php8.5-mbstring php8.5-xml php8.5-zip php8.5-bcmath php8.5-intl']],
  ['php84fpm', ['apt-get install -y php8.4-fpm php8.4-cli php8.4-common php8.4-mysql php8.4-curl php8.4-gd php8.4-mbstring php8.4-xml php8.4-zip php8.4-bcmath php8.4-intl']],
  ['php83fpm', ['apt-get install -y php8.3-fpm php8.3-mysql php8.3-curl php8.3-gd php8.3-mbstring php8.3-xml php8.3-zip']],
  ['php82fpm', ['apt-get install -y php8.2-fpm php8.2-cli php8.2-common php8.2-mysql php8.2-curl php8.2-gd php8.2-mbstring php8.2-xml php8.2-zip php8.2-bcmath php8.2-intl']],
  ['certbot', ['apt-get install -y certbot python3-certbot-nginx python3-certbot-apache']],
  ['phpmyadmin', ['DEBIAN_FRONTEND=noninteractive apt-get install -y phpmyadmin']],
  ['pureftpd', ['apt-get install -y pure-ftpd pure-ftpd-common']],
  ['redis', ['apt-get install -y redis-server']],
]);

const UPDATE_REPO = 'DreamyMonk/hostQ';

function validTag(tag) {
  return /^v\d+\.\d+\.\d+([.-][A-Za-z0-9]+)?$/.test(String(tag || ''));
}

function runShell(command) {
  return new Promise((resolve) => {
    execFile('/bin/sh', ['-lc', command], { timeout: 180000, maxBuffer: 10 * 1024 * 1024 }, (error, stdout, stderr) => {
      resolve({ success: !error, stdout: stdout.trim(), stderr: stderr.trim(), error: error?.message });
    });
  });
}

function serviceName(id) {
  const name = SERVICES.get(String(id || ''));
  if (!name) throw new Error('Service is not allowed');
  return name;
}

async function main() {
  const input = JSON.parse(process.argv[2] || '{}');
  const { task, payload = {} } = input;
  let result;

  if (task === 'service.install') {
    const id = String(payload.serviceId || '');
    const commands = INSTALLS.get(id);
    if (!commands) throw new Error('Install task is not allowed');
    result = await runShell(`export DEBIAN_FRONTEND=noninteractive; apt-get update -qq; ${commands.join(' && ')}`);
    const systemd = SERVICES.get(id);
    if (result.success && systemd) await runShell(`systemctl enable ${systemd} && systemctl start ${systemd}`);
  } else if (task === 'service.control') {
    const action = String(payload.action || '');
    if (!['start', 'stop', 'restart', 'enable'].includes(action)) throw new Error('Service action is not allowed');
    result = await runShell(`systemctl ${action} ${serviceName(payload.serviceId)}`);
  } else if (task === 'web.reload') {
    const server = String(payload.server || 'nginx');
    if (server === 'apache') result = await runShell('apache2ctl configtest && systemctl reload apache2');
    else result = await runShell('nginx -t && systemctl reload nginx');
  } else if (task === 'panel.update') {
    const tag = String(payload.tag || '');
    if (!validTag(tag)) throw new Error('Release tag is not allowed');
    const panelDir = '/opt/hostq';
    const backup = `/var/backups/hostq/panel-${Date.now()}.tar.gz`;
    const archive = `/tmp/hostq-${tag}.tar.gz`;
    const unpack = `/tmp/hostq-${tag}`;
    result = await runShell([
      'set -e',
      `mkdir -p /var/backups/hostq ${unpack}`,
      `[ -d ${panelDir} ]`,
      `tar -czf ${backup} -C ${panelDir} .`,
      `curl -fsSL -o ${archive} https://codeload.github.com/${UPDATE_REPO}/tar.gz/refs/tags/${tag}`,
      `tar -xzf ${archive} -C ${unpack} --strip-components=1`,
      `rsync -a --delete --exclude .env.local --exclude node_modules --exclude .next ${unpack}/ ${panelDir}/`,
      `cd ${panelDir}`,
      'npm ci',
      'npm run build',
      'npm prune --omit=dev',
      'pm2 restart hostq || systemctl restart hostq',
      `rm -rf ${unpack} ${archive}`,
      `echo "Updated hostQ to ${tag}. Backup: ${backup}"`,
    ].join(' && '));
  } else {
    throw new Error('Task is not allowed');
  }

  process.stdout.write(JSON.stringify(result));
  process.exit(result.success ? 0 : 1);
}

main().catch((error) => {
  process.stderr.write(error instanceof Error ? error.message : String(error));
  process.exit(1);
});
