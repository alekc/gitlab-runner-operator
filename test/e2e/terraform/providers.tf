# The bootstrap token authenticates every API call this stack makes (create the
# group, project, CI file, and the project access token). It needs the `api`
# scope. Supply it with `export TF_VAR_gitlab_token=...` or the provider's
# native `GITLAB_TOKEN` env var; never commit it. base_url is derived from
# var.gitlab_url so a single variable drives both the provider and the outputs.
provider "gitlab" {
  base_url = "${var.gitlab_url}/api/v4/"
  token    = var.gitlab_token
}
