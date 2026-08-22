---
description: >-
  Run GitLab CI job pods in a dedicated namespace: the operator allow-list flag,
  Pod Security Admission labels, and the NetworkPolicy jobs still need.
---

# Dedicated build namespace

Keeping job pods out of the namespace the runner lives in is worth doing: jobs
are arbitrary code, and a separate namespace gives you a quota boundary, a
NetworkPolicy boundary, and a Pod Security Admission level of their own.

## The short version

Allow the namespace on the operator first, or the runner will refuse it:

```yaml
# Helm values
allowedBuildNamespaces:
  - gitlab-ci-builds
```

Then point the runner at it:

```yaml
apiVersion: gitlab.k8s.alekc.dev/v1beta2
kind: Runner
metadata:
  name: runner-sample
  namespace: gitlab-runners
spec:
  authentication:
    token:
      secret_key_ref:
        name: gitlab-runner-token
  executor_config:
    namespace: gitlab-ci-builds
```

The operator creates the RoleBinding in `gitlab-ci-builds` while the
ServiceAccount stays in `gitlab-runners`, and removes that binding when the
runner is deleted.

## Why the allow-list exists

Without it, whoever can create a `Runner` could name any namespace and have the
operator bind a ServiceAccount, and run pods, there. `kube-system` included.
The reconciler enforces the list, so it cannot be bypassed by a flag on the
object. A namespace that is not allowed leaves the runner NotReady with an error
and provisions no RBAC at all.

`allowedBuildNamespaces: ["*"]` allows any, and is a reasonable choice on a
single-tenant cluster and a poor one anywhere else. Read
[RBAC and namespaces](../operations/rbac-and-namespaces.md) before choosing.

## What else the namespace needs

**Secrets move with the jobs.** Everything the job pod reads is read from *this*
namespace: image pull secrets, mounted Secrets and ConfigMaps, the service
account. Moving the build namespace and leaving the Secrets behind is the single
most common way this setup breaks.

**Pod Security Admission.** Label it for the strictest level your jobs can live
with:

```bash
kubectl label ns gitlab-ci-builds \
  pod-security.kubernetes.io/enforce=baseline \
  pod-security.kubernetes.io/warn=restricted
```

`restricted` forbids privileged pods, so [DinD](docker-in-docker.md) will not run
under it. That is the trade: `baseline` if you need DinD, `restricted` with
[rootless builds](rootless-builds.md) if you do not.

**A ResourceQuota.** A runaway pipeline can otherwise consume the cluster. Size it
above `concurrent` multiplied by your per-job requests, or jobs fail to schedule
for reasons that look nothing like a quota.

**A NetworkPolicy that still lets jobs out.** A default-deny namespace breaks CI
in a confusing way, because the job pod needs to reach GitLab, your registry, and
usually the public internet for dependencies. Allow egress to those explicitly
and remember DNS.

## Gotchas

**`namespace_per_job` and `namespace_overwrite_allowed` are rejected.** The CRD
refuses both at admission. RBAC is provisioned ahead of time for a namespace the
operator knows about, and a dynamic namespace would mean granting runners
cluster-scoped permissions. See
[limitations](../reference/limitations.md).

**Turning the allow-list off later revokes bindings.** Remove a namespace from
the flag and the operator prunes the RoleBinding it created there on the next
reconcile. Jobs starting after that fail with a forbidden error.

**One namespace per trust level, not per project.** Every runner sharing a build
namespace shares its blast radius. Splitting by team or trust level is worth it;
splitting per project multiplies the allow-list without buying much.

## Related

- [RBAC and namespaces](../operations/rbac-and-namespaces.md)
- [Mount secrets and configmaps](mount-secrets.md)
- [Pull from a private registry](private-registry.md)
