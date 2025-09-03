// Security list for the VCN/subnet
resource "oci_core_security_list" "sec_list" {
  compartment_id = var.compartment_ocid
  vcn_id         = oci_core_virtual_network.vcn.id
  display_name   = "quimera-sec-list"

  ingress_security_rules = [
    {
      protocol = "6"
      source   = "0.0.0.0/0"
      tcp_options = {
        destination_port_range = {
          min = 22
          max = 22
        }
      }
    },
    {
      protocol = "6"
      source   = "0.0.0.0/0"
      tcp_options = {
        destination_port_range = {
          min = var.lb_listen_port
          max = var.lb_listen_port
        }
      }
    },
    {
      protocol = "6"
      source   = "0.0.0.0/0"
      tcp_options = {
        destination_port_range = {
          min = var.backend_port
          max = var.backend_port
        }
      }
    }
  ]

  egress_security_rules = [
    {
      protocol    = "all"
      destination = "0.0.0.0/0"
    }
  ]
}

