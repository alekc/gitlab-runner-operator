---
description: >-
  Task guides and a symptom index for running GitLab CI on Kubernetes with the
  operator: DinD, secrets, node placement, resources, spot nodes, stuck pods.
---

# Guides

Two ways in. If something is broken, start with the symptom. If you are setting
something up, start with the task.

## By symptom

| What you are seeing | Cause | Go to |
| --- | --- | --- |
| `Cannot connect to the Docker daemon at tcp://docker:2375` | TLS port mismatch, or no shared cert volume | [Docker-in-Docker](docker-in-docker.md) |
| `privileged` pod rejected on admission | Pod Security Admission `restricted` | [Rootless builds](rootless-builds.md) |
| `exec format error`, helper container crashes at once | amd64 helper image on an arm64 node | [Node placement](node-placement.md) |
| `Waiting for pod ...` then the job fails after 3 minutes | `poll_timeout`, usually a slow image pull | [Jobs stuck in Pending](stuck-pods.md) |
| `0/N nodes are available` | selector or toleration matches nothing | [Node placement](node-placement.md) |
| Exit code 137, job dies at the same step every time | OOM kill | [Sizing jobs](sizing-jobs.md) |
| Job dies during `uploading artifacts` | the helper container's own limit | [Sizing jobs](sizing-jobs.md) |
| Pod evicted, node looks healthy | ephemeral storage with no request | [Persistent and ephemeral storage](storage.md) |
| `pods is forbidden` | RBAC, or a build namespace that is not allowed | [RBAC and namespaces](../operations/rbac-and-namespaces.md) |
| `poddisruptionbudgets is forbidden` | the optional PDB grant, or an upgrade window | [Spot and preemptible nodes](spot-nodes.md) |
| Mounted secret is missing inside the job | wrong namespace, or a key that needs `items` | [Mount secrets and configmaps](mount-secrets.md) |
| `ImagePullBackOff` on the job's own image | pull secret missing, or Docker Hub rate limit | [Pull from a private registry](private-registry.md) |
| Connection refused reaching postgres or redis | service not ready yet, or OOM-killed | [Service containers](service-containers.md) |
| Runner is online but picks up almost nothing | `limit` and `request_concurrency`, which default low | [Concurrency](concurrency.md) |
| Jobs fail as `script_failure` after a node disappeared | spot eviction is not classified as a system failure | [Spot and preemptible nodes](spot-nodes.md) |
| `cache:` declared but never restores | no distributed cache support yet | [Limitations](../reference/limitations.md) |
| `x509: certificate signed by unknown authority` | GitLab behind a private CA | [Authentication](../authentication.md#custom-ca-for-a-self-signed-gitlab) |
| Runner NotReady, no obvious reason | read `status.error` and the `Ready` condition | [Watching a runner](monitoring.md) |

## By task

**Building images**

- [Docker-in-Docker](docker-in-docker.md): privileged builds, and the certificate share that makes TLS work
- [Rootless builds](rootless-builds.md): kaniko or buildkit, for clusters that forbid privileged

**Scheduling**

- [Node placement](node-placement.md): node pools, taints, arm64, and mixed-arch fleets
- [Spot and preemptible nodes](spot-nodes.md): cheap capacity, and making retry actually fire

**Volumes and secrets**

- [Mount secrets and configmaps](mount-secrets.md): deploy keys, kubeconfigs, `items` mapping
- [Pull from a private registry](private-registry.md): pull secrets, service accounts, rate limits
- [Persistent and ephemeral storage](storage.md): emptyDir, PVCs, CSI, and eviction

**Capacity**

- [Sizing jobs](sizing-jobs.md): three containers, three budgets, and per-job overrides
- [Concurrency](concurrency.md): what `concurrent` really buys you here
- [Jobs stuck in Pending](stuck-pods.md): finding out why, not just waiting longer

**Isolation and networking**

- [Dedicated build namespace](build-namespace.md): the allow-list, PSA, quotas, NetworkPolicy
- [Service containers](service-containers.md): postgres, redis, aliases as hostnames

**Operations**

- [Watching a runner](monitoring.md): status fields, runner metrics on 9090, operator metrics

## Before you file a bug

Four things are known gaps rather than misconfiguration, and each has an issue:

| Gap | Issue |
| --- | --- |
| No distributed cache (`[runners.cache]`) | [#81](https://github.com/alekc/gitlab-runner-operator/issues/81) |
| No environment variables on the manager container, so no outbound proxy | [#82](https://github.com/alekc/gitlab-runner-operator/issues/82) |
| No `nodeSelector` / `tolerations` / `affinity` for the manager pod | [#83](https://github.com/alekc/gitlab-runner-operator/issues/83) |
| Manager does not drain gracefully on rollout or eviction | [#84](https://github.com/alekc/gitlab-runner-operator/issues/84) |

The full list, including fields that are accepted but inert, is in
[limitations](../reference/limitations.md).
