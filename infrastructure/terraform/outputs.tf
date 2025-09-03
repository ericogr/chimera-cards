output "instance_public_ip" {
  description = "Public IP of the created instance (primary VNIC)"
  value       = data.oci_core_vnic.primary_vnic.public_ip
}

output "load_balancer_ips" {
  description = "Public IP addresses assigned to the load balancer"
  value       = oci_load_balancer_load_balancer.lb.ip_addresses
}

output "load_balancer_ip" {
  description = "Primary public IP of the load balancer (first)"
  value       = length(oci_load_balancer_load_balancer.lb.ip_addresses) > 0 ? oci_load_balancer_load_balancer.lb.ip_addresses[0].ip_address : ""
}

