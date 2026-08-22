---
description: >-
  Watch a GitLab runner managed by the operator: status conditions, the runner
  metrics endpoint on port 9090, and the operator's own metrics.
---

# Watching a runner

## Is it healthy

```bash
kubectl get runners
kubectl get multirunners
```

`Ready` is a print column, so a fleet fits on one screen. When it is false, the
reason is in the object:

```bash
kubectl describe runner runner-sample
kubectl get runner runner-sample -o jsonpath='{.status}' | jq
```

| Status field | What it tells you |
| --- | --- |
| `ready` | Everything the operator provisions is in place. |
| `error` | The last reconcile error, verbatim. |
| `conditions[Ready]` | Same signal with `reason`, `message` and a transition time, so you can see *when* it broke. |
| `observed_generation` | The spec generation the controller last acted on. Behind `metadata.generation` means your edit has not been processed. |
| `runner_id` | GitLab's numeric id, for a managed runner. Zero in bring-your-own-token mode. |
| `token_expires_at` | When GitLab will expire a managed token. The operator recreates the runner within 24h of this. |
| `config_map_version` | Hash of the rendered config. A change here is what rolls the Deployment. |

A runner that is `ready` but takes no jobs is a different problem: see
[concurrency](concurrency.md).

## Runner metrics

Each runner manager serves gitlab-runner's own Prometheus metrics on **port
9090**, named `metrics` on the pod. The operator points the container's readiness
and liveness probes at that port, so a manager that cannot serve metrics is
already being restarted for you.

Scrape it with a PodMonitor:

```yaml
apiVersion: monitoring.coreos.com/v1
kind: PodMonitor
metadata:
  name: gitlab-runners
  namespace: gitlab-runners
spec:
  selector:
    matchLabels:
      # The operator labels runner pods with the object's name under
      # "deployment". Match one runner, or use matchExpressions with Exists to
      # scrape every runner in the namespace.
      deployment: runner-sample
  podMetricsEndpoints:
    - port: metrics
```

The metrics worth alerting on are gitlab-runner's own:
`gitlab_runner_jobs` for in-flight work, `gitlab_runner_errors_total` for API
trouble, and `gitlab_runner_request_concurrency_exceeded_total`, which tells you
the queue-polling limit is biting.

## Operator metrics

The operator exposes controller-runtime metrics separately, on **8080** by
default, with the chart creating a Service in front of it:

```yaml
# Helm values
metrics:
  enabled: true
  port: 8080
  service:
    enabled: true
    annotations:
      prometheus.io/scrape: "true"
```

`controller_runtime_reconcile_errors_total` and
`workqueue_depth` are the two that matter: a rising error count means runners are
failing to reconcile, and a growing queue means the operator is behind.

!!! warning

    The operator's metrics endpoint is unauthenticated and binds to all
    interfaces. Restrict it with a NetworkPolicy, or set `metrics.enabled: false`
    if that is not acceptable in your cluster.

## When a job fails rather than a runner

Runner-level health tells you nothing about individual jobs, because each job is
a pod that is created and deleted inside a few minutes. For those, watch the
build namespace while a job runs, or turn on `print_pod_warning_events` so the
pod's events reach the job log. See
[jobs stuck in Pending](stuck-pods.md).

## Related

- [Concurrency](concurrency.md)
- [Jobs stuck in Pending](stuck-pods.md)
- [Authentication](../authentication.md), for what `token_expires_at` implies
