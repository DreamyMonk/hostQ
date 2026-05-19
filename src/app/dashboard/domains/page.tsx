'use client';
import { useEffect, useState, useCallback } from 'react';
import {
  Globe, Plus, Trash2, RefreshCw,
  ExternalLink, FolderOpen, ChevronRight, Terminal,
  ToggleLeft, ToggleRight, ShieldCheck, Server
} from 'lucide-react';

interface Domain {
  domain: string;
  type: 'domain' | 'subdomain';
  docRoot: string;
  enabled: boolean;
  server: string;
  ssl: boolean;
}

export default function DomainsPage() {
  const [domains, setDomains]     = useState<Domain[]>([]);
  const [webserver, setWebserver] = useState('nginx');
  const [loading, setLoading]     = useState(true);
  const [demo, setDemo]           = useState(false);
  const [busy, setBusy]           = useState(false);
  const [output, setOutput]       = useState('');
  const [msg, setMsg]             = useState<{ type: 'success' | 'error'; text: string } | null>(null);
  const [showForm, setShowForm]   = useState(false);
  const [tab, setTab]             = useState<'domain' | 'subdomain'>('domain');

  const [form, setForm] = useState({
    domain: '', parentDomain: '', phpVersion: '8.4',
    server: 'nginx', type: 'domain' as 'domain' | 'subdomain',
    siteType: 'php',
  });

  const showMsg = (type: 'success' | 'error', text: string) => {
    setMsg({ type, text });
    setTimeout(() => setMsg(null), 5000);
  };

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const r = await fetch('/api/domains');
      const d = await r.json();
      setDomains(d.domains || []);
      setWebserver(d.webserver || 'nginx');
      setDemo(d.demo || false);
    } finally { setLoading(false); }
  }, []);

  useEffect(() => {
    const id = setTimeout(() => { void load(); }, 0);
    return () => clearTimeout(id);
  }, [load]);

  const addDomain = async () => {
    const domainVal = tab === 'subdomain'
      ? `${form.domain}.${form.parentDomain}`
      : form.domain;

    if (!domainVal) { showMsg('error', 'Enter a domain name'); return; }
    setBusy(true);
    setOutput('Creating domain…\n');
    try {
      const r = await fetch('/api/domains', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          domain: domainVal,
          type: tab,
          phpVersion: form.phpVersion,
          server: form.server,
          parentDomain: form.parentDomain,
          siteType: form.siteType,
        }),
      });
      const d = await r.json();
      setOutput(d.output || d.message || '');
      if (d.success) { showMsg('success', d.message); setShowForm(false); load(); }
      else showMsg('error', d.error || 'Failed');
    } finally { setBusy(false); }
  };

  const toggleDomain = async (domain: Domain) => {
    setBusy(true);
    try {
      const r = await fetch('/api/domains', {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ domain: domain.domain, action: domain.enabled ? 'disable' : 'enable', server: domain.server }),
      });
      const d = await r.json();
      if (d.success) { showMsg('success', d.message); load(); }
      else showMsg('error', d.error);
    } finally { setBusy(false); }
  };

  const deleteDomain = async (domain: Domain, deleteFiles: boolean) => {
    if (!confirm(`Delete ${domain.domain}?${deleteFiles ? ' (Files will be deleted!)' : ''}`)) return;
    setBusy(true);
    try {
      const r = await fetch('/api/domains', {
        method: 'DELETE',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ domain: domain.domain, deleteFiles, server: domain.server }),
      });
      const d = await r.json();
      if (d.success) { showMsg('success', d.message); load(); }
      else showMsg('error', d.error);
    } finally { setBusy(false); }
  };

  const rootDomains    = domains.filter(d => d.type === 'domain');
  const subdomains     = domains.filter(d => d.type === 'subdomain');
  const activeDomains  = domains.filter(d => d.enabled).length;
  const sslDomains     = domains.filter(d => d.ssl).length;

  return (
    <div className="fade-in">
      {/* Header */}
      <div className="page-header" style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <div>
          <h1 className="page-title">Domain Manager</h1>
          <p className="page-subtitle">Add, configure and manage domains & subdomains</p>
        </div>
        <div style={{ display: 'flex', gap: 8 }}>
          <button id="add-domain-btn" onClick={() => { setShowForm(true); setTab('domain'); }} className="btn btn-primary btn-sm">
            <Plus size={14} /> Add Domain
          </button>
          <button id="add-subdomain-btn" onClick={() => { setShowForm(true); setTab('subdomain'); }} className="btn btn-ghost btn-sm">
            <Plus size={14} /> Add Subdomain
          </button>
          <button onClick={load} className="btn btn-ghost btn-sm">
            <RefreshCw size={14} style={{ animation: loading ? 'spin 1s linear infinite' : 'none' }} />
          </button>
        </div>
      </div>

      {demo && (
        <div className="alert alert-warning">
          ⚠️ Demo mode — running on Windows/macOS. On a Linux VPS, real Nginx/Apache vhosts will be created and managed.
        </div>
      )}
      {msg && <div className={`alert ${msg.type === 'success' ? 'alert-success' : 'alert-error'}`}>{msg.text}</div>}

      {/* Stats */}
      <div className="stat-grid" style={{ marginBottom: 24 }}>
        {[
          { label: 'Total Domains', value: domains.length, icon: '🌐', color: '#06b6d4' },
          { label: 'Subdomains', value: subdomains.length, icon: '🔗', color: '#a855f7' },
          { label: 'Active', value: activeDomains, icon: '✅', color: '#22c55e' },
          { label: 'SSL Secured', value: sslDomains, icon: '🔒', color: '#f59e0b' },
        ].map(stat => (
          <div key={stat.label} className="glass-card" style={{ padding: 20 }}>
            <div style={{ fontSize: 24, marginBottom: 6 }}>{stat.icon}</div>
            <div style={{ fontSize: 26, fontWeight: 800, color: stat.color }}>{stat.value}</div>
            <div style={{ fontSize: 11, color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: '0.5px' }}>{stat.label}</div>
          </div>
        ))}
      </div>

      {/* Active web server banner */}
      <div style={{
        padding: '12px 18px', marginBottom: 20, borderRadius: 10,
        background: webserver === 'nginx' ? 'rgba(34,197,94,0.08)' : 'rgba(59,130,246,0.08)',
        border: `1px solid ${webserver === 'nginx' ? 'rgba(34,197,94,0.2)' : 'rgba(59,130,246,0.2)'}`,
        display: 'flex', alignItems: 'center', gap: 12
      }}>
        <Server size={16} color={webserver === 'nginx' ? '#22c55e' : '#3b82f6'} />
        <span style={{ fontSize: 13, fontWeight: 600 }}>
          Active Web Server: <span style={{ color: webserver === 'nginx' ? '#22c55e' : '#3b82f6' }}>
            {webserver === 'nginx' ? 'Nginx' : webserver === 'apache' ? 'Apache2' : 'Not detected'}
          </span>
        </span>
        <span style={{ marginLeft: 'auto', fontSize: 12, color: 'var(--text-muted)' }}>
          Domains will use this server by default
        </span>
      </div>

      {/* Domains table */}
      {loading ? (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
          {[...Array(3)].map((_, i) => (
            <div key={i} className="glass-card" style={{ padding: 16, height: 56 }}>
              <div style={{ background: 'var(--bg-elevated)', borderRadius: 6, height: 10, width: '40%' }} />
            </div>
          ))}
        </div>
      ) : domains.length === 0 ? (
        <div className="glass-card" style={{ padding: 48, textAlign: 'center' }}>
          <Globe size={42} color="var(--text-muted)" style={{ margin: '0 auto 14px' }} />
          <div style={{ fontSize: 15, fontWeight: 600, marginBottom: 6 }}>No domains configured</div>
          <div style={{ color: 'var(--text-muted)', fontSize: 13, marginBottom: 18 }}>
            Add your first domain to start serving websites
          </div>
          <button onClick={() => { setShowForm(true); setTab('domain'); }} className="btn btn-primary btn-sm">
            <Plus size={14} /> Add Domain
          </button>
        </div>
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
          {/* Root domains with their subdomains */}
          {rootDomains.map(d => {
            const subs = subdomains.filter(s => s.domain.endsWith(`.${d.domain}`));
            return (
              <div key={d.domain} className="glass-card" style={{ overflow: 'hidden' }}>
                {/* Domain row */}
                <div style={{
                  padding: '14px 18px', display: 'flex', alignItems: 'center', gap: 12,
                  borderBottom: subs.length > 0 ? '1px solid var(--border-subtle)' : 'none'
                }}>
                  <div style={{
                    width: 36, height: 36, borderRadius: 8,
                    background: d.enabled ? 'rgba(6,182,212,0.12)' : 'rgba(139,148,158,0.1)',
                    border: `1px solid ${d.enabled ? 'rgba(6,182,212,0.25)' : 'var(--border-subtle)'}`,
                    display: 'flex', alignItems: 'center', justifyContent: 'center',
                  }}>
                    <Globe size={16} color={d.enabled ? '#06b6d4' : '#8b949e'} />
                  </div>

                  <div style={{ flex: 1 }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 2 }}>
                      <span style={{ fontWeight: 700, fontSize: 15 }}>{d.domain}</span>
                      {d.ssl && <ShieldCheck size={13} color="#22c55e" />}
                      <span className={`badge ${d.enabled ? 'badge-green' : 'badge-red'}`}>
                        {d.enabled ? 'Active' : 'Disabled'}
                      </span>
                      {subs.length > 0 && (
                        <span className="badge badge-purple">{subs.length} subdomain{subs.length > 1 ? 's' : ''}</span>
                      )}
                    </div>
                    <div style={{ fontSize: 11, color: 'var(--text-muted)', fontFamily: 'monospace' }}>
                      {d.docRoot}
                    </div>
                  </div>

                  <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                    <a href={`http://${d.domain}`} target="_blank" rel="noopener noreferrer"
                      className="btn btn-ghost btn-sm" title="Open in browser">
                      <ExternalLink size={12} />
                    </a>
                    <button id={`toggle-${d.domain}`} onClick={() => toggleDomain(d)} className="btn btn-ghost btn-sm" title={d.enabled ? 'Disable' : 'Enable'} disabled={busy}>
                      {d.enabled ? <ToggleRight size={16} color="#22c55e" /> : <ToggleLeft size={16} />}
                    </button>
                    <button id={`files-${d.domain}`} className="btn btn-ghost btn-sm" title="File Manager">
                      <FolderOpen size={12} />
                    </button>
                    <button id={`delete-domain-${d.domain}`} onClick={() => deleteDomain(d, false)} className="btn btn-danger btn-sm" disabled={busy}>
                      <Trash2 size={12} />
                    </button>
                  </div>
                </div>

                {/* Subdomains */}
                {subs.map(sub => (
                  <div key={sub.domain} style={{
                    padding: '10px 18px 10px 54px', display: 'flex', alignItems: 'center', gap: 12,
                    borderBottom: '1px solid var(--border-subtle)', background: 'rgba(255,255,255,0.015)',
                  }}>
                    <ChevronRight size={12} color="var(--text-muted)" />
                    <Globe size={13} color="#a855f7" />
                    <div style={{ flex: 1 }}>
                      <span style={{ fontWeight: 500, fontSize: 13, color: '#a855f7' }}>{sub.domain}</span>
                      <span style={{ fontSize: 11, color: 'var(--text-muted)', marginLeft: 10, fontFamily: 'monospace' }}>
                        {sub.docRoot}
                      </span>
                    </div>
                    <span className={`badge ${sub.enabled ? 'badge-green' : 'badge-red'}`} style={{ fontSize: 10 }}>
                      {sub.enabled ? 'Active' : 'Off'}
                    </span>
                    {sub.ssl && <ShieldCheck size={12} color="#22c55e" />}
                    <div style={{ display: 'flex', gap: 4 }}>
                      <button id={`toggle-sub-${sub.domain}`} onClick={() => toggleDomain(sub)} className="btn btn-ghost btn-sm" disabled={busy}>
                        {sub.enabled ? <ToggleRight size={14} color="#22c55e" /> : <ToggleLeft size={14} />}
                      </button>
                      <button id={`delete-sub-${sub.domain}`} onClick={() => deleteDomain(sub, false)} className="btn btn-danger btn-sm" disabled={busy}>
                        <Trash2 size={11} />
                      </button>
                    </div>
                  </div>
                ))}
              </div>
            );
          })}

          {/* Orphan subdomains (no parent in list) */}
          {subdomains.filter(s => !rootDomains.some(r => s.domain.endsWith(`.${r.domain}`))).map(sub => (
            <div key={sub.domain} className="glass-card" style={{ padding: '12px 18px', display: 'flex', alignItems: 'center', gap: 12 }}>
              <Globe size={16} color="#a855f7" />
              <div style={{ flex: 1 }}>
                <span style={{ fontWeight: 600, fontSize: 14 }}>{sub.domain}</span>
                <span className="badge badge-purple" style={{ marginLeft: 8 }}>subdomain</span>
              </div>
              <button id={`delete-orphan-${sub.domain}`} onClick={() => deleteDomain(sub, false)} className="btn btn-danger btn-sm" disabled={busy}>
                <Trash2 size={12} />
              </button>
            </div>
          ))}
        </div>
      )}

      {/* Output terminal */}
      {output && (
        <div style={{ marginTop: 20 }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 8, fontSize: 13, fontWeight: 600, color: 'var(--text-secondary)' }}>
            <Terminal size={14} />Command Output
          </div>
          <div className="terminal">
            {output.split('\n').map((line, i) => (
              <div key={i} className={
                line.startsWith('✓') ? 'line-success' :
                  line.startsWith('✗') || line.includes('ERROR') ? 'line-error' :
                    line.startsWith('▶') ? 'line-info' : ''}>
                {line || ' '}
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Add Domain/Subdomain Modal */}
      {showForm && (
        <div style={{ position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.78)', display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 1000 }}>
          <div className="glass-card fade-in" style={{ padding: 28, width: 500, maxWidth: '95vw' }}>
            <div style={{ fontWeight: 800, fontSize: 17, marginBottom: 20, display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
              <span>{tab === 'domain' ? '🌐 Add Domain' : '🔗 Add Subdomain'}</span>
              <button onClick={() => setShowForm(false)} style={{ background: 'none', border: 'none', cursor: 'pointer', color: 'var(--text-muted)', fontSize: 18 }}>✕</button>
            </div>

            {/* Tab switcher */}
            <div style={{ display: 'flex', gap: 4, marginBottom: 20, background: 'var(--bg-elevated)', borderRadius: 8, padding: 4 }}>
              {(['domain', 'subdomain'] as const).map(t => (
                <button key={t} id={`tab-type-${t}`} onClick={() => setTab(t)}
                  style={{
                    flex: 1, padding: '7px 12px', borderRadius: 6, border: 'none', cursor: 'pointer',
                    background: tab === t ? 'var(--bg-card)' : 'transparent',
                    color: tab === t ? 'var(--text-primary)' : 'var(--text-muted)',
                    fontWeight: tab === t ? 600 : 400, fontSize: 13,
                    borderBottom: tab === t ? '1px solid var(--border-active)' : 'none',
                  }}>
                  {t === 'domain' ? '🌐 Domain' : '🔗 Subdomain'}
                </button>
              ))}
            </div>

            <div style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
              {tab === 'subdomain' ? (
                <>
                  <div style={{ display: 'grid', gridTemplateColumns: '1fr auto 1fr', gap: 8, alignItems: 'end' }}>
                    <div>
                      <label style={{ fontSize: 13, fontWeight: 500, color: 'var(--text-secondary)', display: 'block', marginBottom: 5 }}>Subdomain Prefix</label>
                      <input id="sub-prefix" className="input mono" placeholder="shop" value={form.domain}
                        onChange={e => setForm({ ...form, domain: e.target.value })} />
                    </div>
                    <div style={{ paddingBottom: 10, fontSize: 18, color: 'var(--text-muted)', fontWeight: 700 }}>.</div>
                    <div>
                      <label style={{ fontSize: 13, fontWeight: 500, color: 'var(--text-secondary)', display: 'block', marginBottom: 5 }}>Parent Domain</label>
                      <input id="sub-parent" className="input mono" placeholder="example.com" value={form.parentDomain}
                        onChange={e => setForm({ ...form, parentDomain: e.target.value })} />
                    </div>
                  </div>
                  {form.domain && form.parentDomain && (
                    <div style={{ fontSize: 12, color: '#06b6d4', padding: '6px 10px', background: 'rgba(6,182,212,0.08)', borderRadius: 6 }}>
                      Full subdomain: <strong>{form.domain}.{form.parentDomain}</strong>
                    </div>
                  )}
                </>
              ) : (
                <div>
                  <label style={{ fontSize: 13, fontWeight: 500, color: 'var(--text-secondary)', display: 'block', marginBottom: 5 }}>Domain Name</label>
                  <input id="domain-input" className="input mono" placeholder="example.com" value={form.domain}
                    onChange={e => setForm({ ...form, domain: e.target.value })} />
                </div>
              )}

              <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12 }}>
                <div>
                  <label style={{ fontSize: 13, fontWeight: 500, color: 'var(--text-secondary)', display: 'block', marginBottom: 5 }}>Web Server</label>
                  <select id="domain-server" className="input" value={form.server} onChange={e => setForm({ ...form, server: e.target.value })}>
                    <option value="nginx">Nginx (recommended)</option>
                    <option value="apache">Apache2</option>
                  </select>
                </div>
                <div>
                  <label style={{ fontSize: 13, fontWeight: 500, color: 'var(--text-secondary)', display: 'block', marginBottom: 5 }}>PHP Version</label>
                  <select id="domain-php" className="input" value={form.phpVersion} onChange={e => setForm({ ...form, phpVersion: e.target.value })}>
                    {['8.5', '8.4', '8.3', '8.2'].map(v => (
                      <option key={v} value={v}>PHP {v}</option>
                    ))}
                  </select>
                </div>
              </div>
              <div>
                <label style={{ fontSize: 13, fontWeight: 500, color: 'var(--text-secondary)', display: 'block', marginBottom: 5 }}>Website Type</label>
                <select id="domain-site-type" className="input" value={form.siteType} onChange={e => setForm({ ...form, siteType: e.target.value })}>
                  <option value="php">PHP website</option>
                  <option value="html">HTML / CSS static website</option>
                  <option value="wordpress">WordPress ready folder</option>
                </select>
              </div>

              {/* Preview doc root */}
              <div style={{ fontSize: 12, color: 'var(--text-muted)', padding: '8px 12px', background: 'var(--bg-base)', borderRadius: 6 }}>
                📁 Document root: <span className="mono" style={{ color: '#f59e0b' }}>
                  {tab === 'subdomain' && form.domain && form.parentDomain
                    ? `/var/www/${form.parentDomain}/htdocs/${form.domain}`
                    : form.domain
                      ? `/var/www/${form.domain}/htdocs`
                      : '/var/www/[domain]/htdocs'}
                </span>
              </div>

              <div style={{ display: 'flex', gap: 8, justifyContent: 'flex-end', marginTop: 8 }}>
                <button onClick={() => setShowForm(false)} className="btn btn-ghost">Cancel</button>
                <button id="submit-domain-btn" onClick={addDomain} className="btn btn-primary" disabled={busy}>
                  {busy ? <span className="spinner" /> : <Globe size={14} />}
                  {busy ? 'Creating…' : tab === 'subdomain' ? 'Add Subdomain' : 'Add Domain'}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
