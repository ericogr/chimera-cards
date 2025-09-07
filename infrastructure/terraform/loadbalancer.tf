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
}

resource "oci_load_balancer_backend_set" "backend_https_set" {
  load_balancer_id = oci_load_balancer_load_balancer.lb.id
  name             = "app-backend-https-set"
  policy           = "ROUND_ROBIN"

  health_checker {
    protocol = "TCP"
    port     = var.backend_https_port
  }
}

resource "oci_load_balancer_backend_set" "backend_http_set" {
  load_balancer_id = oci_load_balancer_load_balancer.lb.id
  name             = "app-backend-http-set"
  policy           = "ROUND_ROBIN"

  health_checker {
    protocol = "TCP"
    port     = var.backend_http_port
  }
}

resource "oci_load_balancer_listener" "listener_https" {
  load_balancer_id          = oci_load_balancer_load_balancer.lb.id
  name                      = "tcp-https-listener"
  default_backend_set_name  = oci_load_balancer_backend_set.backend_https_set.name
  port                      = var.lb_listen_https_port
  protocol                  = "TCP"
}

resource "oci_load_balancer_listener" "listener_http" {
  load_balancer_id          = oci_load_balancer_load_balancer.lb.id
  name                      = "tcp-http-listener"
  default_backend_set_name  = oci_load_balancer_backend_set.backend_http_set.name
  port                      = var.lb_listen_http_port
  protocol                  = "TCP"
}

resource "oci_load_balancer_backend" "backend_https" {
  backendset_name = oci_load_balancer_backend_set.backend_https_set.name
  load_balancer_id = oci_load_balancer_load_balancer.lb.id
  ip_address       = oci_core_instance.vm.private_ip
  port             = var.backend_https_port
  weight           = 1
}

resource "oci_load_balancer_backend" "backend_http" {
  backendset_name = oci_load_balancer_backend_set.backend_http_set.name
  load_balancer_id = oci_load_balancer_load_balancer.lb.id
  ip_address       = oci_core_instance.vm.private_ip
  port             = var.backend_http_port
  weight           = 1
}
