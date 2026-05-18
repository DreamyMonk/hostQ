'use client';
import { useEffect, useState } from 'react';
import { Globe, Plus, ExternalLink, RefreshCw, Terminal, Trash2, CheckCircle, FolderOpen } from 'lucide-react';

interface WPInstall { domain: string; path: string; status: string; wpVersion: string; db: string; }

export default function WordPressPage() {
  const [loading, setLoading]   = useState(true);
  const [installs, setInstalls] = useState<WPInstall[]>([]);
  const [demo, setDemo]         = useState(false);
  const [busy, setBusy]         = useState(false);
  const [output, setOutput]     = useState('');
  const [loginUrl, setLoginUrl] = useState('');
  const [msg, setMsg]           = useState<{ type: 'success'|'error'; text: string } | null>(null);
  const [showForm, setShowForm] = useState(false);

  const [form, setForm] = useState({
    domain: '', dbName: '', dbUser: '', dbPassword: '',
    adminEmail: '', siteTitle: '', adminUser: 'admin', adminPass: ''
  });

  const showMsg = (type: 'success'|'error', text: string) => {
    setMsg({ type, text });
    setTimeout(() => setMsg(null), 6000);
  };

  const load = async () => {
    setLoading(true);
    try {
      const r = await fetch('/api/wordpress');
      const d = await r.json();
      setInstalls(d.installations || []);
      setDemo(d.demo || false);
    } finally { setLoading(false); }
  };

  useEffect(() => {
    const id = setTimeout(() => { void load(); }, 0);
    return () => clearTimeout(id);
  }, []);

  const autoFill = (domain: string) => {
    const slug = domain.replace(/[^a-z0-9]/gi, '_').toLowerCase().substring(0, 16);
    setForm(f => ({
      ...f,
      domain,
      dbName: `wp_${slug}`,
      dbUser: `wp_${slug}`.substring(0, 16),
      dbPassword: `Wp${Math.random().toString(36).slice(2,10)}!`,
    }));
  };

  const install = async () => {
    if (!form.domain || !form.dbName || !form.dbUser || !form.dbPassword || !form.adminEmail) {
      showMsg('error', 'Please fill all required fields'); return;
    }
    setBusy(true);
    setOutput('Starting WordPress installation…\n');
    try {
      const r = await fetch('/api/wordpress', {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(form),
      });
      const d = await r.json();
      setOutput(d.output || '');
      if (d.success) {
        setLoginUrl(d.loginUrl || '');
        showMsg('success', d.message);
        setShowForm(false);
        load();
      } else {
        showMsg('error', d.error || 'Installation failed');
      }
    } finally { setBusy(false); }
  };

  const deleteSite = async (wp: WPInstall) => {
    if (!confirm(`Delete WordPress site ${wp.domain}?`)) return;
    setBusy(true);
    try {
      const r = await fetch('/api/wordpress', {
        method: 'DELETE',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ path: wp.path, dbName: wp.db, deleteFiles: true, deleteDatabase: Boolean(wp.db) }),
      });
      const d = await r.json();
      setOutput(d.output || d.message || '');
      if (d.success) { showMsg('success', d.message); load(); }
      else showMsg('error', d.error || 'Delete failed');
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="fade-in">
      <div className="page-header" style={{ display:'flex', alignItems:'center', justifyContent:'space-between' }}>
        <div>
          <h1 className="page-title">WordPress Installer</h1>
          <p className="page-subtitle">One-click WordPress install via WP-CLI</p>
        </div>
        <div style={{ display:'flex', gap:8 }}>
          <button id="install-wp-btn" onClick={() => setShowForm(true)} className="btn btn-primary btn-sm">
            <Plus size={14}/>Install WordPress
          </button>
          <button onClick={load} className="btn btn-ghost btn-sm">
            <RefreshCw size={14} style={{ animation: loading ? 'spin 1s linear infinite' : 'none' }}/>
          </button>
        </div>
      </div>

      {demo && (
        <div className="alert alert-warning">
          ⚠️ Demo mode — WP-CLI not detected. On a Linux VPS with WP-CLI installed, real installations will appear here.
        </div>
      )}
      {msg && <div className={`alert ${msg.type === 'success' ? 'alert-success' : 'alert-error'}`}>{msg.text}</div>}

      {/* Success banner after install */}
      {loginUrl && (
        <div style={{
          padding:'16px 20px', marginBottom:20, borderRadius:10,
          background:'linear-gradient(135deg, rgba(34,197,94,0.12), rgba(6,182,212,0.08))',
          border:'1px solid rgba(34,197,94,0.25)',
          display:'flex', alignItems:'center', justifyContent:'space-between'
        }}>
          <div style={{ display:'flex', alignItems:'center', gap:12 }}>
            <CheckCircle size={22} color="#22c55e" />
            <div>
              <div style={{ fontWeight:700, color:'#22c55e' }}>WordPress Installed Successfully!</div>
              <div style={{ fontSize:12, color:'var(--text-muted)', marginTop:2 }}>{loginUrl}</div>
            </div>
          </div>
          <a href={loginUrl} target="_blank" rel="noopener noreferrer" className="btn btn-success btn-sm">
            <ExternalLink size={13}/>Open WP Admin
          </a>
        </div>
      )}

      {/* What you need info */}
      <div className="card-grid-3" style={{ marginBottom:24 }}>
        {[
          { icon:'🐘', title:'PHP 8.x', desc:'Recommended for WordPress 6+' },
          { icon:'🗄️', title:'MySQL/MariaDB', desc:'Database for your WP site' },
          { icon:'⚡', title:'WP-CLI', desc:'Used for automated installation' },
        ].map(item => (
          <div key={item.title} className="glass-card" style={{ padding:18, display:'flex', alignItems:'center', gap:14 }}>
            <span style={{ fontSize:24 }}>{item.icon}</span>
            <div>
              <div style={{ fontWeight:600, fontSize:14 }}>{item.title}</div>
              <div style={{ fontSize:11, color:'var(--text-muted)' }}>{item.desc}</div>
            </div>
          </div>
        ))}
      </div>

      {/* Installations list */}
      <div className="glass-card" style={{ marginBottom:24 }}>
        <div style={{ padding:'16px 20px', borderBottom:'1px solid var(--border-subtle)', fontWeight:600 }}>
          WordPress Installations ({installs.length})
        </div>
        {loading ? (
          <div style={{ padding:40, textAlign:'center', color:'var(--text-muted)' }}>
            <div className="spinner" style={{ margin:'0 auto 10px' }}/>Loading…
          </div>
        ) : installs.length === 0 ? (
          <div style={{ padding:40, textAlign:'center' }}>
            <Globe size={36} color="var(--text-muted)" style={{ margin:'0 auto 10px' }}/>
            <div style={{ color:'var(--text-muted)', fontSize:14, marginBottom:12 }}>No WordPress installations found</div>
            <button onClick={() => setShowForm(true)} className="btn btn-primary btn-sm">
              <Plus size={14}/>Install WordPress
            </button>
          </div>
        ) : (
          <table className="data-table">
            <thead><tr><th>Domain</th><th>Path</th><th>WP Version</th><th>Status</th><th>Actions</th></tr></thead>
            <tbody>
              {installs.map(wp => (
                <tr key={wp.domain}>
                  <td style={{ fontWeight:600 }}>
                    <div style={{ display:'flex', alignItems:'center', gap:8 }}>
                      <Globe size={14} color="#3b82f6"/>{wp.domain}
                    </div>
                  </td>
                  <td><span className="mono" style={{ fontSize:12, color:'var(--text-muted)' }}>{wp.path}</span></td>
                  <td style={{ color:'var(--text-muted)' }}>{wp.wpVersion}</td>
                  <td><span className="badge badge-green">● {wp.status}</span></td>
                  <td>
                    <div style={{ display:'flex', gap:6 }}>
                      <a id={`wp-admin-${wp.domain}`} href={`http://${wp.domain}/wp-admin`} target="_blank" rel="noopener noreferrer" className="btn btn-ghost btn-sm">
                        <ExternalLink size={12}/>WP Admin
                      </a>
                      <a href={`/dashboard/files?path=${encodeURIComponent(wp.path)}`} className="btn btn-ghost btn-sm">
                        <FolderOpen size={12}/>Files
                      </a>
                      <button onClick={() => deleteSite(wp)} className="btn btn-danger btn-sm" disabled={busy}>
                        <Trash2 size={12}/>Delete
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      {/* Terminal output */}
      {output && (
        <div>
          <div style={{ display:'flex', alignItems:'center', gap:8, marginBottom:8, fontSize:13, fontWeight:600, color:'var(--text-secondary)' }}>
            <Terminal size={14}/>Installation Log
          </div>
          <div className="terminal">
            {output.split('\n').map((line, i) => (
              <div key={i} className={
                line.startsWith('✓') ? 'line-success' :
                line.startsWith('✗') || line.toLowerCase().includes('error') ? 'line-error' :
                line.startsWith('▶') ? 'line-info' : ''
              }>{line || ' '}</div>
            ))}
          </div>
        </div>
      )}

      {/* Install form modal */}
      {showForm && (
        <div style={{ position:'fixed', inset:0, background:'rgba(0,0,0,0.78)', display:'flex', alignItems:'center', justifyContent:'center', zIndex:1000, overflowY:'auto', padding:20 }}>
          <div className="glass-card fade-in" style={{ padding:28, width:520, maxWidth:'100%', maxHeight:'90vh', overflowY:'auto' }}>
            <div style={{ fontWeight:800, fontSize:17, marginBottom:20, display:'flex', alignItems:'center', justifyContent:'space-between' }}>
              <span>🌐 Install WordPress</span>
              <button onClick={() => setShowForm(false)} style={{ background:'none', border:'none', cursor:'pointer', color:'var(--text-muted)', fontSize:18 }}>✕</button>
            </div>

            <div style={{ display:'flex', flexDirection:'column', gap:14 }}>
              {/* Domain */}
              <div>
                <label style={{ fontSize:13, fontWeight:500, color:'var(--text-secondary)', display:'block', marginBottom:5 }}>Domain Name *</label>
                <input id="wp-domain" className="input" placeholder="myblog.com" value={form.domain}
                  onChange={e => autoFill(e.target.value)} />
                <div style={{ fontSize:11, color:'var(--text-muted)', marginTop:4 }}>DB name and user will auto-fill</div>
              </div>

              <div style={{ display:'grid', gridTemplateColumns:'1fr 1fr', gap:12 }}>
                <div>
                  <label style={{ fontSize:13, fontWeight:500, color:'var(--text-secondary)', display:'block', marginBottom:5 }}>Database Name *</label>
                  <input id="wp-dbname" className="input mono" value={form.dbName} onChange={e => setForm({...form, dbName: e.target.value})} placeholder="wp_myblog" />
                </div>
                <div>
                  <label style={{ fontSize:13, fontWeight:500, color:'var(--text-secondary)', display:'block', marginBottom:5 }}>DB Username *</label>
                  <input id="wp-dbuser" className="input mono" value={form.dbUser} onChange={e => setForm({...form, dbUser: e.target.value})} placeholder="wp_user" />
                </div>
              </div>

              <div>
                <label style={{ fontSize:13, fontWeight:500, color:'var(--text-secondary)', display:'block', marginBottom:5 }}>DB Password *</label>
                <input id="wp-dbpass" className="input mono" type="password" value={form.dbPassword} onChange={e => setForm({...form, dbPassword: e.target.value})} placeholder="strong password" />
              </div>

              <div style={{ borderTop:'1px solid var(--border-subtle)', paddingTop:14 }}>
                <div style={{ fontSize:12, fontWeight:600, color:'var(--text-muted)', textTransform:'uppercase', letterSpacing:'0.5px', marginBottom:12 }}>WordPress Admin</div>
                <div style={{ display:'grid', gridTemplateColumns:'1fr 1fr', gap:12 }}>
                  <div>
                    <label style={{ fontSize:13, fontWeight:500, color:'var(--text-secondary)', display:'block', marginBottom:5 }}>Admin Username</label>
                    <input id="wp-adminuser" className="input" value={form.adminUser} onChange={e => setForm({...form, adminUser: e.target.value})} placeholder="admin" />
                  </div>
                  <div>
                    <label style={{ fontSize:13, fontWeight:500, color:'var(--text-secondary)', display:'block', marginBottom:5 }}>Admin Password</label>
                    <input id="wp-adminpass" className="input" type="password" value={form.adminPass} onChange={e => setForm({...form, adminPass: e.target.value})} placeholder="admin password" />
                  </div>
                </div>
              </div>

              <div>
                <label style={{ fontSize:13, fontWeight:500, color:'var(--text-secondary)', display:'block', marginBottom:5 }}>Admin Email *</label>
                <input id="wp-email" className="input" type="email" value={form.adminEmail} onChange={e => setForm({...form, adminEmail: e.target.value})} placeholder="admin@example.com" />
              </div>

              <div>
                <label style={{ fontSize:13, fontWeight:500, color:'var(--text-secondary)', display:'block', marginBottom:5 }}>Site Title</label>
                <input id="wp-title" className="input" value={form.siteTitle} onChange={e => setForm({...form, siteTitle: e.target.value})} placeholder="My WordPress Site" />
              </div>

              <div style={{ display:'flex', gap:8, justifyContent:'flex-end', marginTop:8, borderTop:'1px solid var(--border-subtle)', paddingTop:16 }}>
                <button onClick={() => setShowForm(false)} className="btn btn-ghost">Cancel</button>
                <button id="submit-install-wp" onClick={install} className="btn btn-primary" disabled={busy}>
                  {busy ? <><span className="spinner"/>Installing…</> : <><Globe size={15}/>Install WordPress</>}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
