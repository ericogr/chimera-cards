// Lookup zone by name when the zone ID is not provided. The data source
// is only evaluated when `cloudflare_enabled` is true and a zone name is set.
data "cloudflare_zone" "selected" {
  filter =  {
    name = var.cloudflare_zone_name
  }
}

// Create or update a single DNS record pointing to the load balancer IP.
resource "cloudflare_dns_record" "chimera" {
  name    = var.cloudflare_dns_record_name
  ttl     = var.cloudflare_dns_ttl
  type    = var.cloudflare_dns_type
  zone_id = data.cloudflare_zone.selected.zone_id
  proxied = var.cloudflare_dns_proxied
  content = oci_load_balancer_load_balancer.lb.ip_address_details[0].ip_address
  comment = "Managed by Terraform to chimera game"
}
