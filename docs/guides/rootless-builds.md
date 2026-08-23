---
description: >-
  Build container images without privileged mode, using kaniko or buildkit on a
  Runner object, for clusters that enforce restricted Pod Security Admission.
---

# Rootless image builds

Use this when the cluster refuses privileged pods, which is the default under
Pod Security Admission `restricted`. Neither kaniko nor rootless buildkit needs
`privileged: true`.

## The short version

```yaml
apiVersion: gitlab.k8s.alekc.dev/v1beta2
kind: Runner
metadata:
  name: runner-rootless
spec:
  authentication:
    token:
      secret_key_ref:
        name: gitlab-runner-token
  executor_config:
    volumes:
      secret:
        # kaniko reads registry credentials from /kaniko/.docker/config.json.
        # The Secret must live in the namespace the job pod runs in.
        - name: kaniko-docker-config
          mount_path: /kaniko/.docker
          items:
            ".dockerconfigjson": "config.json"
    build_container_security_context:
      allow_privilege_escalation: false
      run_as_non_root: true
      capabilities:
        drop: ["ALL"]
      seccomp_profile:
        type: RuntimeDefault
```

In `.gitlab-ci.yml`:

```yaml
build:
  image:
    name: gcr.io/kaniko-project/executor:debug
    entrypoint: [""]
  script:
    - /kaniko/executor
      --context "$CI_PROJECT_DIR"
      --dockerfile "$CI_PROJECT_DIR/Dockerfile"
      --destination "$CI_REGISTRY_IMAGE:$CI_COMMIT_SHA"
```

The Secret itself, in the build namespace:

```bash
kubectl create secret docker-registry kaniko-docker-config \
  --namespace <build-namespace> \
  --docker-server=registry.example.com \
  --docker-username=<user> \
  --docker-password=<token>
```

## Why the items mapping

A `kubernetes.io/dockerconfigjson` Secret stores the credentials under the key
`.dockerconfigjson`, but kaniko looks for a file called `config.json`. Mounting
the Secret without `items` gives you `/kaniko/.docker/.dockerconfigjson`, which
kaniko ignores, and the build fails at push time with an authentication error
rather than at mount time. The `items` map renames the key on the way in.

## Gotchas

**Nothing tells you the mount is wrong.** A missing or misnamed Secret key
produces a job that runs happily until the push. Check by running
`ls -la /kaniko/.docker` in the job script while setting this up the first time.

**Namespace.** The Secret is read from the namespace the **job pod** runs in,
which is `executor_config.namespace` when set, otherwise the runner's own
namespace. Putting it next to the Runner object and then setting a build
namespace is a common way to break this. See
[dedicated build namespace](build-namespace.md).

**kaniko cannot use the Docker cache.** It has its own `--cache` flag that needs
a registry to push cache layers to. GitLab's `cache:` keyword does not help here,
and on this operator there is no distributed cache at all yet
([#81](https://github.com/alekc/gitlab-runner-operator/issues/81)).

**buildkit instead.** `moby/buildkit:rootless` works the same way, needs
`securityContext` tuning rather than privileged, and wants
`BUILDKITD_FLAGS=--oci-worker-no-process-sandbox`. It is faster than kaniko for
multi-stage builds and its cache export is more flexible.

## Related

- [Docker-in-Docker](docker-in-docker.md), the privileged alternative
- [Mount secrets and configmaps](mount-secrets.md), for the general mechanism
- [Pull from a private registry](private-registry.md), which is a different
  problem: that one is about pulling the job's own image
