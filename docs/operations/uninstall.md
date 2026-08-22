# Uninstalling

## Order matters

Delete your `Runner` and `MultiRunner` objects first, and wait for them to
actually go, before removing the operator.

They carry a finalizer that deregisters them from GitLab and prunes the
RoleBindings the operator created in other namespaces. Only a running operator
can complete it. Take the operator away first and the objects wedge in
`Terminating`.

```bash
kubectl delete runners --all -A
kubectl delete multirunners --all -A
kubectl get runners,multirunners -A     # confirm they are gone
helm uninstall gitlab-runner-operator
```

!!! tip "Already wedged?"

    If you removed the operator first, reinstall it, let it reconcile the
    pending deletions, then uninstall in the right order. Stripping the
    finalizer by hand works too, but then nothing deregisters the runner from
    GitLab or cleans up cross-namespace RoleBindings.

## What gets left behind

### Executor ClusterRoles

The executor ClusterRoles are created by the operator at runtime rather than by
the install manifest, so neither `helm uninstall` nor
`kubectl delete -k config/default` removes them. They carry no ownerReferences
either, since a cluster-scoped role cannot be owned by a namespaced runner and
is shared between them regardless, so they outlive the last runner as well as
the operator.

Unbound they grant nothing, so this is clutter rather than a security problem:

```bash
kubectl get clusterrole -l app.kubernetes.io/managed-by=gitlab-runner-operator
kubectl delete clusterrole -l app.kubernetes.io/managed-by=gitlab-runner-operator
```

Run the delete **last**: while a runner still exists, the next reconcile
recreates them.

Today the selector matches `gitlab-runner-operator-executor` and
`gitlab-runner-operator-executor-pdb`, and each future optional grant adds one.
It matches only what the operator created: the chart's own ClusterRoles are
labelled `app.kubernetes.io/managed-by: Helm`, and the kustomize ones carry no
labels at all.

### CRDs

`helm uninstall` leaves the CRDs behind, because Helm never removes anything
installed from a chart's `crds/` directory. Remove them by hand if you want
them gone:

```bash
kubectl delete crd runners.gitlab.k8s.alekc.dev multirunners.gitlab.k8s.alekc.dev
```

Deleting the CRDs deletes every object of those kinds, which fires the
finalizers with no operator around to complete them. Do this after the operator
is gone and the objects are already deleted, not as a shortcut instead of
deleting them.

### Runners in GitLab

Bring-your-own-token runners are never deregistered by the operator: it never
knew them as anything but a token. Managed runners are deregistered by the
finalizer, unless the fallback path also failed, in which case the operator
logged the runner as possibly orphaned. Check the GitLab runner list afterwards
and clean up anything stale.
