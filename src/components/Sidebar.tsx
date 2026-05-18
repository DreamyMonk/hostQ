'use client';
import Link from 'next/link';
import { usePathname, useRouter } from 'next/navigation';
import {
  LayoutDashboard, FileCode2, FolderOpen, Globe, ShieldCheck,
  Database, Settings, Server, LogOut, ChevronRight, Boxes
} from 'lucide-react';

const NAV_ITEMS = [
  { href: '/dashboard',           icon: LayoutDashboard, label: 'Dashboard',    color: '#3b82f6' },
  { href: '/dashboard/domains',   icon: Globe,           label: 'Domains',      color: '#06b6d4' },
  { href: '/dashboard/php',       icon: FileCode2,       label: 'PHP Manager',  color: '#a855f7' },
  { href: '/dashboard/services',  icon: Server,          label: 'Services',     color: '#22c55e' },
  { href: '/dashboard/files',     icon: FolderOpen,      label: 'File Manager', color: '#f59e0b' },
  { href: '/dashboard/wordpress', icon: Boxes,           label: 'WordPress',    color: '#3b82f6' },
  { href: '/dashboard/ssl',       icon: ShieldCheck,     label: 'SSL Manager',  color: '#f97316' },
  { href: '/dashboard/databases', icon: Database,        label: 'Databases',    color: '#8b5cf6' },
  { href: '/dashboard/settings',  icon: Settings,        label: 'Settings',     color: '#8b949e' },
];

export default function Sidebar() {
  const pathname = usePathname();
  const router   = useRouter();

  const handleLogout = async () => {
    await fetch('/api/auth', { method: 'DELETE' });
    router.push('/');
  };

  return (
    <nav className="sidebar">
      {/* Logo */}
      <div style={{
        padding: '18px 20px', borderBottom: '1px solid var(--border-subtle)',
        display: 'flex', alignItems: 'center', gap: 10,
      }}>
        <div style={{
          width: 36, height: 36,
          background: 'linear-gradient(135deg, #3b82f6, #8b5cf6)',
          borderRadius: 9, display: 'flex', alignItems: 'center',
          justifyContent: 'center', flexShrink: 0,
        }}>
          <Server size={18} color="white" />
        </div>
        <div>
          <div style={{ fontWeight: 800, fontSize: 15, letterSpacing: '-0.3px' }}>HostPanel</div>
          <div style={{ fontSize: 10, color: 'var(--text-muted)', fontWeight: 500 }}>Control Panel</div>
        </div>
      </div>

      {/* Nav items */}
      <div style={{ flex: 1, padding: '12px 10px', display: 'flex', flexDirection: 'column', gap: 2 }}>
        <div style={{
          fontSize: 10, fontWeight: 600, letterSpacing: '0.8px', textTransform: 'uppercase',
          color: 'var(--text-muted)', padding: '8px 10px 6px'
        }}>Main Menu</div>

        {NAV_ITEMS.map(({ href, icon: Icon, label, color }) => {
          const isActive = href === '/dashboard'
            ? pathname === '/dashboard'
            : pathname.startsWith(href);

          return (
            <Link key={href} href={href} style={{ textDecoration: 'none' }}>
              <div style={{
                display: 'flex', alignItems: 'center', gap: 10,
                padding: '9px 10px', borderRadius: 8, cursor: 'pointer',
                transition: 'all 0.15s',
                background: isActive ? `${color}18` : 'transparent',
                border: `1px solid ${isActive ? `${color}30` : 'transparent'}`,
                color: isActive ? color : 'var(--text-secondary)',
                position: 'relative',
              }}>
                {isActive && (
                  <div style={{
                    position: 'absolute', left: 0, top: '50%', transform: 'translateY(-50%)',
                    width: 3, height: 20, background: color, borderRadius: '0 2px 2px 0'
                  }} />
                )}
                <Icon size={16} style={{ flexShrink: 0 }} />
                <span style={{ fontSize: 13, fontWeight: isActive ? 600 : 400 }}>{label}</span>
                {isActive && <ChevronRight size={13} style={{ marginLeft: 'auto', opacity: 0.5 }} />}
              </div>
            </Link>
          );
        })}
      </div>

      {/* Logout */}
      <div style={{ padding: '12px 10px', borderTop: '1px solid var(--border-subtle)' }}>
        <button
          id="logout-btn"
          onClick={handleLogout}
          style={{
            width: '100%', display: 'flex', alignItems: 'center', gap: 10,
            padding: '9px 10px', borderRadius: 8, cursor: 'pointer',
            background: 'transparent', border: 'none', color: 'var(--text-muted)',
            fontSize: 13, fontWeight: 400, transition: 'all 0.15s',
          }}
          onMouseEnter={e => {
            (e.currentTarget as HTMLButtonElement).style.background = 'rgba(239,68,68,0.1)';
            (e.currentTarget as HTMLButtonElement).style.color = '#ef4444';
          }}
          onMouseLeave={e => {
            (e.currentTarget as HTMLButtonElement).style.background = 'transparent';
            (e.currentTarget as HTMLButtonElement).style.color = 'var(--text-muted)';
          }}
        >
          <LogOut size={16} />
          Logout
        </button>
      </div>
    </nav>
  );
}
