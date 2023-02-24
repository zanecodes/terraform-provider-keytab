resource "keytab_file" "example" {
  entry {
  }
}

output "keytab" {
  sensitive = true
  value     = keytab_file.example.content_base64
}
