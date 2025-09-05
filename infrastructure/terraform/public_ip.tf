// Reserve a public IP for the load balancer
// This creates a RESERVED public IP and assigns it to the load balancer
// Assumption: provider is Oracle Cloud (OCI) and the provider version supports
// `oci_core_public_ip` with `assigned_entity_id` to attach to a load balancer.
resource "oci_core_public_ip" "lb_reserved_ip" {
  compartment_id = var.compartment_ocid
  display_name   = "${var.lb_display_name}-reserved-ip"
  lifetime       = "RESERVED"
}

