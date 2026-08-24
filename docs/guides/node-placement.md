---
description: >-
  Pin GitLab CI job pods and the runner manager pod to a node pool or CPU
  architecture with node_selector, tolerations and affinity, including arm64
  and mixed-arch fleets.
---

# Node placement and architecture

## The short version

Put jobs on a tainted arm64 CI pool:

```yaml
apiVersion: gitlab.k8s.alekc.dev/v1beta2
kind: Runner
metadata:
  name: runner-arm64
spec:
  authentication:
    token:
      secret_key_ref:
        name: gitlab-runner-token
  executor_config:
    node_selector:
      kubernetes.io/arch: arm64
      node-pool: ci
    node_tolerations:
      # Keyed "key=value", valued with the effect.
      "dedicated=ci": "NoSchedule"
    # Pick the helper image matching the node's arch and OS instead of
    # defaulting to amd64.
    helper_image_autoset_arch_and_os: true
```

## Mixed-arch fleets

One `MultiRunner` per GitLab instance, one entry per architecture, tags telling
jobs where to go. All entries share a single manager pod, so this costs one pod
rather than two.

```yaml
apiVersion: gitlab.k8s.alekc.dev/v1beta2
kind: MultiRunner
metadata:
  name: builders
spec:
  concurrent: 8
  entries:
    - name: amd64
      authentication:
        access_token:
          secret_key_ref:
            name: gitlab-pat
        create_options:
          runner_type: project_type
          project_id: 1234567
          tag_list: [amd64]
      executor_config:
        node_selector:
          kubernetes.io/arch: amd64
        helper_image_autoset_arch_and_os: true
    - name: arm64
      authentication:
        access_token:
          secret_key_ref:
            name: gitlab-pat
        create_options:
          runner_type: project_type
          project_id: 1234567
          tag_list: [arm64]
      executor_config:
        node_selector:
          kubernetes.io/arch: arm64
        helper_image_autoset_arch_and_os: true
```

Jobs then choose with `tags: [arm64]`.

## Placing the manager pod

Everything above places **job** pods. The manager pod that runs gitlab-runner
itself is placed by four spec-level fields, on both `Runner` and `MultiRunner`:

```yaml
apiVersion: gitlab.k8s.alekc.dev/v1beta2
kind: Runner
metadata:
  name: runner-ci-pool
spec:
  authentication:
    token:
      secret_key_ref:
        name: gitlab-runner-token
  # Where the manager runs.
  runner_node_selector:
    node-pool: ci
  runner_tolerations:
    - key: dedicated
      operator: Equal
      value: ci
      effect: NoSchedule
  # A class you create; see the gotcha below on why not a built-in one.
  runner_priority_class_name: runner-manager
  # Where the jobs run. Separate setting, separate shape.
  executor_config:
    node_selector:
      node-pool: ci
    node_tolerations:
      "dedicated=ci": "NoSchedule"
```

A `MultiRunner` serves every entry from one manager pod, so these are spec
level there too, never per entry.

`runner_affinity` covers what a selector cannot express:

```yaml
spec:
  # Keep the manager off spot capacity: an evicted manager loses the jobs it
  # was tracking, even when the job pods themselves were fine.
  runner_affinity:
    nodeAffinity:
      requiredDuringSchedulingIgnoredDuringExecution:
        nodeSelectorTerms:
          - matchExpressions:
              - key: node-lifecycle
                operator: NotIn
                values: [spot]
```

Changing any of these rolls the manager pod, which drops the jobs it is
tracking. Set them when you create the runner, or expect a restart. That
includes edits Kubernetes would treat as identical: the operator compares the
lists as written, so reordering two tolerations or two `matchExpressions` rolls
the pod even though it changes nothing. Reformat when the queue is quiet.

## Gotchas

**`exec format error`, or the helper crashing immediately.** The helper image
defaults to an amd64 build. Land it on an arm64 node and it cannot execute. That
is what `helper_image_autoset_arch_and_os` is for. Set it on every runner that
can schedule onto more than one architecture, not just the arm64 one.

**`node_tolerations` is a map, not a list.** The key is `"key=value"` and the
value is the effect: `"dedicated=ci": "NoSchedule"`. A toleration for a taint
with no value is written `"key=": "NoSchedule"`.

**Use your own PriorityClass, not `system-cluster-critical`.** The two built-in
classes sit at the top of the range (`system-cluster-critical` is 2000000000), so
a manager using one can preempt the control-plane and CNI pods it depends on. The
apiserver does accept it outside `kube-system`, which is what makes it an easy
mistake. Create a class above your normal workloads and well below the system
ones, and point `runner_priority_class_name` at that:

```yaml
apiVersion: scheduling.k8s.io/v1
kind: PriorityClass
metadata:
  name: runner-manager
value: 100000
description: "GitLab runner managers. Above CI jobs, below cluster components."
```

Nothing needs a priority class to work. Set one when the manager competes for
capacity with workloads that could otherwise evict it.

**An unplaceable manager still reports `Ready`.** The runner status goes
`Ready=true` once the operator has written the Deployment; it does not wait for
the pod. So a `runner_node_selector` matching no node, a taint you did not
tolerate, or a `runner_priority_class_name` that does not exist all leave the
Runner looking healthy with no manager pod running and no jobs being picked up.
Check the pod, not the CR, after changing placement:

```bash
kubectl get pods -l deployment=<runner-name> -n <namespace>
kubectl describe rs -l deployment=<runner-name> -n <namespace>
```

A missing PriorityClass is rejected when the ReplicaSet tries to create the
pod, so the reason is on the ReplicaSet rather than anywhere on the Runner.

**The manager fields and the executor fields take different shapes.** They
configure the same Kubernetes concepts, but `runner_*` is passed to Kubernetes
directly while `executor_config` is passed to gitlab-runner, which has its own
config format. So:

| Concept | Manager (`runner_*`) | Jobs (`executor_config`) |
| --- | --- | --- |
| Tolerations | a list of `{key, operator, value, effect}` | a map of `"key=value": "effect"` |
| Affinity | `nodeAffinity`, `requiredDuringScheduling...` | `node_affinity`, `required_during_scheduling...` |
| Node selector | a map of labels, same either way | a map of labels, same either way |

Copying an affinity block from one to the other does not fail, it is **silently
dropped**: the CRD prunes keys it does not recognise, so the field reads back
empty and the pod schedules as if you had set nothing. Check with
`kubectl get runner <name> -o yaml` after applying.

**Per-job overrides are off by default.** `KUBERNETES_NODE_SELECTOR_*` and
`KUBERNETES_NODE_TOLERATIONS_*` in a job only take effect if the runner sets
`node_selector_overwrite_allowed` / `node_tolerations_overwrite_allowed` to a
regex that matches. Leaving them unset means CI cannot move its own jobs, which
is usually what you want on a shared cluster.

**Affinity is available but verbose.** Use `node_selector` for equality matching.
Reach for `affinity.node_affinity` only when you need `In` / `NotIn` / `Exists`
or a soft preference. The full shape is in the
[API reference](../reference/api.md#kubernetesaffinity).

## Related

- [Spot and preemptible nodes](spot-nodes.md), for placing jobs on cheap capacity
- [Jobs stuck in Pending](stuck-pods.md), the usual symptom of a selector that
  matches no node
