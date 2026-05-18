'use client';
import Link from 'next/link';
import { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Archive, Database, ExternalLink, FileCode2, FolderOpen, Globe, HardDriveDownload,
  Plus, RefreshCw, Server, Settings2, ShieldCheck, ToggleLeft, ToggleRight, Trash2,
  UserPlus, Wrench, X, AlertTriangle, CheckCircle2
} from 'lucide-react';

interface Site {
  domain: string;
  type: 'domain' | 'subdomain';
  docRoot: string;
  enabled: boolean;
  server: string;
  ssl: boolean;
}

interface SiteUser {
  username: string;
  role: 'owner' | 'developer' | 'viewer';
}

interface SiteSafety {
  findings: { path: string; type: string; severity: 'low' | 'medium' | 'high'; message: string }[];
  database: { detected: boolean; dbName?: string; dbUser?: string; exists?: boolean; message: string };
  mode: 'demo' | 'live';
}

const EMPTY_FORM = {
  domain: '',
  parentDomain: '',
  type: 'domain' as 'domain' | 'subdomain',
  siteType: 'php',
  phpVersion: '8.4',
  server: 'nginx',
};

export default function SitesPage() {
  const [sites, setSites] = useState<Site[]>([]);
  const [webserver, setWebserver] = useState('nginx');
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [demo, setDemo] = useState(false);
  const [form, setForm] = useState(EMPTY_FORM);
  const [showCreate, setShowCreate] = useState(false);
  const [selected, setSelected] = useState<Site | null>(null);
  const [message, setMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null);
  const [output, setOutput] = useState('');
  const [siteUsers, setSiteUsers] = useState<SiteUser[]>([]);
  const [userForm, setUserForm] = useState({ username: '', password: '', role: 'developer' as SiteUser['role'] });
  const [siteSafety, setSiteSafety] = useState<SiteSafety | null>(null);
  const [safetyLoading, setSafetyLoading] = useState(false);

  const showMessage = (type: 'success' | 'error', text: string) => {
    setMessage({ type, text });
    setTimeout(() => setMessage(null), 5000);
  };

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const response = await fetch('/api/domains');
      const data = await response.json();
      setSites(data.domains || []);
      setWebserver(data.webserver || 'nginx');
      setDemo(Boolean(data.demo));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    const id = setTimeout(() => { void load(); }, 0);
    return () => clearTimeout(id);
  }, [load]);

  const stats = useMemo(() => ({
    total: sites.length,
    active: sites.filter(site => site.enabled).length,
    ssl: sites.filter(site => site.ssl).length,
    subdomains: sites.filter(site => site.type === 'subdomain').length,
  }), [sites]);

  const createSite = async () => {
    const domain = form.type === 'subdomain' ? `${form.domain}.${form.parentDomain}` : form.domain;
    if (!domain) { showMessage('error', 'Enter a domain name'); return; }

    setBusy(true);
    setOutput('Creating site...');
    try {
      const response = await fetch('/api/domains', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ ...form, domain, parentDomain: form.parentDomain }),
      });
      const data = await response.json();
      setOutput(data.output || data.message || '');
      if (data.success) {
        showMessage('success', data.message);
        setShowCreate(false);
        setForm(EMPTY_FORM);
        load();
      } else {
        showMessage('error', data.error || 'Site creation failed');
      }
    } finally {
      setBusy(false);
    }
  };

  const siteAction = async (site: Site, action: 'enable' | 'disable' | 'permissions') => {
    setBusy(true);
    try {
      const response = await fetch('/api/domains', {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ domain: site.domain, action, server: site.server }),
      });
      const data = await response.json();
      if (data.success) {
        showMessage('success', data.message);
        load();
        setSelected(current => current?.domain === site.domain ? { ...current, enabled: action === 'enable' ? true : action === 'disable' ? false : current.enabled } : current);
      } else {
        showMessage('error', data.error || 'Action failed');
      }
    } finally {
      setBusy(false);
    }
  };

  const backupSite = async (site: Site) => {
    if (!confirm(`Create a full file backup for ${site.domain}? Large sites can take a while.`)) return;
    setBusy(true);
    setOutput(`Creating backup for ${site.domain}...`);
    try {
      const response = await fetch('/api/domains', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ domain: site.domain }),
      });
      const data = await response.json();
      setOutput(data.output || data.backupPath || data.message || '');
      if (data.success) showMessage('success', data.message);
      else showMessage('error', data.message || data.error || 'Backup failed');
    } finally {
      setBusy(false);
    }
  };

  const deleteSite = async (site: Site, deleteFiles: boolean) => {
    if (!confirm(`Delete ${site.domain}?${deleteFiles ? ' Site files will be soft-deleted to backup trash.' : ' Only the web server vhost will be removed.'}`)) return;
    setBusy(true);
    try {
      const response = await fetch('/api/domains', {
        method: 'DELETE',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ domain: site.domain, deleteFiles, server: site.server }),
      });
      const data = await response.json();
      if (data.success) {
        showMessage('success', data.message);
        setSelected(null);
        load();
      } else {
        showMessage('error', data.error || 'Delete failed');
      }
    } finally {
      setBusy(false);
    }
  };

  const loadSiteUsers = useCallback(async (domain: string) => {
    const response = await fetch(`/api/site-users?domain=${encodeURIComponent(domain)}`);
    const data = await response.json();
    setSiteUsers(data.users || []);
  }, []);

  const loadSiteSafety = useCallback(async (site: Site) => {
    setSafetyLoading(true);
    try {
      const response = await fetch(`/api/site-safety?domain=${encodeURIComponent(site.domain)}&docRoot=${encodeURIComponent(site.docRoot)}`);
      const data = await response.json();
      if (response.ok) setSiteSafety(data);
      else showMessage('error', data.error || 'Unable to inspect site safety');
    } finally {
      setSafetyLoading(false);
    }
  }, []);

  const openManager = (site: Site) => {
    setSelected(site);
    setUserForm({ username: '', password: '', role: 'developer' });
    setSiteSafety(null);
    void loadSiteUsers(site.domain);
    void loadSiteSafety(site);
  };

  const sanitizeSite = async (site: Site) => {
    if (!confirm(`Sanitize files and repair permissions for ${site.domain}? Secret-like files will be moved to quarantine.`)) return;
    setBusy(true);
    try {
      const response = await fetch('/api/site-safety', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ domain: site.domain, docRoot: site.docRoot, action: 'sanitize' }),
      });
      const data = await response.json();
      if (data.success) {
        showMessage('success', data.message);
        setOutput(`Quarantined: ${data.quarantined ?? 0}\nQuarantine: ${data.quarantineRoot || 'demo mode'}`);
        loadSiteSafety(site);
      } else {
        showMessage('error', data.error || 'Sanitize failed');
      }
    } finally {
      setBusy(false);
    }
  };

  const addSiteUser = async () => {
    if (!selected || !userForm.username) return;
    const response = await fetch('/api/site-users', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ domain: selected.domain, ...userForm }),
    });
    const data = await response.json();
    if (data.success) {
      showMessage('success', data.message);
      setUserForm({ username: '', password: '', role: 'developer' });
      loadSiteUsers(selected.domain);
    } else {
      showMessage('error', data.error || 'Unable to add user');
    }
  };

  const removeSiteUser = async (username: string) => {
    if (!selected || !confirm(`Remove ${username} access from ${selected.domain}?`)) return;
    const response = await fetch('/api/site-users', {
      method: 'DELETE',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ domain: selected.domain, username }),
    });
    const data = await response.json();
    if (data.success) {
      showMessage('success', data.message);
      loadSiteUsers(selected.domain);
    } else {
      showMessage('error', data.error || 'Unable to remove user');
    }
  };

  return (
    <div className="fade-in">
      <div className="page-header" style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 16 }}>
        <div>
          <h1 className="page-title">Sites</h1>
          <p className="page-subtitle">User mode workspace for adding sites and managing day-to-day site operations</p>
        </div>
        <div style={{ display: 'flex', gap: 8 }}>
          <button onClick={() => setShowCreate(true)} className="btn btn-primary btn-sm">
            <Plus size={14} /> Add Site
          </button>
          <button onClick={load} className="btn btn-ghost btn-sm" disabled={loading}>
            <RefreshCw size={14} style={{ animation: loading ? 'spin 1s linear infinite' : 'none' }} />
          </button>
        </div>
      </div>

      {demo && <div className="alert alert-warning">Demo mode: site actions are simulated on non-Linux systems.</div>}
      {message && <div className={`alert ${message.type === 'success' ? 'alert-success' : 'alert-error'}`}>{message.text}</div>}

      <div className="stat-grid" style={{ marginBottom: 20 }}>
        {[
          { label: 'Sites', value: stats.total, color: '#06b6d4' },
          { label: 'Active', value: stats.active, color: '#22c55e' },
          { label: 'SSL', value: stats.ssl, color: '#f59e0b' },
          { label: 'Subdomains', value: stats.subdomains, color: '#a855f7' },
        ].map(item => (
          <div key={item.label} className="glass-card" style={{ padding: 18 }}>
            <div style={{ fontSize: 26, fontWeight: 800, color: item.color }}>{item.value}</div>
            <div style={{ fontSize: 11, color: 'var(--text-muted)', textTransform: 'uppercase', letterSpacing: 0.5 }}>{item.label}</div>
          </div>
        ))}
      </div>

      <div className="glass-card" style={{ padding: 14, marginBottom: 18, display: 'flex', alignItems: 'center', gap: 10 }}>
        <Server size={16} color="#22c55e" />
        <span style={{ fontSize: 13, color: 'var(--text-secondary)' }}>Default server:</span>
        <strong style={{ fontSize: 13 }}>{webserver === 'apache' ? 'Apache2' : webserver === 'none' ? 'Not detected' : 'Nginx'}</strong>
        <span style={{ marginLeft: 'auto', fontSize: 12, color: 'var(--text-muted)' }}>Heavy server tasks keep running in the background APIs.</span>
      </div>

      {loading ? (
        <div className="glass-card" style={{ padding: 36, textAlign: 'center', color: 'var(--text-muted)' }}>
          <span className="spinner" /> Loading sites...
        </div>
      ) : sites.length === 0 ? (
        <div className="glass-card" style={{ padding: 48, textAlign: 'center' }}>
          <Globe size={42} color="var(--text-muted)" style={{ margin: '0 auto 14px' }} />
          <div style={{ fontSize: 16, fontWeight: 700, marginBottom: 6 }}>No sites yet</div>
          <div style={{ color: 'var(--text-muted)', fontSize: 13, marginBottom: 18 }}>Add a domain, PHP site, static site, or WordPress-ready site.</div>
          <button onClick={() => setShowCreate(true)} className="btn btn-primary btn-sm"><Plus size={14} /> Add Site</button>
        </div>
      ) : (
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(320px, 1fr))', gap: 12 }}>
          {sites.map(site => (
            <div key={site.domain} className="glass-card glass-card-hover" style={{ padding: 16 }}>
              <div style={{ display: 'flex', alignItems: 'flex-start', gap: 12, marginBottom: 14 }}>
                <div style={{
                  width: 40, height: 40, borderRadius: 8, display: 'flex', alignItems: 'center', justifyContent: 'center',
                  background: site.enabled ? 'rgba(6,182,212,0.12)' : 'rgba(139,148,158,0.1)',
                  border: `1px solid ${site.enabled ? 'rgba(6,182,212,0.25)' : 'var(--border-subtle)'}`,
                }}>
                  <Globe size={18} color={site.enabled ? '#06b6d4' : '#8b949e'} />
                </div>
                <div style={{ flex: 1, minWidth: 0 }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 4 }}>
                    <strong style={{ fontSize: 15 }}>{site.domain}</strong>
                    {site.ssl && <ShieldCheck size={13} color="#22c55e" />}
                  </div>
                  <div className="mono" style={{ color: 'var(--text-muted)', fontSize: 11, overflow: 'hidden', textOverflow: 'ellipsis' }}>{site.docRoot}</div>
                </div>
                <span className={`badge ${site.enabled ? 'badge-green' : 'badge-red'}`}>{site.enabled ? 'Active' : 'Off'}</span>
              </div>

              <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 8 }}>
                <button onClick={() => openManager(site)} className="btn btn-primary btn-sm" style={{ justifyContent: 'center' }}>
                  <Settings2 size={13} /> Manage Site
                </button>
                <a href={`http://${site.domain}`} target="_blank" rel="noopener noreferrer" className="btn btn-ghost btn-sm" style={{ justifyContent: 'center' }}>
                  <ExternalLink size={13} /> Open
                </a>
              </div>
            </div>
          ))}
        </div>
      )}

      {output && (
        <div style={{ marginTop: 20 }}>
          <div className="terminal">{output.split('\n').map((line, index) => <div key={index}>{line || ' '}</div>)}</div>
        </div>
      )}

      {showCreate && (
        <div style={{ position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.78)', display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 1000 }}>
          <div className="glass-card fade-in" style={{ padding: 24, width: 520, maxWidth: '95vw' }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 18 }}>
              <strong>Add Site</strong>
              <button onClick={() => setShowCreate(false)} className="btn btn-ghost btn-sm">Close</button>
            </div>
            <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
              <div style={{ display: 'flex', gap: 6 }}>
                {(['domain', 'subdomain'] as const).map(type => (
                  <button key={type} onClick={() => setForm({ ...form, type })} className={`btn ${form.type === type ? 'btn-primary' : 'btn-ghost'} btn-sm`} style={{ flex: 1, justifyContent: 'center' }}>
                    {type === 'domain' ? 'Domain' : 'Subdomain'}
                  </button>
                ))}
              </div>
              {form.type === 'subdomain' ? (
                <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 10 }}>
                  <input className="input mono" placeholder="shop" value={form.domain} onChange={event => setForm({ ...form, domain: event.target.value })} />
                  <input className="input mono" placeholder="example.com" value={form.parentDomain} onChange={event => setForm({ ...form, parentDomain: event.target.value })} />
                </div>
              ) : (
                <input className="input mono" placeholder="example.com" value={form.domain} onChange={event => setForm({ ...form, domain: event.target.value })} />
              )}
              <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: 10 }}>
                <select className="input" value={form.siteType} onChange={event => setForm({ ...form, siteType: event.target.value })}>
                  <option value="php">PHP site</option>
                  <option value="html">HTML/CSS site</option>
                  <option value="wordpress">WordPress ready</option>
                </select>
                <select className="input" value={form.phpVersion} onChange={event => setForm({ ...form, phpVersion: event.target.value })}>
                  {['8.5', '8.4', '8.3', '8.2'].map(version => <option key={version} value={version}>PHP {version}</option>)}
                </select>
                <select className="input" value={form.server} onChange={event => setForm({ ...form, server: event.target.value })}>
                  <option value="nginx">Nginx</option>
                  <option value="apache">Apache2</option>
                </select>
              </div>
              <button onClick={createSite} className="btn btn-primary" disabled={busy} style={{ justifyContent: 'center' }}>
                {busy ? <span className="spinner" /> : <Plus size={15} />} Create Site
              </button>
            </div>
          </div>
        </div>
      )}

      {selected && (
        <div style={{ position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.78)', display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 1000, padding: 20 }}>
          <div className="glass-card fade-in" style={{ width: 760, maxWidth: '100%', maxHeight: '92vh', overflowY: 'auto' }}>
            <div style={{ padding: 20, borderBottom: '1px solid var(--border-subtle)', display: 'flex', alignItems: 'center', gap: 12 }}>
              <Globe size={22} color="#06b6d4" />
              <div style={{ flex: 1 }}>
                <div style={{ fontSize: 18, fontWeight: 800 }}>{selected.domain}</div>
                <div className="mono" style={{ fontSize: 11, color: 'var(--text-muted)' }}>{selected.docRoot}</div>
              </div>
              <span className={`badge ${selected.enabled ? 'badge-green' : 'badge-red'}`}>{selected.enabled ? 'Active' : 'Disabled'}</span>
              <button onClick={() => setSelected(null)} className="btn btn-ghost btn-sm">Close</button>
            </div>

            <div style={{ padding: 20 }}>
              <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(170px, 1fr))', gap: 10 }}>
                <a href={`http://${selected.domain}`} target="_blank" rel="noopener noreferrer" className="btn btn-ghost" style={{ justifyContent: 'center' }}><ExternalLink size={15} /> Open Site</a>
                <Link href={`/dashboard/files?path=${encodeURIComponent(selected.docRoot)}`} className="btn btn-ghost" style={{ justifyContent: 'center' }}><FolderOpen size={15} /> Files</Link>
                <Link href={`/dashboard/ssl?domain=${encodeURIComponent(selected.domain)}`} className="btn btn-ghost" style={{ justifyContent: 'center' }}><ShieldCheck size={15} /> SSL</Link>
                <Link href={`/dashboard/wordpress?domain=${encodeURIComponent(selected.domain)}`} className="btn btn-ghost" style={{ justifyContent: 'center' }}><Globe size={15} /> WordPress</Link>
                <Link href="/dashboard/databases" className="btn btn-ghost" style={{ justifyContent: 'center' }}><Database size={15} /> Database</Link>
                <Link href="/dashboard/php" className="btn btn-ghost" style={{ justifyContent: 'center' }}><FileCode2 size={15} /> PHP</Link>
                <button onClick={() => siteAction(selected, selected.enabled ? 'disable' : 'enable')} className="btn btn-ghost" disabled={busy} style={{ justifyContent: 'center' }}>
                  {selected.enabled ? <ToggleRight size={15} /> : <ToggleLeft size={15} />} {selected.enabled ? 'Disable' : 'Enable'}
                </button>
                <button onClick={() => siteAction(selected, 'permissions')} className="btn btn-ghost" disabled={busy} style={{ justifyContent: 'center' }}><Wrench size={15} /> Fix Permissions</button>
                <button onClick={() => backupSite(selected)} className="btn btn-ghost" disabled={busy} style={{ justifyContent: 'center' }}><Archive size={15} /> Backup</button>
                <button onClick={() => deleteSite(selected, false)} className="btn btn-danger" disabled={busy} style={{ justifyContent: 'center' }}><Trash2 size={15} /> Delete Vhost</button>
                <button onClick={() => deleteSite(selected, true)} className="btn btn-danger" disabled={busy} style={{ justifyContent: 'center' }}><HardDriveDownload size={15} /> Delete All</button>
              </div>

              <div className="glass-card" style={{ padding: 16, marginTop: 18, background: 'var(--bg-base)' }}>
                <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(160px, 1fr))', gap: 12 }}>
                  <div><div style={{ color: 'var(--text-muted)', fontSize: 11 }}>Server</div><strong>{selected.server}</strong></div>
                  <div><div style={{ color: 'var(--text-muted)', fontSize: 11 }}>SSL</div><strong>{selected.ssl ? 'Installed' : 'Not installed'}</strong></div>
                  <div><div style={{ color: 'var(--text-muted)', fontSize: 11 }}>Type</div><strong>{selected.type}</strong></div>
                  <div><div style={{ color: 'var(--text-muted)', fontSize: 11 }}>Root</div><strong className="mono" style={{ fontSize: 11 }}>{selected.docRoot}</strong></div>
                </div>
              </div>

              <div className="glass-card" style={{ padding: 16, marginTop: 18, background: 'var(--bg-base)' }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', gap: 12, marginBottom: 12 }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                    {siteSafety && siteSafety.findings.length === 0 ? <CheckCircle2 size={16} color="var(--accent-green)" /> : <AlertTriangle size={16} color="var(--accent-yellow)" />}
                    <strong>File permissions & database</strong>
                  </div>
                  <div style={{ display: 'flex', gap: 8 }}>
                    <button onClick={() => loadSiteSafety(selected)} className="btn btn-ghost btn-sm" disabled={safetyLoading}>
                      <RefreshCw size={14} style={{ animation: safetyLoading ? 'spin 1s linear infinite' : 'none' }} /> Scan
                    </button>
                    <button onClick={() => sanitizeSite(selected)} className="btn btn-primary btn-sm" disabled={busy}>
                      <Wrench size={14} /> Sanitize
                    </button>
                  </div>
                </div>
                {!siteSafety ? (
                  <div style={{ fontSize: 12, color: 'var(--text-muted)' }}>{safetyLoading ? 'Scanning site...' : 'No scan loaded yet.'}</div>
                ) : (
                  <div style={{ display: 'grid', gap: 12 }}>
                    <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(180px, 1fr))', gap: 10 }}>
                      <div style={{ padding: 10, border: '1px solid var(--border-subtle)', borderRadius: 8 }}>
                        <div style={{ fontSize: 11, color: 'var(--text-muted)' }}>Findings</div>
                        <strong>{siteSafety.findings.length}</strong>
                      </div>
                      <div style={{ padding: 10, border: '1px solid var(--border-subtle)', borderRadius: 8 }}>
                        <div style={{ fontSize: 11, color: 'var(--text-muted)' }}>Database</div>
                        <strong>{siteSafety.database.detected ? (siteSafety.database.exists ? 'Exists' : 'Missing') : 'Not detected'}</strong>
                        {siteSafety.database.dbName && <div className="mono" style={{ fontSize: 11, color: 'var(--text-muted)' }}>{siteSafety.database.dbName}</div>}
                      </div>
                    </div>
                    {siteSafety.findings.length > 0 && (
                      <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
                        {siteSafety.findings.slice(0, 8).map((finding, index) => (
                          <div key={`${finding.path}-${index}`} style={{ display: 'flex', gap: 10, padding: 10, border: '1px solid var(--border-subtle)', borderRadius: 8, background: '#fff' }}>
                            <span className={`badge ${finding.severity === 'high' ? 'badge-red' : finding.severity === 'medium' ? 'badge-yellow' : 'badge-blue'}`}>{finding.severity}</span>
                            <div style={{ minWidth: 0 }}>
                              <div style={{ fontSize: 13, fontWeight: 700 }}>{finding.message}</div>
                              <div className="mono" style={{ fontSize: 11, color: 'var(--text-muted)', overflowWrap: 'anywhere' }}>{finding.path}</div>
                            </div>
                          </div>
                        ))}
                      </div>
                    )}
                  </div>
                )}
              </div>

              <div className="glass-card" style={{ padding: 16, marginTop: 18, background: 'var(--bg-base)' }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', gap: 12, marginBottom: 12 }}>
                  <strong>Site users</strong>
                  <span style={{ fontSize: 11, color: 'var(--text-muted)' }}>Per-site access roles</span>
                </div>
                <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 150px auto', gap: 8, marginBottom: 12 }}>
                  <input className="input" placeholder="username" value={userForm.username} onChange={event => setUserForm({ ...userForm, username: event.target.value })} />
                  <input className="input" type="password" placeholder="new user password" value={userForm.password} onChange={event => setUserForm({ ...userForm, password: event.target.value })} />
                  <select className="input" value={userForm.role} onChange={event => setUserForm({ ...userForm, role: event.target.value as SiteUser['role'] })}>
                    <option value="owner">Owner</option>
                    <option value="developer">Developer</option>
                    <option value="viewer">Viewer</option>
                  </select>
                  <button onClick={addSiteUser} className="btn btn-primary btn-sm" disabled={busy || !userForm.username}><UserPlus size={14} /> Add</button>
                </div>
                {siteUsers.length === 0 ? (
                  <div style={{ fontSize: 12, color: 'var(--text-muted)' }}>No extra users assigned.</div>
                ) : (
                  <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
                    {siteUsers.map(user => (
                      <div key={`${selected.domain}-${user.username}`} style={{ display: 'flex', alignItems: 'center', gap: 10, padding: '8px 10px', border: '1px solid var(--border-subtle)', borderRadius: 8 }}>
                        <strong style={{ flex: 1 }}>{user.username}</strong>
                        <span className="badge">{user.role}</span>
                        <button onClick={() => removeSiteUser(user.username)} className="btn btn-ghost btn-sm" title="Remove access"><X size={14} /></button>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
