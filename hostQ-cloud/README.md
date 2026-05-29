# hostQ-cloud

Premium multi-tenant hosting control panel. Ground-up redesign of [hostQ](https://github.com/DreamyMonk/hostQ) for SaaS-grade deployments.

## What's different from hostQ

| | hostQ | hostQ-cloud |
|---|-------|-------------|
| Frontend | Inline HTML/CSS/JS in Go binary | Next.js 15 + Tailwind + shadcn/ui |
| Backend | Go single-binary, no DB | Go REST API + Postgres + Redis |
| Tenancy | Single admin | Full multi-tenant (users own sites) |
| Web stack | Nginx + PHP-FPM | **Nginx (front) + Apache (back) + mod_php** |
| Deploy | One binary + nginx | API + frontend + db + queue |
| Audience | Single-VPS operators | Hosting providers, agencies, SaaS |

## Architecture

```
                      ┌─────────────────────────┐
                      │ Next.js (panel UI)      │
                      │ • /admin/* (operator)   │
                      │ • /user/*  (tenant)     │
                      └────────────┬────────────┘
                                   │ JSON / REST
                      ┌────────────▼────────────┐
                      │ Go API server           │
                      │ • Auth (JWT sessions)   │
                      │ • Sites / DB / SSL CRUD │
                      │ • Vhost emission        │
                      │ • Job queue dispatcher  │
                      └────────────┬────────────┘
                                   │
        ┌─────────────────┬────────┴─────────┬──────────────────┐
        ▼                 ▼                  ▼                  ▼
   ┌─────────┐      ┌──────────┐       ┌─────────┐       ┌──────────┐
   │Postgres │      │  Redis   │       │ Apache  │       │  Nginx   │
   │ (state) │      │ (queue + │       │ (mod_php│       │  (front, │
   │         │      │  cache)  │       │  on 8080)│       │  TLS,    │
   └─────────┘      └──────────┘       └─────────┘       │  static) │
                                                          └──────────┘
```

**Nginx terminates TLS, serves static files, proxies dynamic `.php` to Apache on `127.0.0.1:8080`.** Best of both: nginx perf for static + mod_php for compat with `.htaccess`, WordPress, classic LAMP apps.

## Repository layout

```
hostQ-cloud/
├── backend/                Go API server
│   ├── cmd/api/            Entry point
│   ├── internal/
│   │   ├── auth/           JWT sessions, password hashing, TOTP
│   │   ├── db/             sqlc-generated queries + migrations
│   │   ├── api/            HTTP handlers
│   │   ├── sites/          Vhost emission (nginx + apache)
│   │   ├── users/          User & tenant CRUD
│   │   ├── audit/          Audit log writes
│   │   ├── middleware/     Auth, RBAC, rate limit, request ID
│   │   └── config/         Env parsing
│   └── go.mod
├── frontend/               Next.js 15 panel UI
│   ├── app/
│   │   ├── (auth)/login/   Login flow
│   │   ├── (admin)/admin/  Server operator dashboard
│   │   ├── (user)/user/    Tenant dashboard
│   │   └── api/            BFF endpoints (token refresh, etc.)
│   ├── components/         shadcn/ui + custom
│   ├── lib/                API client, auth helpers
│   └── package.json
├── db/
│   └── schema.sql          Postgres schema (run once via psql -f)
├── infra/
│   ├── nginx/              Vhost templates (front)
│   ├── apache/             Vhost templates (back)
│   └── systemd/            hostq-cloud-api.service
├── docs/
│   ├── ARCHITECTURE.md     Deep architecture doc
│   └── DEPLOY.md           Production deployment guide
└── install.sh              One-shot Ubuntu/Debian bootstrap
```

## Quick start (development)

```bash
# 1. Backend
cd backend
cp .env.example .env
go run ./cmd/api

# 2. Frontend (in another shell)
cd frontend
cp .env.example .env.local
pnpm install
pnpm dev
```

Open http://localhost:3000 — login with seeded admin credentials printed in the API startup log.

## Production install (single VPS)

```bash
curl -fsSL https://raw.githubusercontent.com/DreamyMonk/hostQ-cloud/main/install.sh | sudo bash
```

Installs nginx, apache2 (mod_php, port 8080), postgres, redis, the Go API, the frontend (built + served via systemd + Next.js standalone), and seeds an admin account.

## License

TBD — likely AGPL or commercial dual.
