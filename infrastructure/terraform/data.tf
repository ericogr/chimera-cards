// Data sources and local resolution
data "oci_identity_tenancy" "tenancy" {
  tenancy_id = var.tenancy_ocid
}

data "oci_identity_availability_domains" "ads" {
  compartment_id = var.compartment_ocid
}

// Attempt to find the image by display name when image_ocid is empty.
data "oci_core_images" "selected_image" {
  count          = var.image_ocid == "" ? 1 : 0
  compartment_id = data.oci_identity_tenancy.tenancy.id
  filter {
    name   = "display_name"
    values = [var.image_display_name]
  }
  sort_by    = "TIMECREATED"
  sort_order = "DESC"
}

locals {
  image_id = var.image_ocid != "" ? var.image_ocid : (length(data.oci_core_images.selected_image) > 0 && length(data.oci_core_images.selected_image[0].images) > 0 ? data.oci_core_images.selected_image[0].images[0].id : "")
}
