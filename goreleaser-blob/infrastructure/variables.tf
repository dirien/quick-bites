variable "gcp_project" {
  type = string
}

variable "gcp_region" {
  default = "europe-west6"
}

variable "gcp_zone" {
  default = "europe-west6-a"
}

variable "gcp_bucket_location" {
  default = "EU"
}

variable "gcp_auth_file" {
  default = "./auth.json"
  description = "Path to the GCP auth file"
}

variable "aws_region" {
  default = "eu-central-1"
}

variable "azure_location" {
  default = "West Europe"
}

variable "name" {
  default = "gorleaser-quickbites"
}