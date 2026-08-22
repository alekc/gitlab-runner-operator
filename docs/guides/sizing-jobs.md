---
description: >-
  Set CPU and memory for GitLab CI build, helper and service containers, fix
  exit code 137 OOM kills, and let jobs tune their own limits within a ceiling.
---

# Sizing jobs

## The short version

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
    # The build container: your script runs here.
    cpu_request: "500m"
    cpu_limit: "2"
    memory_request: "1Gi"
    memory_limit: "4Gi"

    # The helper: clone, artifacts, cache. The 256Mi-ish default is not enough
    # for a large artifact upload.
    helper_cpu_request: "100m"
    helper_memory_request: "256Mi"
    helper_memory_limit: "512Mi"

    # Service containers, per service. A database service needs a real budget.
    service_cpu_request: "100m"
    service_memory_request: "256Mi"
    service_memory_limit: "1Gi"
```

## Three containers, three budgets

Every job pod has a build container, a helper container, and one container per
service. They have separate settings, and the common mistake is sizing only the
first. A job that dies during `uploading artifacts` is usually the helper being
killed, not your script.

## Letting jobs tune themselves

Hard-coding one limit for every job on a runner means sizing for the worst case
and wasting it on the rest. The `*_overwrite_max_allowed` keys set a ceiling and
let the job ask for what it needs:

```yaml
spec:
  executor_config:
    memory_request: "1Gi"
    memory_limit: "2Gi"
    memory_limit_overwrite_max_allowed: "8Gi"
    cpu_limit_overwrite_max_allowed: "4"
```

A job then asks with variables:

```yaml
integration-tests:
  variables:
    KUBERNETES_MEMORY_LIMIT: "6Gi"
    KUBERNETES_CPU_LIMIT: "3"
```

Anything above the ceiling is rejected rather than silently clamped. Leave the
`_overwrite_max_allowed` keys unset and CI cannot change its limits at all,
which is the right default on a shared cluster.

## Gotchas

**Exit code 137 is an OOM kill.** Not a script bug. The container hit its memory
limit and was killed. Raise `memory_limit`, or find out what allocated.

**A JVM or Node process can be OOM-killed while apparently under the limit.**
There is a known pattern where a process allocating a large block at once is
killed even with headroom left, because the cgroup reacts to the allocation rate.
If a build dies at exactly the same step every time with memory to spare, cap the
runtime's heap explicitly (`-Xmx`, `--max-old-space-size`) rather than raising the
container limit further.

**On cgroup v2 an OOM-killed pod can hang instead of failing.** The runner does
not always notice, and the job sits until its timeout. If you see jobs burning
their full timeout with no output, check whether the pod was OOM-killed.

**Requests are what the scheduler sees.** A limit with no request means the pod
is scheduled as if it needs nothing, and the node gets over-packed. Always set
both.

**CPU limits throttle, they do not kill.** A build that got mysteriously slower
after you added `cpu_limit` is being throttled. Requests guarantee, limits cap.

## Related

- [Persistent and ephemeral storage](storage.md), the other resource that gets
  jobs killed
- [Jobs stuck in Pending](stuck-pods.md), which is what happens when requests
  exceed what any node has free
