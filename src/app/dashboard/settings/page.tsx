'use client';
import { useEffect, useState } from 'react';
import { AlertTriangle, CheckCircle, Database, DownloadCloud, Eye, EyeOff, FolderOpen, Globe2, Key, LockKeyhole, RefreshCw, Save, Server, XCircle } from 'lucide-react';

type AuditEntry = { ts: string; action: string; actor?: string; target?: string; status: string };
type Session = { id: string; username: string; role: string; lastSeenAt: string; revokedAt?: string };
type SecurityCheck = { id: string; label: string; detail: string; status: 'pass' | 'warn' | 'fail' };
type UpdateInfo = {
  current: string;
  repo: string;
  updateAvailable: boolean;
  latest: null | { tag: string; name: string; notes: string; url: string; publishedAt: string; prerelease: boolean };
};
type PanelConfig = { panelDomain: string; panelUrl: string; allowInsecureHttp: boolean; envFile: string };

export default function SettingsPage() {
  const [saved, setSaved] = useState(false);
  const [showPass, setShowPass] = useState(false);
  const [audit, setAudit] = useState<{ entries: AuditEntry[]; chainValid: boolean } | null>(null);
  const [sessions, setSessions] = useState<Session[]>([]);
  const [security, setSecurity] = useState<{ score: number; ready: boolean; checks: SecurityCheck[] } | null>(null);
  const [update, setUpdate] = useState<UpdateInfo | null>(null);
  const [panelConfig, setPanelConfig] = useState<PanelConfig | null>(null);
  const [panelSaveMessage, setPanelSaveMessage] = useState('');
  const [updating, setUpdating] = useState(false);
  const [updateOutput, setUpdateOutput] = useState('');
  const [passwordForm, setPasswordForm] = useState({ currentPassword: '', newPassword: '', confirmPassword: '' });
  const [otpSetup, setOtpSetup] = useState<{ otpSecret: string; otpAuthUrl: string } | null>(null);
  const [otpCode, setOtpCode] = useState('');
  const [settings, setSettings] = useState({
    panelUsername: 'admin',
    panelPassword: '',
    panelPasswordConfirm: '',
    phpmyadminUrl: 'http://localhost/phpmyadmin',
    fileManagerRoot: '/var/www',
    webRoot: '/var/www/html',
    dbHost: 'localhost',
    dbPort: '3306',
    dbRootUser: 'root',
    panelUrl: 'https://panel.domain.com',
  });

  const refreshSecurity = async () => {
    const [auditResponse, sessionResponse, securityResponse, updateResponse, panelResponse] = await Promise.all([
      fetch('/api/audit'),
      fetch('/api/sessions'),
      fetch('/api/security'),
      fetch('/api/update'),
      fetch('/api/panel'),
    ]);
    if (auditResponse.ok) setAudit(await auditResponse.json());
    if (sessionResponse.ok) setSessions((await sessionResponse.json()).sessions || []);
    if (securityResponse.ok) setSecurity(await securityResponse.json());
    if (updateResponse.ok) setUpdate(await updateResponse.json());
    if (panelResponse.ok) setPanelConfig(await panelResponse.json());
  };

  useEffect(() => {
    const id = setTimeout(() => { void refreshSecurity(); }, 0);
    return () => clearTimeout(id);
  }, []);

  const savePanelHost = async () => {
    if (!panelConfig) return;
    setPanelSaveMessage('');
    const response = await fetch('/api/panel', {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        panelDomain: panelConfig.panelDomain,
        allowInsecureHttp: panelConfig.allowInsecureHttp,
      }),
    });
    const data = await response.json();
    if (response.ok) {
      setPanelConfig(data.config);
      setPanelSaveMessage(data.message || 'Panel host saved.');
      setSaved(true);
      setTimeout(() => setSaved(false), 3000);
      await refreshSecurity();
    } else {
      setPanelSaveMessage(data.error || 'Unable to save panel host');
    }
  };

  const revokeSession = async (sessionId: string) => {
    if (!confirm('Revoke this session?')) return;
    await fetch('/api/sessions', {
      method: 'DELETE',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ sessionId }),
    });
    await refreshSecurity();
  };

  const runUpdate = async () => {
    if (!update?.latest) return;
    if (!confirm(`Update hostQ from ${update.current} to ${update.latest.tag}? A panel backup will be created first.`)) return;
    setUpdating(true);
    setUpdateOutput('Starting update...');
    try {
      const response = await fetch('/api/update', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ tag: update.latest.tag, confirm: update.latest.tag }),
      });
      const data = await response.json();
      setUpdateOutput(data.output || data.message || '');
      if (!data.success) alert(data.error || data.message || 'Update failed');
    } finally {
      setUpdating(false);
    }
  };

  const changeAccountPassword = async () => {
    if (passwordForm.newPassword !== passwordForm.confirmPassword) {
      alert('New password and confirm password do not match');
      return;
    }
    const response = await fetch('/api/account', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ action: 'change_password', ...passwordForm }),
    });
    const data = await response.json();
    if (data.success) {
      alert(data.message);
      setPasswordForm({ currentPassword: '', newPassword: '', confirmPassword: '' });
    } else {
      alert(data.error || 'Password change failed');
    }
  };

  const start2fa = async () => {
    const response = await fetch('/api/account', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ action: 'start_2fa' }),
    });
    const data = await response.json();
    if (data.success) setOtpSetup({ otpSecret: data.otpSecret, otpAuthUrl: data.otpAuthUrl });
    else alert(data.error || 'Unable to start 2FA setup');
  };

  const enable2fa = async () => {
    const response = await fetch('/api/account', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ action: 'enable_2fa', otp: otpCode }),
    });
    const data = await response.json();
    if (data.success) {
      alert('2FA enabled');
      setOtpSetup(null);
      setOtpCode('');
    } else {
      alert(data.error || 'Unable to enable 2FA');
    }
  };

  const sections = [
    {
      title: 'Authentication',
      icon: <Key size={16} color="#3b82f6" />,
      fields: [
        { label: 'Panel Username', key: 'panelUsername', type: 'text', placeholder: 'admin' },
        { label: 'New Password', key: 'panelPassword', type: showPass ? 'text' : 'password', placeholder: 'Leave blank to keep current' },
        { label: 'Confirm Password', key: 'panelPasswordConfirm', type: showPass ? 'text' : 'password', placeholder: 'Confirm new password' },
      ]
    },
    {
      title: 'Database',
      icon: <Database size={16} color="#06b6d4" />,
      fields: [
        { label: 'DB Host', key: 'dbHost', type: 'text', placeholder: 'localhost' },
        { label: 'DB Port', key: 'dbPort', type: 'text', placeholder: '3306' },
        { label: 'DB Root User', key: 'dbRootUser', type: 'text', placeholder: 'root' },
      ]
    },
    {
      title: 'Paths And URLs',
      icon: <FolderOpen size={16} color="#f59e0b" />,
      fields: [
        { label: 'phpMyAdmin URL', key: 'phpmyadminUrl', type: 'url', placeholder: 'https://panel.domain.com/phpmyadmin' },
        { label: 'File Manager Root', key: 'fileManagerRoot', type: 'text', placeholder: '/var/www' },
        { label: 'Web Root', key: 'webRoot', type: 'text', placeholder: '/var/www/html' },
        { label: 'Panel URL', key: 'panelUrl', type: 'url', placeholder: 'https://panel.domain.com' },
      ]
    },
  ];

  return (
    <div className="fade-in">
      <div className="page-header" style={{ display:'flex', alignItems:'center', justifyContent:'space-between', gap:16 }}>
        <div>
          <h1 className="page-title">Security</h1>
          <p className="page-subtitle">Production readiness, active sessions, audit chain and deployment controls</p>
        </div>
        <button onClick={refreshSecurity} className="btn btn-primary"><RefreshCw size={15}/> Refresh Checks</button>
      </div>

      {security && (
        <div className="glass-card" style={{ padding:20, marginBottom:20 }}>
          <div style={{ display:'flex', justifyContent:'space-between', alignItems:'center', gap:16, marginBottom:16 }}>
            <div>
              <div style={{ fontSize:13, color:'var(--text-muted)', marginBottom:4 }}>Production readiness</div>
              <div style={{ fontSize:34, fontWeight:850, color: security.ready ? 'var(--accent-green)' : security.checks.some(c => c.status === 'fail') ? 'var(--accent-red)' : 'var(--accent-yellow)' }}>{security.score}%</div>
            </div>
            <span className={`badge ${security.ready ? 'badge-green' : security.checks.some(c => c.status === 'fail') ? 'badge-red' : 'badge-yellow'}`}>
              {security.ready ? 'Ready signal' : 'Action needed'}
            </span>
          </div>
          <div style={{ display:'grid', gridTemplateColumns:'repeat(auto-fit, minmax(260px, 1fr))', gap:10 }}>
            {security.checks.map(check => (
              <div key={check.id} style={{ display:'flex', gap:10, padding:12, border:'1px solid var(--border-subtle)', borderRadius:8, background:'var(--bg-base)' }}>
                {check.status === 'pass' ? <CheckCircle size={16} color="var(--accent-green)" /> : check.status === 'fail' ? <XCircle size={16} color="var(--accent-red)" /> : <AlertTriangle size={16} color="var(--accent-yellow)" />}
                <div style={{ minWidth:0 }}>
                  <div style={{ fontWeight:750, fontSize:13 }}>{check.label}</div>
                  <div style={{ fontSize:12, color:'var(--text-muted)', overflowWrap:'anywhere' }}>{check.detail}</div>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      <div className="glass-card" style={{ marginBottom:20, overflow:'hidden' }}>
        <div style={{ padding:'14px 20px', borderBottom:'1px solid var(--border-subtle)', display:'flex', alignItems:'center', gap:10, fontWeight:700 }}>
          <LockKeyhole size={16} color="#16a34a" /> Control Plane Security
          {audit && <span className={`badge ${audit.chainValid ? 'badge-green' : 'badge-red'}`} style={{ marginLeft:'auto' }}>{audit.chainValid ? 'Audit chain valid' : 'Audit tamper warning'}</span>}
        </div>
        <div style={{ padding:20, display:'grid', gridTemplateColumns:'repeat(auto-fit, minmax(300px, 1fr))', gap:16 }}>
          <div>
            <div style={{ fontWeight:750, marginBottom:10 }}>Active Sessions</div>
            <div style={{ display:'flex', flexDirection:'column', gap:8 }}>
              {sessions.slice(0, 10).map(session => (
                <div key={session.id} style={{ border:'1px solid var(--border-subtle)', borderRadius:8, padding:10, background:'var(--bg-base)' }}>
                  <div style={{ display:'flex', justifyContent:'space-between', gap:8 }}><strong>{session.username}</strong><span className="badge badge-blue">{session.role}</span></div>
                  <div className="mono" style={{ fontSize:11, color:'var(--text-muted)', marginTop:4 }}>{session.id.slice(0, 8)} - {new Date(session.lastSeenAt).toLocaleString()}</div>
                  <button onClick={() => revokeSession(session.id)} className="btn btn-danger btn-sm" style={{ marginTop:8 }}>Revoke</button>
                </div>
              ))}
              {sessions.length === 0 && <div style={{ color:'var(--text-muted)', fontSize:13 }}>No sessions visible.</div>}
            </div>
          </div>
          <div>
            <div style={{ fontWeight:750, marginBottom:10 }}>Recent Audit Events</div>
            <div style={{ display:'flex', flexDirection:'column', gap:8 }}>
              {(audit?.entries || []).slice(0, 10).map((entry, index) => (
                <div key={`${entry.ts}-${index}`} style={{ border:'1px solid var(--border-subtle)', borderRadius:8, padding:10, background:'var(--bg-base)' }}>
                  <div style={{ display:'flex', justifyContent:'space-between', gap:8 }}><strong>{entry.action}</strong><span className={`badge ${entry.status === 'success' ? 'badge-green' : 'badge-red'}`}>{entry.status}</span></div>
                  <div className="mono" style={{ fontSize:11, color:'var(--text-muted)', marginTop:4 }}>{entry.actor || 'system'} - {entry.target || '-'} - {new Date(entry.ts).toLocaleString()}</div>
                </div>
              ))}
              {!audit?.entries?.length && <div style={{ color:'var(--text-muted)', fontSize:13 }}>No audit events yet.</div>}
            </div>
          </div>
        </div>
      </div>

      <div className="glass-card" style={{ marginBottom:20, overflow:'hidden' }}>
        <div style={{ padding:'14px 20px', borderBottom:'1px solid var(--border-subtle)', display:'flex', alignItems:'center', gap:10, fontWeight:700 }}>
          <Key size={16} color="#3b82f6" /> Account Access
        </div>
        <div style={{ padding:20, display:'grid', gridTemplateColumns:'repeat(auto-fit, minmax(300px, 1fr))', gap:16 }}>
          <div style={{ display:'grid', gap:10 }}>
            <strong>Change password</strong>
            <input className="input" type="password" placeholder="Current password" value={passwordForm.currentPassword} onChange={event => setPasswordForm({ ...passwordForm, currentPassword: event.target.value })} />
            <input className="input" type="password" placeholder="New password" value={passwordForm.newPassword} onChange={event => setPasswordForm({ ...passwordForm, newPassword: event.target.value })} />
            <input className="input" type="password" placeholder="Confirm new password" value={passwordForm.confirmPassword} onChange={event => setPasswordForm({ ...passwordForm, confirmPassword: event.target.value })} />
            <button onClick={changeAccountPassword} className="btn btn-primary btn-sm" style={{ justifyContent:'center' }}>Change Password</button>
          </div>
          <div style={{ display:'grid', gap:10 }}>
            <strong>Two-factor authentication</strong>
            {!otpSetup ? (
              <>
                <div style={{ color:'var(--text-muted)', fontSize:13, lineHeight:1.6 }}>2FA is optional after SSH setup. Start setup, add the secret to your authenticator app, then verify a 6-digit code.</div>
                <button onClick={start2fa} className="btn btn-ghost btn-sm" style={{ justifyContent:'center' }}>Start 2FA Setup</button>
              </>
            ) : (
              <>
                <div className="alert alert-warning" style={{ marginBottom:0 }}>
                  Secret: <strong className="mono">{otpSetup.otpSecret}</strong>
                </div>
                <input className="input mono" inputMode="numeric" placeholder="6-digit code" value={otpCode} onChange={event => setOtpCode(event.target.value)} />
                <button onClick={enable2fa} className="btn btn-primary btn-sm" style={{ justifyContent:'center' }}>Verify And Enable 2FA</button>
              </>
            )}
          </div>
        </div>
      </div>

      <div className="glass-card" style={{ marginBottom:20, overflow:'hidden' }}>
        <div style={{ padding:'14px 20px', borderBottom:'1px solid var(--border-subtle)', display:'flex', alignItems:'center', gap:10, fontWeight:700 }}>
          <DownloadCloud size={16} color="#2563eb" /> Updates
          {update && <span className={`badge ${update.updateAvailable ? 'badge-yellow' : 'badge-green'}`} style={{ marginLeft:'auto' }}>{update.updateAvailable ? 'Update available' : 'Up to date'}</span>}
        </div>
        <div style={{ padding:20, display:'grid', gridTemplateColumns:'minmax(0, 1fr) auto', gap:16, alignItems:'start' }}>
          <div>
            <div style={{ display:'grid', gridTemplateColumns:'repeat(auto-fit, minmax(160px, 1fr))', gap:10, marginBottom:12 }}>
              <div style={{ padding:10, border:'1px solid var(--border-subtle)', borderRadius:8, background:'var(--bg-base)' }}>
                <div style={{ fontSize:11, color:'var(--text-muted)' }}>Installed</div>
                <strong>{update?.current || 'Checking...'}</strong>
              </div>
              <div style={{ padding:10, border:'1px solid var(--border-subtle)', borderRadius:8, background:'var(--bg-base)' }}>
                <div style={{ fontSize:11, color:'var(--text-muted)' }}>Latest Release</div>
                <strong>{update?.latest?.tag || 'No release found'}</strong>
              </div>
              <div style={{ padding:10, border:'1px solid var(--border-subtle)', borderRadius:8, background:'var(--bg-base)' }}>
                <div style={{ fontSize:11, color:'var(--text-muted)' }}>Repository</div>
                <strong>{update?.repo || 'DreamyMonk/hostQ'}</strong>
              </div>
            </div>
            {update?.latest?.notes && (
              <div style={{ fontSize:12, color:'var(--text-secondary)', lineHeight:1.6, whiteSpace:'pre-wrap', maxHeight:120, overflow:'auto' }}>
                {update.latest.notes}
              </div>
            )}
            {updateOutput && <div className="terminal" style={{ marginTop:12 }}>{updateOutput.split('\n').map((line, index) => <div key={index}>{line || ' '}</div>)}</div>}
          </div>
          <div style={{ display:'flex', flexDirection:'column', gap:8 }}>
            <button onClick={refreshSecurity} className="btn btn-ghost btn-sm"><RefreshCw size={14}/> Check</button>
            <button onClick={runUpdate} className="btn btn-primary btn-sm" disabled={!update?.updateAvailable || updating}>
              {updating ? <span className="spinner" /> : <DownloadCloud size={14}/>} Update
            </button>
          </div>
        </div>
      </div>

      {saved && <div className="alert alert-success">Settings saved. Restart hostQ to apply environment-backed changes.</div>}

      <div className="glass-card" style={{ marginBottom:20, overflow:'hidden' }}>
        <div style={{ padding:'14px 20px', borderBottom:'1px solid var(--border-subtle)', display:'flex', alignItems:'center', gap:10, fontWeight:700 }}>
          <Globe2 size={16} color="#2563eb" /> Panel Host
          {panelConfig && <span className={`badge ${panelConfig.allowInsecureHttp ? 'badge-yellow' : 'badge-green'}`} style={{ marginLeft:'auto' }}>{panelConfig.allowInsecureHttp ? 'HTTP setup mode' : 'HTTPS enforced'}</span>}
        </div>
        <div style={{ padding:20, display:'grid', gridTemplateColumns:'minmax(280px, 1fr) minmax(280px, 1fr)', gap:16, alignItems:'start' }}>
          <div style={{ display:'grid', gap:12 }}>
            <label style={{ display:'grid', gap:6 }}>
              <span style={{ fontSize:13, fontWeight:650, color:'var(--text-secondary)' }}>Panel domain or subdomain</span>
              <input
                className="input"
                placeholder="panel.example.com"
                value={panelConfig?.panelDomain || ''}
                onChange={event => setPanelConfig({ ...(panelConfig || { panelUrl:'', envFile:'', allowInsecureHttp:true }), panelDomain: event.target.value })}
              />
            </label>
            <label style={{ display:'flex', alignItems:'center', gap:10, fontSize:13, color:'var(--text-secondary)' }}>
              <input
                type="checkbox"
                checked={panelConfig?.allowInsecureHttp || false}
                onChange={event => setPanelConfig({ ...(panelConfig || { panelDomain:'', panelUrl:'', envFile:'' }), allowInsecureHttp: event.target.checked })}
              />
              Allow temporary HTTP direct-IP setup access
            </label>
            <button onClick={savePanelHost} className="btn btn-primary btn-sm" style={{ justifyContent:'center', width:'fit-content' }}>
              <Save size={14}/> Save Panel Host
            </button>
            {panelSaveMessage && <div className={panelSaveMessage.includes('Unable') || panelSaveMessage.includes('valid') ? 'alert alert-error' : 'alert alert-success'} style={{ marginBottom:0 }}>{panelSaveMessage}</div>}
          </div>
          <div style={{ border:'1px solid var(--border-subtle)', borderRadius:8, padding:14, background:'var(--bg-base)', display:'grid', gap:8 }}>
            <div style={{ fontSize:12, color:'var(--text-muted)' }}>Panel URL after restart</div>
            <strong style={{ overflowWrap:'anywhere' }}>{panelConfig?.allowInsecureHttp ? 'http' : 'https'}://{panelConfig?.panelDomain || 'panel.example.com'}</strong>
            <div style={{ fontSize:12, color:'var(--text-muted)', lineHeight:1.6 }}>
              Point this DNS record to the VPS IP, issue SSL for the domain, then turn off HTTP setup mode. Changes are written to <span className="mono">{panelConfig?.envFile || '/opt/hostq/.env.local'}</span>.
            </div>
          </div>
        </div>
      </div>

      <div style={{ display:'flex', flexDirection:'column', gap:20 }}>
        {sections.map(section => (
          <div key={section.title} className="glass-card" style={{ overflow:'hidden' }}>
            <div style={{ padding:'14px 20px', borderBottom:'1px solid var(--border-subtle)', display:'flex', alignItems:'center', gap:10, fontWeight:700 }}>
              {section.icon}{section.title}
            </div>
            <div style={{ padding:20, display:'flex', flexDirection:'column', gap:14 }}>
              {section.fields.map(field => (
                <div key={field.key} style={{ display:'grid', gridTemplateColumns:'200px 1fr', alignItems:'center', gap:16 }}>
                  <label style={{ fontSize:13, fontWeight:650, color:'var(--text-secondary)' }}>{field.label}</label>
                  <div style={{ position:'relative' }}>
                    <input
                      id={`setting-${field.key}`}
                      className="input"
                      type={field.type}
                      placeholder={field.placeholder}
                      value={settings[field.key as keyof typeof settings]}
                      onChange={event => setSettings({ ...settings, [field.key]: event.target.value })}
                    />
                    {field.key === 'panelPassword' && (
                      <button type="button" onClick={() => setShowPass(!showPass)} style={{ position:'absolute', right:10, top:'50%', transform:'translateY(-50%)', background:'none', border:'none', cursor:'pointer', color:'var(--text-muted)' }}>
                        {showPass ? <EyeOff size={14}/> : <Eye size={14}/>}
                      </button>
                    )}
                  </div>
                </div>
              ))}
            </div>
          </div>
        ))}
      </div>

      <div className="glass-card" style={{ marginTop:20, padding:'16px 20px' }}>
        <div style={{ fontWeight:700, marginBottom:14, display:'flex', alignItems:'center', gap:8 }}>
          <Server size={16} color="#a855f7" /> Production Commands
        </div>
        <div style={{ display:'grid', gridTemplateColumns:'repeat(auto-fit, minmax(280px, 1fr))', gap:12 }}>
          {[
            { label:'Build check', cmd:'npm run lint && npm run security:test && npm run build' },
            { label:'Restart', cmd:'pm2 restart hostq' },
            { label:'Logs', cmd:'pm2 logs hostq' },
            { label:'Helper enforcement', cmd:'HOSTQ_REQUIRE_HELPER=true' },
            { label:'Create release', cmd:'gh release create v0.2.0 --generate-notes' },
          ].map(item => (
            <div key={item.label} style={{ display:'grid', gap:6 }}>
              <span style={{ fontSize:12, color:'var(--text-muted)' }}>{item.label}</span>
              <div className="mono" style={{ background:'var(--bg-base)', padding:'8px 10px', borderRadius:7, border:'1px solid var(--border-subtle)', fontSize:12, color:'#027a48', userSelect:'all' }}>{item.cmd}</div>
            </div>
          ))}
        </div>
      </div>

      <div style={{ display:'flex', justifyContent:'flex-end', marginTop:20 }}>
        <button id="save-settings-btn" onClick={savePanelHost} className="btn btn-primary">
          {saved ? <><CheckCircle size={15}/>Saved</> : <><Save size={15}/>Save Settings</>}
        </button>
      </div>
    </div>
  );
}
