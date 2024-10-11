<!-- BEGIN_TF_DOCS -->
## This is a custom header imported from docs/.header.md

[![pre-commit.ci status](https://results.pre-commit.ci/badge/github/tbriot/terraform-test-workflow/main.svg)](https://results.pre-commit.ci/latest/github/tbriot/terraform-test-workflow/main)

#### Table of Contents
1. [Usage](#usage)
2. [Requirements](#requirements)
3. [Resources](#resources)
4. [Inputs](#inputs)
5. [Outputs](#outputs)

## Usage

Text describing how to use the terraform configuration.

## Requirements

| Name | Version |
|------|---------|
| <a name="requirement_terraform"></a> [terraform](#requirement\_terraform) | ~> 1.9.6 |
| <a name="requirement_aws"></a> [aws](#requirement\_aws) | ~> 5.69.0 |

## Resources

| Name | Type |
|------|------|
| [aws_iam_policy.example](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_policy) | resource |
| [aws_security_group.allow_tls](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/security_group) | resource |
| [aws_iam_policy_document.test](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/iam_policy_document) | data source |
| [aws_vpc.selected](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/vpc) | data source |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_s3_bucket_name"></a> [s3\_bucket\_name](#input\_s3\_bucket\_name) | n/a | `string` | `"test-tbriot-workflow"` | no |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_instance_ip_addr"></a> [instance\_ip\_addr](#output\_instance\_ip\_addr) | n/a |

## This is a custom footer imported from docs/.footer.md
<!-- END_TF_DOCS -->
