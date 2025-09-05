// Load balancer (TCP) public
resource "oci_load_balancer_load_balancer" "lb" {
  compartment_id = var.compartment_ocid
  display_name   = var.lb_display_name
  subnet_ids     = [oci_core_subnet.subnet.id]
  is_private     = false
  shape          = "flexible"
  shape_details {
    minimum_bandwidth_in_mbps = 10
    maximum_bandwidth_in_mbps = 10
  }

  reserved_ips {
    id = try(oci_core_public_ip.lb_reserved_ip.id, null)
  }
}

resource "oci_load_balancer_backend_set" "backend_set" {
  load_balancer_id = oci_load_balancer_load_balancer.lb.id
  name             = "app-backend-set"
  policy           = "ROUND_ROBIN"

  health_checker {
    protocol = "TCP"
    port     = var.backend_port
  }
}

resource "oci_load_balancer_listener" "listener" {
  load_balancer_id          = oci_load_balancer_load_balancer.lb.id
  name                      = "tcp-listener"
  default_backend_set_name  = oci_load_balancer_backend_set.backend_set.name
  port                      = var.lb_listen_port
  protocol                  = "TCP"
}

resource "oci_load_balancer_backend" "backend" {
  backendset_name = oci_load_balancer_backend_set.backend_set.name
  load_balancer_id = oci_load_balancer_load_balancer.lb.id
  # ip_address       = data.oci_core_vnic.primary_vnic.public_ip_address
  ip_address       = oci_core_instance.vm.private_ip
  port             = var.backend_port
  weight           = 1
}
