# Structure provisioned in the dedicated e2e GitLab account:
#
#   project (private, in var.namespace_id or the personal namespace)
#     ├── .gitlab-ci.yml on the default branch (build-job, tagged)
#     └── project access token (api + create_runner, Maintainer)
#
# No top-level group is created: gitlab.com forbids creating one via the API
# (POST /groups -> 403). By default the project lands in the token owner's
# personal namespace; set var.namespace_id to nest it under an existing group.
#
# The suite in test/e2e reads the project id and the token from the outputs of
# this stack. Everything here is throwaway: `tofu destroy` removes it cleanly.

locals {
  # $CI_* stays literal: OpenTofu only interpolates ${...}, not a bare $.
  # build-job needs the configured tag and an explicit image so the Kubernetes
  # executor has something to run.
  gitlab_ci_yml = <<-YAML
    # Managed by OpenTofu (test/e2e/terraform). Do not edit in the GitLab UI:
    # changes are overwritten on the next `tofu apply`.
    stages:
      - build

    variables:
      # The e2e suite overrides this per pipeline so a run's job can only be
      # picked up by that run's runner. Without it every concurrent run shares
      # one tag and a job can land on a sibling's runner, which then tears its
      # cluster down mid-job. The default keeps a manual pipeline working.
      RUNNER_TAG: ${var.job_tag}

    build-job:
      stage: build
      image: ${var.ci_job_image}
      tags:
        - $RUNNER_TAG
      script:
        - echo "e2e build-job on runner $CI_RUNNER_ID ($CI_RUNNER_DESCRIPTION)"
        - echo "commit $CI_COMMIT_SHORT_SHA on ref $CI_COMMIT_REF_NAME"
  YAML

  # gitlab.com rejects a project access token without expires_at. When the
  # caller does not pin one, default to ~11 months out (8016h = 334d), safely
  # under gitlab.com's 365-day cap. Computed at creation; the resource ignores
  # later drift so a re-apply on a different day does not rotate the token.
  token_expires_at = coalesce(
    var.token_expires_at,
    formatdate("YYYY-MM-DD", timeadd(timestamp(), "8016h")),
  )
}

resource "gitlab_project" "e2e" {
  name         = var.project_name
  namespace_id = var.namespace_id # null => the token owner's personal namespace
  description  = "Throwaway project for the gitlab-runner-operator e2e suite. Managed by OpenTofu."

  # A default branch with content must exist before we can commit the CI file
  # and before the suite can trigger a pipeline on it.
  initialize_with_readme = true
  default_branch         = "main"

  # The suite sets RUNNER_TAG as a pipeline variable so a run's job can only be
  # picked up by that run's runner. GitLab refuses that with "Insufficient
  # permissions to set pipeline variables" unless this is at or below the
  # token's role, and new projects default to no_one_allowed. The e2e token is
  # a Maintainer, so maintainer is the least-privilege value that works.
  ci_pipeline_variables_minimum_override_role = "maintainer"

  # The suite asserts build-job ran on our operator-managed runner. Keep shared
  # runners out of the picture and silence Auto DevOps pipelines.
  shared_runners_enabled = false
  auto_devops_enabled    = false

  visibility_level = "private"
}

resource "gitlab_repository_file" "ci" {
  project        = gitlab_project.e2e.id
  branch         = gitlab_project.e2e.default_branch
  file_path      = ".gitlab-ci.yml"
  encoding       = "text"
  content        = local.gitlab_ci_yml
  author_email   = var.commit_author_email
  author_name    = var.commit_author_name
  commit_message = "Add e2e pipeline definition (managed by OpenTofu)"
}

# The token the suite consumes as GITLAB_E2E_TOKEN. It reads the project (api),
# triggers pipelines (api + Maintainer), and mints project runners
# (create_runner). Maintainer is the provider default; set explicitly for
# clarity.
resource "gitlab_project_access_token" "e2e" {
  project      = gitlab_project.e2e.id
  name         = "gitlab-runner-operator-e2e"
  description  = "Consumed by the e2e suite as GITLAB_E2E_TOKEN. Managed by OpenTofu."
  access_level = "maintainer"
  scopes       = ["api", "create_runner"]
  expires_at   = local.token_expires_at

  # expires_at is derived from timestamp() when not pinned, so it would drift on
  # every plan. Ignore that drift: updating it would destroy and recreate the
  # token (a new value). Use `tofu apply -replace` to get a fresh window.
  lifecycle {
    ignore_changes = [expires_at]
  }
}
