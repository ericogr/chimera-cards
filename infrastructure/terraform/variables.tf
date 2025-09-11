variable "region" {
  description = "OCI region (ex: us-ashburn-1)"
  type        = string
  default     = "us-ashburn-1"
}

variable "compartment_ocid" {
  description = "Compartment OCID where resources will be created"
  type        = string
}

variable "tenancy_ocid" {
  description = "Tenancy OCID (ocid1.tenancy...) used by some data sources"
  type        = string
}

variable "vcn_cidr" {
  description = "VCN CIDR block"
  type        = string
  default     = "10.0.0.0/16"
}

variable "subnet_cidr" {
  description = "Subnet CIDR block"
  type        = string
  default     = "10.0.1.0/24"
}

variable "dns_label" {
  description = "DNS label used for VCN/subnet hostnames"
  type        = string
  default     = "chimera"
}

variable "ssh_public_key" {
  description = "SSH public key (openssh format) to inject in the instance (required)"
  type        = string
  default     = ""
}

variable "backend_https_port" {
  description = "Port on the instance where the application listens (backend HTTPS port)"
  type        = number
  default     = 8443
}

variable "backend_http_port" {
  description = "Port on the instance where the application listens (backend HTTP port)"
  type        = number
  default     = 8080
}

variable "lb_listen_https_port" {
  description = "Port the load balancer listens on (external)"
  type        = number
  default     = 443
}

variable "lb_listen_http_port" {
  description = "Port the load balancer listens on (external)"
  type        = number
  default     = 80
}

variable "image_ocid" {
  description = "(Optional) Image OCID to use for the instance. If empty, Terraform will attempt to find an image by name (may fail depending on tenancy/region)"
  type        = string
  default     = ""
}

variable "image_display_name" {
  description = "Friendly name to search for the image if image_ocid is not provided"
  type        = string
  default     = "Canonical-Ubuntu-24.04-Minimal-2025.07.23-0"
}

variable "shape" {
  description = "Instance shape. Use a flex shape (eg. VM.Standard.E3.Flex) together with use_flex_shape=true to customize ocpus/memory. Default is VM.Standard.E2.1.Micro"
  type        = string
  default     = "VM.Standard.E2.1.Micro"
}

variable "use_flex_shape" {
  description = "Whether to enable shape_config (for flex shapes). Set to true when using a flexible shape."
  type        = bool
  default     = false
}

variable "ocpus" {
  description = "Number of OCPUs (used only when use_flex_shape = true)"
  type        = number
  default     = 1
}

variable "memory_in_gbs" {
  description = "Memory in GB (used only when use_flex_shape = true)"
  type        = number
  default     = 1
}

variable "storage_size_gb" {
  description = "Boot volume size in GB for the instance (default 47). This sets the instance's boot volume size."
  type        = number
  default     = 47
}

variable "instance_display_name" {
  description = "Display name for the compute instance"
  type        = string
  default     = "chimera-instance"
}

variable "lb_display_name" {
  description = "Display name for the load balancer"
  type        = string
  default     = "chimera-lb"
}

variable "user_ocid" {
  description = "User OCID (ocid1.user.oc1..)"
  type        = string
}
variable "fingerprint" {
  description = "User OCID (ocid1.user.oc1..)"
  type        = string
}
variable "private_key_path" {
  description = "User OCID (ocid1.user.oc1..)"
  type        = string
}

variable "cloudflare_api_token" {
  description = "Cloudflare API token. If not set, provider reads from the environment variable CLOUDFLARE_API_TOKEN."
  type        = string
  default     = ""
}

variable "cloudflare_zone_name" {
  description = "DNS zone name (domain) in Cloudflare (e.g., example.com)"
  type        = string
  default     = ""
}

variable "cloudflare_dns_record_name" {
  description = "DNS record name to create/update in Cloudflare (e.g., 'www' or 'app.example.com'). If empty, uses var.dns_label"
  type        = string
  default     = ""
}

variable "cloudflare_dns_ttl" {
  description = "TTL for the Cloudflare DNS record (seconds). Use 1 for automatic TTL."
  type        = number
  default     = 1
}

variable "cloudflare_dns_type" {
  description = "DNS record type to create (e.g., A, CNAME)"
  type        = string
  default     = "A"
}

variable "cloudflare_dns_proxied" {
  description = "Whether the Cloudflare record should be proxied (Cloudflare CDN)."
  type        = bool
  default     = false
}

