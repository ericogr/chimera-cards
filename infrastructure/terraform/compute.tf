// Compute instance
resource "oci_core_instance" "vm" {
  compartment_id      = var.compartment_ocid
  availability_domain = data.oci_identity_availability_domains.ads.availability_domains[0].name
  display_name        = var.instance_display_name
  shape               = var.shape

  source_details {
    source_type = "image"
    image_id    = local.image_id
  }

  create_vnic_details {
    subnet_id        = oci_core_subnet.subnet.id
    assign_public_ip = true
    display_name     = "quimera-vnic"
  }

  metadata = {
    ssh_authorized_keys = var.ssh_public_key
  }

  # Set custom boot volume size (in GB) from variable
  boot_volume_size_in_gbs = var.storage_size_gb

  dynamic "shape_config" {
    for_each = var.use_flex_shape ? [1] : []
    content {
      ocpus         = var.ocpus
      memory_in_gbs = var.memory_in_gbs
    }
  }
}

// Get the primary VNIC public IP
data "oci_core_vnic" "primary_vnic" {
  vnic_id = oci_core_instance.vm.primary_vnic_id
}

