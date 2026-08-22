---
description: >-
  Pin GitLab CI job pods to a node pool or CPU architecture with node_selector,
  node_tolerations and affinity, including arm64 and mixed-arch fleets.
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

## Gotchas

**`exec format error`, or the helper crashing immediately.** The helper image
defaults to an amd64 build. Land it on an arm64 node and it cannot execute. That
is what `helper_image_autoset_arch_and_os` is for. Set it on every runner that
can schedule onto more than one architecture, not just the arm64 one.

**`node_tolerations` is a map, not a list.** The key is `"key=value"` and the
value is the effect: `"dedicated=ci": "NoSchedule"`. A toleration for a taint
with no value is written `"key=": "NoSchedule"`.

**The manager pod cannot be placed.** Everything on this page applies to **job**
pods. The runner manager pod has no `nodeSelector`, `tolerations` or `affinity`
yet, so on a fully tainted CI pool the manager schedules elsewhere, and on a
mixed-arch cluster you cannot steer it onto an architecture matching its own
image. Tracked in
[#83](https://github.com/alekc/gitlab-runner-operator/issues/83).

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
