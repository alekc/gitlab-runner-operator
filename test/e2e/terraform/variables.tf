variable "gitlab_url" {
  description = "Instance root URL, no trailing slash and no /api/v4 suffix."
  type        = string
  default     = "https://gitlab.com"
}

variable "gitlab_token" {
  description = <<-DESC
    Bootstrap token used to run this stack (api scope). Leave null to let the
    provider read GITLAB_TOKEN from the environment. Prefer a token that belongs
    to the dedicated e2e account, not a personal admin account.
  DESC
  type        = string
  default     = null
  sensitive   = true
}

variable "namespace_id" {
  description = <<-DESC
    Namespace the e2e project is created in. Leave null to use the token owner's
    personal namespace, which is the default and works on gitlab.com. gitlab.com
    forbids creating top-level groups via the API (POST /groups returns 403), so
    this stack does not create one. To nest the project under a group, create the
    group once in the web UI (or an API-created subgroup) and pass its numeric id
    here.
  DESC
  type        = number
  default     = null
}

variable "project_name" {
  description = "Name of the throwaway project the suite runs pipelines against."
  type        = string
  default     = "e2e-runner"
}

variable "job_tag" {
  description = <<-DESC
    Default RUNNER_TAG in the generated .gitlab-ci.yml, used when a pipeline
    does not set one. MUST match the defaultJobTag constant in
    test/e2e/e2e_suite_test.go, or a locally run suite never picks the job up.
    CI overrides it per run via GITLAB_E2E_RUNNER_TAG so concurrent runs cannot
    steal each other's jobs.
  DESC
  type        = string
  default     = "test-gitlab-runner"
}

variable "ci_job_image" {
  description = <<-DESC
    Container image build-job runs in. The Kubernetes executor needs an explicit
    image or the job fails with "no image specified"; that is why the CI file
    sets one rather than relying on a runner default.
  DESC
  type        = string
  default     = "alpine:latest"
}

variable "token_expires_at" {
  description = <<-DESC
    Expiry for the e2e project access token, YYYY-MM-DD. gitlab.com REQUIRES an
    expiry and rejects creation without one ("expires_at is missing"). Leave null
    to default to ~11 months out, under the 365-day maximum. The resource ignores
    later changes to expires_at (a change would rotate the token); use
    `tofu apply -replace=gitlab_project_access_token.e2e` for a fresh window.
  DESC
  type        = string
  default     = null
}

variable "commit_author_name" {
  description = "Author name recorded on the CI-file commit."
  type        = string
  default     = "OpenTofu"
}

variable "commit_author_email" {
  description = "Author email recorded on the CI-file commit."
  type        = string
  default     = "tofu@example.com"
}
