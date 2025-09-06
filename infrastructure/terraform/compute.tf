// Compute instance
resource "oci_core_instance" "vm" {
  compartment_id      = var.compartment_ocid
  availability_domain = data.oci_identity_availability_domains.ads.availability_domains[0].name
  display_name        = var.instance_display_name
  shape               = var.shape

  source_details {
    source_type = "image"
    source_id   = local.image_id
    # Set custom boot volume size (in GB) from variable when launching from image
    boot_volume_size_in_gbs = var.storage_size_gb
  }

  create_vnic_details {
    subnet_id        = oci_core_subnet.subnet.id
    assign_public_ip = true
    display_name     = "chimera-vnic"
  }

  metadata = {
    ssh_authorized_keys = var.ssh_public_key
  }

  dynamic "shape_config" {
    for_each = var.use_flex_shape ? [1] : []
    content {
      ocpus         = var.ocpus
      memory_in_gbs = var.memory_in_gbs
    }
  }
}

// Get the primary VNIC public IP
// Get the primary VNIC via listing vnic attachments for the instance
data "oci_core_vnic_attachments" "instance_attachments" {
  compartment_id = var.compartment_ocid
  instance_id    = oci_core_instance.vm.id
}
