output "project_id" {
  description = "GITLAB_E2E_PROJECT_ID: numeric id of the e2e project."
  value       = gitlab_project.e2e.id
}

output "project_web_url" {
  description = "Human-facing URL of the e2e project."
  value       = gitlab_project.e2e.web_url
}

output "gitlab_e2e_url" {
  description = "GITLAB_E2E_URL: instance root the suite and Runner CRs point at."
  value       = "${var.gitlab_url}/"
}

output "e2e_token" {
  description = "GITLAB_E2E_TOKEN: project access token consumed by the suite."
  value       = gitlab_project_access_token.e2e.token
  sensitive   = true
}

# Convenience bundle. Write it to the repo-root .envrc (gitignored) with:
#   tofu output -raw e2e_env > "$(git rev-parse --show-toplevel)/.envrc"
output "e2e_env" {
  description = "Ready-to-source exports for the three GITLAB_E2E_* variables."
  sensitive   = true
  value       = <<-ENV
    export GITLAB_E2E_URL="${var.gitlab_url}/"
    export GITLAB_E2E_PROJECT_ID="${gitlab_project.e2e.id}"
    export GITLAB_E2E_TOKEN="${gitlab_project_access_token.e2e.token}"
  ENV
}
