# Contributing

## Local cluster

```bash
make kind-create            # idempotent; override KIND_CLUSTER_NAME / KIND_K8S_VERSION
make install                # CRDs into the current context
make kind-destroy
```

To run the controller against that cluster without building an image:

```bash
make run
```

## Building and deploying an image

kind will pull an image tagged `latest` even after `kind load`, so tag it with
something else:

```bash
make docker-build IMG=controller:dev
kind load docker-image controller:dev --name kind
make deploy IMG=controller:dev
```

## Tests

```bash
make test                   # unit + envtest suites
make lint                   # golangci-lint
make govulncheck            # CVE scan of the built binary
```

`make test` runs `manifests` and `generate` first, so a change to the API types
that you forgot to regenerate shows up as a dirty tree rather than as a
mysterious test failure.

### Live e2e

`make test-e2e` runs against the current kube context and a real GitLab
project. It needs a deployed operator and three environment variables, and
skips when they are unset:

| Variable | Notes |
| --- | --- |
| `GITLAB_E2E_URL` | GitLab instance to register against. |
| `GITLAB_E2E_TOKEN` | Access token with **both** `api` and `create_runner`, Maintainer on the project. Use a throwaway project. |
| `GITLAB_E2E_PROJECT_ID` | Numeric project id. |

Copy `.envrc.example` to `.envrc` (gitignored) and fill them in, or let the
OpenTofu stack in `test/e2e/terraform` provision the project, CI file and token
and emit the file for you:

```bash
cd test/e2e/terraform
tofu output -raw e2e_env > "$(git rev-parse --show-toplevel)/.envrc"
```

## Changing the API

The generated pieces are committed, so any change to `api/` needs three
regenerations before the tree is clean again:

```bash
make manifests              # CRDs in config/crd/bases
make generate               # deepcopy functions
make docs                   # docs/reference/api.md
```

CI runs `make docs-verify`, which regenerates the reference into a temp file and
fails on any diff against the committed one. There is no way to land an API
change with a stale reference.

## Working on the docs

```bash
make docs-deps              # pip install mkdocs-material + mike
make docs-serve             # http://localhost:8000
```

`mkdocs.yml` is in strict mode, so a broken internal link or a nav entry
pointing at a missing file fails the build locally exactly as it does in CI.

Every code fence that shows a full manifest should come from a real file under
`config/samples` via a snippet include, rather than being retyped into the page.
That way the examples are covered by whatever validates the samples.

### How docs get published

`docs.yml` deploys the `dev` alias on every push to main that touches
`docs/**` or `mkdocs.yml`. The release workflow deploys a numbered version and
moves the `latest` alias, once the image is published. Both push to the
`gh-pages` branch, which is what GitHub Pages serves.

To publish a docs-only fix for the current release without cutting a new
version, run the docs workflow manually and give it the version to overwrite.

The site is served from the `gh-pages` branch at
<https://gitlab-runner-operator.alekc.dev/>, with the custom domain set in the
repository's Pages settings. GitHub keeps that domain as a `CNAME` file at the
root of `gh-pages`; mike only writes version directories, `versions.json` and
the root redirect, so it leaves that file alone. Deleting it drops the site back
to `alekc.github.io`.
