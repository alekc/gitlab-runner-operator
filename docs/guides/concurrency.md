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
  # Ceiling on job pods this manager will have in flight. Unset, this floors
  # to 1, so it is the one value you always want to set.
  concurrent: 50
  # Optional per-entry cap. Unset, the entry is bounded by concurrent alone.
  # Set it only to hold an entry below the budget.
  # limit: 20
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

| Setting | Scope | Bounds | Unset means |
| --- | --- | --- | --- |
| `concurrent` | The manager process | Jobs running across every entry | **1**, floored by the operator |
| `limit` | One runner entry | Jobs running for that entry | **no cap**, so `concurrent` governs |
| `request_concurrency` | One runner entry | Job requests in flight, not jobs | **1**, gitlab-runner's own default |

`concurrent` and `limit` both apply and the lower one wins. With `limit` unset
the entry is bounded by `concurrent`, so **`concurrent: 50` runs fifty jobs**.

## What the operator defaults, and what it leaves to gitlab-runner

The operator defaults **neither** `limit` nor `request_concurrency`. Left unset,
the key is omitted from the rendered `config.toml` entirely and gitlab-runner
applies its own default. The operator does not invent a ceiling your spec never
asked for, which it used to: every entry once carried a hardcoded `limit = 10`,
so `concurrent: 50` quietly ran ten jobs.

What upstream does with an absent value, from gitlab-runner's own source:

| Key | Absent or zero behaves as | Where |
| --- | --- | --- |
| `limit` | no per-entry cap | `acquireBuild` enforces a limit only `if runner.Limit > 0` |
| `request_concurrency` | 1 | `GetRequestConcurrency()` returns `max(1, x)` |
| `concurrent` | never absent | the operator floors it to 1, matching upstream's default |

So `concurrent` is the one knob you always want to set. A lone `limit: 50` with
`concurrent` unset still runs **one** job, because `concurrent` floors to 1 and
the lower value wins. That combination is accepted rather than rejected, because
gitlab-runner accepts it too and the operator does not add validation upstream
does not have.

### What gitlab-runner will and will not tell you

`checkConfigConcurrency` logs warnings at manager startup, but they are narrower
than they sound and none of them covers the case above:

- *"Worker starvation bottleneck: 'concurrent' setting (N) is less than number of
  runners (M)"*. The test is `concurrent < number of entries`. A `Runner` renders
  exactly one entry, so this **cannot fire for a `Runner` at all**. It is a
  `MultiRunner` warning in practice.
- *"Request bottleneck: N runners have request_concurrency=1, causing job delays
  during long polling"*, suggesting 2 to 4. Since the operator does not render
  `request_concurrency`, every entry that does not set it sits at 1, so this
  fires for **every** runner until you set the field. Useful the first time,
  noise thereafter.
- An entry with `limit` of 1 or 2 **and** `request_concurrency` at 1, flagged as
  restrictive.

So `limit: 50` with `concurrent` unset produces no warning from any of the three:
one entry, so no starvation check; `limit` above 2, so nothing restrictive. It
runs one job in silence. Read `concurrent` as the number you will get, and do not
wait for a log line to tell you otherwise.

!!! warning "`gitlab_runner_limit` is not the effective cap"

    Upstream exports `gitlab_runner_limit` as a gauge, but it reports the raw
    configured `limit`, not the ceiling actually in force. Because the operator
    no longer renders the key, it reads **0** on every runner it manages, meaning
    "no per-entry cap", not "no jobs allowed". Do not alert on it as though it
    were the effective parallelism. Upstream's own HELP string for it says "The
    current value of concurrent setting", which is wrong too.

!!! warning "Raising one without the other is half a change"

    `limit` caps jobs running; `request_concurrency` caps how many jobs the
    manager can be asking for at once. At `request_concurrency: 1` an entry
    acquires work one round trip at a time, so `limit: 50` fills fifty slots
    over fifty sequential requests, paced by `check_interval`. On a queue other
    runners are also draining it may never catch up, which is the "online but
    takes almost nothing" symptom. Unset it is 1, so raising `limit` without also
    raising this is half a change. Upstream guidance for a busy fleet is 4 to
    20.

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
replica. Scale with `concurrent`, more entries, or more `Runner` objects. Raising
`limit` past `concurrent` does nothing, since the lower value wins.

**`output_limit` is still not exposed.** The job log size cap stays at
gitlab-runner's default; see [limitations](../reference/limitations.md).

**A limit above `concurrent` is inert, not an error.** The lower value wins and
nothing reports the difference, so `limit: 50` under `concurrent: 10` runs ten
jobs and the 50 is dead config. Setting every entry high and steering with
`concurrent` alone is a legitimate style; just do not read the high number as
what you will get.

**One MultiRunner shares the budget.** `spec.concurrent` is spread across all
entries in whatever mix GitLab hands out, not applied per entry. If one workload
must be guaranteed capacity, give it its own object.

## Related

- [Sizing jobs](sizing-jobs.md)
- [Jobs stuck in Pending](stuck-pods.md)
- [Watching a runner](monitoring.md)
- [Limitations](../reference/limitations.md)
