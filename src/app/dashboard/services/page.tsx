'use client';
import { useEffect, useState, useCallback } from 'react';
import {
  Server, RefreshCw, Play, Square, RotateCcw, Download,
  CheckCircle, Terminal, Settings2, Zap
} from 'lucide-react';

interface ServiceStatus {
  id: string;
  name: string;
  category: string;
  installed: boolean;
  running: boolean;
  status: string;
  version: string;
  port: number | null;
}

const CATEGORY_LABELS: Record<string, { label: string; icon: string; color: string }> = {
  web:      { label: 'Web Servers',     icon: '🌐', color: '#06b6d4' },
  database: { label: 'Database',        icon: '🗄️', color: '#a855f7' },
  php:      { label: 'PHP-FPM',         icon: '🐘', color: '#8b5cf6' },
  security: { label: 'Security / SSL',  icon: '🔒', color: '#f59e0b' },
  tools:    { label: 'Tools',           icon: '🔧', color: '#3b82f6' },
};

function StatusBadge({ status }: { status: string }) {
  if (status === 'active')       return <span className="badge badge-green">● Running</span>;
  if (status === 'inactive')     return <span className="badge badge-red">○ Stopped</span>;
  if (status === 'installed')    return <span className="badge badge-blue">✓ Installed</span>;
  if (status === 'not-installed') return <span className="badge" style={{ background: 'rgba(139,148,158,0.1)', color: '#8b949e', border: '1px solid rgba(139,148,158,0.2)' }}>Not Installed</span>;
  return <span className="badge badge-yellow">{status}</span>;
}

export default function ServicesPage() {
  const [services, setServices] = useState<ServiceStatus[]>([]);
  const [loading, setLoading]   = useState(true);
  const [demo, setDemo]         = useState(false);
  const [output, setOutput]     = useState('');
  const [outputTitle, setOutputTitle] = useState('');
  const [busy, setBusy]         = useState<string | null>(null);
  const [msg, setMsg]           = useState<{ type: 'success' | 'error'; text: string } | null>(null);

  const showMsg = (type: 'success' | 'error', text: string) => {
    setMsg({ type, text });
    setTimeout(() => setMsg(null), 6000);
  };

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const r = await fetch('/api/services');
      const d = await r.json();
      setServices(d.services || []);
      setDemo(d.demo || false);
    } finally { setLoading(false); }
  }, []);

  useEffect(() => {
    const id = setTimeout(() => { void load(); }, 0);
    return () => clearTimeout(id);
  }, [load]);

  const doAction = async (serviceId: string, action: string, svcName: string) => {
    setBusy(`${serviceId}-${action}`);
    setOutputTitle(`${svcName} — ${action}`);
    setOutput(`Running: ${action} ${svcName}…\n`);
    try {
      const r = await fetch('/api/services', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ serviceId, action }),
      });
      const d = await r.json();
      setOutput(d.output || d.message || '');
      if (d.success) { showMsg('success', d.message); load(); }
      else showMsg('error', d.message || d.error);
    } finally { setBusy(null); }
  };

  const byCategory = Object.keys(CATEGORY_LABELS).reduce((acc, cat) => {
    acc[cat] = services.filter(s => s.category === cat);
    return acc;
  }, {} as Record<string, ServiceStatus[]>);

  const totalInstalled = services.filter(s => s.installed).length;
  const totalRunning   = services.filter(s => s.running).length;
  const totalServices  = services.length;

  return (
    <div className="fade-in">
      <div className="page-header" style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <div>
          <h1 className="page-title">Services Manager</h1>
          <p className="page-subtitle">Install, start, stop and configure all server services</p>
        </div>
        <button id="refresh-services-btn" onClick={load} className="btn btn-ghost btn-sm" disabled={loading}>
          <RefreshCw size={14} style={{ animation: loading ? 'spin 1s linear infinite' : 'none' }} />Refresh
        </button>
      </div>

      {demo && (
        <div className="alert alert-warning">
          ⚠️ Demo mode — Showing simulated service states. On a Linux VPS, real systemctl commands will run.
        </div>
      )}
      {msg && <div className={`alert ${msg.type === 'success' ? 'alert-success' : 'alert-error'}`}>{msg.text}</div>}

      {/* Overview stats */}
      <div className="card-grid-3" style={{ marginBottom: 24 }}>
        {[
          { label: 'Total Services', value: totalServices, icon: <Server size={20} color="#3b82f6" />, color: '#3b82f6' },
          { label: 'Installed', value: totalInstalled, icon: <CheckCircle size={20} color="#22c55e" />, color: '#22c55e' },
          { label: 'Running', value: totalRunning, icon: <Zap size={20} color="#f59e0b" />, color: '#f59e0b' },
        ].map(stat => (
          <div key={stat.label} className="glass-card" style={{ padding: 20, display: 'flex', alignItems: 'center', gap: 16 }}>
            <div style={{ width: 48, height: 48, background: `${stat.color}15`, border: `1px solid ${stat.color}25`, borderRadius: 10, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
              {stat.icon}
            </div>
            <div>
              <div style={{ fontSize: 28, fontWeight: 800, color: stat.color }}>{stat.value}</div>
              <div style={{ fontSize: 12, color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: '0.5px' }}>{stat.label}</div>
            </div>
          </div>
        ))}
      </div>

      {/* Service categories */}
      {loading ? (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
          {[...Array(3)].map((_, i) => (
            <div key={i} className="glass-card" style={{ padding: 20, height: 120 }}>
              <div style={{ background: 'var(--bg-elevated)', borderRadius: 6, height: 10, width: '30%', marginBottom: 16 }} />
              <div style={{ background: 'var(--bg-elevated)', borderRadius: 6, height: 8, width: '60%' }} />
            </div>
          ))}
        </div>
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 20 }}>
          {Object.entries(CATEGORY_LABELS).map(([cat, catInfo]) => {
            const catServices = byCategory[cat] || [];
            if (catServices.length === 0) return null;
            return (
              <div key={cat} className="glass-card" style={{ overflow: 'hidden' }}>
                {/* Category header */}
                <div style={{
                  padding: '14px 20px',
                  borderBottom: '1px solid var(--border-subtle)',
                  display: 'flex', alignItems: 'center', gap: 10,
                  background: `${catInfo.color}08`,
                }}>
                  <span style={{ fontSize: 18 }}>{catInfo.icon}</span>
                  <span style={{ fontWeight: 700, color: catInfo.color }}>{catInfo.label}</span>
                  <span style={{ marginLeft: 'auto', fontSize: 12, color: 'var(--text-muted)' }}>
                    {catServices.filter(s => s.running || s.status === 'installed').length}/{catServices.length} active
                  </span>
                </div>

                {/* Services in category */}
                {catServices.map((svc, i) => (
                  <div key={svc.id} style={{
                    padding: '14px 20px',
                    borderBottom: i < catServices.length - 1 ? '1px solid var(--border-subtle)' : 'none',
                    display: 'flex', alignItems: 'center', gap: 14,
                  }}>
                    {/* Status indicator */}
                    <div style={{
                      width: 10, height: 10, borderRadius: '50%', flexShrink: 0,
                      background: svc.running ? '#22c55e' : svc.installed ? '#f59e0b' : '#484f58',
                      boxShadow: svc.running ? '0 0 8px #22c55e80' : 'none',
                    }} />

                    {/* Name & version */}
                    <div style={{ flex: 1, minWidth: 0 }}>
                      <div style={{ fontWeight: 600, fontSize: 14 }}>{svc.name}</div>
                      <div style={{ fontSize: 11, color: 'var(--text-muted)', marginTop: 2 }}>
                        {svc.version ? `v${svc.version}` : 'Not installed'}
                        {svc.port && <span style={{ marginLeft: 8 }}>Port {svc.port}</span>}
                      </div>
                    </div>

                    {/* Status badge */}
                    <StatusBadge status={svc.status} />

                    {/* Action buttons */}
                    <div style={{ display: 'flex', gap: 6, flexShrink: 0 }}>
                      {!svc.installed ? (
                        <button
                          id={`install-${svc.id}`}
                          onClick={() => doAction(svc.id, 'install', svc.name)}
                          className="btn btn-primary btn-sm"
                          disabled={busy !== null}
                          style={{ minWidth: 80 }}
                        >
                          {busy === `${svc.id}-install`
                            ? <><span className="spinner" style={{ width: 12, height: 12 }} />Installing…</>
                            : <><Download size={12} />Install</>}
                        </button>
                      ) : (
                        <>
                          {svc.status === 'active' ? (
                            <>
                              <button id={`restart-${svc.id}`} onClick={() => doAction(svc.id, 'restart', svc.name)}
                                className="btn btn-ghost btn-sm" disabled={busy !== null} title="Restart">
                                {busy === `${svc.id}-restart` ? <span className="spinner" style={{ width: 12, height: 12 }} /> : <RotateCcw size={12} />}
                                Restart
                              </button>
                              <button id={`stop-${svc.id}`} onClick={() => doAction(svc.id, 'stop', svc.name)}
                                className="btn btn-danger btn-sm" disabled={busy !== null} title="Stop">
                                {busy === `${svc.id}-stop` ? <span className="spinner" style={{ width: 12, height: 12 }} /> : <Square size={12} />}
                                Stop
                              </button>
                            </>
                          ) : svc.status === 'inactive' ? (
                            <button id={`start-${svc.id}`} onClick={() => doAction(svc.id, 'start', svc.name)}
                              className="btn btn-success btn-sm" disabled={busy !== null} title="Start">
                              {busy === `${svc.id}-start` ? <span className="spinner" style={{ width: 12, height: 12 }} /> : <Play size={12} />}
                              Start
                            </button>
                          ) : (
                            <button id={`restart-${svc.id}`} onClick={() => doAction(svc.id, 'restart', svc.name)}
                              className="btn btn-ghost btn-sm" disabled={busy !== null}>
                              <RotateCcw size={12} />Restart
                            </button>
                          )}
                          {['nginx', 'apache', 'mysql', 'php85fpm', 'php84fpm', 'php83fpm', 'php82fpm'].includes(svc.id) && (
                            <button id={`test-${svc.id}`} onClick={() => doAction(svc.id, 'test', svc.name)}
                              className="btn btn-ghost btn-sm" disabled={busy !== null} title="Test config">
                              <Settings2 size={12} />Test
                            </button>
                          )}
                        </>
                      )}
                    </div>
                  </div>
                ))}
              </div>
            );
          })}
        </div>
      )}

      {/* Output terminal */}
      {output && (
        <div style={{ marginTop: 24 }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 8, fontSize: 13, fontWeight: 600, color: 'var(--text-secondary)' }}>
            <Terminal size={14} />{outputTitle || 'Command Output'}
          </div>
          <div className="terminal">
            {output.split('\n').map((line, i) => (
              <div key={i} className={
                line.includes('OK') || line.includes('success') || line.startsWith('✓') ? 'line-success' :
                  line.includes('error') || line.includes('ERROR') || line.startsWith('✗') ? 'line-error' :
                    line.startsWith('Running:') ? 'line-cmd' : ''}>
                {line || ' '}
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Quick install guide */}
      {!loading && services.some(s => !s.installed) && (
        <div className="glass-card" style={{ marginTop: 24, padding: 20 }}>
          <div style={{ fontWeight: 600, fontSize: 14, marginBottom: 14, color: 'var(--text-secondary)' }}>
            💡 Quick Install Guide
          </div>
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(260px, 1fr))', gap: 12 }}>
            {[
              { title: 'Light LEMP Stack', desc: 'Nginx + MariaDB + PHP 8.4 for 1GB VPS', services: ['nginx', 'mysql', 'php84fpm'], color: '#22c55e' },
              { title: 'FTP + Database Tools', desc: 'Pure-FTPd + phpMyAdmin', services: ['pureftpd', 'phpmyadmin'], color: '#3b82f6' },
              { title: 'SSL + WordPress Tools', desc: 'Certbot + WP-CLI', services: ['certbot', 'wpcli'], color: '#f59e0b' },
            ].map(bundle => (
              <div key={bundle.title} style={{
                padding: 16, borderRadius: 10, border: `1px solid ${bundle.color}20`,
                background: `${bundle.color}06`
              }}>
                <div style={{ fontWeight: 700, marginBottom: 4, color: bundle.color }}>{bundle.title}</div>
                <div style={{ fontSize: 12, color: 'var(--text-muted)', marginBottom: 12 }}>{bundle.desc}</div>
                <button
                  id={`install-bundle-${bundle.title.replace(/\s/g, '-')}`}
                  onClick={async () => {
                    for (const id of bundle.services) {
                      const svc = services.find(s => s.id === id);
                      if (svc && !svc.installed) await doAction(id, 'install', svc.name);
                    }
                  }}
                  className="btn btn-ghost btn-sm"
                  style={{ borderColor: `${bundle.color}30`, color: bundle.color }}
                  disabled={busy !== null}
                >
                  <Download size={12} />Install Bundle
                </button>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
