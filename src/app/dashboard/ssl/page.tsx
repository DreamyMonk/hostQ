'use client';
import { useEffect, useState } from 'react';
import { ShieldCheck, Plus, RefreshCw, RotateCcw, Trash2, AlertTriangle, CheckCircle, Clock, Terminal } from 'lucide-react';

interface Cert {
  domain: string;
  expiry: string;
  daysLeft: number;
  status: string;
  issuer: string;
}

export default function SSLPage() {
  const [loading, setLoading]   = useState(true);
  const [certs, setCerts]       = useState<Cert[]>([]);
  const [demo, setDemo]         = useState(false);
  const [showForm, setShowForm] = useState(false);
  const [output, setOutput]     = useState('');
  const [busy, setBusy]         = useState(false);
  const [msg, setMsg]           = useState<{ type: 'success'|'error'; text: string } | null>(null);

  const [form, setForm] = useState({
    domain: '', email: '', webserver: 'nginx', staging: false,
    mode: 'letsencrypt', certificate: '', privateKey: '', chain: ''
  });

  const showMsg = (type: 'success'|'error', text: string) => {
    setMsg({ type, text });
    setTimeout(() => setMsg(null), 6000);
  };

  const loadCerts = async () => {
    setLoading(true);
    try {
      const r = await fetch('/api/ssl');
      const d = await r.json();
      setCerts(d.certificates || []);
      setDemo(d.demo || false);
    } finally { setLoading(false); }
  };

  useEffect(() => {
    const id = setTimeout(() => { void loadCerts(); }, 0);
    return () => clearTimeout(id);
  }, []);

  const installSSL = async () => {
    if (!form.domain || (form.mode === 'letsencrypt' && !form.email)) return;
    setBusy(true);
    setOutput('Installing SSL certificate…\n');
    try {
      const r = await fetch('/api/ssl', {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(form),
      });
      const d = await r.json();
      setOutput(d.output || '');
      if (d.success) { showMsg('success', d.message); setShowForm(false); loadCerts(); }
      else showMsg('error', d.message);
    } finally { setBusy(false); }
  };

  const renewCert = async (domain?: string) => {
    setBusy(true);
    setOutput(`Renewing ${domain || 'all certificates'}…\n`);
    try {
      const r = await fetch('/api/ssl', {
        method: 'PATCH', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ domain }),
      });
      const d = await r.json();
      setOutput(d.output || '');
      if (d.success) { showMsg('success', d.message); loadCerts(); }
      else showMsg('error', d.message);
    } finally { setBusy(false); }
  };

  const deleteCert = async (domain: string) => {
    if (!confirm(`Delete SSL certificate for ${domain}?`)) return;
    setBusy(true);
    try {
      const r = await fetch('/api/ssl', {
        method: 'DELETE', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ domain }),
      });
      const d = await r.json();
      if (d.success) { showMsg('success', d.message); loadCerts(); }
      else showMsg('error', d.message);
    } finally { setBusy(false); }
  };

  const statusIcon = (status: string) => {
    if (status === 'valid')    return <CheckCircle size={15} color="#22c55e" />;
    if (status === 'expiring') return <Clock size={15} color="#f59e0b" />;
    return <AlertTriangle size={15} color="#ef4444" />;
  };

  return (
    <div className="fade-in">
      <div className="page-header" style={{ display:'flex', alignItems:'center', justifyContent:'space-between' }}>
        <div>
          <h1 className="page-title">SSL Manager</h1>
          <p className="page-subtitle">Free Let&apos;s Encrypt certificates or upload your own PEM certificate</p>
        </div>
        <div style={{ display:'flex', gap:8 }}>
          <button id="renew-all-btn" onClick={() => renewCert()} className="btn btn-ghost btn-sm" disabled={busy}>
            <RotateCcw size={14}/>Renew All
          </button>
          <button id="install-ssl-btn" onClick={() => setShowForm(true)} className="btn btn-primary btn-sm">
            <Plus size={14}/>Install SSL
          </button>
          <button onClick={loadCerts} className="btn btn-ghost btn-sm">
            <RefreshCw size={14} style={{ animation: loading ? 'spin 1s linear infinite' : 'none' }}/>
          </button>
        </div>
      </div>

      {demo && (
        <div className="alert alert-warning">
          ⚠️ Demo mode — Certbot not detected. These are sample certificates. On a Linux VPS with Certbot installed, real certs will appear here.
        </div>
      )}

      {msg && <div className={`alert ${msg.type === 'success' ? 'alert-success' : 'alert-error'}`}>{msg.text}</div>}

      {/* Stats row */}
      <div className="card-grid-3" style={{ marginBottom:24 }}>
        {[
          { label:'Total Certificates', value: certs.length, color:'#3b82f6', icon:'🔒' },
          { label:'Valid Certificates', value: certs.filter(c => c.status === 'valid').length, color:'#22c55e', icon:'✅' },
          { label:'Expiring Soon',      value: certs.filter(c => c.status !== 'valid').length, color:'#f59e0b', icon:'⚠️' },
        ].map(stat => (
          <div key={stat.label} className="glass-card" style={{ padding:20 }}>
            <div style={{ fontSize:28, marginBottom:6 }}>{stat.icon}</div>
            <div style={{ fontSize:26, fontWeight:800, color: stat.color }}>{stat.value}</div>
            <div style={{ fontSize:12, color:'var(--text-muted)', textTransform:'uppercase', letterSpacing:'0.5px' }}>{stat.label}</div>
          </div>
        ))}
      </div>

      {/* Certificates table */}
      <div className="glass-card" style={{ marginBottom:24 }}>
        <div style={{ padding:'16px 20px', borderBottom:'1px solid var(--border-subtle)', display:'flex', alignItems:'center', justifyContent:'space-between' }}>
          <span style={{ fontWeight:600 }}>SSL Certificates</span>
        </div>
        {loading ? (
          <div style={{ padding:40, textAlign:'center', color:'var(--text-muted)' }}>
            <div className="spinner" style={{ margin:'0 auto 10px' }}/>Loading…
          </div>
        ) : certs.length === 0 ? (
          <div style={{ padding:40, textAlign:'center' }}>
            <ShieldCheck size={40} color="var(--text-muted)" style={{ margin:'0 auto 12px' }}/>
            <div style={{ color:'var(--text-muted)', fontSize:14 }}>No SSL certificates installed</div>
            <button onClick={() => setShowForm(true)} className="btn btn-primary btn-sm" style={{ marginTop:12 }}>
              <Plus size={14}/>Install your first SSL
            </button>
          </div>
        ) : (
          <table className="data-table">
            <thead>
              <tr><th>Domain</th><th>Status</th><th>Days Left</th><th>Expiry</th><th>Issuer</th><th>Actions</th></tr>
            </thead>
            <tbody>
              {certs.map(cert => (
                <tr key={cert.domain}>
                  <td style={{ fontWeight:600 }}>{cert.domain}</td>
                  <td>
                    <div style={{ display:'flex', alignItems:'center', gap:6 }}>
                      {statusIcon(cert.status)}
                      <span className={`badge ${cert.status === 'valid' ? 'badge-green' : cert.status === 'expiring' ? 'badge-yellow' : 'badge-red'}`}>
                        {cert.status}
                      </span>
                    </div>
                  </td>
                  <td>
                    <span style={{ fontWeight:700, color: cert.daysLeft < 7 ? '#ef4444' : cert.daysLeft < 30 ? '#f59e0b' : '#22c55e' }}>
                      {cert.daysLeft}d
                    </span>
                  </td>
                  <td style={{ color:'var(--text-muted)', fontSize:12 }}>{cert.expiry}</td>
                  <td style={{ color:'var(--text-muted)', fontSize:12 }}>{cert.issuer}</td>
                  <td>
                    <div style={{ display:'flex', gap:6 }}>
                      <button id={`renew-${cert.domain}`} onClick={() => renewCert(cert.domain)} className="btn btn-ghost btn-sm" disabled={busy}>
                        <RotateCcw size={12}/>Renew
                      </button>
                      <button id={`delete-ssl-${cert.domain}`} onClick={() => deleteCert(cert.domain)} className="btn btn-danger btn-sm" disabled={busy}>
                        <Trash2 size={12}/>
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      {/* Output */}
      {output && (
        <div>
          <div style={{ display:'flex', alignItems:'center', gap:8, marginBottom:8, fontSize:13, fontWeight:600, color:'var(--text-secondary)' }}>
            <Terminal size={14}/>Certbot Output
          </div>
          <div className="terminal">
            {output.split('\n').map((line, i) => (
              <div key={i} className={line.includes('Successfully') ? 'line-success' : line.includes('ERROR') || line.includes('Failed') ? 'line-error' : ''}>
                {line || ' '}
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Install SSL form modal */}
      {showForm && (
        <div style={{ position:'fixed', inset:0, background:'rgba(0,0,0,0.75)', display:'flex', alignItems:'center', justifyContent:'center', zIndex:1000 }}>
          <div className="glass-card fade-in" style={{ padding:28, width:460, maxWidth:'90vw' }}>
            <div style={{ fontSize:17, fontWeight:700, marginBottom:20, display:'flex', alignItems:'center', justifyContent:'space-between' }}>
              Install SSL Certificate
              <button onClick={() => setShowForm(false)} style={{ background:'none', border:'none', cursor:'pointer', color:'var(--text-muted)' }}>✕</button>
            </div>
            <div style={{ display:'flex', flexDirection:'column', gap:14 }}>
              {[
                { label:'Domain Name', id:'ssl-domain', key:'domain', placeholder:'example.com' },
                { label:'Email Address', id:'ssl-email', key:'email', placeholder:'admin@example.com' },
              ].map(f => (
                <div key={f.key}>
                  <label style={{ fontSize:13, fontWeight:500, color:'var(--text-secondary)', marginBottom:6, display:'block' }}>{f.label}</label>
                  <input id={f.id} className="input" placeholder={f.placeholder}
                    value={form[f.key as keyof typeof form] as string}
                    onChange={e => setForm({ ...form, [f.key]: e.target.value })} />
                </div>
              ))}
              <div>
                <label style={{ fontSize:13, fontWeight:500, color:'var(--text-secondary)', marginBottom:6, display:'block' }}>Certificate Source</label>
                <select id="ssl-mode" className="input" value={form.mode} onChange={e => setForm({ ...form, mode: e.target.value })}>
                  <option value="letsencrypt">Let&apos;s Encrypt via Certbot</option>
                  <option value="manual">Upload own certificate</option>
                </select>
              </div>
              <div>
                <label style={{ fontSize:13, fontWeight:500, color:'var(--text-secondary)', marginBottom:6, display:'block' }}>Web Server</label>
                <select id="ssl-webserver" className="input" value={form.webserver} onChange={e => setForm({ ...form, webserver: e.target.value })}>
                  <option value="nginx">Nginx</option>
                  <option value="apache">Apache</option>
                  <option value="standalone">Standalone</option>
                </select>
              </div>
              <label style={{ display:'flex', alignItems:'center', gap:8, fontSize:13, cursor:'pointer' }}>
                <input type="checkbox" id="ssl-staging" checked={form.staging} onChange={e => setForm({ ...form, staging: e.target.checked })} />
                <span style={{ color:'var(--text-secondary)' }}>Use staging (test mode, no rate limits)</span>
              </label>
              {form.mode === 'manual' && (
                <>
                  <textarea id="ssl-cert" className="input" placeholder="-----BEGIN CERTIFICATE-----" value={form.certificate}
                    onChange={e => setForm({ ...form, certificate: e.target.value })} style={{ minHeight:120, fontFamily:'monospace' }} />
                  <textarea id="ssl-key" className="input" placeholder="-----BEGIN PRIVATE KEY-----" value={form.privateKey}
                    onChange={e => setForm({ ...form, privateKey: e.target.value })} style={{ minHeight:120, fontFamily:'monospace' }} />
                  <textarea id="ssl-chain" className="input" placeholder="Optional CA chain / intermediate certificates" value={form.chain}
                    onChange={e => setForm({ ...form, chain: e.target.value })} style={{ minHeight:80, fontFamily:'monospace' }} />
                </>
              )}
              <div style={{ display:'flex', gap:8, justifyContent:'flex-end', marginTop:8 }}>
                <button onClick={() => setShowForm(false)} className="btn btn-ghost">Cancel</button>
                <button id="submit-ssl-btn" onClick={installSSL} className="btn btn-primary" disabled={busy}>
                  {busy ? <span className="spinner"/> : <ShieldCheck size={15}/>}
                  {busy ? 'Installing…' : 'Install SSL'}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
