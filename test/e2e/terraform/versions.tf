# Pin OpenTofu and the GitLab provider. Run every command with `tofu`, not
# `terraform`: this stack is only tested against OpenTofu.
terraform {
  required_version = ">= 1.6.0"

  required_providers {
    gitlab = {
      source  = "gitlabhq/gitlab"
      version = "~> 19.1"
    }
  }
}
