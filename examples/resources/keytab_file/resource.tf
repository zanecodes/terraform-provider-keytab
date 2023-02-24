resource "keytab_file" "example" {
  entry {
    principal       = "example"
    realm           = "example.com"
    key             = "example key"
    key_version     = 0
    encryption_type = "rc4-hmac"
    timestamp       = "1970-01-01T00:00:00Z"
  }
}

output "keytab" {
  sensitive = true
  value     = keytab_file.example.content_base64
}
