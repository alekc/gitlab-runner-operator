---
description: >-
  What RBAC the operator grants each runner, and how to let job pods run in a
  namespace other than the runner's own.
---

# RBAC and namespaces

## What every runner gets

The permissions every kubernetes executor needs (pods and pods/exec,
pods/attach, pods/log, services, secrets, configmaps, serviceaccounts and
events) live in one shared ClusterRole, `gitlab-runner-operator-executor`,
reconciled by the operator.

For each `Runner` or `MultiRunner` the operator then provisions:

- its own ServiceAccount, so each runner is a distinct identity for audit and
  revocation;
- a RoleBinding tying that ServiceAccount to the shared ClusterRole.

A `MultiRunner` shares a single ServiceAccount across all of its entries.
Because the rules live in one ClusterRole, a permission change in a new
operator version applies to every runner at once.

## Optional grants

Permissions only some runners need are not in that role. Each optional grant
has its own ClusterRole and its own RoleBinding, created only where a spec asks
for it, so enabling one does not hand out the others.

Today there is one: `pod_disruption_budget` needs
`policy/poddisruptionbudgets`, held in
`gitlab-runner-operator-executor-pdb` and bound by a RoleBinding named
`pdb-<child-name>`. The grant is per build namespace, so a `MultiRunner` that
sets the flag on one entry does not widen it into the namespaces its other
entries target. Turning the flag off deletes the binding on the next reconcile.

!!! warning "Upgrading from a version that granted this unconditionally"

    Upgrading from an operator version that granted `poddisruptionbudgets` to
    everything revokes it fleet-wide as soon as the first runner reconciles.
    Each runner that still wants it regains it on its own next reconcile. A job
    starting in that gap fails with `poddisruptionbudgets is forbidden`.

## The ceiling

The operator can only grant a runner what the operator itself holds: it has no
RBAC `escalate` verb. So the manager ClusterRole is the explicit ceiling for
runner permissions.

RoleBindings to the ClusterRole are namespaced, so the effective grant is
confined to the build namespace. Nothing cluster-scoped is ever granted to a
runner.

## Where jobs run

Job pods run in `executor_config.namespace` when it is set, otherwise in the
runner's own namespace.

By default a runner may only target its **own** namespace. A Runner author
picking an arbitrary namespace would otherwise have the operator bind their
ServiceAccount, and run their jobs, in somewhere like `kube-system`.

To permit specific build namespaces, start the operator with:

```text
--allowed-build-namespaces=ns-a,ns-b
```

or `--allowed-build-namespaces=*` to allow any. With the Helm chart, set
`allowedBuildNamespaces`.

The reconciler refuses any other `executor_config.namespace`: the runner goes
NotReady with an error, no RBAC is provisioned, and any binding previously
created for a now-disallowed namespace is revoked. When an allowed build
namespace differs from the runner's, the operator creates the RoleBinding there
too, while the ServiceAccount stays in the runner's namespace, and removes it
when the runner is deleted.

!!! danger "Keep the allow-list tight"

    The allow-list is enforced by the reconciler, the component that actually
    provisions the RBAC, so it cannot be turned off by a flag. The operator
    holds the executor permission set cluster-wide, so on a shared cluster an
    unrestricted namespace is a privilege-escalation path: a Runner author
    could reach `kube-system` or another tenant's namespace.

    Restrict who can create `Runner` and `MultiRunner` objects by RBAC as
    well. Creating one of these is, in effect, requesting pod-create rights in
    the target namespace.

## Rejected by design

`namespace_per_job` and `namespace_overwrite_allowed` are rejected at admission
by the CRD schema. Both make the build namespace dynamic, which would require
cluster-scoped RBAC for runners, and the operator pre-provisions RBAC for a
namespace it knows in advance. See [limitations](../reference/limitations.md).
