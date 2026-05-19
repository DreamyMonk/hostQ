variable "vultr_api_key" {
  description = "Vultr API key. Prefer VULTR_API_KEY environment variable."
  type        = string
  sensitive   = true
  default     = null
}

variable "project_name" {
  description = "Name prefix for created resources."
  type        = string
  default     = "hostq"
}

variable "region" {
  description = "Vultr region."
  type        = string
  default     = "bom"
}

variable "plan" {
  description = "Vultr instance plan. vc2-1c-1gb is the minimum."
  type        = string
  default     = "vc2-1c-1gb"
}

variable "os_id" {
  description = "Vultr OS id. 1743 is Ubuntu 22.04 x64 on Vultr at time of writing."
  type        = number
  default     = 1743
}

variable "ssh_public_key_path" {
  description = "Local SSH public key path."
  type        = string
  default     = "~/.ssh/id_ed25519.pub"
}

variable "allowed_ssh_cidr" {
  description = "CIDR allowed to SSH. Use your public IP /32 for production."
  type        = string
  default     = "0.0.0.0/0"
}

variable "allowed_panel_cidr" {
  description = "CIDR allowed to access direct setup port 8090. Use your public IP /32 for production."
  type        = string
  default     = "0.0.0.0/0"
}

variable "domain" {
  description = "Optional DNS zone already hosted in Vultr DNS, such as example.com."
  type        = string
  default     = ""
}

variable "panel_subdomain" {
  description = "Optional panel host record. With domain=example.com, panel_subdomain=panel creates panel.example.com."
  type        = string
  default     = "panel"
}

variable "enable_ipv6" {
  description = "Enable IPv6 on the VPS."
  type        = bool
  default     = true
}

variable "backups" {
  description = "Enable Vultr automatic backups."
  type        = string
  default     = "enabled"
}
