output "server_ip" {
  description = "Public IPv4 address."
  value       = vultr_instance.hostq.main_ip
}

output "panel_setup_url" {
  description = "Temporary direct setup URL."
  value       = "http://${vultr_instance.hostq.main_ip}:8090"
}

output "panel_domain" {
  description = "Panel domain when DNS is managed by this module."
  value       = var.domain == "" ? "" : "${var.panel_subdomain}.${var.domain}"
}

output "ansible_inventory_line" {
  description = "Paste into infra/ansible/inventory.ini."
  value       = "hostq ansible_host=${vultr_instance.hostq.main_ip} ansible_user=root"
}
