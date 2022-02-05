output "gcp-bucket-url" {
  value = google_storage_bucket.goreleaser-gcp-storage-bucket.url
}

output "azure-storage-account-name" {
  value = format("export AZURE_STORAGE_ACCOUNT=%s", azurerm_storage_account.goreleaser-azure-storage-account.name)
}

output "azure-storage-account-key" {
  value     = format("export AZURE_STORAGE_KEY=%s", azurerm_storage_account.goreleaser-azure-storage-account.primary_access_key)
  sensitive = true
}

output "aws-s3-bucket-name" {
  value = aws_s3_bucket.goreleaser-s3-bucket.id
}