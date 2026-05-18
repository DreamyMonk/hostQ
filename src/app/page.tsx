'use client';
import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { Server, Lock, User, Eye, EyeOff, Loader2 } from 'lucide-react';

export default function LoginPage() {
  const router = useRouter();
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [showPass, setShowPass] = useState(false);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError('');
    try {
      const res = await fetch('/api/auth', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username, password }),
      });
      const data = await res.json();
      if (res.ok && data.success) {
        router.push('/dashboard');
      } else {
        setError(data.error || 'Invalid credentials');
      }
    } catch {
      setError('Connection failed. Please try again.');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen flex" style={{ background: 'var(--bg-base)' }}>
      {/* Left panel */}
      <div className="hidden lg:flex flex-col justify-between w-1/2 p-12 relative overflow-hidden"
        style={{
          background: 'linear-gradient(135deg, #0f1117 0%, #161b22 50%, #0a0c10 100%)',
          borderRight: '1px solid var(--border-subtle)'
        }}>
        {/* Animated background orbs */}
        <div style={{
          position:'absolute', top:'-100px', left:'-100px', width:'400px', height:'400px',
          background:'radial-gradient(circle, rgba(59,130,246,0.15) 0%, transparent 70%)',
          borderRadius:'50%', pointerEvents:'none'
        }} />
        <div style={{
          position:'absolute', bottom:'-80px', right:'-80px', width:'350px', height:'350px',
          background:'radial-gradient(circle, rgba(168,85,247,0.12) 0%, transparent 70%)',
          borderRadius:'50%', pointerEvents:'none'
        }} />

        {/* Logo */}
        <div style={{ display:'flex', alignItems:'center', gap:12, position:'relative' }}>
          <div style={{
            width:42, height:42, background:'linear-gradient(135deg, #3b82f6, #8b5cf6)',
            borderRadius:10, display:'flex', alignItems:'center', justifyContent:'center'
          }}>
            <Server size={22} color="white" />
          </div>
          <div>
            <div style={{ fontWeight:800, fontSize:18, letterSpacing:'-0.5px' }}>HostPanel</div>
            <div style={{ fontSize:11, color:'var(--text-muted)', fontWeight:500 }}>Hosting Control Panel</div>
          </div>
        </div>

        {/* Features list */}
        <div style={{ position:'relative' }}>
          <h2 style={{ fontSize:32, fontWeight:800, lineHeight:1.2, marginBottom:8 }}>
            Your server,<br />
            <span className="gradient-text">fully in control.</span>
          </h2>
          <p style={{ color:'var(--text-secondary)', fontSize:15, marginBottom:40, lineHeight:1.6 }}>
            Manage PHP versions, databases, SSL certificates, WordPress installs, and files from one premium dashboard.
          </p>
          <div style={{ display:'flex', flexDirection:'column', gap:16 }}>
            {[
              { icon:'PHP', label:'PHP Version Manager', desc:'Switch supported PHP 8.2, 8.3, 8.4, 8.5' },
              { icon:'📁', label:'File Manager', desc:'Upload, edit, organize server files' },
              { icon:'🌐', label:'WordPress Installer', desc:'One-click WP installs with WP-CLI' },
              { icon:'🔒', label:'SSL Manager', desc:"Free Let's Encrypt SSL certificates" },
              { icon:'🗄️', label:'Database Manager', desc:'MySQL databases with phpMyAdmin' },
            ].map((f) => (
              <div key={f.label} style={{ display:'flex', alignItems:'flex-start', gap:14 }}>
                <div style={{
                  width:38, height:38, background:'var(--bg-card)', border:'1px solid var(--border-default)',
                  borderRadius:8, display:'flex', alignItems:'center', justifyContent:'center',
                  fontSize:16, flexShrink:0
                }}>
                  {f.icon}
                </div>
                <div>
                  <div style={{ fontWeight:600, fontSize:14, marginBottom:2 }}>{f.label}</div>
                  <div style={{ fontSize:12, color:'var(--text-muted)' }}>{f.desc}</div>
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* Bottom badge */}
        <div style={{ display:'flex', gap:8, position:'relative' }}>
          <span className="badge badge-green"> <span className="pulse-dot" style={{width:6,height:6,borderRadius:'50%',background:'#22c55e',display:'inline-block'}} />Single User</span>
          <span className="badge badge-blue">Self-Hosted</span>
          <span className="badge badge-purple">VPS Ready</span>
        </div>
      </div>

      {/* Right panel - Login form */}
      <div className="flex-1 flex items-center justify-center p-8">
        <div className="w-full max-w-sm fade-in">
          {/* Mobile logo */}
          <div className="lg:hidden flex items-center gap-3 mb-8">
            <div style={{
              width:38, height:38, background:'linear-gradient(135deg, #3b82f6, #8b5cf6)',
              borderRadius:10, display:'flex', alignItems:'center', justifyContent:'center'
            }}>
              <Server size={20} color="white" />
            </div>
            <span style={{ fontWeight:700, fontSize:16 }}>HostPanel</span>
          </div>

          <h1 style={{ fontSize:26, fontWeight:800, marginBottom:6 }}>Welcome back</h1>
          <p style={{ color:'var(--text-secondary)', fontSize:14, marginBottom:32 }}>
            Sign in to your hosting panel
          </p>

          {error && (
            <div className="alert alert-error" style={{ marginBottom:20 }}>
              {error}
            </div>
          )}

          <form onSubmit={handleLogin} style={{ display:'flex', flexDirection:'column', gap:16 }}>
            <div>
              <label style={{ fontSize:13, fontWeight:500, color:'var(--text-secondary)', marginBottom:6, display:'block' }}>
                Username
              </label>
              <div style={{ position:'relative' }}>
                <User size={15} style={{ position:'absolute', left:12, top:'50%', transform:'translateY(-50%)', color:'var(--text-muted)' }} />
                <input
                  id="username"
                  className="input"
                  style={{ paddingLeft:36 }}
                  type="text"
                  value={username}
                  onChange={e => setUsername(e.target.value)}
                  placeholder="admin"
                  autoComplete="username"
                  required
                />
              </div>
            </div>

            <div>
              <label style={{ fontSize:13, fontWeight:500, color:'var(--text-secondary)', marginBottom:6, display:'block' }}>
                Password
              </label>
              <div style={{ position:'relative' }}>
                <Lock size={15} style={{ position:'absolute', left:12, top:'50%', transform:'translateY(-50%)', color:'var(--text-muted)' }} />
                <input
                  id="password"
                  className="input"
                  style={{ paddingLeft:36, paddingRight:40 }}
                  type={showPass ? 'text' : 'password'}
                  value={password}
                  onChange={e => setPassword(e.target.value)}
                  placeholder="••••••••"
                  autoComplete="current-password"
                  required
                />
                <button type="button" onClick={() => setShowPass(!showPass)}
                  style={{ position:'absolute', right:12, top:'50%', transform:'translateY(-50%)',
                    background:'none', border:'none', cursor:'pointer', color:'var(--text-muted)', padding:2
                  }}>
                  {showPass ? <EyeOff size={15} /> : <Eye size={15} />}
                </button>
              </div>
            </div>

            <button
              id="login-btn"
              type="submit"
              className="btn btn-primary btn-lg"
              style={{ width:'100%', justifyContent:'center', marginTop:4 }}
              disabled={loading}
            >
              {loading ? <Loader2 size={16} className="animate-spin" /> : <Lock size={16} />}
              {loading ? 'Signing in…' : 'Sign In'}
            </button>
          </form>

          <p style={{ marginTop:24, fontSize:12, color:'var(--text-muted)', textAlign:'center' }}>
            Default: admin / admin123 (change in .env.local)
          </p>

          {/* Security note */}
          <div style={{
            marginTop:28, padding:'12px 14px', background:'rgba(245,158,11,0.06)',
            border:'1px solid rgba(245,158,11,0.15)', borderRadius:8, fontSize:12, color:'#f59e0b'
          }}>
            🔒 Change default credentials in <span className="mono">.env.local</span> before production use.
          </div>
        </div>
      </div>
    </div>
  );
}
