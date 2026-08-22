---
description: >-
  Kubernetes operator for GitLab CI runners: typed CRDs instead of config.toml,
  with registration, config, RBAC and cleanup handled for you.
---

# GitLab Runner Operator

A Kubernetes operator that manages GitLab CI runners using the
[kubernetes executor](https://docs.gitlab.com/runner/executors/kubernetes.html).

It lets you run one or many runners, each configured in YAML instead of a
hand-written `config.toml`. The supported kubernetes executor options are
reachable through the CRD, and the operator handles registration, config
rendering, per-runner RBAC, and deregistration from GitLab on delete. What is
not covered is listed in [limitations](reference/limitations.md).

!!! warning "Alpha"

    Breaking changes are possible and are called out in the release notes. The
    current API group version is `gitlab.k8s.alekc.dev/v1beta2`, which replaced
    `v1beta1` and `v1alpha1` without automatic conversion. See
    [upgrading from v1beta1](install.md#upgrading-from-v1beta1).

## Quickstart

Install the operator:

```bash
helm repo add alekc https://charts.alekc.dev/
helm repo update
helm install gitlab-runner-operator alekc/gitlab-runner-operator
```

Create a runner in the GitLab UI, copy its `glrt-` token, and hand it to a
`Runner` object:

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

Then watch it come up:

```bash
kubectl get runners
kubectl describe runner runner-sample
```

`Ready` turns true once the config Secret, ServiceAccount, RoleBinding and
runner Deployment are all in place. If it stays false, `status.error` and the
`Ready` condition carry the reason.

Putting a token inline is fine for a first try. For anything real, read it from
a Secret instead: see [authentication](authentication.md#token-sources).

## Where to go next

| If you want to | Read |
| --- | --- |
| Solve a specific problem, or look up an error message | [Guides](guides/index.md) |
| Install with kustomize, or pin versions | [Install](install.md) |
| Let the operator create the runner in GitLab for you | [Authentication](authentication.md) |
| Run jobs in a namespace other than the runner's | [RBAC and namespaces](operations/rbac-and-namespaces.md) |
| Look up a specific field | [CRD API reference](reference/api.md) |
| Know what the operator deliberately does not do | [Limitations](reference/limitations.md) |
| Remove it cleanly | [Uninstalling](operations/uninstall.md) |
| Work on the operator itself | [Contributing](contributing.md) |
