---
description: >-
  Use postgres, redis and other service containers with the kubernetes executor:
  aliases as hostnames, per-service resources, and why localhost sometimes works.
---

# Service containers

Databases and caches for integration tests. With the kubernetes executor every
service runs as another container in the **same pod** as the build, which changes
how you reach it and how it is sized.

## The short version

Declared in `.gitlab-ci.yml`, the usual way:

```yaml
integration-tests:
  image: python:3.13
  services:
    - name: postgres:17
      alias: db
      variables:
        # The executor waits for a TCP connection on this port before running
        # the script. A service without it is not waited for at all.
        HEALTHCHECK_TCP_PORT: "5432"
    - name: redis:8
      alias: cache
      variables:
        HEALTHCHECK_TCP_PORT: "6379"
  variables:
    POSTGRES_PASSWORD: ci
    POSTGRES_DB: app_test
    DATABASE_URL: "postgresql://postgres:ci@db:5432/app_test"
    REDIS_URL: "redis://cache:6379/0"
  script:
    - pytest
```

Give the services a budget on the runner, because the defaults are shared with
nothing in mind:

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
    service_cpu_request: "100m"
    service_memory_request: "256Mi"
    service_memory_limit: "1Gi"
```

Those apply to **each** service container, not to all of them together.

## Pinning a service to every job on a runner

If every job on a runner needs the same service, declare it on the runner and
keep it out of every `.gitlab-ci.yml`:

```yaml
spec:
  executor_config:
    services:
      - name: "postgres:17"
        alias: db
        environment:
          - "POSTGRES_PASSWORD=ci"
          - "POSTGRES_DB=app_test"
```

`command` and `entrypoint` are available on the same block when the image needs
arguments.

## How the name resolves

The alias becomes the container name and the hostname the build container uses.
An alias has to be a valid DNS label; when it is missing or unusable the runner
names the container `svc-0`, `svc-1` and so on, which is a confusing thing to
discover from a connection error.

Because all containers share the pod's network namespace, `localhost:5432` also
reaches postgres. Prefer the alias: it survives someone moving the job to a
different executor, and it makes the dependency legible.

## Gotchas

**Two instances of the same image need distinct aliases.** Without them the second
one is unreachable, or worse, the first one is.

**A TCP check is not readiness.** `HEALTHCHECK_TCP_PORT` waits for the port to
accept a connection, which postgres does before it will accept an authenticated
query, and before initialisation has finished. Keep it, because without it the
executor does not wait at all, and still loop on `pg_isready` or an
application-level check in `before_script`. A test suite that fails only in CI,
only sometimes, is usually this.

**An OOM-killed service looks like a network error.** The build container gets
connection refused, with nothing in the job log about memory, because the kill
happened in a different container. Check `kubectl describe pod` for the service
container's last state, and give services a real `service_memory_limit`.

**`FF_NETWORK_PER_BUILD` is a docker executor concept.** Advice about per-build
networks and `/etc/hosts` aliases found in forum threads generally does not
transfer to the kubernetes executor, where the shared pod network already gives
you what that flag is for.

**Restrict what jobs may start.** `allowed_services` is an allowlist. On a shared
cluster it stops a job pulling an arbitrary image as a "service".

## Related

- [Sizing jobs](sizing-jobs.md), for the three separate resource budgets
- [Docker-in-Docker](docker-in-docker.md), which is a service container with
  extra requirements
