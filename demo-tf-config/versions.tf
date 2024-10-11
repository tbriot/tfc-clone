terraform {
  required_version = "~> 1.9.6"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.69.0"
    }
  }
  cloud {
    organization = "tbriot-org"
    workspaces {
      name = "terraform-cloud-test"
    }
  }


}

provider "aws" {
  region = "ca-central-1"

}
