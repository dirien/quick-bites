terraform {
  required_providers {
    google  = {
      source  = "hashicorp/google"
      version = "4.63.0"
    }
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "3.53.0"
    }
    aws     = {
      source  = "hashicorp/aws"
      version = "4.64.0"
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