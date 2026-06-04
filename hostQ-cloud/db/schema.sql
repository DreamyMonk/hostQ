-- hostQ-cloud Postgres schema.
-- Apply: psql -U hostq -d hostq_cloud -f db/schema.sql
-- Idempotent via IF NOT EXISTS where possible. Migrations are append-only.

CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS citext;

-- ---------------------------------------------------------------------------
-- tenants — billing/group entity. 1 tenant = N users.
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS tenants (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name         TEXT NOT NULL,
    plan         TEXT NOT NULL DEFAULT 'starter',
    quota_sites  INT NOT NULL DEFAULT 5,
    quota_disk_mb BIGINT NOT NULL DEFAULT 5120,
    suspended    BOOLEAN NOT NULL DEFAULT FALSE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ---------------------------------------------------------------------------
-- users — accounts. role gates admin vs tenant. tenant_id NULL for admins.
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID REFERENCES tenants(id) ON DELETE CASCADE,
    email         CITEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    role          TEXT NOT NULL CHECK (role IN ('superadmin','admin','tenant')),
    display_name  TEXT,
    totp_secret   TEXT,
    totp_enabled  BOOLEAN NOT NULL DEFAULT FALSE,
    last_login_at TIMESTAMPTZ,
    last_login_ip INET,
    failed_logins INT NOT NULL DEFAULT 0,
    locked_until  TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_users_tenant ON users(tenant_id);
CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);

-- ---------------------------------------------------------------------------
-- sessions — refresh tokens, rotated on use. Short JWT lives only in memory.
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS sessions (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    refresh_hash  TEXT NOT NULL,
    user_agent    TEXT,
    ip            INET,
    expires_at    TIMESTAMPTZ NOT NULL,
    revoked_at    TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_expires ON sessions(expires_at);

-- ---------------------------------------------------------------------------
-- sites — one row per hosted domain. owner_id = tenant user that owns it.
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS sites (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id           UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tenant_id          UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    domain             CITEXT UNIQUE NOT NULL,
    docroot            TEXT NOT NULL,
    php_version        TEXT NOT NULL DEFAULT '8.3',
    stack              TEXT NOT NULL DEFAULT 'nginx-apache',
    vhost_path_nginx   TEXT,
    vhost_path_apache  TEXT,
    suspended          BOOLEAN NOT NULL DEFAULT FALSE,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_sites_owner ON sites(owner_id);
CREATE INDEX IF NOT EXISTS idx_sites_tenant ON sites(tenant_id);

-- ---------------------------------------------------------------------------
-- domains — primary + aliases attached to a site.
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS domains (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    site_id    UUID NOT NULL REFERENCES sites(id) ON DELETE CASCADE,
    fqdn       CITEXT UNIQUE NOT NULL,
    is_primary BOOLEAN NOT NULL DEFAULT FALSE,
    ssl_cert_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_domains_site ON domains(site_id);

-- ---------------------------------------------------------------------------
-- ssl_certs — Let's Encrypt certificates managed by certbot.
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS ssl_certs (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    site_id       UUID REFERENCES sites(id) ON DELETE CASCADE,
    primary_fqdn  CITEXT NOT NULL,
    san_fqdns     TEXT[] NOT NULL DEFAULT '{}',
    issuer        TEXT NOT NULL DEFAULT 'letsencrypt',
    fullchain_path TEXT,
    privkey_path  TEXT,
    issued_at     TIMESTAMPTZ,
    expires_at    TIMESTAMPTZ,
    last_renew_at TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_ssl_site ON ssl_certs(site_id);

-- ---------------------------------------------------------------------------
-- databases — MySQL/MariaDB databases linked to a site (or standalone).
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS databases (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    site_id    UUID REFERENCES sites(id) ON DELETE SET NULL,
    name       TEXT NOT NULL,
    username   TEXT NOT NULL,
    host       TEXT NOT NULL DEFAULT 'localhost',
    engine     TEXT NOT NULL DEFAULT 'mariadb',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (owner_id, name)
);
CREATE INDEX IF NOT EXISTS idx_db_owner ON databases(owner_id);

-- ---------------------------------------------------------------------------
-- jobs — async work queue mirror. Redis holds the live queue; this is the
-- durable history + UI polling source.
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS jobs (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id      UUID REFERENCES users(id) ON DELETE SET NULL,
    site_id       UUID REFERENCES sites(id) ON DELETE SET NULL,
    kind          TEXT NOT NULL,
    payload       JSONB NOT NULL DEFAULT '{}',
    status        TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','running','success','failure','cancelled')),
    progress      INT NOT NULL DEFAULT 0,
    error_msg     TEXT,
    started_at    TIMESTAMPTZ,
    finished_at   TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_jobs_owner ON jobs(owner_id);
CREATE INDEX IF NOT EXISTS idx_jobs_status ON jobs(status);

-- ---------------------------------------------------------------------------
-- audit_log — every state-changing action gets a row.
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS audit_log (
    id         BIGSERIAL PRIMARY KEY,
    actor_id   UUID REFERENCES users(id) ON DELETE SET NULL,
    actor_role TEXT,
    actor_ip   INET,
    action     TEXT NOT NULL,
    target     TEXT,
    status     TEXT NOT NULL CHECK (status IN ('success','failure')),
    metadata   JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_audit_created ON audit_log(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_actor ON audit_log(actor_id);
CREATE INDEX IF NOT EXISTS idx_audit_action ON audit_log(action);
