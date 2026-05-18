'use client';
import Sidebar from '@/components/Sidebar';
import SecurityFetch from '@/components/SecurityFetch';
import TopBar from '@/components/TopBar';

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  return (
    <>
      <SecurityFetch />
      <Sidebar />
      <TopBar />
      <main className="main-content">
        {children}
      </main>
    </>
  );
}
