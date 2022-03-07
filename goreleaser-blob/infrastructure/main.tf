terraform {
  required_providers {
    google  = {
      source  = "hashicorp/google"
      version = "4.12.0"
    }
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "2.98.0"
    }
    aws     = {
      source  = "hashicorp/aws"
      version = "4.4.0"
    }
  }
}


provider "azurerm" {
  features {}
}

provider "google" {
  credentials = file(var.gcp_auth_file)
  project     = var.gcp_project
  region      = var.gcp_region
}

provider "aws" {
  region = var.aws_region
}