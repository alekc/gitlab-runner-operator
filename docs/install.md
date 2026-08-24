---
description: >-
  Install the GitLab Runner Operator with Helm or kustomize, upgrade it, and
  migrate from the v1beta1 API.
---

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

Helm never updates anything installed from a chart's `crds/` directory, so a
chart upgrade that ships new CRD fields does not apply them. Apply the CRDs
**first**, then upgrade:

```bash
helm repo update
helm show crds alekc/gitlab-runner-operator | kubectl apply --server-side -f -
helm upgrade gitlab-runner-operator alekc/gitlab-runner-operator
```

Order matters if you create runners from the chart's `runners:` or
`multiRunners:` values. Those specs pass through unchanged, so a field the
installed CRD does not know is pruned by the API server, silently and with no
error. Applying the CRDs afterwards does not bring the pruned value back: you
would have to run the upgrade again.

Use `--server-side`: these CRDs are large, and a client-side apply stores the
whole schema in a `last-applied-configuration` annotation.

!!! warning "Upgrading past the hardcoded limit changes parallelism and rolls every runner once"

    Every rendered entry used to carry a hardcoded `limit = 10`, so `concurrent`
    above 10 was silently capped at ten jobs. `limit` is now a field the operator
    never defaults, and unset means bounded by `concurrent` alone, so those
    runners are free to run the full budget they always asked for. Check the
    cluster can schedule it first.

    Reaching that budget also needs `request_concurrency`, which the operator no
    longer renders either, so it sits at gitlab-runner's default of 1. At 1 an
    entry acquires work one round trip at a time, so fifty slots fill over fifty
    sequential requests and on a contended queue may never fill. Raise both, or
    the extra capacity sits idle. See
    [concurrency](guides/concurrency.md).

    To keep the old behaviour instead, set `limit: 10` explicitly. That also
    avoids the restart below, since the rendered config is then byte-identical to
    what the previous release produced. Order matters: apply the new CRDs first,
    then set `limit` on every object, then upgrade. A `limit` set before the CRD
    knows the field is pruned by the API server with no error, and for
    chart-managed `runners:` values that leaves you with exactly the parallelism
    jump you were trying to prevent.

    Otherwise, because `limit = 10` stops being rendered, the config hash changes
    for every existing `Runner` and `MultiRunner` even though their specs did
    not. The first reconcile after the upgrade restarts each runner manager, and
    the manager does not drain
    ([#84](https://github.com/alekc/gitlab-runner-operator/issues/84)), so jobs
    in flight at that moment may be lost. It happens once. Upgrade when the
    pipeline queue is quiet.

!!! warning "Manager pod settings that were silently ignored start taking effect"

    `runner_resources`, `runner_image_pull_policy` and `runner_security_context`
    shape the manager pod but never reach `config.toml`, so until now a change to
    one of them did not roll the Deployment and did not apply. The reconcile now
    compares the manager pod's shape directly, so the value you set takes effect
    on the first reconcile after the upgrade, at the cost of one manager restart.
    As above, the manager does not drain
    ([#84](https://github.com/alekc/gitlab-runner-operator/issues/84)).

    A runner that never set any of the three is unaffected: the operator's own
    defaults are stable, so its live pod already matches and nothing rolls.

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
