variable "region" {
  description = "OCI region (ex: us-ashburn-1)"
  type        = string
  default     = "us-ashburn-1"
}

variable "compartment_ocid" {
  description = "Compartment OCID where resources will be created"
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
  default     = "quimera"
}

variable "ssh_public_key" {
  description = "SSH public key (openssh format) to inject in the instance (required)"
  type        = string
  default     = ""
}

variable "backend_port" {
  description = "Port on the instance where the application listens (backend port)"
  type        = number
  default     = 8080
}

variable "lb_listen_port" {
  description = "Port the load balancer listens on (external)"
  type        = number
  default     = 443
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
  default     = "quimera-instance"
}

variable "lb_display_name" {
  description = "Display name for the load balancer"
  type        = string
  default     = "quimera-lb"
}

variable "lb_shape" {
  description = "Load balancer shape (eg. 10Mbps, 100Mbps, 4000Mbps). Default 100Mbps"
  type        = string
  default     = "100Mbps"
}
