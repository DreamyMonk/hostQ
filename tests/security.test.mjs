import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';

const proxy = readFileSync('src/proxy.ts', 'utf8');
assert.match(proxy, /Invalid CSRF token/, 'proxy must reject missing CSRF tokens');
assert.match(proxy, /Strict-Transport-Security/, 'proxy must set HSTS in production');
assert.match(proxy, /x-csrf-token/, 'proxy must validate CSRF header');

const files = readFileSync('src/app/api/files/route.ts', 'utf8');
assert.match(files, /process\.platform === 'linux' \? '\/var\/www'/, 'file manager must lock Linux root to /var/www');
assert.match(files, /BLOCKED_EXTS/, 'file manager must block secret extensions');
assert.match(files, /file\.soft_delete/, 'file manager deletes must be soft deletes');
assert.doesNotMatch(files, /fs\.rm\(/, 'file manager must not hard-delete with fs.rm');

const auth = readFileSync('src/lib/auth.ts', 'utf8');
assert.match(auth, /validateOtp/, 'auth must require TOTP validation');
assert.match(auth, /revokeSession/, 'auth must support session revocation');
assert.match(auth, /SESSION_IDLE_TIMEOUT/, 'auth must enforce idle timeout');
assert.match(auth, /otpEnabled: false/, 'initial admin creation must not force 2FA before setup');

const setup = readFileSync('setup.sh', 'utf8');
assert.match(setup, /Initial hostQ admin login/, 'setup must print generated admin credentials over SSH');
assert.match(setup, /admin\.json/, 'setup must create the admin account file');
assert.match(setup, /hostq-update/, 'setup must install the SSH update command');
assert.match(setup, /PANEL_PUBLIC_PORT="\$\{PANEL_PUBLIC_PORT:-8090\}"/, 'setup must default the direct panel port to 8090');
assert.match(setup, /listen __PANEL_PUBLIC_PORT__;/, 'nginx config must include the direct panel setup port');
assert.match(setup, /ufw allow \$\{PANEL_PUBLIC_PORT\}\/tcp/, 'firewall must open the direct panel setup port');
assert.match(setup, /<<'EOF'/, 'nginx heredoc must not expand proxy header variables');

const authz = readFileSync('src/lib/authz.ts', 'utf8');
assert.match(authz, /canManageSite/, 'RBAC helper must enforce site permissions');
assert.match(authz, /canManagePanel/, 'RBAC helper must enforce admin-only actions');

const helper = readFileSync('scripts/hostq-helper.mjs', 'utf8');
assert.doesNotMatch(helper, /task === 'shell'/, 'helper must not expose a generic shell task');
assert.match(helper, /Task is not allowed/, 'helper must deny unknown tasks');
assert.match(helper, /panel\.update/, 'helper must expose a narrow panel update task');

const update = readFileSync('src/app/api/update/route.ts', 'utf8');
assert.match(update, /confirm !== tag/, 'update API must require tag confirmation');
assert.match(update, /runHelper\('panel.update'/, 'update API must use privileged helper');

const sshUpdate = readFileSync('scripts/hostq-update.sh', 'utf8');
assert.match(sshUpdate, /api\.github\.com\/repos\/\$\{REPO\}\/releases\/latest/, 'SSH updater must be able to resolve latest release');
assert.match(sshUpdate, /panel\.update/, 'SSH updater must call the narrow helper update task');

console.log('Security regression checks passed');
