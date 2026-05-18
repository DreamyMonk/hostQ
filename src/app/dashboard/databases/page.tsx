'use client';
import { useEffect, useState } from 'react';
import { Database, Plus, Trash2, ExternalLink, RefreshCw, User } from 'lucide-react';

interface DB { name: string; size: string; tables: number; created?: string; }
interface DBUser { user: string; host: string; db?: string; }

export default function DatabasesPage() {
  const [loading, setLoading]   = useState(true);
  const [databases, setDatabases] = useState<DB[]>([]);
  const [users, setUsers]         = useState<DBUser[]>([]);
  const [pmaUrl, setPmaUrl]       = useState('');
  const [demo, setDemo]           = useState(false);
  const [msg, setMsg]             = useState<{ type: 'success'|'error'; text: string } | null>(null);
  const [busy, setBusy]           = useState(false);

  // Create DB form
  const [showCreateDB, setShowCreateDB] = useState(false);
  const [dbName, setDbName]             = useState('');

  // Create user form
  const [showCreateUser, setShowCreateUser] = useState(false);
  const [userForm, setUserForm]             = useState({ dbUser:'', dbPassword:'', dbName:'' });

  const [activeTab, setActiveTab] = useState<'databases'|'users'>('databases');

  const showMsg = (type: 'success'|'error', text: string) => {
    setMsg({ type, text });
    setTimeout(() => setMsg(null), 5000);
  };

  const load = async () => {
    setLoading(true);
    try {
      const r = await fetch('/api/databases');
      const d = await r.json();
      setDatabases(d.databases || []);
      setUsers(d.users || []);
      setPmaUrl(d.phpmyadmin || '');
      setDemo(d.demo || false);
    } finally { setLoading(false); }
  };

  useEffect(() => {
    const id = setTimeout(() => { void load(); }, 0);
    return () => clearTimeout(id);
  }, []);

  const createDB = async () => {
    if (!dbName) return;
    setBusy(true);
    try {
      const r = await fetch('/api/databases', {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ action: 'create_db', dbName }),
      });
      const d = await r.json();
      if (d.success) { showMsg('success', d.message); setShowCreateDB(false); setDbName(''); load(); }
      else showMsg('error', d.error);
    } finally { setBusy(false); }
  };

  const createUser = async () => {
    if (!userForm.dbUser || !userForm.dbPassword || !userForm.dbName) return;
    setBusy(true);
    try {
      const r = await fetch('/api/databases', {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ action: 'create_user', ...userForm }),
      });
      const d = await r.json();
      if (d.success) { showMsg('success', d.message); setShowCreateUser(false); setUserForm({ dbUser:'', dbPassword:'', dbName:'' }); load(); }
      else showMsg('error', d.error);
    } finally { setBusy(false); }
  };

  const dropDB = async (name: string) => {
    if (!confirm(`Drop database "${name}"? This cannot be undone.`)) return;
    setBusy(true);
    try {
      const r = await fetch('/api/databases', {
        method: 'DELETE', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ action: 'drop_db', name }),
      });
      const d = await r.json();
      if (d.success) { showMsg('success', d.message); load(); }
      else showMsg('error', d.error);
    } finally { setBusy(false); }
  };

  const dropUser = async (name: string) => {
    if (!confirm(`Drop user "${name}"?`)) return;
    setBusy(true);
    try {
      const r = await fetch('/api/databases', {
        method: 'DELETE', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ action: 'drop_user', name }),
      });
      const d = await r.json();
      if (d.success) { showMsg('success', d.message); load(); }
      else showMsg('error', d.error);
    } finally { setBusy(false); }
  };

  return (
    <div className="fade-in">
      <div className="page-header" style={{ display:'flex', alignItems:'center', justifyContent:'space-between' }}>
        <div>
          <h1 className="page-title">Database Manager</h1>
          <p className="page-subtitle">MySQL / MariaDB databases and users</p>
        </div>
        <div style={{ display:'flex', gap:8 }}>
          {pmaUrl && (
            <a id="phpmyadmin-link" href={pmaUrl} target="_blank" rel="noopener noreferrer" className="btn btn-ghost btn-sm">
              <ExternalLink size={14}/> phpMyAdmin
            </a>
          )}
          <button onClick={load} className="btn btn-ghost btn-sm">
            <RefreshCw size={14} style={{ animation: loading ? 'spin 1s linear infinite' : 'none' }}/>
          </button>
        </div>
      </div>

      {demo && (
        <div className="alert alert-warning">
          ⚠️ Demo mode — MySQL not connected. These are sample records. Configure DB_ROOT_PASSWORD in .env.local to connect.
        </div>
      )}
      {msg && <div className={`alert ${msg.type === 'success' ? 'alert-success' : 'alert-error'}`}>{msg.text}</div>}

      {/* Stats */}
      <div className="card-grid-3" style={{ marginBottom:24 }}>
        {[
          { label:'Databases', value: databases.length, icon:'🗄️', color:'#06b6d4' },
          { label:'Users', value: users.length, icon:'👤', color:'#a855f7' },
          { label:'Total Size', value: databases.reduce((t,d) => t + parseFloat(d.size||'0'), 0).toFixed(1)+' MB', icon:'📊', color:'#f59e0b' },
        ].map(s => (
          <div key={s.label} className="glass-card" style={{ padding:20 }}>
            <div style={{ fontSize:26, marginBottom:6 }}>{s.icon}</div>
            <div style={{ fontSize:24, fontWeight:800, color:s.color }}>{s.value}</div>
            <div style={{ fontSize:12, color:'var(--text-muted)', textTransform:'uppercase', letterSpacing:'0.5px' }}>{s.label}</div>
          </div>
        ))}
      </div>

      {/* phpMyAdmin banner */}
      {pmaUrl && (
        <div style={{
          padding:'14px 20px', marginBottom:20, borderRadius:10,
          background:'linear-gradient(135deg, rgba(6,182,212,0.1), rgba(59,130,246,0.08))',
          border:'1px solid rgba(6,182,212,0.2)',
          display:'flex', alignItems:'center', justifyContent:'space-between'
        }}>
          <div style={{ display:'flex', alignItems:'center', gap:12 }}>
            <span style={{ fontSize:22 }}>🐬</span>
            <div>
              <div style={{ fontWeight:600, fontSize:14 }}>phpMyAdmin</div>
              <div style={{ fontSize:12, color:'var(--text-muted)' }}>Full database management GUI</div>
            </div>
          </div>
          <a id="pma-open-btn" href={pmaUrl} target="_blank" rel="noopener noreferrer" className="btn btn-primary btn-sm">
            <ExternalLink size={13}/> Open phpMyAdmin
          </a>
        </div>
      )}

      {/* Tabs */}
      <div style={{ display:'flex', gap:4, marginBottom:16 }}>
        {['databases','users'].map(tab => (
          <button key={tab} id={`tab-${tab}`} onClick={() => setActiveTab(tab as 'databases'|'users')}
            className={`btn ${activeTab === tab ? 'btn-primary' : 'btn-ghost'} btn-sm`}>
            {tab === 'databases' ? <Database size={14}/> : <User size={14}/>}
            {tab.charAt(0).toUpperCase() + tab.slice(1)}
          </button>
        ))}
        <div style={{ marginLeft:'auto' }}>
          {activeTab === 'databases' ? (
            <button id="create-db-btn" onClick={() => setShowCreateDB(true)} className="btn btn-primary btn-sm">
              <Plus size={14}/>New Database
            </button>
          ) : (
            <button id="create-user-btn" onClick={() => setShowCreateUser(true)} className="btn btn-primary btn-sm">
              <Plus size={14}/>New User
            </button>
          )}
        </div>
      </div>

      {/* Databases list */}
      {activeTab === 'databases' && (
        <div className="glass-card">
          {loading ? (
            <div style={{ padding:40, textAlign:'center', color:'var(--text-muted)' }}>
              <div className="spinner" style={{ margin:'0 auto 10px'}}/>Loading…
            </div>
          ) : databases.length === 0 ? (
            <div style={{ padding:40, textAlign:'center' }}>
              <Database size={36} color="var(--text-muted)" style={{ margin:'0 auto 10px' }}/>
              <div style={{ color:'var(--text-muted)' }}>No databases found</div>
              <button onClick={() => setShowCreateDB(true)} className="btn btn-primary btn-sm" style={{ marginTop:12 }}>
                <Plus size={14}/>Create Database
              </button>
            </div>
          ) : (
            <table className="data-table">
              <thead>
                <tr><th>Database Name</th><th>Tables</th><th>Size</th><th>Actions</th></tr>
              </thead>
              <tbody>
                {databases.map(db => (
                  <tr key={db.name}>
                    <td>
                      <div style={{ display:'flex', alignItems:'center', gap:8 }}>
                        <Database size={14} color="#06b6d4"/>
                        <span className="mono">{db.name}</span>
                      </div>
                    </td>
                    <td style={{ color:'var(--text-muted)' }}>{db.tables} tables</td>
                    <td style={{ color:'var(--text-muted)' }}>{db.size}</td>
                    <td>
                      <div style={{ display:'flex', gap:6 }}>
                        {pmaUrl && (
                          <a href={`${pmaUrl}/index.php?db=${db.name}`} target="_blank" rel="noopener noreferrer"
                            id={`pma-db-${db.name}`} className="btn btn-ghost btn-sm">
                            <ExternalLink size={12}/>phpMyAdmin
                          </a>
                        )}
                        <button id={`drop-db-${db.name}`} onClick={() => dropDB(db.name)} className="btn btn-danger btn-sm" disabled={busy}>
                          <Trash2 size={12}/>Drop
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      )}

      {/* Users list */}
      {activeTab === 'users' && (
        <div className="glass-card">
          {users.length === 0 ? (
            <div style={{ padding:40, textAlign:'center' }}>
              <User size={36} color="var(--text-muted)" style={{ margin:'0 auto 10px' }}/>
              <div style={{ color:'var(--text-muted)' }}>No users found</div>
            </div>
          ) : (
            <table className="data-table">
              <thead><tr><th>Username</th><th>Host</th><th>Database</th><th>Actions</th></tr></thead>
              <tbody>
                {users.map(u => (
                  <tr key={u.user + u.host}>
                    <td><div style={{ display:'flex', alignItems:'center', gap:8 }}><User size={14} color="#a855f7"/><span className="mono">{u.user}</span></div></td>
                    <td style={{ color:'var(--text-muted)' }}>{u.host}</td>
                    <td style={{ color:'var(--text-muted)' }}>{u.db || '—'}</td>
                    <td>
                      <button id={`drop-user-${u.user}`} onClick={() => dropUser(u.user)} className="btn btn-danger btn-sm" disabled={busy}>
                        <Trash2 size={12}/>Drop
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      )}

      {/* Create DB modal */}
      {showCreateDB && (
        <div style={{ position:'fixed', inset:0, background:'rgba(0,0,0,0.75)', display:'flex', alignItems:'center', justifyContent:'center', zIndex:1000 }}>
          <div className="glass-card fade-in" style={{ padding:28, width:380 }}>
            <div style={{ fontWeight:700, fontSize:16, marginBottom:18 }}>Create Database</div>
            <label style={{ fontSize:13, color:'var(--text-secondary)', display:'block', marginBottom:6 }}>Database Name</label>
            <input id="db-name-input" className="input" value={dbName} onChange={e => setDbName(e.target.value)}
              placeholder="my_database" autoFocus onKeyDown={e => e.key === 'Enter' && createDB()} />
            <div style={{ fontSize:11, color:'var(--text-muted)', marginTop:6 }}>Only letters, numbers, underscores</div>
            <div style={{ display:'flex', gap:8, justifyContent:'flex-end', marginTop:18 }}>
              <button onClick={() => setShowCreateDB(false)} className="btn btn-ghost">Cancel</button>
              <button id="submit-create-db" onClick={createDB} className="btn btn-primary" disabled={busy}>
                {busy ? <span className="spinner"/> : <Database size={14}/>}Create
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Create user modal */}
      {showCreateUser && (
        <div style={{ position:'fixed', inset:0, background:'rgba(0,0,0,0.75)', display:'flex', alignItems:'center', justifyContent:'center', zIndex:1000 }}>
          <div className="glass-card fade-in" style={{ padding:28, width:420 }}>
            <div style={{ fontWeight:700, fontSize:16, marginBottom:18 }}>Create Database User</div>
            <div style={{ display:'flex', flexDirection:'column', gap:12 }}>
              {[
                { label:'Username', id:'user-name', key:'dbUser', placeholder:'myapp_user' },
                { label:'Password', id:'user-pass', key:'dbPassword', placeholder:'strong_password', type:'password' },
                { label:'Grant Access To Database', id:'user-db', key:'dbName', placeholder:'my_database' },
              ].map(f => (
                <div key={f.key}>
                  <label style={{ fontSize:13, color:'var(--text-secondary)', display:'block', marginBottom:5 }}>{f.label}</label>
                  <input id={f.id} className="input" type={f.type || 'text'} placeholder={f.placeholder}
                    value={userForm[f.key as keyof typeof userForm]}
                    onChange={e => setUserForm({ ...userForm, [f.key]: e.target.value })} />
                </div>
              ))}
            </div>
            <div style={{ display:'flex', gap:8, justifyContent:'flex-end', marginTop:18 }}>
              <button onClick={() => setShowCreateUser(false)} className="btn btn-ghost">Cancel</button>
              <button id="submit-create-user" onClick={createUser} className="btn btn-primary" disabled={busy}>
                {busy ? <span className="spinner"/> : <User size={14}/>}Create User
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
