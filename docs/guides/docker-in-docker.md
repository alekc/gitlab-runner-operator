---
description: >-
  Run Docker-in-Docker jobs on a Runner object, including the TLS certificate
  share that fixes the Cannot connect to the Docker daemon error on port 2375.
---

# Docker-in-Docker

## The short version

```yaml
apiVersion: gitlab.k8s.alekc.dev/v1beta2
kind: Runner
metadata:
  name: runner-dind
spec:
  authentication:
    token:
      secret_key_ref:
        name: gitlab-runner-token
  environment:
    - "DOCKER_TLS_CERTDIR=/certs"
  executor_config:
    image: "docker:28"
    privileged: true
    volumes:
      empty_dir:
        # The daemon writes its client certs here and the build container reads
        # them. Both containers are in the same pod, so an emptyDir is the
        # share. Without it the client has no certs and falls back to :2375.
        - name: docker-certs
          mount_path: /certs/client
          medium: Memory
```

In `.gitlab-ci.yml`:

```yaml
build:
  image: docker:28
  services:
    - name: docker:28-dind
      variables:
        # Without this the executor does not wait for the daemon at all: it
        # skips the readiness check for any service that does not set it.
        HEALTHCHECK_TCP_PORT: "2376"
  variables:
    DOCKER_HOST: tcp://docker:2376
    DOCKER_TLS_VERIFY: 1
    DOCKER_CERT_PATH: "/certs/client"
  script:
    - docker info
    - docker build -t "$CI_REGISTRY_IMAGE:$CI_COMMIT_SHA" .
```

## Why it is shaped like this

Docker 19.03 and later talks TLS by default. The daemon generates a CA and
client certificate at startup, writes them under `DOCKER_TLS_CERTDIR`, and
listens on **2376**. The client needs those files. In the kubernetes executor
the daemon runs as a service container in the same pod as the build container,
so the two share `localhost` and the pod's volumes, but not their filesystems.
An `empty_dir` mounted at `/certs/client` in both is what carries the
certificate across.

`DOCKER_TLS_CERTDIR=/certs` belongs on `spec.environment`, not in
`executor_config`. That field renders into the runner's `config.toml` as the
build environment, so it reaches every container in the job pod, including the
service.

`privileged: true` is required. The daemon needs it.

## Gotchas

**`Cannot connect to the Docker daemon at tcp://docker:2375. Is the docker daemon running?`**

A port mismatch. 2375 is the plaintext port, 2376 is TLS. Seeing both numbers in
one job log means the client and daemon disagree about TLS. Either give the
client the certs as above and point it at 2376, or turn TLS off entirely with
`DOCKER_TLS_CERTDIR=""` and `DOCKER_HOST=tcp://docker:2375`. Do not mix them.

**The daemon is not up yet.** The build container can start before the service
finishes booting, which looks identical to a misconfiguration. That is what
`HEALTHCHECK_TCP_PORT` in the example above is for: the executor skips the
readiness check entirely for a service that does not set it.

**Never use `docker:latest`.** Pin the tag on both the image and the service, and
keep them the same version. A new Docker major changing its TLS defaults
underneath you is the single most common cause of this breaking on a Friday.

**Your cluster may forbid this outright.** Under Pod Security Admission
`restricted`, a privileged pod is rejected and the job fails before the daemon
starts. There is no configuration that fixes that. See
[rootless builds](rootless-builds.md).

## Related

- [Rootless builds](rootless-builds.md), when privileged is not available
- [Sizing jobs](sizing-jobs.md), because the daemon needs its own memory budget
- [Service containers](service-containers.md), for how aliases become hostnames
