// Load balancer (TCP) public
resource "oci_load_balancer_load_balancer" "lb" {
  compartment_id = var.compartment_ocid
  display_name   = var.lb_display_name
  subnet_ids     = [oci_core_subnet.subnet.id]
  is_private     = false
  shape_name     = var.lb_shape
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
  load_balancer_id = oci_load_balancer_load_balancer.lb.id
  backend_set_name = oci_load_balancer_backend_set.backend_set.name
  name             = "backend-instance-1"
  ip_address       = data.oci_core_vnic.primary_vnic.public_ip
  port             = var.backend_port
  weight           = 1
}

