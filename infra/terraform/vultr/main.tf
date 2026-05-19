locals {
  labels = {
    project = var.project_name
    app     = "hostq"
  }
}

resource "vultr_ssh_key" "hostq" {
  name    = "${var.project_name}-ssh"
  ssh_key = file(pathexpand(var.ssh_public_key_path))
}

resource "vultr_firewall_group" "hostq" {
  description = "${var.project_name} firewall"
}

resource "vultr_firewall_rule" "ssh" {
  firewall_group_id = vultr_firewall_group.hostq.id
  protocol          = "tcp"
  ip_type           = "v4"
  subnet            = split("/", var.allowed_ssh_cidr)[0]
  subnet_size       = tonumber(split("/", var.allowed_ssh_cidr)[1])
  port              = "22"
  notes             = "SSH"
}

resource "vultr_firewall_rule" "http" {
  firewall_group_id = vultr_firewall_group.hostq.id
  protocol          = "tcp"
  ip_type           = "v4"
  subnet            = "0.0.0.0"
  subnet_size       = 0
  port              = "80"
  notes             = "HTTP"
}

resource "vultr_firewall_rule" "https" {
  firewall_group_id = vultr_firewall_group.hostq.id
  protocol          = "tcp"
  ip_type           = "v4"
  subnet            = "0.0.0.0"
  subnet_size       = 0
  port              = "443"
  notes             = "HTTPS"
}

resource "vultr_firewall_rule" "panel_setup" {
  firewall_group_id = vultr_firewall_group.hostq.id
  protocol          = "tcp"
  ip_type           = "v4"
  subnet            = split("/", var.allowed_panel_cidr)[0]
  subnet_size       = tonumber(split("/", var.allowed_panel_cidr)[1])
  port              = "8090"
  notes             = "hostQ setup panel"
}

resource "vultr_instance" "hostq" {
  label              = var.project_name
  region             = var.region
  plan               = var.plan
  os_id              = var.os_id
  enable_ipv6        = var.enable_ipv6
  backups            = var.backups
  ssh_key_ids        = [vultr_ssh_key.hostq.id]
  firewall_group_id  = vultr_firewall_group.hostq.id
  activation_email   = false
  tags               = values(local.labels)
}

resource "vultr_dns_record" "root" {
  count  = var.domain == "" ? 0 : 1
  domain = var.domain
  name   = "@"
  type   = "A"
  data   = vultr_instance.hostq.main_ip
  ttl    = 300
}

resource "vultr_dns_record" "panel" {
  count  = var.domain == "" ? 0 : 1
  domain = var.domain
  name   = var.panel_subdomain
  type   = "A"
  data   = vultr_instance.hostq.main_ip
  ttl    = 300
}
