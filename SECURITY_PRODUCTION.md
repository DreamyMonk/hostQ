# hostQ Production Hardening

hostQ should run behind HTTPS only. Use a real DNS name, install a certificate with Certbot, and keep `PANEL_URL` set to the HTTPS URL. Do not expose the Node.js port publicly.

## Supported Base OS

- Ubuntu 22.04 LTS or 24.04 LTS
- Debian 12
- 1 GB RAM / 1 vCPU minimum for small sites

## Network

- Allow only `22/tcp`, `80/tcp`, `443/tcp`, and FTP ports you intentionally use.
- Bind the Next.js app to `127.0.0.1`.
- Put Nginx in front with `X-Forwarded-Proto` preserved.
- Enable HSTS after HTTPS works.

## Accounts And Sessions

- `setup.sh` generates the first admin username/password over SSH and writes `/etc/hostq/admin.json`.
- TOTP is optional on first login. Enable it from Admin -> Security after changing the generated password.
- User accounts are per-site and enforced by API RBAC.
- Admin sessions can be viewed and revoked through `/api/sessions`.
- Idle sessions expire using `SESSION_IDLE_TIMEOUT_MINUTES`.

## Privileged Helper

`scripts/hostq-helper.mjs` is the only place new privileged tasks should be added. Keep it narrow:

- add one task at a time
- validate every argument
- prefer fixed command templates
- never add a generic shell task

After all privileged routes are migrated to helper tasks, set:

```env
HOSTQ_REQUIRE_HELPER=true
HOSTQ_HELPER=/usr/local/sbin/hostq-helper
```

## Files And Secrets

- File manager is locked to `/var/www` on Linux.
- `.env`, SSH keys, PEM/private-key/certificate containers are blocked by default.
- Uploaded SSL private keys are stored under `/etc/ssl/hostq/<domain>/privkey.pem` with `0600`.

## Backups And Restore

- Site backup creates tarballs under `/var/backups/hostq`.
- Restore requires a dry-run first and a confirmation string equal to the domain.
- Restore rejects absolute paths, path traversal, and secret-like files.
- A pre-restore backup is created before extraction.

## Audit Logs

- Audit log: `/etc/hostq/audit.log`
- Entries include a hash chain. The audit API reports whether the chain is valid.
- Logs rotate when the active file reaches 5 MB.
- For high-trust environments, ship logs to remote append-only storage.

## Updates

- Run OS security updates weekly.
- Rotate `JWT_SECRET` after a suspected compromise.
- Keep Node.js LTS and supported PHP versions only.
- Run `npm run lint`, `npm run build`, and `npm run security:test` before deploy.
- Publish hostQ releases as GitHub tags like `v0.2.0`.
- The panel updater downloads release tarballs from `DreamyMonk/hostQ`, creates `/var/backups/hostq/panel-*.tar.gz`, rebuilds, prunes dev dependencies, and restarts `hostq`.
- Keep update actions inside `scripts/hostq-helper.mjs`; do not add a generic shell update endpoint.
- SSH update command:

```bash
sudo hostq-update          # latest GitHub release
sudo hostq-update v0.2.2   # specific release tag
```

- `hostq-update` calls `/usr/local/sbin/hostq-helper` with the narrow `panel.update` task.
