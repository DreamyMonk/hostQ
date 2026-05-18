'use client';
import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { ArrowRight, Eye, EyeOff, Loader2, Lock, Server, ShieldCheck, User } from 'lucide-react';

export default function LoginPage() {
  const router = useRouter();
  const [setupRequired, setSetupRequired] = useState<boolean | null>(null);
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [otp, setOtp] = useState('');
  const [showPass, setShowPass] = useState(false);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    const id = setTimeout(async () => {
      try {
        const response = await fetch('/api/auth');
        const data = await response.json();
        setSetupRequired(Boolean(data.setupRequired));
      } catch {
        setSetupRequired(false);
      }
    }, 0);
    return () => clearTimeout(id);
  }, []);

  const submit = async (event: React.FormEvent) => {
    event.preventDefault();
    setLoading(true);
    setError('');
    try {
      const response = await fetch('/api/auth', {
        method: setupRequired ? 'PUT' : 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username, password, confirmPassword, otp }),
      });
      const raw = await response.text();
      let data: { success?: boolean; requiresLogin?: boolean; error?: string; otpSecret?: string; otpAuthUrl?: string } = {};
      try {
        data = raw ? JSON.parse(raw) : {};
      } catch {
        setError(raw || `Server returned ${response.status}`);
        return;
      }
      if (response.ok && data.requiresLogin) {
        setSetupRequired(false);
        setPassword('');
        setConfirmPassword('');
        setOtp('');
        setError('');
        return;
      }
      if (response.ok && data.success) {
        router.push('/dashboard/sites');
        return;
      }
      setError(data.error || 'Unable to continue');
    } catch (error) {
      setError(error instanceof Error ? `Connection failed: ${error.message}` : 'Connection failed. Please try again.');
    } finally {
      setLoading(false);
    }
  };

  const isSetup = setupRequired === true;

  return (
    <main style={{ minHeight: '100vh', background: '#f6f8fb', display: 'grid', gridTemplateColumns: 'minmax(0, 1.05fr) minmax(420px, 0.95fr)' }}>
      <section style={{ padding: 48, display: 'flex', flexDirection: 'column', justifyContent: 'space-between' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
          <div style={{ width: 42, height: 42, borderRadius: 10, background: '#111827', color: 'white', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
            <Server size={21} />
          </div>
          <div>
            <div style={{ fontWeight: 850, fontSize: 22, letterSpacing: '-0.04em' }}>hostQ</div>
            <div style={{ color: '#667085', fontSize: 13 }}>Practical hosting control panel</div>
          </div>
        </div>

        <div style={{ maxWidth: 680 }}>
          <div className="badge badge-blue" style={{ marginBottom: 18 }}>v2 control panel</div>
          <h1 style={{ fontSize: 56, lineHeight: 1, letterSpacing: '-0.06em', color: '#101828', marginBottom: 20 }}>
            Manage sites without babysitting the server.
          </h1>
          <p style={{ color: '#475467', fontSize: 18, lineHeight: 1.7, maxWidth: 580 }}>
            Add sites, manage SSL, files, WordPress, databases, PHP and server services from a clean white workspace built for small VPS hosting.
          </p>
          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, minmax(0, 1fr))', gap: 12, marginTop: 34 }}>
            {[
              ['Sites', 'User workspace'],
              ['Admin', 'Server controls'],
              ['1GB VPS', 'Lightweight stack'],
            ].map(([label, value]) => (
              <div key={label} className="glass-card" style={{ padding: 18 }}>
                <div style={{ fontSize: 12, color: '#667085', marginBottom: 6 }}>{label}</div>
                <div style={{ fontWeight: 800, color: '#101828' }}>{value}</div>
              </div>
            ))}
          </div>
        </div>

        <div style={{ display: 'flex', gap: 10, color: '#667085', fontSize: 13 }}>
          <ShieldCheck size={16} color="#16a34a" />
          VPS setup prints the generated admin username and password in SSH.
        </div>
      </section>

      <section style={{ background: 'white', borderLeft: '1px solid #e4e7ec', display: 'flex', alignItems: 'center', justifyContent: 'center', padding: 32 }}>
        <div style={{ width: '100%', maxWidth: 420 }}>
          <div style={{ marginBottom: 28 }}>
            <div style={{ color: '#2563eb', fontSize: 13, fontWeight: 800, textTransform: 'uppercase', letterSpacing: '0.08em', marginBottom: 8 }}>
              {setupRequired === null ? 'Checking panel' : isSetup ? 'First run setup' : 'Welcome back'}
            </div>
            <h2 style={{ fontSize: 30, letterSpacing: '-0.04em', color: '#101828', marginBottom: 8 }}>
              {isSetup ? 'Create your admin account' : 'Sign in to hostQ'}
            </h2>
            <p style={{ color: '#667085', fontSize: 14, lineHeight: 1.6 }}>
              {isSetup ? 'Manual fallback setup. On VPS installs, setup.sh generates this account in SSH.' : 'Use the admin credentials printed at the end of setup.sh.'}
            </p>
          </div>

          {error && <div className="alert alert-error">{error}</div>}

          <form onSubmit={submit} style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
            <label style={{ display: 'grid', gap: 6 }}>
              <span style={{ fontSize: 13, color: '#344054', fontWeight: 650 }}>Username</span>
              <div style={{ position: 'relative' }}>
                <User size={16} style={{ position: 'absolute', left: 12, top: '50%', transform: 'translateY(-50%)', color: '#98a2b3' }} />
                <input className="input" style={{ paddingLeft: 38 }} value={username} onChange={event => setUsername(event.target.value)} placeholder="admin" autoComplete="username" required />
              </div>
            </label>

            <label style={{ display: 'grid', gap: 6 }}>
              <span style={{ fontSize: 13, color: '#344054', fontWeight: 650 }}>Password</span>
              <div style={{ position: 'relative' }}>
                <Lock size={16} style={{ position: 'absolute', left: 12, top: '50%', transform: 'translateY(-50%)', color: '#98a2b3' }} />
                <input className="input" style={{ paddingLeft: 38, paddingRight: 42 }} type={showPass ? 'text' : 'password'} value={password} onChange={event => setPassword(event.target.value)} placeholder="At least 10 characters" autoComplete={isSetup ? 'new-password' : 'current-password'} required />
                <button type="button" onClick={() => setShowPass(value => !value)} style={{ position: 'absolute', right: 12, top: '50%', transform: 'translateY(-50%)', border: 0, background: 'transparent', color: '#667085', cursor: 'pointer' }}>
                  {showPass ? <EyeOff size={16} /> : <Eye size={16} />}
                </button>
              </div>
            </label>

            {isSetup && (
              <label style={{ display: 'grid', gap: 6 }}>
                <span style={{ fontSize: 13, color: '#344054', fontWeight: 650 }}>Confirm password</span>
                <input className="input" type={showPass ? 'text' : 'password'} value={confirmPassword} onChange={event => setConfirmPassword(event.target.value)} placeholder="Repeat password" autoComplete="new-password" required />
              </label>
            )}

            {!isSetup && (
              <label style={{ display: 'grid', gap: 6 }}>
                <span style={{ fontSize: 13, color: '#344054', fontWeight: 650 }}>2FA code</span>
                <input className="input mono" inputMode="numeric" pattern="[0-9]*" value={otp} onChange={event => setOtp(event.target.value)} placeholder="Only if enabled" autoComplete="one-time-code" />
              </label>
            )}

            <button className="btn btn-primary btn-lg" style={{ justifyContent: 'center', marginTop: 8 }} disabled={loading || setupRequired === null}>
              {loading || setupRequired === null ? <Loader2 size={17} className="animate-spin" /> : <ArrowRight size={17} />}
              {setupRequired === null ? 'Checking...' : isSetup ? 'Create account' : 'Sign in'}
            </button>
          </form>
        </div>
      </section>
    </main>
  );
}
