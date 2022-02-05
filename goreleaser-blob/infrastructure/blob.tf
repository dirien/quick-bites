resource "google_storage_bucket" "goreleaser-gcp-storage-bucket" {
  name                        = var.name
  location                    = var.gcp_bucket_location
  force_destroy               = true
  uniform_bucket_level_access = false
}
resource "google_storage_bucket_access_control" "goreleaser-gcp-storage-bucket-access-control" {
  bucket = google_storage_bucket.goreleaser-gcp-storage-bucket.name
  role   = "READER"
  entity = "allUsers"
}

resource "azurerm_resource_group" "goreleaser-azure-resource-group" {
  name     = var.name
  location = var.azure_location
}

resource "azurerm_storage_account" "goreleaser-azure-storage-account" {
  name                     = "gorleaserquickbites"
  resource_group_name      = azurerm_resource_group.goreleaser-azure-resource-group.name
  location                 = azurerm_resource_group.goreleaser-azure-resource-group.location
  account_tier             = "Standard"
  account_replication_type = "LRS"
  allow_blob_public_access = true
  network_rules {
    default_action = "Allow"
  }
}

resource "azurerm_storage_container" "goreleaser-storage-container" {
  name                  = var.name
  storage_account_name  = azurerm_storage_account.goreleaser-azure-storage-account.name
  container_access_type = "container"
}

resource "aws_s3_bucket" "goreleaser-s3-bucket" {
  bucket = var.name
  acl    = "public-read"
}