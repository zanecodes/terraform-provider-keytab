resource "keytab_file" "example" {
  entry {
    principal = "example"
    realm     = "example.com"
  }
}

output "keytab" {
  sensitive = true
  value     = keytab_file.example.content_base64
}
