# Gitlab Runner Operator

Kubernetes operator that manages GitLab CI runners using the kubernetes executor.

## What is this operator for

It lets you run one or many GitLab runners, each with its own configuration
expressed in YAML (no more hand-written `config.toml`), following an
infrastructure-as-code approach. Every option exposed by the
[kubernetes executor](https://docs.gitlab.com/runner/executors/kubernetes.html)
is configurable through the CRD.

## Status

Alpha. Breaking changes are possible and will be called out in the release
notes. Please open an issue if you hit a bug.

> **API version**: the current API group version is
> `gitlab.k8s.alekc.dev/v1beta2`. It replaces the older `v1beta1` /
> `v1alpha1` versions. The change is breaking: the authentication block was
> reworked (see below) and existing objects are not converted automatically.

## Installation with helm

Once the CRDs are installed you can deploy the operator into your preferred
namespace:

```
helm repo add alekc https://charts.alekc.dev/
helm repo update
helm install gitlab-runner-operator alekc/gitlab-runner-operator
```

## Authentication

GitLab deprecated the registration-token workflow (in 16.0) and disabled it by
default from 18.0 onward, so this operator uses runner authentication tokens
(the `glrt-` tokens). There are two ways to authenticate, set under
`spec.authentication`:

1. **Bring your own token.** Create the runner in GitLab yourself (UI or the
   `POST /user/runners` API) and give the operator the resulting `glrt-` token.
   The operator makes no GitLab API calls; it just writes the token into the
   runner config.

2. **Operator-managed.** Give the operator an access token (personal, group, or
   project) that holds the `create_runner` scope plus a `create_options` block.
   The operator creates the runner through `POST /user/runners`, stores the
   returned token, and deletes the runner from GitLab when the object is
   removed.

Exactly one of the two modes must be configured; the admission webhook rejects
objects that set both or neither.

## Configuration

Top-level `spec` fields (all optional unless noted):

| Key | Description |
| --- | --- |
| `authentication` | Required. How the runner authenticates (see above). |
| `concurrent` | Maximum number of jobs run concurrently across this runner. Minimum 1. |
| `check_interval` | Seconds between checks for new jobs. Minimum 3. |
| `log_level` | One of `panic`, `fatal`, `error`, `warning`, `info`, `debug`. |
| `log_format` | One of `runner`, `text`, `json`. |
| `gitlab_instance_url` | GitLab URL. Defaults to `https://gitlab.com/`. |
| `executor_config` | Kubernetes executor options, see the [keywords reference](https://docs.gitlab.com/runner/executors/kubernetes.html#configuration-settings). |
| `environment` | Custom environment variables injected into the build environment. |
| `runner_image` | Override the gitlab-runner image. Defaults to a recent `gitlab/gitlab-runner:alpine-vX.Y.Z`. |

### `authentication` fields

| Key | Description |
| --- | --- |
| `token` | Bring-your-own mode: the pre-created `glrt-` token, as a token source (see below). |
| `access_token` | Managed mode: an access token with the `create_runner` scope, as a token source. |
| `create_options` | Managed mode: `runner_type` (`instance_type`/`group_type`/`project_type`), `group_id`, `project_id`, `description`, `tag_list`, `run_untagged`, `locked`, `paused`, `access_level`, `maximum_timeout`. |

Both `token` and `access_token` are **token sources** with two mutually
exclusive ways to supply the value:

| Key | Description |
| --- | --- |
| `value` | The literal token, inline. Convenient for testing. |
| `secret_key_ref` | Read the token from a Secret in the runner namespace: `name` (required), `key` (optional, defaults to `token`), `optional` (when `true`, a missing secret or key resolves to an empty token instead of failing). |

## Examples

### Bring-your-own token

```yaml
apiVersion: gitlab.k8s.alekc.dev/v1beta2
kind: Runner
metadata:
  name: runner-sample
spec:
  authentication:
    token:
      value: "glrt-XXXXXXXXXXXXXXXXXXXX"
```

### Operator-managed runner

```yaml
apiVersion: gitlab.k8s.alekc.dev/v1beta2
kind: Runner
metadata:
  name: runner-managed
spec:
  authentication:
    access_token:
      value: "glpat-XXXXXXXXXXXXXXXXXXXX"
    create_options:
      runner_type: project_type
      project_id: 1234567
      run_untagged: true
      tag_list:
        - test-gitlab-runner
```

### Token from a secret

`key` defaults to `token`; set `secret_key_ref.key` to read a different key.

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: gitlab-runner-token
type: Opaque
stringData:
  token: "glrt-XXXXXXXXXXXXXXXXXXXX"
---
apiVersion: gitlab.k8s.alekc.dev/v1beta2
kind: Runner
metadata:
  name: runner-sample
spec:
  authentication:
    token:
      secret_key_ref:
        name: gitlab-runner-token
        # key omitted -> defaults to "token"
```

### Mounting secrets or config maps as volumes

```yaml
apiVersion: gitlab.k8s.alekc.dev/v1beta2
kind: Runner
metadata:
  name: runner-sample
spec:
  log_level: debug
  executor_config:
    image: "debian:slim"
    memory_limit: "150Mi"
    memory_request: "150Mi"
    volumes:
      config_map:
        - mount_path: /cm/
          name: test-config
      secret:
        - mount_path: /secrets/1/
          name: test-secret
  authentication:
    token:
      value: "glrt-XXXXXXXXXXXXXXXXXXXX"
```

### Multiple runners in one object

See `config/samples/gitlab_v1beta2_multirunner.yaml` for a `MultiRunner`
example that mixes both authentication modes across entries.

## RBAC and namespaces

The kubernetes executor permission set (pods and pods/exec, pods/attach,
pods/log, services, secrets, configmaps, serviceaccounts, events) lives in one
shared ClusterRole, `gitlab-runner-operator-executor`, reconciled by the
operator. For each Runner or MultiRunner the operator then provisions its own
ServiceAccount (a distinct identity for audit and revocation) and a RoleBinding
that binds that ServiceAccount to the shared ClusterRole. A MultiRunner shares a
single ServiceAccount across all its entries. Because the rules live in one
ClusterRole, a permission change in a new operator version applies to every
runner at once.

The operator can only grant a runner what the operator itself holds (it has no
RBAC `escalate` verb), so the manager ClusterRole is the explicit ceiling for
runner permissions. RoleBindings to the ClusterRole are namespaced, so the
effective grant is confined to the build namespace; nothing cluster-scoped is
granted to a runner.

Job pods run in `executor_config.namespace` when set, otherwise in the runner's
own namespace. When that namespace differs from the runner's, the operator
creates the RoleBinding there too (the ServiceAccount stays in the runner
namespace) and removes it when the runner is deleted. Because the operator
pre-provisions RBAC for a known namespace, `namespace_per_job` and
`namespace_overwrite_allowed` are rejected at admission: both make the build
namespace dynamic, which would require cluster-scoped RBAC.

## License

Apache License 2.0. See [LICENSE](LICENSE).
