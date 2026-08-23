---
description: >-
  How concurrent, limit and request_concurrency interact on this operator, what
  they default to, and why raising one without the other does little.
---

# Concurrency

## The short version

```yaml
apiVersion: gitlab.k8s.alekc.dev/v1beta2
kind: Runner
metadata:
  name: runner-sample
spec:
  # Ceiling on job pods this manager will have in flight.
  concurrent: 50
  # Ceiling for this runner entry. The lower of the two wins, so leaving this
  # at its default of 10 caps you at 10 whatever concurrent says.
  limit: 50
  # Job requests in flight to GitLab. Not a job limit: a polling limit.
  request_concurrency: 8
  # How often it asks GitLab for work, in seconds. Minimum 3.
  check_interval: 3
  authentication:
    token:
      secret_key_ref:
        name: gitlab-runner-token
```

## The three settings

| Setting | Scope | Bounds | Default here |
| --- | --- | --- | --- |
| `concurrent` | The manager process | Jobs running across every entry | **1** (floored by the operator) |
| `limit` | One runner entry | Jobs running for that entry | **10** |
| `request_concurrency` | One runner entry | Job requests in flight, not jobs | **3** |

`concurrent` and `limit` both apply and the lower one wins. A `Runner` renders
exactly one entry, so **`concurrent: 50` with the default `limit` still runs ten
jobs**. Raise both, or accept ten.

!!! warning "Raising one without the other is half a change"

    `limit` caps jobs running; `request_concurrency` caps how many jobs the
    manager can be asking for at once. At `request_concurrency: 1` an entry
    acquires work one round trip at a time, so `limit: 50` fills fifty slots
    over fifty sequential requests, paced by `check_interval`. On a queue other
    runners are also draining it may never catch up, which is the "online but
    takes almost nothing" symptom. The default of 3 is a starting point, not a
    target: upstream guidance for a busy fleet is 4 to 20.

## Per entry on a MultiRunner

Both settings live on the **entry**, because that is the only way to stop one
entry consuming the whole shared `concurrent` budget:

```yaml
apiVersion: gitlab.k8s.alekc.dev/v1beta2
kind: MultiRunner
metadata:
  name: builders
spec:
  concurrent: 30
  entries:
    # A long-running integration suite that must not starve the others.
    - name: integration
      limit: 4
      request_concurrency: 2
      authentication:
        token:
          secret_key_ref:
            name: gitlab-runner-token-integration
    # Short unit jobs, given the bulk of the budget.
    - name: unit
      limit: 24
      request_concurrency: 8
      authentication:
        token:
          secret_key_ref:
            name: gitlab-runner-token-unit
```

Each entry is a separate runner in GitLab. The example splits the budget, not
the work: to route jobs deliberately you also need distinct tags, and `tag_list`
only exists under `authentication.access_token.create_options`, so entries using
a pre-created token cannot set it. See
[authentication](../authentication.md#create_options).

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
replica. Scale with `limit`, more entries, or more `Runner` objects.

**`output_limit` is still not exposed.** The job log size cap stays at
gitlab-runner's default; see [limitations](../reference/limitations.md).

**There is no unlimited.** gitlab-runner treats `limit = 0` as no limit, but the
schema requires at least 1, so an entry always carries a cap. Bound it by
`concurrent` instead by setting `limit` to the same value.

**One MultiRunner shares the budget.** `spec.concurrent` is spread across all
entries in whatever mix GitLab hands out, not applied per entry. If one workload
must be guaranteed capacity, give it its own object.

## Related

- [Sizing jobs](sizing-jobs.md)
- [Jobs stuck in Pending](stuck-pods.md)
- [Watching a runner](monitoring.md)
- [Limitations](../reference/limitations.md)
