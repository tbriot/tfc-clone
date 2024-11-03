terraform {
  required_version = "~> 1.8.3"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.69.0"
    }
  }
}

provider "aws" {
  region = "ca-central-1"
}
