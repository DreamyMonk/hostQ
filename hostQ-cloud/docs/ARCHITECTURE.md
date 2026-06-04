# hostQ-cloud Architecture

Deep design doc — read this before extending the codebase.

## Goals

1. **Multi-tenant by default** — many user accounts share one server. Each user sees only their own sites/databases/files. Admin sees everyone.
2. **Premium SaaS UX** — Next.js + shadcn/ui front-end, polished, fast, no full-page reloads.
3. **Classic LAMP compatibility** — nginx terminates, Apache runs PHP via mod_php so `.htaccess` "just works" for legacy apps.
4. **Operationally simple** — single VPS install, optional horizontal scaling for the API + frontend later.

## Roles & RBAC

Three roles, stored on the `users.role` column:

| Role | Can do |
|------|--------|
| `superadmin` | Everything. Manage other admins. Server-wide ops. |
| `admin` | Server-wide ops. Cannot create other admins. |
| `tenant` | Their own sites/databases/files/SSL/cron only. |

All API handlers go through `middleware.RequireRole(...)`. Tenant-scoped handlers also enforce `resource.owner_id == request.user.id` (or the user is an admin).

## Auth

- **Password**: bcrypt cost 12.
- **Sessions**: short-lived JWT (15 min) + long-lived refresh token (httpOnly cookie, 30d). Rotated on use.
- **2FA**: TOTP optional per user, required by org policy for admin role.
- **Sign-in audit**: every login (success + failure) goes into `audit_log` with IP, user agent.

## Database (Postgres)

Schema lives in `db/schema.sql`. Highlights:

- `users` — accounts (role, password, TOTP secret, email)
- `tenants` — billing/group entity (1 tenant = N users, owner is one user)
- `sites` — one row per hosted domain. `owner_id` → users. `vhost_path_nginx`, `vhost_path_apache` cached for cleanup.
- `databases` — MySQL/MariaDB databases owned by sites (or standalone)
- `domains` — primary + aliases per site, with SSL state
- `ssl_certs` — issued certs (LE), path on disk, renewal status
- `jobs` — async work (issue cert, scan malware, install WP, run backup) — picked up by the worker
- `audit_log` — every state-changing action
- `sessions` — refresh tokens (rotation tracking + revocation)

All write paths use transactions. Queries are generated via [sqlc](https://sqlc.dev/) so no string-builder SQL.

## Vhost emission (the LAMP-with-nginx bit)

For each site we write **two** vhost files atomically:

1. `/etc/nginx/sites-available/{site}.conf` — nginx terminates HTTPS on 443, serves static (`location ~ \.(css|js|png|jpg|...)$`), reverse-proxies dynamic to `127.0.0.1:8080`.
2. `/etc/apache2/sites-available/{site}.conf` — Apache listens on `127.0.0.1:8080` only (no public exposure), serves PHP via mod_php, honours `.htaccess`.

A `Forwarded: ` / `X-Forwarded-Proto: https` header tells Apache the original scheme so PHP apps that check `$_SERVER['HTTPS']` work.

Both files are emitted by `internal/sites/vhost.go`. After write we run `nginx -t` and `apache2ctl configtest`. If either fails we roll back. Reload is two systemctl calls.

## Job queue

Long-running ops (cert issuance, malware scans, WP install, backups) are queued via Redis (`asynq` library). The API enqueues, a separate `cmd/worker` process consumes. Status pushed back into `jobs` table, polled by the UI for progress.

This is what lets the UI feel instant — clicking "Install WordPress" returns immediately with a job-id, the frontend polls and shows progress.

## Frontend

- **Next.js 15 App Router** — file-system routing, layouts per role.
- **Tailwind v4** — utility CSS, design tokens from `tailwind.config.ts`.
- **shadcn/ui** — copy-in components (not a dep), full Radix primitives underneath.
- **React Query (TanStack)** — every API call. Auto cache + refetch + loading states.
- **next-themes** — dark/light.
- **next-safe-action** — typed server actions for mutations.

### Route layout

```
app/
  layout.tsx              Root html
  (auth)/
    layout.tsx            No chrome
    login/page.tsx
    setup-2fa/page.tsx
  (admin)/
    layout.tsx            Admin shell (sidebar, topbar)
    admin/
      page.tsx            Dashboard
      users/page.tsx
      sites/page.tsx      (all sites across tenants)
      services/page.tsx
      firewall/page.tsx
      audit/page.tsx
      account/page.tsx
  (user)/
    layout.tsx            Tenant shell (sidebar, topbar)
    user/
      page.tsx            "My sites" dashboard
      sites/[domain]/
        page.tsx          Site overview
        files/page.tsx
        databases/page.tsx
        ssl/page.tsx
        cron/page.tsx
        backups/page.tsx
        analytics/page.tsx
```

### Design system

Token names (Tailwind theme):

- `bg-surface` / `bg-canvas` / `bg-elevated`
- `text-default` / `text-muted` / `text-faint`
- `border-default` / `border-muted`
- `accent` (brand blue), `success`, `warning`, `danger`

Same names work in both light and dark theme via CSS variables. Components never use raw `slate-500` etc — they reference tokens. This is the key to a consistent "premium" feel across pages without case-by-case styling.

## Install

`install.sh` (Ubuntu 22.04+ / Debian 12+):

1. `apt install nginx apache2 libapache2-mod-php8.3 postgresql redis-server certbot`
2. Configure Apache to listen on `127.0.0.1:8080` only
3. Create `hostq` postgres role + db, apply schema
4. Download API binary + frontend standalone bundle
5. Write systemd units, enable + start
6. Print initial admin credentials

## What's NOT in scope (v1)

- DNS provider integrations (Cloudflare API, etc.)
- Email server management
- Container support (we run PHP via mod_php, not Docker)
- Billing integration (Stripe etc.) — tenants are admin-provisioned for now

These are all reasonable v2 additions.
