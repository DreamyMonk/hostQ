import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';

const proxy = readFileSync('src/proxy.ts', 'utf8');
assert.match(proxy, /Invalid CSRF token/, 'proxy must reject missing CSRF tokens');
assert.match(proxy, /Strict-Transport-Security/, 'proxy must set HSTS in production');
assert.match(proxy, /x-csrf-token/, 'proxy must validate CSRF header');
assert.match(proxy, /HOSTQ_ALLOW_INSECURE_HTTP/, 'proxy must support explicit temporary HTTP setup mode');

const files = readFileSync('src/app/api/files/route.ts', 'utf8');
assert.match(files, /process\.platform === 'linux' \? '\/var\/www'/, 'file manager must lock Linux root to /var/www');
assert.match(files, /BLOCKED_EXTS/, 'file manager must block secret extensions');
assert.match(files, /file\.soft_delete/, 'file manager deletes must be soft deletes');
assert.match(files, /path\.isAbsolute\(normalized\)/, 'file manager must accept absolute paths under /var/www without prefixing root twice');
assert.match(files, /action === 'copy'/, 'file manager must support copying files and folders');
assert.match(files, /action === 'chmod'/, 'file manager must support permission changes');
assert.doesNotMatch(files, /fs\.rm\(/, 'file manager must not hard-delete with fs.rm');

const auth = readFileSync('src/lib/auth.ts', 'utf8');
assert.match(auth, /validateOtp/, 'auth must require TOTP validation');
assert.match(auth, /revokeSession/, 'auth must support session revocation');
assert.match(auth, /SESSION_IDLE_TIMEOUT/, 'auth must enforce idle timeout');
assert.match(auth, /otpEnabled: false/, 'initial admin creation must not force 2FA before setup');

const authRoute = readFileSync('src/app/api/auth/route.ts', 'utf8');
assert.match(authRoute, /shouldUseSecureCookies/, 'auth cookies must adapt to HTTPS or explicit setup mode');

const setup = readFileSync('setup.sh', 'utf8');
assert.match(setup, /Initial hostQ admin login/, 'setup must print generated admin credentials over SSH');
assert.match(setup, /admin\.json/, 'setup must create the admin account file');
assert.match(setup, /hostq-update/, 'setup must install the SSH update command');
assert.match(setup, /PANEL_PUBLIC_PORT="\$\{PANEL_PUBLIC_PORT:-8090\}"/, 'setup must default the direct panel port to 8090');
assert.match(setup, /HOSTQ_ALLOW_INSECURE_HTTP=true/, 'setup must allow direct-IP HTTP login until HTTPS is configured');
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

const panel = readFileSync('src/app/api/panel/route.ts', 'utf8');
assert.match(panel, /canManagePanel/, 'panel host settings must be admin-only');
assert.match(panel, /validPanelDomain/, 'panel host settings must validate domains');
assert.match(panel, /HOSTQ_ALLOW_INSECURE_HTTP/, 'panel host settings must manage temporary HTTP setup mode');

const wordpress = readFileSync('src/app/api/wordpress/route.ts', 'utf8');
assert.match(wordpress, /command -v wp/, 'WordPress API must check WP-CLI before showing demo mode');
assert.match(wordpress, /installations: \[\], demo: false/, 'WordPress API must return an empty real list instead of demo data when no installs exist');
assert.match(wordpress, /validEmail/, 'WordPress install must validate admin email before running WP-CLI');
assert.match(wordpress, /status: failed \? 'failure' : 'success'/, 'WordPress install must audit failed installs as failures');
assert.match(wordpress, /WordPress installation failed/, 'WordPress install must return failure when a step fails');
assert.match(wordpress, /\.hostq-trash/, 'WordPress scanner must ignore soft-deleted sites in trash');
assert.match(wordpress, /wp core download --path=\$\{qSitePath\} --force/, 'WordPress install must tolerate retrying over partial files');
assert.match(wordpress, /wp config create --path=\$\{qSitePath\}.*--force/, 'WordPress config creation must tolerate retrying partial installs');
assert.match(wordpress, /ALTER USER/, 'WordPress retry must reset existing database user passwords');
assert.match(wordpress, /siteDocRoot/, 'WordPress installs must use the site document root layout');
assert.match(wordpress, /Configure Nginx vhost/, 'WordPress installs must configure the webserver vhost');

const domains = readFileSync('src/app/api/domains/route.ts', 'utf8');
assert.match(domains, /\/htdocs/, 'new sites must use htdocs document roots');

const terraform = readFileSync('infra/terraform/vultr/main.tf', 'utf8');
assert.match(terraform, /vultr_firewall_rule" "panel_setup"/, 'Terraform must open the hostQ setup panel port');
assert.match(terraform, /vultr_instance" "hostq"/, 'Terraform must provision a hostQ VPS');

const ansible = readFileSync('infra/ansible/roles/hostq/tasks/main.yml', 'utf8');
assert.match(ansible, /bash setup\.sh/, 'Ansible must run hostQ setup');
assert.match(ansible, /hostq-update/, 'Ansible must support hostQ tagged updates');
assert.match(ansible, /\/swapfile/, 'Ansible must create swap for 1GB VPS deployments');

const sshUpdate = readFileSync('scripts/hostq-update.sh', 'utf8');
assert.match(sshUpdate, /api\.github\.com\/repos\/\$\{REPO\}\/releases\/latest/, 'SSH updater must be able to resolve latest release');
assert.match(sshUpdate, /panel\.update/, 'SSH updater must call the narrow helper update task');

console.log('Security regression checks passed');
