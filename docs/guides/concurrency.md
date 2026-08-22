---
description: >-
  How concurrent, the hardcoded per-entry limit of 10, and request_concurrency
  interact on this operator, and why a runner tops out at ten jobs.
---

# Concurrency

## The short version

```yaml
apiVersion: gitlab.k8s.alekc.dev/v1beta2
kind: Runner
metadata:
  name: runner-sample
spec:
  # Ceiling on job pods this manager will have in flight. Anything above 10
  # has no effect: see below.
  concurrent: 10
  # How often it asks GitLab for work, in seconds. Minimum 3.
  check_interval: 3
  authentication:
    token:
      secret_key_ref:
        name: gitlab-runner-token
```

## A single Runner tops out at 10 jobs

Three settings govern this in gitlab-runner, and only one of them is yours to
set on this operator:

| Setting | Scope | On this operator |
| --- | --- | --- |
| `concurrent` | The manager process | `spec.concurrent`, yours to set. |
| `limit` | One runner entry | **Hardcoded to 10** by the config generator. Not exposed ([#85](https://github.com/alekc/gitlab-runner-operator/issues/85)). |
| `request_concurrency` | One runner entry | **Never set**, so it takes gitlab-runner's default of 1. Not exposed. |

`concurrent` and `limit` both apply, and the lower one wins. A `Runner` object
renders exactly one entry, so its `limit` of 10 is the real ceiling.
`concurrent: 50` on a single `Runner` still runs at most ten jobs at a time.

To go past ten, add entries or objects:

```yaml
apiVersion: gitlab.k8s.alekc.dev/v1beta2
kind: MultiRunner
metadata:
  name: builders
spec:
  # Now meaningful: 3 entries x limit 10 = 30 possible, capped here at 24.
  concurrent: 24
  entries:
    - name: shard-a
      authentication:
        token:
          secret_key_ref:
            name: gitlab-runner-token-a
    - name: shard-b
      authentication:
        token:
          secret_key_ref:
            name: gitlab-runner-token-b
    - name: shard-c
      authentication:
        token:
          secret_key_ref:
            name: gitlab-runner-token-c
```

Each entry is a separate runner in GitLab, so give them the same `tag_list` if
you want GitLab to spread work across them.

## The queue-polling default

`request_concurrency` controls how many job requests the manager holds open
against GitLab's queue at once, and gitlab-runner defaults it to 1. On a busy
queue that is the reason a runner reports online, looks idle, and drains the
backlog far more slowly than its `concurrent` suggests it should. Upstream
guidance is to raise it into the 4 to 20 range for a busy fleet.

That is not reachable through the CRD today
([#85](https://github.com/alekc/gitlab-runner-operator/issues/85)). Running
several entries is the available workaround, since each entry polls on its own.

## Sizing it

`concurrent` is bounded by the cluster, not by the runner. Ten concurrent jobs at
`memory_request: 1Gi` need 10Gi of schedulable memory, plus helper and service
containers on top. Set it above what the cluster can actually schedule and the
surplus jobs sit in `Pending` until `poll_timeout` fails them, which reads as a
runner fault and is really a capacity fault. Work the ceiling out from
[sizing jobs](sizing-jobs.md) first.

## Gotchas

**`check_interval` has a floor of 3 seconds**, enforced by the CRD. Polling harder
is rarely the bottleneck anyway.

**A pending pod holds a slot.** Jobs waiting on an image pull or a node scale-up
consume `concurrent` while doing nothing, so a slow cluster can make a runner
look stalled. See [jobs stuck in Pending](stuck-pods.md).

**More replicas is not more throughput.** The manager Deployment is fixed at one
replica. Scale with more entries or more `Runner` objects.

**One MultiRunner shares the budget.** `spec.concurrent` is spread across all
entries in whatever mix GitLab hands out, not applied per entry. If one workload
must be guaranteed capacity, give it its own object.

## Related

- [Sizing jobs](sizing-jobs.md)
- [Jobs stuck in Pending](stuck-pods.md)
- [Watching a runner](monitoring.md)
- [Limitations](../reference/limitations.md)
