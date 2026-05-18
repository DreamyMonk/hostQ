import { NextRequest, NextResponse } from 'next/server';
import { cookies } from 'next/headers';
import { verifyToken } from '@/lib/auth';
import { runCommand } from '@/lib/exec';

async function auth() {
  const cookieStore = await cookies();
  const token = cookieStore.get('panel_token')?.value;
  return token ? verifyToken(token) : null;
}

// ──────────────────────────────────────────────
// Service definitions
// ──────────────────────────────────────────────
interface ServiceDef {
  id: string;
  name: string;
  systemd: string;
  checkCmd: string;
  installCmd: string;
  configCmd: string;
  port: number | null;
  category: string;
}

const SERVICES: ServiceDef[] = [
  {
    id: 'nginx',
    name: 'Nginx',
    systemd: 'nginx',
    checkCmd: 'which nginx && nginx -v',
    installCmd: 'apt-get install -y nginx',
    configCmd: 'nginx -t',
    port: 80,
    category: 'web',
  },
  {
    id: 'apache',
    name: 'Apache2',
    systemd: 'apache2',
    checkCmd: 'which apache2 && apache2 -v',
    installCmd: 'apt-get install -y apache2',
    configCmd: 'apache2ctl configtest',
    port: 80,
    category: 'web',
  },
  {
    id: 'mysql',
    name: 'MariaDB 10.11',
    systemd: 'mariadb',
    checkCmd: 'which mysql && mysql --version',
    installCmd: 'apt-get install -y mariadb-server mariadb-client',
    configCmd: 'mysql -u root -e "SHOW DATABASES;"',
    port: 3306,
    category: 'database',
  },
  {
    id: 'php85fpm',
    name: 'PHP 8.5-FPM',
    systemd: 'php8.5-fpm',
    checkCmd: 'php8.5 --version',
    installCmd: 'apt-get install -y php8.5-fpm php8.5-cli php8.5-common php8.5-mysql php8.5-curl php8.5-gd php8.5-mbstring php8.5-xml php8.5-zip php8.5-bcmath php8.5-intl',
    configCmd: 'php-fpm8.5 -t',
    port: null,
    category: 'php',
  },
  {
    id: 'php84fpm',
    name: 'PHP 8.4-FPM',
    systemd: 'php8.4-fpm',
    checkCmd: 'php8.4 --version',
    installCmd: 'apt-get install -y php8.4-fpm php8.4-cli php8.4-common php8.4-mysql php8.4-curl php8.4-gd php8.4-mbstring php8.4-xml php8.4-zip php8.4-bcmath php8.4-intl',
    configCmd: 'php-fpm8.4 -t',
    port: null,
    category: 'php',
  },
  {
    id: 'php83fpm',
    name: 'PHP 8.3-FPM',
    systemd: 'php8.3-fpm',
    checkCmd: 'php8.3 --version',
    installCmd: 'apt-get install -y php8.3-fpm php8.3-mysql php8.3-curl php8.3-gd php8.3-mbstring php8.3-xml php8.3-zip',
    configCmd: 'php-fpm8.3 -t',
    port: null,
    category: 'php',
  },
  {
    id: 'php82fpm',
    name: 'PHP 8.2-FPM',
    systemd: 'php8.2-fpm',
    checkCmd: 'php8.2 --version',
    installCmd: 'apt-get install -y php8.2-fpm php8.2-cli php8.2-common php8.2-mysql php8.2-curl php8.2-gd php8.2-mbstring php8.2-xml php8.2-zip php8.2-bcmath php8.2-intl',
    configCmd: 'php-fpm8.2 -t',
    port: null,
    category: 'php',
  },
  {
    id: 'certbot',
    name: 'Certbot (SSL)',
    systemd: '',
    checkCmd: 'certbot --version',
    installCmd: 'apt-get install -y certbot python3-certbot-nginx python3-certbot-apache',
    configCmd: 'certbot --version',
    port: null,
    category: 'security',
  },
  {
    id: 'wpcli',
    name: 'WP-CLI',
    systemd: '',
    checkCmd: 'wp --version --allow-root',
    installCmd:
      'curl -s -O https://raw.githubusercontent.com/wp-cli/builds/gh-pages/phar/wp-cli.phar && chmod +x wp-cli.phar && mv wp-cli.phar /usr/local/bin/wp',
    configCmd: 'wp --info --allow-root',
    port: null,
    category: 'tools',
  },
  {
    id: 'phpmyadmin',
    name: 'phpMyAdmin 5.2',
    systemd: '',
    checkCmd: 'ls /usr/share/phpmyadmin/index.php',
    installCmd: 'DEBIAN_FRONTEND=noninteractive apt-get install -y phpmyadmin',
    configCmd: 'ls /usr/share/phpmyadmin/',
    port: null,
    category: 'tools',
  },
  {
    id: 'pureftpd',
    name: 'Pure-FTPd 1.0.49',
    systemd: 'pure-ftpd',
    checkCmd: 'pure-ftpd --help 2>&1 | head -1 || pure-ftpd-mysql --help 2>&1 | head -1',
    installCmd: 'apt-get install -y pure-ftpd pure-ftpd-common',
    configCmd: 'systemctl status pure-ftpd --no-pager',
    port: 21,
    category: 'tools',
  },
];

// ──────────────────────────────────────────────
// Check a single service status
// ──────────────────────────────────────────────
async function checkService(svc: ServiceDef) {
  const isLinux = process.platform === 'linux';

  // Demo data for non-Linux
  if (!isLinux) {
      const demoStatus: Record<string, string> = {
      nginx: 'active', mysql: 'active', php84fpm: 'active',
      certbot: 'installed', wpcli: 'installed',
      apache: 'inactive', php82fpm: 'inactive', php83fpm: 'inactive', php85fpm: 'inactive', phpmyadmin: 'installed', pureftpd: 'active',
    };
    return {
      id: svc.id, name: svc.name, category: svc.category,
      installed: ['nginx','mysql','php84fpm','certbot','wpcli','phpmyadmin','pureftpd'].includes(svc.id),
      running: ['nginx','mysql','php84fpm','pureftpd'].includes(svc.id),
      status: demoStatus[svc.id] || 'inactive',
      version: svc.id === 'nginx' ? '1.24.0' : svc.id === 'mysql' ? '10.11.7-MariaDB' : 'N/A',
      port: svc.port,
    };
  }

  // Check installed
  const installCheck = await runCommand(svc.checkCmd + ' 2>/dev/null');
  const installed = installCheck.success;

  // Check systemd status
  let running = false;
  let status = 'not-installed';
  if (svc.systemd) {
    const statusR = await runCommand(`systemctl is-active ${svc.systemd} 2>/dev/null`);
    running = statusR.stdout.trim() === 'active';
    if (!installed) status = 'not-installed';
    else status = running ? 'active' : 'inactive';
  } else if (installed) {
    status = 'installed';
  }

  // Get version
  let version = '';
  if (installed) {
    const versionStr = installCheck.stdout.split('\n')[0];
    const vMatch = versionStr.match(/[\d.]+/);
    version = vMatch ? vMatch[0] : '';
  }

  return { id: svc.id, name: svc.name, category: svc.category, installed, running, status, version, port: svc.port };
}

// ──────────────────────────────────────────────
// GET - return all service statuses
// ──────────────────────────────────────────────
export async function GET() {
  if (!await auth()) return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });

  const statuses = await Promise.all(SERVICES.map(checkService));
  return NextResponse.json({ services: statuses, demo: process.platform !== 'linux' });
}

// ──────────────────────────────────────────────
// POST - install / start / stop / restart / test service
// ──────────────────────────────────────────────
export async function POST(request: NextRequest) {
  if (!await auth()) return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });

  const { serviceId, action } = await request.json();
  const svc = SERVICES.find(s => s.id === serviceId);
  if (!svc) return NextResponse.json({ error: 'Unknown service' }, { status: 400 });

  if (process.platform !== 'linux') {
    return NextResponse.json({
      success: true,
      output: `▶ Demo mode\n✓ Would ${action} ${svc.name}\n✓ Done (no real action on Windows/macOS dev)`,
      message: `${svc.name} ${action} simulated (demo mode)`,
    });
  }

  let cmd = '';
  switch (action) {
    case 'install':
      cmd = `export DEBIAN_FRONTEND=noninteractive; apt-get update -qq; ${svc.installCmd} 2>&1`;
      break;
    case 'start':
      cmd = `systemctl start ${svc.systemd} 2>&1`;
      break;
    case 'stop':
      cmd = `systemctl stop ${svc.systemd} 2>&1`;
      break;
    case 'restart':
      cmd = `systemctl restart ${svc.systemd} 2>&1`;
      break;
    case 'enable':
      cmd = `systemctl enable ${svc.systemd} 2>&1`;
      break;
    case 'test':
      cmd = `${svc.configCmd} 2>&1`;
      break;
    case 'config':
      // Return the config file content
      const confPaths: Record<string, string> = {
        nginx: '/etc/nginx/nginx.conf',
        apache: '/etc/apache2/apache2.conf',
        mysql: '/etc/mysql/mariadb.conf.d/50-server.cnf',
        php85fpm: '/etc/php/8.5/fpm/php.ini',
        php84fpm: '/etc/php/8.4/fpm/php.ini',
        php83fpm: '/etc/php/8.3/fpm/php.ini',
        php82fpm: '/etc/php/8.2/fpm/php.ini',
      };
      const confPath = confPaths[serviceId];
      if (confPath) {
        const r = await runCommand(`cat "${confPath}" 2>/dev/null | head -100`);
        return NextResponse.json({ success: true, output: r.stdout || 'Config file not found', configPath: confPath });
      }
      return NextResponse.json({ success: false, error: 'No config known for this service' });
    default:
      return NextResponse.json({ error: 'Unknown action' }, { status: 400 });
  }

  const r = await runCommand(cmd, 180000);

  // Auto-configure after install
  if (action === 'install' && r.success) {
    if (svc.id === 'nginx') {
      await runCommand('systemctl enable nginx && systemctl start nginx');
    } else if (svc.id === 'apache') {
      await runCommand('a2enmod rewrite && a2enmod ssl && systemctl enable apache2 && systemctl start apache2');
    } else if (svc.id === 'mysql') {
      await runCommand('systemctl enable mariadb && systemctl start mariadb');
    } else if (svc.id.startsWith('php')) {
      const verMatch = svc.id.match(/^php(\d)(\d)fpm$/);
      const ver = verMatch ? `${verMatch[1]}.${verMatch[2]}` : '';
      if (ver) await runCommand(`systemctl enable php${ver}-fpm && systemctl start php${ver}-fpm`);
    } else if (svc.id === 'pureftpd') {
      await runCommand('systemctl enable pure-ftpd && systemctl start pure-ftpd');
    }
  }

  return NextResponse.json({
    success: r.success,
    output: r.stdout || r.stderr || '(no output)',
    message: r.success ? `${svc.name} ${action} completed` : `${svc.name} ${action} failed`,
  });
}
