# Install

## Requirements

| Requirement | Why |
| --- | --- |
| Kubernetes 1.25 or newer | The CRDs express their validation with CEL (`x-kubernetes-validations`), which the API server only enforces from 1.25. On anything older the CRDs install but validate nothing, so a bad `Runner` is accepted and fails later at runtime. |
| A GitLab runner authentication token, or an access token with `create_runner` | See [authentication](authentication.md). Registration tokens are not supported; GitLab deprecated them in 16.0 and disabled them by default from 18.0. |

## Helm

```bash
helm repo add alekc https://charts.alekc.dev/
helm repo update
helm install gitlab-runner-operator alekc/gitlab-runner-operator
```

The chart installs the CRDs from its `crds/` directory on first install. Values
worth setting at install time:

| Value | Default | Notes |
| --- | --- | --- |
| `allowedBuildNamespaces` | `[]` | Namespaces besides a runner's own where the operator may provision executor RBAC. Empty means every runner is confined to its own namespace. See [RBAC and namespaces](operations/rbac-and-namespaces.md). |
| `metrics.enabled` | `true` | Serves controller-runtime metrics on port 8080. The endpoint is unauthenticated, so restrict it with a NetworkPolicy or turn it off. |
| `image.tag` | chart `appVersion` | Pin this if you want the operator version fixed independently of the chart. |
| `runners`, `multiRunners` | `[]` | Create runner objects from the chart itself, so the operator and its runners land in one release. |

The full values reference lives in the
[chart README](https://gitlab.com/alexander-chernov/helm/gitlab-runner-operator).

### Upgrading

```bash
helm repo update
helm upgrade gitlab-runner-operator alekc/gitlab-runner-operator
```

Helm never updates anything installed from a chart's `crds/` directory, so a
chart upgrade that ships new CRD fields does not apply them. Apply them
yourself:

```bash
helm show crds alekc/gitlab-runner-operator | kubectl apply --server-side -f -
```

Use `--server-side`: these CRDs are large, and a client-side apply stores the
whole schema in a `last-applied-configuration` annotation.

## Kustomize

From a checkout of the repo:

```bash
make install                      # CRDs only
make deploy IMG=ghcr.io/alekc/gitlab-runner-operator:v2.0.1
```

`make deploy` renders `config/default` with the image you name and applies it.
There is no `allowedBuildNamespaces` value in this path; edit the manager args
in `config/manager` directly.

## Versions

Three version numbers are in play and they move independently:

| Number | Example | What it tracks |
| --- | --- | --- |
| Chart version | `2.2.1` | Packaging: templates, values, chart metadata. |
| Operator version (`appVersion`, image tag) | `v2.0.1` | The controller binary. This is what these docs are versioned against. |
| API group version | `v1beta2` | The CRD schema. Changes rarely, breaks compatibility when it does. |

## Upgrading from v1beta1

There is no conversion webhook and no automatic migration. `v1beta1` and
`v1alpha1` objects are not readable by this operator, and the authentication
block was reworked, so the move is a manual export and reapply:

1. Note the runners you have and how each one authenticates.
2. Create a runner in GitLab for each one (UI or `POST /user/runners`) and keep
   the `glrt-` token, or prepare an access token with `create_runner` if you
   want the operator to create them. Registration tokens no longer work.
3. Delete the old objects **while the old operator is still running**, so its
   finalizer can deregister them. See [uninstalling](operations/uninstall.md).
4. Install this version, then write the objects again against
   `gitlab.k8s.alekc.dev/v1beta2` using the new
   [authentication](authentication.md) shape.

The chart advertises `Basic Install` on Artifact Hub rather than
`Seamless Upgrades` for exactly this reason.
