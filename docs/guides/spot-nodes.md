---
description: >-
  Run GitLab CI jobs on spot or preemptible nodes: pod disruption budgets, why
  evicted jobs report script_failure, and how to make retry actually fire.
---

# Spot and preemptible nodes

Jobs are the ideal spot workload: interruptible, retryable, and the majority of
a CI bill. The catch is that GitLab does not classify an evicted job the way you
would expect, so naive retry rules do not fire.

## The short version

```yaml
apiVersion: gitlab.k8s.alekc.dev/v1beta2
kind: Runner
metadata:
  name: runner-spot
spec:
  authentication:
    token:
      secret_key_ref:
        name: gitlab-runner-token
  executor_config:
    node_selector:
      node-lifecycle: spot
    node_tolerations:
      "node-lifecycle=spot": "NoSchedule"
    # Ask Kubernetes not to evict a running job during a voluntary disruption
    # such as a drain. Read the gotcha below before enabling this.
    pod_disruption_budget: true
    # Retry the Kubernetes API calls that fail while a node is going away.
    retry_limit: 5
    retry_backoff_max: 5000
```

And in `.gitlab-ci.yml`, because the runner cannot do this part for you:

```yaml
default:
  retry:
    max: 2
    when:
      - runner_system_failure
      - stuck_or_timeout_failure
      - script_failure
```

## Why `script_failure` has to be in that list

When a spot node is reclaimed, the job pod disappears mid-script. The runner
reports that as `script_failure`, not `runner_system_failure`, because from its
point of view the script stopped returning. This is a long-standing upstream
complaint: the retry reasons that exist for infrastructure faults do not match
what an eviction produces.

The consequence is that a `retry: when: [runner_system_failure]` rule, which is
the intuitive thing to write, never fires on spot evictions. You have to include
`script_failure`, which also retries genuine test failures. That is the trade,
and it is worth knowing before you enable spot rather than after.

## Gotchas

**A PodDisruptionBudget can wedge a node drain.** `pod_disruption_budget: true`
tells Kubernetes to protect a running job, which is what you want for a
voluntary drain, and it means the drain waits for the job. A long job can block a
node upgrade for its full duration. It does not help with an involuntary
disruption at all: a reclaimed spot instance goes away regardless.

**It needs a permission that is granted per build namespace.** The operator binds
`policy/poddisruptionbudgets` through a separate ClusterRole only where a spec
asks for it. Enabling the flag on a `MultiRunner` entry does not widen it into
the namespaces the other entries use. See
[RBAC and namespaces](../operations/rbac-and-namespaces.md).

**Upgrading from an older operator revokes it fleet-wide for a moment.** A
version that granted PDB permissions unconditionally has them removed as soon as
the first runner reconciles, and each runner that wants them gets them back on
its own next reconcile. A job starting in that window fails with
`poddisruptionbudgets is forbidden`.

**Put the manager on stable capacity.** The manager pod tracks in-flight jobs; if
it is evicted, those jobs are lost even though the job pods were fine. Today the
manager pod has no placement controls, so you cannot pin it off spot
([#83](https://github.com/alekc/gitlab-runner-operator/issues/83)), and it also
cannot drain gracefully on a rollout
([#84](https://github.com/alekc/gitlab-runner-operator/issues/84)). Until those
land, run the manager on a cluster or node pool that is not preemptible.

## Related

- [Node placement](node-placement.md)
- [Jobs stuck in Pending](stuck-pods.md), for the scale-up delay spot pools cause
