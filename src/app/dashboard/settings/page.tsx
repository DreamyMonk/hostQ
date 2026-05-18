'use client';
import { useState } from 'react';
import { Settings, Save, Eye, EyeOff, CheckCircle, Key, Server, Database, Link, FolderOpen } from 'lucide-react';

export default function SettingsPage() {
  const [saved, setSaved] = useState(false);
  const [showPass, setShowPass] = useState(false);

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
    panelUrl: 'http://localhost:3000',
  });

  const handleSave = async () => {
    // In production, this would call an API to update the .env file
    // For demo, just show saved state
    setSaved(true);
    setTimeout(() => setSaved(false), 3000);
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
      title: 'phpMyAdmin',
      icon: <Link size={16} color="#22c55e" />,
      fields: [
        { label: 'phpMyAdmin URL', key: 'phpmyadminUrl', type: 'url', placeholder: 'http://localhost/phpmyadmin' },
      ]
    },
    {
      title: 'File Paths',
      icon: <FolderOpen size={16} color="#f59e0b" />,
      fields: [
        { label: 'File Manager Root', key: 'fileManagerRoot', type: 'text', placeholder: '/var/www' },
        { label: 'Web Root', key: 'webRoot', type: 'text', placeholder: '/var/www/html' },
      ]
    },
    {
      title: 'Panel',
      icon: <Server size={16} color="#a855f7" />,
      fields: [
        { label: 'Panel URL', key: 'panelUrl', type: 'url', placeholder: 'https://panel.domain.com' },
      ]
    },
  ];

  return (
    <div className="fade-in">
      <div className="page-header" style={{ display:'flex', alignItems:'center', justifyContent:'space-between' }}>
        <div>
          <h1 className="page-title">Settings</h1>
          <p className="page-subtitle">Configure your hosting panel</p>
        </div>
        <button id="save-settings-btn" onClick={handleSave} className="btn btn-primary">
          {saved ? <><CheckCircle size={15}/>Saved!</> : <><Save size={15}/>Save Settings</>}
        </button>
      </div>

      {saved && (
        <div className="alert alert-success" style={{ marginBottom:20 }}>
          ✅ Settings saved. Restart the panel server to apply changes.
        </div>
      )}

      <div style={{ display:'flex', flexDirection:'column', gap:20 }}>
        {sections.map(section => (
          <div key={section.title} className="glass-card" style={{ overflow:'hidden' }}>
            <div style={{
              padding:'14px 20px', borderBottom:'1px solid var(--border-subtle)',
              display:'flex', alignItems:'center', gap:10, fontWeight:600
            }}>
              {section.icon}{section.title}
            </div>
            <div style={{ padding:20, display:'flex', flexDirection:'column', gap:14 }}>
              {section.fields.map(field => (
                <div key={field.key} style={{ display:'grid', gridTemplateColumns:'200px 1fr', alignItems:'center', gap:16 }}>
                  <label style={{ fontSize:13, fontWeight:500, color:'var(--text-secondary)' }}>{field.label}</label>
                  <div style={{ position:'relative' }}>
                    <input
                      id={`setting-${field.key}`}
                      className="input"
                      type={field.type}
                      placeholder={field.placeholder}
                      value={settings[field.key as keyof typeof settings]}
                      onChange={e => setSettings({ ...settings, [field.key]: e.target.value })}
                    />
                    {field.key === 'panelPassword' && (
                      <button
                        type="button"
                        onClick={() => setShowPass(!showPass)}
                        style={{ position:'absolute', right:10, top:'50%', transform:'translateY(-50%)',
                          background:'none', border:'none', cursor:'pointer', color:'var(--text-muted)' }}
                      >
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

      {/* Info box */}
      <div className="glass-card" style={{ marginTop:24, padding:'16px 20px', borderColor:'rgba(59,130,246,0.2)',
        background:'rgba(59,130,246,0.05)' }}>
        <div style={{ display:'flex', gap:10 }}>
          <Settings size={16} color="#3b82f6" style={{ flexShrink:0, marginTop:2 }}/>
          <div>
            <div style={{ fontWeight:600, marginBottom:6, color:'#60a5fa' }}>Configuration Notes</div>
            <ul style={{ fontSize:13, color:'var(--text-secondary)', lineHeight:1.7, paddingLeft:16 }}>
              <li>Settings are stored in <span className="mono">.env.local</span> on the server</li>
              <li>Restart the panel (<span className="mono">pm2 restart hostq</span>) after changes</li>
              <li>phpMyAdmin URL must be accessible from the browser, not the server</li>
              <li>File Manager Root restricts browsing to that directory for security</li>
              <li>Change default credentials immediately on production</li>
            </ul>
          </div>
        </div>
      </div>

      {/* Deployment info */}
      <div className="glass-card" style={{ marginTop:16, padding:'16px 20px' }}>
        <div style={{ fontWeight:600, marginBottom:14, display:'flex', alignItems:'center', gap:8 }}>
          <Server size={16} color="#a855f7" /> Deployment Commands
        </div>
        <div style={{ display:'flex', flexDirection:'column', gap:12 }}>
          {[
            { label:'Start (PM2)', cmd:'pm2 start npm --name hostq -- start' },
            { label:'Restart',     cmd:'pm2 restart hostq' },
            { label:'View Logs',   cmd:'pm2 logs hostq' },
            { label:'Docker',      cmd:'docker-compose up -d' },
          ].map(d => (
            <div key={d.label} style={{ display:'grid', gridTemplateColumns:'120px 1fr', gap:12, alignItems:'center' }}>
              <span style={{ fontSize:12, color:'var(--text-muted)' }}>{d.label}</span>
              <div className="mono" style={{
                background:'var(--bg-base)', padding:'6px 12px', borderRadius:6,
                border:'1px solid var(--border-subtle)', fontSize:12, color:'#22c55e',
                cursor:'pointer', userSelect:'all'
              }}
                onClick={() => navigator.clipboard.writeText(d.cmd)}
                title="Click to copy"
              >
                {d.cmd}
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
