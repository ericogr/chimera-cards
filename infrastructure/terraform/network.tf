// VCN, Internet Gateway, Route Table, Subnet
resource "oci_core_virtual_network" "vcn" {
  compartment_id = var.compartment_ocid
  display_name   = "chimera-vcn"
  cidr_block     = var.vcn_cidr
  dns_label      = var.dns_label
}

resource "oci_core_internet_gateway" "igw" {
  compartment_id = var.compartment_ocid
  display_name   = "chimera-igw"
  vcn_id         = oci_core_virtual_network.vcn.id
}

resource "oci_core_route_table" "rt" {
  compartment_id = var.compartment_ocid
  display_name   = "chimera-rt"
  vcn_id         = oci_core_virtual_network.vcn.id

  route_rules {
    destination       = "0.0.0.0/0"
    network_entity_id = oci_core_internet_gateway.igw.id
  }
}

resource "oci_core_subnet" "subnet" {
  compartment_id              = var.compartment_ocid
  vcn_id                      = oci_core_virtual_network.vcn.id
  display_name                = "chimera-subnet"
  cidr_block                  = var.subnet_cidr
  dns_label                   = "${var.dns_label}sub"
  route_table_id              = oci_core_route_table.rt.id
  security_list_ids           = [oci_core_security_list.sec_list.id]
  availability_domain         = data.oci_identity_availability_domains.ads.availability_domains[0].name
  prohibit_public_ip_on_vnic  = false
}
