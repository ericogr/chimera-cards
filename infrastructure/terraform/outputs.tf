output "instance_public_ip" {
  description = "Public IP of the created instance (primary VNIC)"
  value       = oci_core_instance.vm.public_ip
}

output "load_balancer_ips" {
  description = "Public IP addresses assigned to the load balancer"
  # ip_addresses can change shape between provider versions; try to normalize
  value = try(
    [for ipobj in oci_load_balancer_load_balancer.lb.ip_address_details : try(ipobj.ip_address, ipobj)]
  )
}
