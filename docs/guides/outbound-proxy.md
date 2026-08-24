---
description: >-
  Give the runner manager container HTTP_PROXY, HTTPS_PROXY and NO_PROXY with
  runner_env so it can reach GitLab through an outbound proxy, and the
  secretKeyRef and fieldRef gotchas that come with it.
---

# Outbound proxy

## The short version

A runner behind an egress proxy never registers: the object sits `NotReady`
with a connection error that looks like a network fault rather than a missing
setting. `spec.environment` does not help, it only reaches the **job** build
environment via `config.toml`. Set `runner_env` instead, which reaches the
manager process itself, the one that actually talks to GitLab:

```yaml
apiVersion: gitlab.k8s.alekc.dev/v1beta2
kind: Runner
metadata:
  name: runner-sample
spec:
  authentication:
    token:
      secret_key_ref:
        name: gitlab-runner-token
  runner_env:
    - name: HTTP_PROXY
      value: http://proxy.internal:3128
    - name: HTTPS_PROXY
      value: http://proxy.internal:3128
    - name: NO_PROXY
      value: ".svc,.cluster.local,10.0.0.0/8"
```

On a `MultiRunner`, `runner_env` is set once on the spec and shared by every
entry, same as `runner_resources` and `runner_security_context`: there is one
manager pod, so there is one process environment.

## Credentials belong in a Secret

A proxy URL with embedded basic-auth credentials, or a bearer token some
proxies want in an env var, should not go in a plain `value`. The operator's
reconcile logs the full manager pod shape, `runner_env` included, whenever it
hits the settle-and-log path described below, and a plain `value` renders
verbatim into that log line. Reference a Secret instead:

```yaml
  runner_env:
    - name: HTTPS_PROXY
      valueFrom:
        secretKeyRef:
          name: proxy-credentials
          key: url
```

## Gotchas

**A `fieldRef` needs an explicit `apiVersion`.** The Kubernetes apiserver
defaults `valueFrom.fieldRef.apiVersion` to `v1` the moment the manager pod is
written, but the operator sends it empty when you omit it. The two never
match, so every reconcile takes the "cluster did not store the spec as sent"
path and logs an INFO line, forever. Harmless (no pod roll, no requeue loop),
but noisy. Set `apiVersion: v1` explicitly on any `fieldRef` under `runner_env`
to avoid it.

**Distinct from `spec.environment`.** `spec.environment` renders into
`config.toml`'s `[[runners]]` block, reaching every **job** the runner picks
up. `runner_env` sets Kubernetes env vars on the **manager container** only,
the process that registers with GitLab and dispatches jobs, not the jobs
themselves. Getting these two confused is the single most common reason this
guide exists: a proxy variable in `spec.environment` builds jobs behind the
proxy without ever unblocking manager registration.

## Related

- [Limitations](../reference/limitations.md): the rest of the manager pod's
  configurable set (resources, security context, placement).
