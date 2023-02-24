resource "keytab_file" "example" {
  entry {
    principal   = "example"
    realm       = "example.com"
    key         = "example key"
    key_version = 0
  }
}

output "keytab" {
  sensitive = true
  value     = keytab_file.example.content_base64
}
