---
description: >-
  How the runner manager finishes its running jobs when it is rolled, evicted
  or deleted, what runner_termination_grace_period_seconds bounds, and the
  custom-image and spot-node caveats.
---

# Draining the runner manager

## The short version

When the runner manager pod is stopped, it stops asking GitLab for new jobs and
finishes the ones it is running. `runner_termination_grace_period_seconds`
bounds how long that may take. It defaults to 3600, the upstream Helm chart's
value. A job still running when it runs out is killed, and its build pod may be
left behind.

```yaml
spec:
  # At least your longest job timeout. 0 kills the manager at once.
  runner_termination_grace_period_seconds: 7200
```

## How it works

Kubernetes stops a container with the stop signal its image declares. The
gitlab-runner images declare `SIGQUIT`, which gitlab-runner treats as a graceful
shutdown: no new jobs, wait for the running ones. The operator sets no
`command`, so that signal reaches the runner through the image's own
entrypoint. No `preStop` hook is involved.

That covers every way the manager stops:

- **A rollout.** A change to the rendered config, the runner image or the
  manager pod settings replaces the pod. The new manager starts first and takes
  new jobs; the old one finishes its jobs alongside it.
- **A node drain or eviction.** `kubectl drain` waits up to the grace period
  for the manager to exit.
- **Deleting the Runner or MultiRunner.** The operator deletes the manager
  first and waits for it to exit, up to the grace period plus a minute. Only
  then does it remove the runner from GitLab and clean up RBAC in other build
  namespaces, which a draining manager still needs. A delete can therefore
  take as long as the longest running job. To cut it short, force-delete the
  manager pod (`kubectl delete pod <pod> --grace-period=0 --force`); its
  running jobs are lost. Setting the grace period to 0 first does not help,
  because the pod being replaced keeps the grace period it started with.

## Caveats

**A custom `spec.image` must keep `STOPSIGNAL SIGQUIT`.** An image that drops
it gets `SIGTERM`, which gitlab-runner treats as "abort every job now", so the
grace period buys nothing. Check with
`docker image inspect --format '{{.Config.StopSignal}}' <image>`.

**Spot capacity overrides all of this.** A reclaimed spot or preemptible node
gets a short notice from the cloud (two minutes on AWS, 30 seconds on GCP),
and the node goes when it runs out regardless of any grace period. Keep the
manager on on-demand nodes; see [spot and preemptible nodes](spot-nodes.md).

**Upgrading rolls every manager once.** The first operator release with this
setting adds the 3600s default to every manager pod. That roll still stops the
old pod with the 30s it was created with, so jobs running at that moment can be
lost. Upgrade when the pipeline queue is quiet.

## Related

- [Spot and preemptible nodes](spot-nodes.md)
- [Node placement](node-placement.md)
- [Install: upgrading](../install.md#upgrading)
