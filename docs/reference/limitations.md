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

## No API version conversion

There is no conversion webhook. `v1alpha1` and `v1beta1` objects cannot be read
by this operator, and the authentication block changed shape, so moving from
v1 to v2 is a manual export and reapply. See
[upgrading from v1beta1](../install.md#upgrading-from-v1beta1).

## MultiRunner entries share more than you might expect

A `MultiRunner` entry can only vary three things: `authentication`,
`executor_config` and `environment`. Everything else is set once on the spec
and shared by every entry, including `gitlab_instance_url`, `caCertificate`,
`concurrent`, `check_interval`, `log_level`, `log_format`, `runner_image`,
`runner_resources`, `runner_image_pull_policy` and
`runner_security_context`.

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
