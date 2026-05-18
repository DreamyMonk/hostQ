'use client';
import { useEffect, useState } from 'react';
import { Cpu, MemoryStick, HardDrive, Clock, Globe, Database, FileCode2, RefreshCw } from 'lucide-react';
import Link from 'next/link';

interface ServerStats {
  cpu: { usage: number; model: string; cores: number };
  memory: { total: number; used: number; free: number; percent: number };
  disk: { total: number; used: number; percent: number };
  uptime: string;
  hostname: string;
  phpVersion: string;
  services: { nginx: boolean; apache: boolean };
  platform: string;
}

function StatCard({ icon: Icon, label, value, sub, percent, color, href }: {
  icon: React.ComponentType<{size?: number; color?: string}>;
  label: string;
  value: string;
  sub?: string;
  percent?: number;
  color: string;
  href?: string;
}) {
  const inner = (
    <div className="glass-card glass-card-hover" style={{ padding: 20, cursor: href ? 'pointer' : 'default' }}>
      <div style={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', marginBottom: 14 }}>
        <div style={{
          width: 40, height: 40, background: `${color}18`,
          border: `1px solid ${color}30`, borderRadius: 10,
          display: 'flex', alignItems: 'center', justifyContent: 'center'
        }}>
          <Icon size={18} color={color} />
        </div>
        {percent !== undefined && (
          <span style={{ fontSize: 12, fontWeight: 600, color: percent > 80 ? '#ef4444' : percent > 60 ? '#f59e0b' : '#22c55e' }}>
            {percent}%
          </span>
        )}
      </div>
      <div style={{ fontSize: 22, fontWeight: 800, marginBottom: 2 }}>{value}</div>
      <div style={{ fontSize: 12, color: 'var(--text-muted)', fontWeight: 500, textTransform: 'uppercase', letterSpacing: '0.5px' }}>{label}</div>
      {sub && <div style={{ fontSize: 11, color: 'var(--text-muted)', marginTop: 4 }}>{sub}</div>}
      {percent !== undefined && (
        <div className="progress-bar" style={{ marginTop: 12 }}>
          <div className="progress-fill" style={{
            width: `${percent}%`,
            background: percent > 80 ? '#ef4444' : percent > 60 ? '#f59e0b' : color
          }} />
        </div>
      )}
    </div>
  );
  return href ? <Link href={href} style={{ textDecoration: 'none', color: 'inherit' }}>{inner}</Link> : inner;
}

export default function DashboardPage() {
  const [stats, setStats] = useState<ServerStats | null>(null);
  const [loading, setLoading] = useState(true);
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null);

  const fetchStats = async () => {
    setLoading(true);
    try {
      const res = await fetch('/api/server');
      if (res.ok) {
        setStats(await res.json());
        setLastUpdated(new Date());
      }
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    const id = setTimeout(() => { void fetchStats(); }, 0);
    const interval = setInterval(fetchStats, 30000);
    return () => {
      clearTimeout(id);
      clearInterval(interval);
    };
  }, []);

  const quickActions = [
    { icon: '🌐', label: 'Install WordPress', desc: 'One-click WP setup', href: '/dashboard/wordpress', color: '#3b82f6' },
    { icon: '🔒', label: 'Add SSL Certificate', desc: "Free Let's Encrypt SSL", href: '/dashboard/ssl', color: '#22c55e' },
    { icon: '🗄️', label: 'Create Database', desc: 'MySQL database + user', href: '/dashboard/databases', color: '#06b6d4' },
    { icon: '🐘', label: 'Switch PHP Version', desc: 'PHP 7.4 → 8.x', href: '/dashboard/php', color: '#a855f7' },
    { icon: '📁', label: 'Open File Manager', desc: 'Browse server files', href: '/dashboard/files', color: '#f59e0b' },
  ];

  return (
    <div className="fade-in">
      {/* Header */}
      <div className="page-header" style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <div>
          <h1 className="page-title">Server Overview</h1>
          <p className="page-subtitle">
            {stats?.hostname || 'Loading…'} &nbsp;·&nbsp; {stats?.platform || ''}
            {lastUpdated && <span style={{ marginLeft: 8, color: 'var(--text-muted)' }}>
              · Updated {lastUpdated.toLocaleTimeString()}
            </span>}
          </p>
        </div>
        <button
          id="refresh-stats-btn"
          onClick={fetchStats}
          className="btn btn-ghost btn-sm"
          disabled={loading}
          style={{ gap: 6 }}
        >
          <RefreshCw size={14} style={{ animation: loading ? 'spin 1s linear infinite' : 'none' }} />
          Refresh
        </button>
      </div>

      {/* Stat cards */}
      {loading && !stats ? (
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: 16, marginBottom: 28 }}>
          {[...Array(4)].map((_, i) => (
            <div key={i} className="glass-card" style={{ padding: 20, height: 130 }}>
              <div style={{ background: 'var(--bg-elevated)', borderRadius: 8, height: 10, width: '60%', marginBottom: 10 }} />
              <div style={{ background: 'var(--bg-elevated)', borderRadius: 8, height: 24, width: '40%' }} />
            </div>
          ))}
        </div>
      ) : stats ? (
        <div className="stat-grid" style={{ marginBottom: 28 }}>
          <StatCard icon={Cpu} label="CPU Usage" value={`${stats.cpu.usage}%`}
            sub={`${stats.cpu.cores} cores · ${stats.cpu.model.substring(0, 30)}...`}
            percent={stats.cpu.usage} color="#3b82f6" />
          <StatCard icon={MemoryStick} label="Memory" value={`${Math.round(stats.memory.used / 1024)} GB`}
            sub={`${Math.round(stats.memory.total / 1024)} GB total`}
            percent={stats.memory.percent} color="#a855f7" />
          <StatCard icon={HardDrive} label="Disk Usage" value={stats.disk.total ? `${stats.disk.used} GB` : 'N/A'}
            sub={stats.disk.total ? `${stats.disk.total} GB total` : 'Linux only'}
            percent={stats.disk.percent || undefined} color="#f59e0b" />
          <StatCard icon={Clock} label="Uptime" value={stats.uptime} sub="System running" color="#22c55e" />
        </div>
      ) : null}

      {/* Services & PHP */}
      {stats && (
        <div className="card-grid-2" style={{ marginBottom: 28 }}>
          <div className="glass-card" style={{ padding: 20 }}>
            <div style={{ fontSize: 13, fontWeight: 600, color: 'var(--text-secondary)', marginBottom: 16 }}>
              Services
            </div>
            <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
              {[
                { label: 'Nginx', running: stats.services.nginx, icon: '🌐' },
                { label: 'Apache2', running: stats.services.apache, icon: '🔴' },
                { label: 'MySQL / MariaDB', running: true, icon: '🗄️' },
              ].map(svc => (
                <div key={svc.label} style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                  <span style={{ fontSize: 13, display: 'flex', alignItems: 'center', gap: 8 }}>
                    <span>{svc.icon}</span> {svc.label}
                  </span>
                  <span className={`badge ${svc.running ? 'badge-green' : 'badge-red'}`}>
                    {svc.running ? '● Running' : '○ Stopped'}
                  </span>
                </div>
              ))}
            </div>
          </div>

          <div className="glass-card" style={{ padding: 20 }}>
            <div style={{ fontSize: 13, fontWeight: 600, color: 'var(--text-secondary)', marginBottom: 16 }}>
              Environment
            </div>
            <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
              {[
                { label: 'PHP Version', value: stats.phpVersion, icon: <FileCode2 size={14} color="#a855f7" /> },
                { label: 'Hostname', value: stats.hostname, icon: <Globe size={14} color="#3b82f6" /> },
                { label: 'Platform', value: stats.platform, icon: <Database size={14} color="#06b6d4" /> },
              ].map(row => (
                <div key={row.label} style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                  <span style={{ fontSize: 13, display: 'flex', alignItems: 'center', gap: 8 }}>
                    {row.icon} {row.label}
                  </span>
                  <span className="mono" style={{ color: 'var(--text-primary)', fontSize: 12 }}>{row.value}</span>
                </div>
              ))}
            </div>
          </div>
        </div>
      )}

      {/* Quick actions */}
      <div>
        <h2 style={{ fontSize: 14, fontWeight: 600, color: 'var(--text-secondary)', marginBottom: 14,
          textTransform: 'uppercase', letterSpacing: '0.6px' }}>
          Quick Actions
        </h2>
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(5, 1fr)', gap: 12 }}>
          {quickActions.map(qa => (
            <Link key={qa.href} href={qa.href} style={{ textDecoration: 'none', color: 'inherit' }}>
              <div className="glass-card glass-card-hover" style={{ padding: 18, textAlign: 'center', cursor: 'pointer' }}>
                <div style={{ fontSize: 28, marginBottom: 10 }}>{qa.icon}</div>
                <div style={{ fontSize: 13, fontWeight: 600, marginBottom: 4 }}>{qa.label}</div>
                <div style={{ fontSize: 11, color: 'var(--text-muted)' }}>{qa.desc}</div>
              </div>
            </Link>
          ))}
        </div>
      </div>
    </div>
  );
}
