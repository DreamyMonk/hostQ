import { NextRequest, NextResponse } from 'next/server';
import { cookies } from 'next/headers';
import fs from 'fs';
import { verifyToken } from '@/lib/auth';
import { auditLogPath } from '@/lib/security';
import { canManagePanel } from '@/lib/authz';

async function auth() {
  const cookieStore = await cookies();
  const token = cookieStore.get('panel_token')?.value;
  return token ? verifyToken(token) : null;
}

function verifyChain(lines: string[]) {
  let previous = '';
  for (const line of lines) {
    try {
      const entry = JSON.parse(line);
      if ((entry.prevHash || '') !== previous) return false;
      previous = entry.hash || '';
    } catch {
      return false;
    }
  }
  return true;
}

export async function GET(request: NextRequest) {
  const actor = await auth();
  if (!actor) return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });
  if (!canManagePanel(actor)) return NextResponse.json({ error: 'Forbidden' }, { status: 403 });

  const limit = Math.min(Number(request.nextUrl.searchParams.get('limit') || 200), 1000);
  try {
    const lines = fs.readFileSync(auditLogPath(), 'utf8').trim().split('\n').filter(Boolean);
    return NextResponse.json({
      entries: lines.slice(-limit).reverse().map((line) => JSON.parse(line)),
      chainValid: verifyChain(lines),
      rotated: fs.readdirSync(auditLogPath().replace(/audit\.log$/, '')).filter((file) => file.startsWith('audit.log.')),
    });
  } catch {
    return NextResponse.json({ entries: [], chainValid: true, rotated: [] });
  }
}
