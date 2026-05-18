import { NextResponse } from 'next/server';
import { cookies } from 'next/headers';
import { verifyToken } from '@/lib/auth';
import { mysqlIdentifier, mysqlString, runMysql } from '@/lib/exec';
import { audit, clientIp } from '@/lib/security';
import { canManagePanel } from '@/lib/authz';

async function auth() {
  const cookieStore = await cookies();
  const token = cookieStore.get('panel_token')?.value;
  return token ? verifyToken(token) : null;
}

// GET - list all databases
export async function GET() {
  const actor = await auth();
  if (!actor) return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });
  if (!canManagePanel(actor)) return NextResponse.json({ error: 'Forbidden' }, { status: 403 });

  const r = await runMysql('SHOW DATABASES;');
  if (!r.success) {
    // Return demo data if MySQL not available
    return NextResponse.json({
      databases: [
        { name: 'wordpress_site1', size: '24 MB', tables: 12, created: '2024-01-15' },
        { name: 'wordpress_site2', size: '8 MB', tables: 12, created: '2024-02-10' },
        { name: 'myapp_db', size: '156 MB', tables: 28, created: '2024-03-01' },
      ],
      users: [
        { user: 'wp_user1', host: 'localhost', db: 'wordpress_site1' },
        { user: 'wp_user2', host: 'localhost', db: 'wordpress_site2' },
        { user: 'app_user', host: 'localhost', db: 'myapp_db' },
      ],
      phpmyadmin: process.env.PHPMYADMIN_URL || 'http://localhost/phpmyadmin',
      demo: true,
    });
  }

  const excluded = ['information_schema', 'performance_schema', 'mysql', 'sys'];
  const dbs = r.stdout.split('\n')
    .slice(1)
    .filter(db => db && !excluded.includes(db.trim()))
    .map(db => db.trim())
    .filter(Boolean);

  // Get sizes
  const dbsWithSize = await Promise.all(dbs.map(async (db) => {
    const sizeR = await runMysql(
      `SELECT ROUND(SUM(data_length + index_length) / 1024 / 1024, 2) AS size_mb FROM information_schema.TABLES WHERE table_schema = '${db}';`
    );
    const lines = sizeR.stdout.split('\n');
    const size = lines[1]?.trim() || '0';
    const tableR = await runMysql(`SELECT COUNT(*) FROM information_schema.TABLES WHERE table_schema = '${db}';`);
    const tables = parseInt(tableR.stdout.split('\n')[1] || '0');
    return { name: db, size: `${size} MB`, tables };
  }));

  // Get users
  const usersR = await runMysql("SELECT user, host FROM mysql.user WHERE user != 'root' AND user != '';");
  const users = usersR.stdout.split('\n').slice(1).map(line => {
    const [user, host] = line.trim().split('\t');
    return { user, host };
  }).filter(u => u.user);

  return NextResponse.json({
    databases: dbsWithSize,
    users,
    phpmyadmin: process.env.PHPMYADMIN_URL || 'http://localhost/phpmyadmin',
    demo: false,
  });
}

// POST - create database / user
export async function POST(request: Request) {
  const actor = await auth();
  if (!actor) return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });
  if (!canManagePanel(actor)) return NextResponse.json({ error: 'Forbidden' }, { status: 403 });

  const body = await request.json();
  const { action, dbName, dbUser, dbPassword } = body;

  if (action === 'create_db') {
    if (!dbName || !/^[a-zA-Z0-9_]+$/.test(dbName)) {
      return NextResponse.json({ error: 'Invalid database name' }, { status: 400 });
    }
    const r = await runMysql(`CREATE DATABASE IF NOT EXISTS ${mysqlIdentifier(dbName)};`);
    if (!r.success) return NextResponse.json({ error: r.error || r.stderr }, { status: 500 });
    audit({ actor: actor.username, action: 'database.create', target: dbName, ip: clientIp(request) });
    return NextResponse.json({ success: true, message: `Database '${dbName}' created` });
  }

  if (action === 'create_user') {
    if (!dbUser || !dbPassword || !dbName || !/^[a-zA-Z0-9_]+$/.test(dbUser) || !/^[a-zA-Z0-9_]+$/.test(dbName)) {
      return NextResponse.json({ error: 'dbUser, dbPassword, dbName required' }, { status: 400 });
    }
    const cmds = [
      `CREATE USER IF NOT EXISTS ${mysqlString(dbUser)}@'localhost' IDENTIFIED BY ${mysqlString(dbPassword)};`,
      `GRANT ALL PRIVILEGES ON ${mysqlIdentifier(dbName)}.* TO ${mysqlString(dbUser)}@'localhost';`,
      `FLUSH PRIVILEGES;`,
    ];
    for (const sql of cmds) {
      const r = await runMysql(sql);
      if (!r.success) return NextResponse.json({ error: r.error || r.stderr }, { status: 500 });
    }
    audit({ actor: actor.username, action: 'database_user.create', target: `${dbName}:${dbUser}`, ip: clientIp(request) });
    return NextResponse.json({ success: true, message: `User '${dbUser}' created and granted on '${dbName}'` });
  }

  return NextResponse.json({ error: 'Unknown action' }, { status: 400 });
}

// DELETE - drop database or user
export async function DELETE(request: Request) {
  const actor = await auth();
  if (!actor) return NextResponse.json({ error: 'Unauthorized' }, { status: 401 });
  if (!canManagePanel(actor)) return NextResponse.json({ error: 'Forbidden' }, { status: 403 });

  const { action, name } = await request.json();
  
  if (action === 'drop_db') {
    if (!name || !/^[a-zA-Z0-9_]+$/.test(name)) return NextResponse.json({ error: 'Invalid database name' }, { status: 400 });
    const r = await runMysql(`DROP DATABASE IF EXISTS ${mysqlIdentifier(name)};`);
    if (!r.success) return NextResponse.json({ error: r.stderr }, { status: 500 });
    audit({ actor: actor.username, action: 'database.drop', target: name, ip: clientIp(request) });
    return NextResponse.json({ success: true, message: `Database '${name}' dropped` });
  }

  if (action === 'drop_user') {
    if (!name || !/^[a-zA-Z0-9_]+$/.test(name)) return NextResponse.json({ error: 'Invalid user name' }, { status: 400 });
    const r = await runMysql(`DROP USER IF EXISTS ${mysqlString(name)}@'localhost';`);
    if (!r.success) return NextResponse.json({ error: r.stderr }, { status: 500 });
    audit({ actor: actor.username, action: 'database_user.drop', target: name, ip: clientIp(request) });
    return NextResponse.json({ success: true, message: `User '${name}' dropped` });
  }

  return NextResponse.json({ error: 'Unknown action' }, { status: 400 });
}
