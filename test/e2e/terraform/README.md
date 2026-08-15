# e2e GitLab fixture (OpenTofu)

This stack provisions the GitLab side of the end-to-end suite in `test/e2e`. The
suite needs a live GitLab project it can register runners against and trigger
real pipelines on. Standing that up by hand (project, CI file, a token with the
right scopes) is fiddly and easy to get subtly wrong, so it lives here as code.

Run everything with `tofu`, not `terraform`.

## What it creates

```text
project (private)                       var.project_name
  ├── .gitlab-ci.yml                    build-job, tagged var.job_tag
  └── project access token              scopes: api, create_runner (Maintainer)
```

By default the project is created in the token owner's **personal namespace**.
No group is created: gitlab.com forbids creating a top-level group via the API
(`POST /groups` returns 403, even for a token with `api` scope and an account
whose `can_create_group` is true). To nest the project under a group, create
the group once in the web UI and pass its numeric id as `namespace_id` (see
`terraform.tfvars.example`). A dedicated e2e account's personal namespace is the
simplest and fully-IaC option.

- **Project** is a throwaway. Shared runners and Auto DevOps are disabled so the
  only thing that can run `build-job` is the operator-managed runner the suite
  creates. All e2e matrix jobs share this single project, so the CI workflow runs
  the k8s matrix with `max-parallel: 1`; parallel jobs would let a build-job land
  on a sibling's runner and die when that sibling tears down first.
- **`.gitlab-ci.yml`** defines a single `build-job` tagged with `var.job_tag`.
  The suite registers its managed runner with `run_untagged = false` and that
  same tag, then asserts the job ran on *our* runner id.
- **Project access token** is what the suite consumes as `GITLAB_E2E_TOKEN`. It
  reads the project (`api`), triggers pipelines (`api` + Maintainer), and mints
  project runners (`create_runner`).

## Why a dedicated GitLab account

The suite creates and deletes runners, triggers pipelines, and reads project
data using a token with `api` scope. Point it at a throwaway account, never at
anything with access to real projects. The whole fixture is disposable: destroy
and re-create it whenever you like.

## Prerequisites

1. A dedicated GitLab account (gitlab.com or self-managed).
2. A **bootstrap** personal access token for that account with the `api` scope.
   This is what runs the stack. Create it under User settings, Access tokens.
   It is separate from the token the stack itself produces.
3. OpenTofu >= 1.6. The GitLab provider is downloaded on `tofu init`.

## Usage

```bash
cd test/e2e/terraform

# Bootstrap token for the dedicated account (api scope). TF_VAR_gitlab_token or
# the provider's native GITLAB_TOKEN both work.
export TF_VAR_gitlab_token="glpat-xxxxxxxxxxxxxxxxxxxx"

# Self-managed? also: export TF_VAR_gitlab_url="https://gitlab.example.com"

tofu init
tofu plan
tofu apply
```

## Wiring the outputs into the suite

The stack exposes exactly the three values the suite requires.

The plain `tofu output` view masks sensitive values as `(sensitive value)`.
Use `-raw` to print a single value unmasked, or `-json` for everything. This
reads local state only, so it needs no token and no network.

```bash
tofu output -raw project_id      # -> GITLAB_E2E_PROJECT_ID
tofu output -raw gitlab_e2e_url  # -> GITLAB_E2E_URL
tofu output -raw e2e_token       # -> GITLAB_E2E_TOKEN (sensitive)
tofu output -json                # all outputs, sensitive ones included
```

### Local run (`make test-e2e`)

Write the ready-to-source env file to the repo root, where `.envrc.example`
lives and `make test-e2e` runs, in one shot (the token is piped, never printed):

```bash
tofu output -raw e2e_env > "$(git rev-parse --show-toplevel)/.envrc"
direnv allow "$(git rev-parse --show-toplevel)"   # if you use direnv
```

Then follow the local-run steps in the operator repo (kind cluster,
cert-manager, `make deploy`, `make test-e2e`).

### GitHub Actions (`.github/workflows/e2e.yaml`)

Set the three repository secrets the workflow reads:

```bash
gh secret set GITLAB_E2E_URL        --body "$(tofu output -raw gitlab_e2e_url)"
gh secret set GITLAB_E2E_PROJECT_ID --body "$(tofu output -raw project_id)"
gh secret set GITLAB_E2E_TOKEN      --body "$(tofu output -raw e2e_token)"
```

## Token scopes and the fallback

The e2e token needs **both** `api` and `create_runner`, and at least Maintainer
on the project. The suite mints project runners through `POST /user/runners`,
which acts as the token's owner. A project access token bot with Maintainer and
`create_runner` is expected to satisfy this.

If your instance rejects runner creation with the project access token, the
simplest fallback is to give your **bootstrap** PAT the `create_runner` scope
too and use it directly as `GITLAB_E2E_TOKEN`. A personal access token always
has a real user owner and will mint the runner. The rest of the fixture (group,
project, CI file) is unaffected.

## Keep the CI tag in sync

`var.job_tag` defaults to `test-gitlab-runner`, which must match the `jobTag`
constant in `test/e2e/e2e_suite_test.go`. Change one and you must change the
other, otherwise the managed runner never picks up `build-job` and the suite
times out.

## Cleanup

```bash
tofu destroy
```

Removes the project and its token. A group referenced via `namespace_id` is not
managed by this stack, so it is left untouched.

## State safety

The state file maps this config to live GitLab resources and can contain the
project access token in plaintext. It is gitignored here. For shared or
long-lived use, put it in a remote backend (for example an object-storage
backend with encryption at rest) rather than leaving it on a laptop. Never
hand-edit or delete `*.tfstate`; let `tofu` manage it.
