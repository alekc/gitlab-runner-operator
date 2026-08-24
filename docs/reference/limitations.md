---
description: >-
  What the GitLab Runner Operator deliberately does not do: no distributed
  cache, static build namespace, inert fields, and what the manager pod still
  cannot be told.
---

# Limitations

What the operator deliberately does not do, and what it accepts but cannot make
work. Read this before assuming a missing field is a bug.

## No distributed cache

There is no way to configure `[runners.cache]`. The CRD has no cache types, the
rendered `config.toml` has no cache section, and there is no raw `config.toml`
passthrough to work around it. S3, GCS and Azure Blob are all unreachable.

The consequence is worse than a missing feature: `cache:` in `.gitlab-ci.yml`
looks configured but never restores anything, because with the kubernetes
executor the local cache directory lives in the build pod and dies with it.

Tracked in
[issue #81](https://github.com/alekc/gitlab-runner-operator/issues/81).

## Kubernetes executor only

`spec.executor_config` is a kubernetes executor config and nothing else. There
is no docker, shell, ssh or custom executor, and no plan for one. If you need a
shell runner, run it outside this operator.

## The build namespace must be static

`namespace_per_job` and `namespace_overwrite_allowed` are rejected at admission
by the CRD schema, with a CEL message explaining why. Both make the build
namespace dynamic, and the operator pre-provisions namespaced RBAC for a
namespace it has to know in advance. Supporting them would mean granting
runners cluster-scoped permissions.

Any namespace other than the runner's own also has to be allow-listed on the
operator. See [RBAC and namespaces](../operations/rbac-and-namespaces.md).

## Fields accepted but inert

| Field | What actually happens |
| --- | --- |
| `executor_config.terminationGracePeriodSeconds` | Accepted, rendered into `config.toml`, then silently dropped by the runner. gitlab-runner removed the key in v17.0.0. Use `pod_termination_grace_period_seconds` and `cleanup_grace_period_seconds` instead. |
| `executor_config.pod_spec` | Rendered, but the runner ignores it unless the job also sets the `FF_USE_ADVANCED_POD_SPEC_CONFIGURATION` [feature flag](https://docs.gitlab.com/runner/configuration/feature-flags.html). |

## The runner manager pod is only partly configurable

`executor_config` shapes **job** pods. The manager pod takes a narrower set of
`runner_*` fields: its image, resources, pull policy, security context, and its
placement (`runner_node_selector`, `runner_tolerations`, `runner_affinity`,
`runner_priority_class_name`, see [node placement](../guides/node-placement.md)).
Two things production clusters ask for are still missing:

| Missing | Consequence | Issue |
| --- | --- | --- |
| Environment variables on the container | No `HTTP_PROXY` / `NO_PROXY`, so a runner behind an outbound proxy cannot register. `spec.environment` is the *build* environment and does not help. | [#82](https://github.com/alekc/gitlab-runner-operator/issues/82) |
| `terminationGracePeriodSeconds` and a drain hook | Stuck at Kubernetes' 30s default with no `preStop`, so a rollout or eviction kills in-flight jobs instead of draining. | [#84](https://github.com/alekc/gitlab-runner-operator/issues/84) |

The Deployment is also fixed at one replica, which is correct (two managers would
double the effective concurrency) but means throughput scales by adding runner
objects or `MultiRunner` entries, not replicas.

Three consequences of how the manager Deployment is reconciled, all sharpened by
the placement fields because they give it more to compare:

- **Hand edits to the Deployment are reverted.** Each roll rewrites the spec
  from scratch, so labels, annotations, extra containers and a changed `replicas`
  are dropped on the next roll. This was always true; it now happens on more
  triggers. A mutating webhook is the exception rather than an example: it
  re-applies during the operator's own write, so its additions survive, and
  anything it adds outside the compared subset does not even trigger a roll.
- **A mutating webhook that rewrites the manager pod wins.** If a policy engine
  rewrites `nodeSelector`, `tolerations`, `priorityClassName` or `resources` on
  the Deployment, the operator notices its own write did not stick and leaves the
  webhook's version in place rather than fighting it. Your spec value is then
  ignored, and the reconcile logs what it asked for against what was stored.
- **A field this CRD accepts but your cluster cannot store is silently ignored.**
  The CRDs are generated against a newer Kubernetes than the oldest one the
  operator supports, so a gated or unknown field (`runner_resources.claims`
  without `DynamicResourceAllocation`, `appArmorProfile` below 1.30,
  `matchLabelKeys` below 1.31) is dropped by the apiserver with a warning rather
  than an error. The runner still becomes ready; the field just does nothing.

## Concurrency settings

`limit` and `request_concurrency` are settable per runner and per `MultiRunner`
entry. Neither is defaulted by the operator: unset, the key is omitted and
gitlab-runner applies its own default, so `concurrent: 50` runs fifty jobs. A
`limit` above `concurrent` is accepted and inert, since the lower of the two
wins. Nothing reports the effective ceiling back: upstream's startup warnings do
not cover a single-entry runner, and its `gitlab_runner_limit` gauge reports the
configured value rather than the cap in force. See
[concurrency](../guides/concurrency.md).

One related setting is not exposed:

| Setting | State | Effect |
| --- | --- | --- |
| `output_limit` | Never set, so the upstream default | Job log size cap is not tunable per runner. |

## No API version conversion

There is no conversion webhook. `v1alpha1` and `v1beta1` objects cannot be read
by this operator, and the authentication block changed shape, so moving from
v1 to v2 is a manual export and reapply. See
[upgrading from v1beta1](../install.md#upgrading-from-v1beta1).

## MultiRunner entries share more than you might expect

A `MultiRunner` entry can only vary five things: `authentication`,
`executor_config`, `environment`, `limit` and `request_concurrency`. Everything else is set once on the spec
and shared by every entry, including `gitlab_instance_url`, `caCertificate`,
`concurrent`, `check_interval`, `log_level`, `log_format`, `runner_image`,
`runner_resources`, `runner_image_pull_policy`, `runner_security_context`,
`runner_node_selector`, `runner_tolerations`, `runner_affinity` and
`runner_priority_class_name`. The placement fields are shared because there is
one manager pod, so per-entry placement is not a thing that could exist.

So one `MultiRunner` cannot span two GitLab instances, or use a different CA
per entry. Use separate `Runner` objects for that.

All entries are also served by one runner manager pod, which is the point of
the kind: `concurrent` is a budget shared across every entry, not a per-entry
limit.

## Asymmetries between Runner and MultiRunner

`sentry_dsn` exists on `MultiRunner` only. A plain `Runner` cannot report system
level errors to Sentry.

## Registration tokens are not supported

Only runner authentication tokens (`glrt-`) and access tokens with
`create_runner`. GitLab deprecated registration tokens in 16.0 and disabled
them by default from 18.0, so there is nothing to fall back to on a current
GitLab.
