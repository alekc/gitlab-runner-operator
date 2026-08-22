---
description: >-
  Diagnose GitLab CI job pods stuck in Pending or ContainerCreating, and tune
  poll_timeout, poll_interval and pod warning events on the kubernetes executor.
---

# Jobs stuck in Pending

## Symptom

The job log shows `Waiting for pod <ns>/runner-...-concurrent-0` and nothing
else, then after three minutes the job fails and the pod is deleted before you
can look at it.

Three minutes is `poll_timeout`. The pod never reached `Running`.

## First: see the reason

Turn on pod warning events so the reason reaches the job log instead of dying
with the pod:

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
    print_pod_warning_events: true
    # Give a slow image pull room, rather than masking the problem: 3 minutes
    # is not enough for a multi-gigabyte image on a cold node.
    poll_timeout: 600
    poll_interval: 3
```

Live, while a job is pending:

```bash
kubectl get pods -n <build-namespace> -w
kubectl describe pod -n <build-namespace> <pod>   # Events at the bottom
kubectl describe node <node>                      # Allocatable vs requests
```

## The actual causes, in rough order of frequency

**Image pull.** The most common by far. A large image on a node that has never
pulled it, or Docker Hub rate limiting the whole cluster from one IP. Fix with
[registry credentials and a sane pull policy](private-registry.md), not with a
bigger timeout.

**Nothing matches the selector.** A `node_selector` or toleration that no node
satisfies means `Pending` forever with `0/N nodes are available`. Check the
Events line, it says exactly which predicate failed. See
[node placement](node-placement.md).

**Requests exceed anything free.** A 16Gi memory request on 8Gi nodes never
schedules. `kubectl describe node` and compare allocatable against the sum of
requests. See [sizing jobs](sizing-jobs.md).

**IP exhaustion.** On AWS VPC CNI and similar, a subnet with no free addresses
leaves pods stuck in `ContainerCreating` with a sandbox creation error. Nothing
in the runner config fixes this.

**A missing dependency.** A `service_account` or an image pull secret that does
not exist in the build namespace. `resource_availability_check_max_attempts`
controls how long the runner waits for those before giving up, with five seconds
between attempts.

**Cluster autoscaler.** If a new node has to be provisioned, `Pending` is
correct and expected. Size `poll_timeout` above your node provisioning time or
every scale-up event costs you a failed pipeline.

## Gotchas

**Raising `poll_timeout` treats the symptom.** It is the right call for slow node
provisioning and large images, and the wrong call for a selector that matches
nothing, where it just delays the failure.

**The pod is deleted on timeout.** By the time you run `kubectl describe`, it is
gone. Either watch while the job runs, or set `print_pod_warning_events` and read
the job log.

**A `Pending` pod still holds a concurrency slot.** Jobs stuck for ten minutes
each will starve a runner with a low `concurrent`. See
[concurrency](concurrency.md).

## Related

- [Pull from a private registry](private-registry.md)
- [Node placement](node-placement.md)
- [Sizing jobs](sizing-jobs.md)
