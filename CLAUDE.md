# Working in this repository

A Kubernetes operator that manages GitLab CI runners through CRDs, using the
kubernetes executor. Contributor setup and the docs toolchain are in
[docs/contributing.md](docs/contributing.md); the user-facing documentation is
in [docs/](docs/) and published at <https://gitlab-runner-operator.alekc.dev/>.

These instructions apply to any AI agent working in this repository.

## The mental model

A `Runner` (one runner) or `MultiRunner` (several entries, one manager) is
reconciled into four things:

1. a Secret holding the rendered `config.toml` and the runner token;
2. a ServiceAccount, one per object, so each runner is a distinct identity;
3. a RoleBinding tying that ServiceAccount to a shared executor ClusterRole,
   created in the build namespace;
4. a Deployment running `gitlab-runner`, one replica, which then creates job
   pods itself.

Two invariants matter when changing any of this:

- **The namespace RBAC is provisioned for must match the namespace the rendered
  config tells the runner to use.** `KubernetesConfig.EffectiveNamespace` is the
  single source of that rule and is shared by both paths deliberately. Do not
  reimplement the defaulting.
- **The config hash rolls the Deployment.** `status.config_map_version` is a
  digest of the rendered config; the controller writes it as a pod annotation so
  a config change restarts the runner. Anything that should take effect on the
  running runner has to be inside that hash.

## Layout

| Path | What lives there |
| --- | --- |
| `api/v1beta2/` | CRD types. Validation is CEL via `+kubebuilder:validation:XValidation`, not a webhook. |
| `config/config.go` | The `config.toml` shape the operator renders. Mirrors gitlab-runner's own config struct. |
| `internal/generate/` | Renders `config.toml` and the runner system ID. Tests assert on rendered output, not on structs. |
| `internal/controller/` | Reconcilers. `shared.go` holds the managed-runner decision tree (create, recreate, verify). |
| `internal/crud/` | Creates and prunes the child objects, including cross-namespace RoleBindings. |
| `internal/validate/` | Builds the runner **Deployment** and reconciles it. The manager pod template lives here, which is not where you would look for it first. |
| `hack/*.sh` | CI helpers, each with a `_test.sh` beside it. See below. |
| `test/e2e/` | Live suite against a real GitLab project, plus an OpenTofu fixture that provisions one. |
| `docs/` | The published site. `docs/reference/api.md` is generated. |

## Documentation is part of the change

**Any change to the CRDs, to the API types, or to observable behaviour must
leave the documentation consistent in the same change.** Not in a follow-up, not
in an issue. A merged commit that makes a doc page wrong is a regression, and a
reader has no way to tell that a page is stale.

| You changed | Check and amend |
| --- | --- |
| `api/` types, kubebuilder markers, CEL rules | `make docs` to regenerate the reference. Then the prose page describing the field: a new field usually needs a sentence somewhere in `docs/`, not just a generated table row. |
| A default value | Every page stating the old default. Search for the value, not the field name. |
| A field's behaviour, or made one inert | `docs/reference/limitations.md`. An accepted-but-ignored field belongs in the inert table there. |
| A hardcoded value in `internal/generate/` | `docs/reference/limitations.md`, and `docs/guides/concurrency.md` if it caps something. |
| RBAC, namespaces, finalizers | `docs/operations/`. |
| Authentication or the GitLab API calls | `docs/authentication.md`. |
| Anything a guide's example manifest uses | The affected page in `docs/guides/`. A guide with a manifest that no longer applies is worse than no guide. |
| Install, upgrade or uninstall steps | `docs/install.md`, `docs/operations/uninstall.md`, and the README if the quickstart moved. |
| Make targets, CI workflows, the docs pipeline | `docs/contributing.md`. |

The test to apply: **could someone follow the existing documentation and reach a
wrong outcome, or miss a required step?** If yes, the docs are part of this
change. This is not a requirement to write new documentation for every change,
it is a requirement that existing documentation does not contradict reality.

## Generated files

Never hand-edit these. The edit is silently reverted by the next regeneration.

```bash
make manifests   # config/crd/bases/*.yaml
make generate    # api/v1beta2/zz_generated.deepcopy.go
make docs        # docs/reference/api.md
```

`make test` runs `manifests` and `generate` first, so a forgotten regeneration
shows up as a dirty tree rather than a confusing test failure.

## Commands and their traps

```bash
make test          # unit + envtest. Downloads envtest assets on first run.
make lint          # golangci-lint, standard set plus misspell and unconvert
make govulncheck   # binary-mode CVE scan
make docs-verify   # fails if docs/reference/api.md drifted from the types
make docs-serve    # local docs at :8000, strict
```

- **`GOTOOLCHAIN=auto` matters in CI.** `actions/setup-go` exports
  `GOTOOLCHAIN=local`, which ignores the `toolchain` directive in `go.mod`. That
  is how v2.0.0 shipped built against a different stdlib patch than intended
  (#49). Every Go step in a workflow sets it per command; keep doing that.
- **Build local images from `Dockerfile.e2e`, not `Dockerfile`.** The root
  Dockerfile is the release image: goreleaser assembles it from a prebuilt
  `dist/` tarball, so a plain `docker build` fails on the tar step.
  `make docker-build` uses it and has the same problem.
- **The `source/gitlab-runner` submodule is not checked out** and does not need
  to be for normal work. `hack/runner-release-watch.sh` uses upstream to detect
  new kubernetes executor keys we have not modelled yet.
- **`hack/*.sh` scripts each have a `_test.sh`**, run by the PR workflow, because
  a silent bug in one of them either hides live CVEs or lets an untested commit
  merge. Change a script, update its test.

## CI gates worth knowing before you push

- The PR workflow runs build, vet, `make test`, the three `hack` script tests,
  lint, govulncheck, and a docs job that runs `make docs-verify` plus
  `mkdocs build --strict`.
- **`e2e-gate` is the required status check**, deliberately named for its
  stability rather than the per-version e2e legs, whose matrix is computed at
  run time. It treats a skipped suite as a failure, so a broken upstream job
  cannot wave a PR through.
- **A release tag must point at a commit e2e has already passed on.** The
  release workflow looks up a successful `e2e.yaml` run for that exact SHA and
  refuses to publish otherwise. Renaming that workflow file breaks the lookup
  and blocks every release until the name is updated in `release.yml` too.
- Pin third-party actions to a commit SHA with the version in a trailing
  comment, matching what is already there.

## The chart is a separate repository

The Helm chart lives at
`gitlab.com/alexander-chernov/helm/gitlab-runner-operator`, not here. A change
that adds or renames a values key, or changes what the operator needs at install
time, leaves that repo's `README.md` (generated by helm-docs from
`README.md.gotmpl`) and `values.schema.json` stale. Say so explicitly in the PR
description when you cannot fix it in the same change.

Keep the explanation of any mechanism in one place: the chart README covers
install and values, everything else links to the docs site instead of restating
it.

## Conventions

- **Validation belongs in CEL** on the type, so a bad object is rejected at
  admission rather than failing at reconcile. Two keys are refused outright this
  way (`namespace_per_job`, `namespace_overwrite_allowed`) because they make the
  build namespace dynamic, which would require cluster-scoped RBAC for runners.
- **The operator holds the executor permissions cluster-wide and has no RBAC
  `escalate` verb.** Its own ClusterRole is therefore the ceiling on what any
  runner can be granted. Widening it widens every runner: treat that as a
  security change, not a convenience.
- **Comments explain why.** Several of the ones already in the tree record a
  specific incident; keep that style and keep them short.
- Conventional Commits, imperative and lowercase after the type, no trailing
  period. Sign off with `git commit -s`.
- Do not add AI attribution or co-author trailers to commits, PR descriptions,
  or anything else this repository publishes.
