import { NextResponse } from 'next/server';
import { cookies } from 'next/headers';
import { verifyToken } from '@/lib/auth';
import { runCommand } from '@/lib/exec';
import os from 'os';

async function getStats() {
  const platform = os.platform();
  const isLinux = platform === 'linux';

  // CPU usage
  let cpuUsage = 0;
  if (isLinux) {
    const r = await runCommand("top -bn1 | grep 'Cpu(s)' | awk '{print $2}' | cut -d'%' -f1");
    cpuUsage = parseFloat(r.stdout) || 0;
  } else {
    // Windows/Mac fallback: use load average or simulate
    const loadAvg = os.loadavg()[0];
    const cpuCount = os.cpus().length;
    cpuUsage = Math.min(100, Math.round((loadAvg / cpuCount) * 100));
  }

  // Memory
  const totalMem = os.totalmem();
  const freeMem = os.freemem();
  const usedMem = totalMem - freeMem;
  const memPercent = Math.round((usedMem / totalMem) * 100);

  // Disk usage (Linux only)
  let diskUsed = 0, diskTotal = 0, diskPercent = 0;
  if (isLinux) {
    const r = await runCommand("df -BG / | tail -1 | awk '{print $2, $3, $5}'");
    if (r.success && r.stdout) {
      const parts = r.stdout.split(' ');
      diskTotal = parseInt(parts[0]) || 0;
      diskUsed  = parseInt(parts[1]) || 0;
      diskPercent = parseInt(parts[2]) || 0;
    }
  }

  // Uptime
  const uptimeSecs = os.uptime();
  const days    = Math.floor(uptimeSecs / 86400);
  const hours   = Math.floor((uptimeSecs % 86400) / 3600);
  const minutes = Math.floor((uptimeSecs % 3600) / 60);
  const uptime  = days > 0 ? `${days}d ${hours}h ${minutes}m` : `${hours}h ${minutes}m`;

  // Nginx / Apache status (Linux)
  let nginxRunning = false, apacheRunning = false;
  if (isLinux) {
    const ng = await runCommand('systemctl is-active nginx');
    nginxRunning = ng.stdout.trim() === 'active';
    const ap = await runCommand('systemctl is-active apache2');
    apacheRunning = ap.stdout.trim() === 'active';
  }

  // PHP version
  let phpVersion = 'N/A';
  const phpR = await runCommand('php --version');
  if (phpR.success) {
    const match = phpR.stdout.match(/PHP (\d+\.\d+\.\d+)/);
    phpVersion = match ? match[1] : phpR.stdout.split('\n')[0];
  }

  const hostname = os.hostname();
  const cpuModel = os.cpus()[0]?.model || 'N/A';
  const cpuCount = os.cpus().length;

  return {
    cpu: { usage: Math.round(cpuUsage), model: cpuModel, cores: cpuCount },
    memory: {
      total: Math.round(totalMem / 1024 / 1024),
      used:  Math.round(usedMem / 1024 / 1024),
      free:  Math.round(freeMem / 1024 / 1024),
      percent: memPercent,
    },
    disk: { total: diskTotal, used: diskUsed, percent: diskPercent },
    uptime,
    hostname,
    phpVersion,
    services: { nginx: nginxRunning, apache: apacheRunning },
    platform,
  };
}

export async function GET() {
  const cookieStore = await cookies();
  const token = cookieStore.get('panel_token')?.value;
  if (!token || !verifyToken(token)) {
    return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });
  }
  try {
    const stats = await getStats();
    return NextResponse.json(stats);
  } catch (err) {
    return NextResponse.json({ error: String(err) }, { status: 500 });
  }
}
