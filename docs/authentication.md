# Authentication

GitLab deprecated the registration-token workflow in 16.0 and disabled it by
default from 18.0, so this operator authenticates with runner authentication
tokens (the `glrt-` ones). There are two modes, both set under
`spec.authentication`.

The CRD schema rejects an object that configures both modes or neither, so this
is not something you can get half right.

## Bring your own token

Create the runner in GitLab yourself, in the UI or through
`POST /user/runners`, and give the operator the resulting `glrt-` token. The
operator makes no GitLab API calls at all: it writes the token into the runner
config and leaves the runner's existence in GitLab entirely to you.

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

Deleting this object does not remove the runner from GitLab. Clean it up
yourself, or GitLab keeps showing a runner that no longer exists.

## Operator-managed

Give the operator an access token (personal, group or project) holding the
`create_runner` scope, plus a `create_options` block describing the runner you
want. The operator calls `POST /user/runners`, stores the returned token, and
removes the runner from GitLab when the object is deleted.

```yaml
apiVersion: gitlab.k8s.alekc.dev/v1beta2
kind: Runner
metadata:
  name: runner-managed
spec:
  authentication:
    access_token:
      secret_key_ref:
        name: gitlab-access-token
    create_options:
      runner_type: project_type
      project_id: 1234567
      run_untagged: true
      tag_list:
        - test-gitlab-runner
```

### Scopes needed

`create_runner` alone is enough for the normal lifecycle. Deletion first tries
the runner's own authentication token (`DELETE /runners` by token), which needs
no access-token scope at all.

If that call fails, or the token is unavailable, the operator falls back to
deleting by runner id (`DELETE /runners/:id`), which only succeeds when the
access token **also** holds `api`. Without it the runner is logged as possibly
orphaned and left in GitLab.

### create_options

| Key | Notes |
| --- | --- |
| `runner_type` | Required. One of `instance_type`, `group_type`, `project_type`. |
| `group_id` | Required when `runner_type` is `group_type`. |
| `project_id` | Required when `runner_type` is `project_type`. |
| `description` | Shown in the GitLab runner list. |
| `tag_list` | Job tags this runner picks up. |
| `run_untagged` | Whether it also takes untagged jobs. |
| `locked` | Prevents the runner being assigned to other projects. |
| `paused` | Registers the runner but does not let it take jobs. |
| `access_level` | `not_protected` or `ref_protected`. |
| `maximum_timeout` | Per-job timeout ceiling, in seconds. |
| `maintenance_note` | Free-text note stored on the runner in GitLab. |

The last two CEL rules on the schema enforce the `group_id` and `project_id`
pairings, so a `project_type` runner with no `project_id` is rejected at
admission rather than failing at reconcile.

### When a managed runner gets recreated

A recreate is a delete in GitLab followed by a create, so the runner ends up
with a new id and a new token. Four things trigger it:

| Trigger | Detail |
| --- | --- |
| `create_options` changed | The operator stores a hash of the block in `status.registration_hash` and compares on every reconcile. Editing `tag_list` is the usual way people meet this. |
| Token within 24h of expiry | GitLab has no reset-by-token endpoint and resetting by id needs the `api` scope, so the operator recreates instead. That keeps `create_runner` sufficient. |
| Token missing from the config Secret | Without it the operator can neither verify nor delete by token, so it recreates and treats the old runner as possibly orphaned. |
| Token rejected by GitLab | Checked with a verify call each reconcile. A runner deleted in the GitLab UI lands here. |

!!! warning

    Recreating is not free. The old runner is deleted in GitLab, so anything
    still running on it goes away with it. Treat an edit to `create_options` on
    a busy runner as a restart, not a config tweak.

## Token sources

Both `token` and `access_token` are token sources with two mutually exclusive
ways to supply the value.

| Key | Notes |
| --- | --- |
| `value` | The literal token, inline in the spec. Convenient for testing; it puts the credential in the object where anyone with read access to the CR can see it. |
| `secret_key_ref` | Read the token from a Secret in the **runner's** namespace. `name` is required, `key` defaults to `token`, and `optional: true` lets a missing secret or key resolve to an empty token instead of failing the reconcile. |

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
        # key omitted, so it reads "token"
```

`optional: true` exists for the bootstrap case where the Secret arrives after
the Runner (an External Secrets sync, a sealed secret controller). Be aware
that it turns a missing credential into an empty one, so the runner comes up
and fails to authenticate rather than telling you the Secret is not there.

## What ends up where

The resolved token is written into the runner's config Secret, not into the
object. `status` carries the non-secret parts of the registration:

| Field | Meaning |
| --- | --- |
| `runner_id` | GitLab's numeric id, for managed runners. Zero in bring-your-own mode. |
| `token_expires_at` | GitLab's expiry for a managed runner token, when it sets one. |
| `registration_hash` | Digest of the create options that produced the current runner. |

## Custom CA for a self-signed GitLab

If your GitLab presents a certificate from a private CA, registration fails
with `x509: certificate signed by unknown authority`. Set `spec.caCertificate`
to a PEM bundle, inline or from a Secret or ConfigMap key, and the operator
uses it both for its own API calls and for the runner's connection.

```yaml
spec:
  gitlab_instance_url: https://gitlab.internal.example.com/
  caCertificate:
    configMapKeyRef:
      name: gitlab-ca
      # key defaults to ca.crt
```

Set at most one of `value`, `secretKeyRef` or `configMapKeyRef`; the schema
rejects more than one. The referenced object must live in the runner's
namespace. Full field list in the
[API reference](reference/api.md#casource).
