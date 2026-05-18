'use client';
import { usePathname } from 'next/navigation';
import { Bell, ShieldCheck } from 'lucide-react';
import { useEffect, useState } from 'react';

const PAGE_TITLES: Record<string, { title: string; subtitle: string }> = {
  '/dashboard':            { title: 'Dashboard',      subtitle: 'Server overview & resource usage' },
  '/dashboard/sites':      { title: 'Sites',          subtitle: 'User mode site workspace' },
  '/dashboard/domains':    { title: 'Domain Manager',  subtitle: 'Add domains, subdomains & vhost configs' },
  '/dashboard/php':        { title: 'PHP Manager',     subtitle: 'Switch PHP versions & manage extensions' },
  '/dashboard/services':   { title: 'Services',         subtitle: 'Manage Nginx, Apache, MySQL & auto-install' },
  '/dashboard/files':      { title: 'File Manager',    subtitle: 'Browse, upload and manage server files' },
  '/dashboard/wordpress':  { title: 'WordPress',       subtitle: 'Install and manage WordPress sites' },
  '/dashboard/ssl':        { title: 'SSL Manager',     subtitle: "Manage Let's Encrypt SSL certificates" },
  '/dashboard/databases':  { title: 'Databases',       subtitle: 'Create and manage MySQL databases' },
  '/dashboard/settings':   { title: 'Security',        subtitle: 'Production readiness, audit logs and sessions' },
};

export default function TopBar() {
  const pathname = usePathname();
  const info = PAGE_TITLES[pathname] || { title: 'hostQ', subtitle: '' };
  const now = new Date().toLocaleDateString('en-US', { weekday:'long', month:'long', day:'numeric' });
  const [me, setMe] = useState<{ username: string; role: string } | null>(null);

  useEffect(() => {
    void fetch('/api/me').then(response => response.ok ? response.json() : null).then(data => data && setMe(data)).catch(() => {});
  }, []);

  return (
    <div className="topbar">
      <div style={{ flex: 1 }}>
        <div style={{ fontWeight: 700, fontSize: 16 }}>{info.title}</div>
        <div style={{ fontSize: 12, color: 'var(--text-muted)' }}>{info.subtitle}</div>
      </div>
      <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
        <span style={{ fontSize: 12, color: 'var(--text-muted)' }}>{now}</span>
        <div className="badge badge-green" style={{ height: 28 }}>
          <ShieldCheck size={13} /> Secure
        </div>
        <div style={{
          width: 34, height: 34, background: 'var(--bg-card)',
          border: '1px solid var(--border-default)', borderRadius: 8,
          display: 'flex', alignItems: 'center', justifyContent: 'center', cursor: 'pointer'
        }}>
          <Bell size={15} color="var(--text-secondary)" />
        </div>
        <div style={{
          display: 'flex', alignItems: 'center', gap: 8,
          background: 'var(--bg-card)', border: '1px solid var(--border-default)',
          borderRadius: 8, padding: '5px 10px'
        }}>
          <div style={{
            width: 22, height: 22, background: 'linear-gradient(135deg, #3b82f6, #8b5cf6)',
            borderRadius: '50%', display: 'flex', alignItems: 'center', justifyContent: 'center',
            fontSize: 11, fontWeight: 700, color: 'white'
          }}>{(me?.username || 'A').slice(0, 1).toUpperCase()}</div>
          <span style={{ fontSize: 13, fontWeight: 500 }}>{me?.username || 'admin'}</span>
          <span className="badge badge-blue">{me?.role || 'admin'}</span>
        </div>
      </div>
    </div>
  );
}
