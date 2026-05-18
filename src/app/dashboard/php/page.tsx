'use client';
import { useEffect, useState } from 'react';
import { FileCode2, RefreshCw, ChevronRight, Terminal, CheckCircle, AlertCircle } from 'lucide-react';

export default function PHPPage() {
  const [loading, setLoading] = useState(true);
  const [switching, setSwitching] = useState(false);
  const [data, setData] = useState<{ active: string; versions: string[]; currentOutput: string; demo?: boolean } | null>(null);
  const [output, setOutput] = useState('');
  const [msg, setMsg] = useState<{ type: 'success' | 'error'; text: string } | null>(null);

  const fetchData = async () => {
    setLoading(true);
    try {
      const r = await fetch('/api/php');
      setData(await r.json());
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    const id = setTimeout(() => { void fetchData(); }, 0);
    return () => clearTimeout(id);
  }, []);

  const switchVersion = async (version: string) => {
    if (version === data?.active) return;
    setSwitching(true);
    setMsg(null);
    setOutput('Switching PHP version…');
    try {
      const r = await fetch('/api/php', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ version }),
      });
      const d = await r.json();
      setOutput(d.output || '');
      if (d.success) {
        setMsg({ type: 'success', text: d.message });
        fetchData();
      } else {
        setMsg({ type: 'error', text: d.error || 'Failed to switch' });
      }
    } finally {
      setSwitching(false);
    }
  };

  const VERSION_INFO: Record<string, { label: string; eol: string; color: string; status: string }> = {
    '8.2': { label: 'PHP 8.2', eol: 'Security fixes until Dec 2026', color: '#3b82f6', status: 'security' },
    '8.3': { label: 'PHP 8.3', eol: 'Security fixes until Dec 2027', color: '#22c55e', status: 'stable' },
    '8.4': { label: 'PHP 8.4', eol: 'Active support until Dec 2026', color: '#a855f7', status: 'active' },
    '8.5': { label: 'PHP 8.5', eol: 'Active support until Dec 2027', color: '#06b6d4', status: 'latest' },
  };

  return (
    <div className="fade-in">
      <div className="page-header" style={{ display:'flex', alignItems:'center', justifyContent:'space-between' }}>
        <div>
          <h1 className="page-title">PHP Version Manager</h1>
          <p className="page-subtitle">Switch PHP versions for your server</p>
        </div>
        <button id="refresh-php-btn" onClick={fetchData} className="btn btn-ghost btn-sm" disabled={loading}>
          <RefreshCw size={14} style={{ animation: loading ? 'spin 1s linear infinite' : 'none' }} />Refresh
        </button>
      </div>

      {data?.demo && (
        <div className="alert alert-warning" style={{ marginBottom:20 }}>
          ⚠️ Demo mode — PHP or update-alternatives not detected. On a Linux VPS, this will show real installed PHP versions.
        </div>
      )}

      {msg && (
        <div className={`alert ${msg.type === 'success' ? 'alert-success' : 'alert-error'}`} style={{ marginBottom:20, display:'flex', alignItems:'center', gap:8 }}>
          {msg.type === 'success' ? <CheckCircle size={15}/> : <AlertCircle size={15}/>}
          {msg.text}
        </div>
      )}

      {/* Current version */}
      {data && (
        <div className="glass-card" style={{ padding:20, marginBottom:20, display:'flex', alignItems:'center', gap:16 }}>
          <div style={{
            width:52, height:52, background:'rgba(168,85,247,0.12)', border:'1px solid rgba(168,85,247,0.25)',
            borderRadius:12, display:'flex', alignItems:'center', justifyContent:'center'
          }}>
            <FileCode2 size={24} color="#a855f7" />
          </div>
          <div>
            <div style={{ fontSize:13, color:'var(--text-muted)', marginBottom:2 }}>Currently Active</div>
            <div style={{ fontSize:26, fontWeight:800, color:'#a855f7' }}>PHP {data.active || 'N/A'}</div>
          </div>
          {data.active && VERSION_INFO[data.active] && (
            <span className="badge" style={{ marginLeft:'auto',
              background: data.active?.startsWith('8.3') ? 'rgba(168,85,247,0.12)' : 'rgba(34,197,94,0.12)',
              color: data.active?.startsWith('8.3') ? '#a855f7' : '#22c55e',
              border: `1px solid ${data.active?.startsWith('8.3') ? 'rgba(168,85,247,0.25)' : 'rgba(34,197,94,0.25)'}`
            }}>
              {VERSION_INFO[data.active]?.eol}
            </span>
          )}
        </div>
      )}

      {/* Version cards */}
      <div style={{ display:'grid', gridTemplateColumns:'repeat(auto-fill, minmax(200px, 1fr))', gap:14, marginBottom:24 }}>
        {loading ? (
          Array.from({length:5}).map((_,i) => (
            <div key={i} className="glass-card" style={{ padding:20, height:120 }}>
              <div style={{ background:'var(--bg-elevated)', borderRadius:6, height:10, width:'60%', marginBottom:8 }}/>
              <div style={{ background:'var(--bg-elevated)', borderRadius:6, height:8, width:'40%' }}/>
            </div>
          ))
        ) : data?.versions.map(ver => {
          const info = VERSION_INFO[ver] || { label:`PHP ${ver}`, eol:'', color:'#8b949e', status:'unknown' };
          const isActive = ver === data.active;
          return (
            <div
              key={ver}
              id={`php-${ver}-btn`}
              className="glass-card glass-card-hover"
              style={{
                padding:20, cursor: switching ? 'not-allowed' : 'pointer',
                border: isActive ? `1px solid ${info.color}40` : '1px solid var(--border-default)',
                background: isActive ? `${info.color}10` : 'var(--bg-card)',
                opacity: switching && !isActive ? 0.7 : 1,
              }}
              onClick={() => !switching && switchVersion(ver)}
            >
              <div style={{ display:'flex', alignItems:'center', justifyContent:'space-between', marginBottom:10 }}>
                <span style={{ fontSize:18, fontWeight:800, color: info.color }}>{info.label}</span>
                {isActive ? (
                  <CheckCircle size={16} color={info.color} />
                ) : (
                  <ChevronRight size={14} color="var(--text-muted)" />
                )}
              </div>
              <div style={{ fontSize:11, color:'var(--text-muted)', marginBottom:12 }}>{info.eol}</div>
              <div>
                {isActive ? (
                  <span className="badge badge-green">● Active</span>
                ) : (
                  <button
                    className="btn btn-ghost btn-sm"
                    style={{ fontSize:11, padding:'3px 10px' }}
                    disabled={switching}
                  >
                    {switching ? <span className="spinner" style={{width:10,height:10}} /> : null}
                    Switch
                  </button>
                )}
              </div>
            </div>
          );
        })}
      </div>

      {/* Terminal output */}
      {output && (
        <div>
          <div style={{ display:'flex', alignItems:'center', gap:8, marginBottom:8, fontSize:13, fontWeight:600, color:'var(--text-secondary)' }}>
            <Terminal size={14} /> Command Output
          </div>
          <div className="terminal">
            {output.split('\n').map((line, i) => {
              let cls = '';
              if (line.startsWith('✓')) cls = 'line-success';
              else if (line.startsWith('✗') || line.toLowerCase().includes('error')) cls = 'line-error';
              else if (line.startsWith('$')) cls = 'line-cmd';
              else if (line.startsWith('▶')) cls = 'line-info';
              return <div key={i} className={cls}>{line || ' '}</div>;
            })}
          </div>
        </div>
      )}

      {/* Version info table */}
      <div className="glass-card" style={{ marginTop:24 }}>
        <div style={{ padding:'16px 20px', borderBottom:'1px solid var(--border-subtle)', fontWeight:600, fontSize:14 }}>
          PHP Version Reference
        </div>
        <table className="data-table">
          <thead>
            <tr>
              <th>Version</th><th>Status</th><th>End of Life</th><th>Recommended For</th>
            </tr>
          </thead>
          <tbody>
            {[
              ['8.5', 'latest', 'Dec 2029 security', 'Newest supported PHP branch'],
              ['8.4', 'active', 'Dec 2028 security', 'New production sites'],
              ['8.3', 'stable', 'Dec 2027 security', 'Most modern WordPress and PHP apps'],
              ['8.2', 'security', 'Dec 2026 security', 'Compatibility fallback only'],
            ].map(([v, status, eol, rec]) => (
              <tr key={v}>
                <td className="mono">PHP {v}</td>
                <td>
                  <span className={`badge ${status === 'latest' ? 'badge-purple' : status === 'stable' || status === 'active' ? 'badge-blue' : status === 'security' ? 'badge-green' : 'badge-red'}`}>
                    {status}
                  </span>
                </td>
                <td style={{ color:'var(--text-secondary)', fontSize:12 }}>{eol}</td>
                <td style={{ color:'var(--text-muted)', fontSize:12 }}>{rec}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
