resource "keytab_file" "example" {
}

output "keytab" {
  sensitive = true
  value     = keytab_file.example.content_base64
}
