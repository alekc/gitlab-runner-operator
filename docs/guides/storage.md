---
description: >-
  Give GitLab CI jobs disk: emptyDir with a size limit, PVCs, CSI and NFS
  volumes, and ephemeral storage requests that stop pods being evicted.
---

# Persistent and ephemeral storage

## The short version

Give jobs a bigger scratch area and stop the node evicting them:

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
  executor_config:
    # Counted against the node's ephemeral storage. Without a request the
    # scheduler assumes zero and packs the node until the kubelet evicts.
    ephemeral_storage_request: "4Gi"
    ephemeral_storage_limit: "16Gi"
    volumes:
      empty_dir:
        - name: scratch
          mount_path: /scratch
          size_limit: "8Gi"
```

## Volume types available

| Type | Use it for |
| --- | --- |
| `empty_dir` | Scratch space for one job. `medium: Memory` makes it a tmpfs, which is fast and counts against the memory limit. `size_limit` caps it. |
| `pvc` | An existing PersistentVolumeClaim, shared by every job on this runner. Needs `ReadWriteMany` if jobs can overlap. |
| `csi` | A CSI driver volume with `volume_attributes`, for secrets-store drivers and the like. |
| `nfs` | An NFS export, by `server` and `path`. |
| `host_path` | A path on the node. Avoid: it ties jobs to nodes and is a container escape if writable. |

## Why ephemeral storage requests matter

Job pods write a lot: the clone, dependencies, build output, artifacts before
upload, and the log itself. The kubelet measures that usage whether or not you
requested any, and evicts pods when the node runs short. What a request changes
is the ranking: a pod over its request is evicted before one within it, so the
job with no request is the first to go. In practice this shows up as jobs dying
part way through an artifact upload, on a node that looks healthy.

Setting a request also stops the scheduler over-packing the node in the first
place, which is the actual fix rather than the mitigation.

## Sharing a cache directory between jobs

You cannot, safely, with a PVC. Two jobs on the same runner can run
concurrently, and both will write to the same paths. That is a corrupted cache
rather than a shared one.

A distributed cache is the mechanism for this, and it is not implemented in this
operator yet: there is no `[runners.cache]` support and no raw `config.toml`
passthrough
([#81](https://github.com/alekc/gitlab-runner-operator/issues/81)). Until that
lands, treat every job as starting from nothing and lean on registry-side caching
for image layers.

## Gotchas

**`empty_dir` with `medium: Memory` eats your memory limit.** A tmpfs is charged
to the container's memory cgroup. A 8Gi tmpfs under a 4Gi memory limit is an
OOM kill waiting for a big enough artifact.

**`size_limit` is not a quota you can rely on for isolation.** It bounds the
volume, but the pod can still exhaust the node through other paths if the
ephemeral storage limit is unset.

**`volumes.csi` and persistence.** There is a known upstream bug where CSI
volumes are created as ephemeral rather than persistent volumes. If you are
relying on data surviving the job, verify it does before building a workflow on
it.

**Artifacts and logs are separate budgets.** A job producing an enormous log can
fail on the log size cap rather than on disk. That cap (`output_limit`) is not
exposed by this operator, so it stays at gitlab-runner's default: see
[limitations](../reference/limitations.md).

## Related

- [Sizing jobs](sizing-jobs.md), for CPU and memory
- [Mount secrets and configmaps](mount-secrets.md), for credentials rather than
  data
- [Limitations](../reference/limitations.md), for the cache gap
