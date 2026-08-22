---
description: >-
  Pull GitLab CI job images from a private registry with image_pull_secrets or a
  service account, and avoid Docker Hub rate limits stalling job pods.
---

# Pull from a private registry

This is about the image the **job pod itself** runs, pulled by the kubelet. If
you want credentials available to a tool inside the job, that is
[mount secrets](mount-secrets.md) instead.

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
    image: "registry.example.com/ci/base:1.4"
    image_pull_secrets:
      - registry-example-com
    pull_policy:
      - if-not-present
```

The Secret, in the namespace the job pods run in:

```bash
kubectl create secret docker-registry registry-example-com \
  --docker-server=registry.example.com \
  --docker-username=<user> \
  --docker-password=<token>
```

## Using the service account instead

If your identity story already puts pull credentials on service accounts (a
registry controller, IRSA, an imagePullSecret patched onto the SA), point the
runner at that rather than listing secrets:

```yaml
spec:
  executor_config:
    service_account: ci-builder
    use_service_account_image_pull_secrets: true
```

The operator provisions its own ServiceAccount for the runner, so
`service_account` here names a **different**, pre-existing one you manage, in the
build namespace.

## Docker Hub rate limits

Unauthenticated pulls from Docker Hub are rate limited per source IP, and a
cluster is one IP. When the limit is hit, pods sit in `Pending` or
`ImagePullBackOff` until the runner gives up at `poll_timeout` and the job fails
with something that looks nothing like a rate limit.

Three mitigations, in order of effectiveness:

1. Authenticate. Even a free account raises the limit substantially. Same
   `image_pull_secrets` mechanism, with `docker.io` as the server.
2. `pull_policy: [if-not-present]`, so a node that already has the image does not
   re-pull it.
3. Mirror the handful of base images you actually use into your own registry.

## Gotchas

**`pull_policy` is a list.** It takes an ordered list of policies to attempt, not
a single string. `["always"]` and `["if-not-present", "always"]` are both valid;
`"always"` on its own is not.

**Restricting what jobs may run.** `allowed_images` and `allowed_pull_policies`
are allowlists checked at job start. Setting `allowed_images` to your own
registry prefix is a cheap way to stop CI pulling arbitrary images onto your
nodes.

**The secret is namespaced.** Same trap as everywhere else on this site: the
kubelet reads it from the job pod's namespace.

## Related

- [Mount secrets and configmaps](mount-secrets.md)
- [Jobs stuck in Pending](stuck-pods.md), where slow or failing pulls actually
  show up
