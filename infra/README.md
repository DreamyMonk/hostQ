# hostQ Infrastructure

This folder contains optional provider-style automation for repeatable deployments:

- Terraform provisions the VPS, firewall, SSH key, and optional DNS records.
- Ansible hardens the OS and installs hostQ. It does **not** auto-update: the source is checked out once, and re-running the playbook will not advance a running host to a newer commit.
- After first boot, hostQ manages sites, databases, SSL, files, PHP, and WordPress. Panel updates are explicit only — via a published GitHub release + `sudo hostq-update <tag>`, or `ansible-playbook … --tags update -e hostq_version=vX.Y.Z`.

The primary hostQ deployment remains the cPanel/Plesk-style installer: clone the repo on a VPS and run `install.sh`. Docker is intentionally not used for production because hostQ controls host-level services such as Nginx, MariaDB, PHP-FPM, Certbot, Pure-FTPd, systemd, and `/var/www`.

## Requirements

Install locally:

```bash
terraform -version
ansible --version
ssh -V
```

Set the Vultr API token without committing it:

```bash
export VULTR_API_KEY="..."
```

## 1. Provision VPS

```bash
cd infra/terraform/vultr
cp terraform.tfvars.example terraform.tfvars
```

Edit `terraform.tfvars`, then:

```bash
terraform init
terraform plan
terraform apply
```

Terraform prints the server IP and an Ansible inventory command.

## 2. Install hostQ With Ansible

From the repository root:

```bash
cd infra/ansible
cp inventory.ini.example inventory.ini
```

Put the Terraform server IP in `inventory.ini`, then run:

```bash
ansible-playbook -i inventory.ini playbook.yml
```

The hostQ installer prints the first admin username and generated password in SSH output.

## 3. Update hostQ Later

```bash
ansible-playbook -i inventory.ini playbook.yml --tags update -e hostq_version=v0.3.5
```

Or SSH directly:

```bash
sudo hostq-update v0.3.5
```

## Production Notes

- Point a real domain or subdomain to the VPS.
- Install SSL for the panel domain from the SSL manager.
- After HTTPS works, restrict or close the direct setup port.
- Keep ports `22`, `80`, `443`, and `8090` restricted to trusted IPs where possible.
