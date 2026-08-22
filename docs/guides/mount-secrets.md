---
description: >-
  Mount Kubernetes Secrets and ConfigMaps into GitLab CI job pods: SSH deploy
  keys, kubeconfigs, registry credentials, and the items key mapping.
---

# Mount secrets and configmaps

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
    volumes:
      secret:
        - name: deploy-ssh-key
          mount_path: /etc/ci/ssh
          read_only: true
          items:
            # secret key -> filename inside mount_path
            "id_ed25519": "id_ed25519"
      config_map:
        - name: ci-settings
          mount_path: /etc/ci/conf
```

Every container in the job pod gets the mount, so the build script can read
`/etc/ci/ssh/id_ed25519` directly.

## The three fields that matter

| Field | Notes |
| --- | --- |
| `name` | The Secret or ConfigMap, read from the namespace the **job pod** runs in. |
| `mount_path` | Directory inside the containers. Mounting over a populated path hides what was there. |
| `items` | Maps a key in the object to a filename in the mount. Without it, every key becomes a file named after the key. |

`sub_path` mounts a single key at a file path instead of a directory, and
`read_only` is worth setting on anything a job has no business writing to.

## Worked example: git submodules over SSH

```yaml
spec:
  environment:
    - "GIT_SSH_COMMAND=ssh -i /etc/ci/ssh/id_ed25519 -o StrictHostKeyChecking=accept-new"
  executor_config:
    volumes:
      secret:
        - name: deploy-ssh-key
          mount_path: /etc/ci/ssh
          read_only: true
          items:
            "id_ed25519": "id_ed25519"
```

A key mounted from a Secret arrives mode `0644`, and OpenSSH refuses a private
key that is group or world readable. The secret volume type exposes no file-mode
setting, and `fs_group` sets group ownership rather than the mode, so it does not
help here. Copy the key and fix the mode in a `before_script`:

```yaml
before_script:
  - install -m 600 -D /etc/ci/ssh/id_ed25519 ~/.ssh/id_ed25519
```

## Gotchas

**A wrong key name fails silently.** The mount succeeds, the file simply is not
there, and the job fails much later with an unrelated error. When wiring one up
for the first time, put `ls -la <mount_path>` in the script and look once.

**Namespace.** Secrets are read from the job pod's namespace, which is
`executor_config.namespace` if set and the runner's namespace otherwise. This is
the most common reason a mount that "should work" does not. See
[dedicated build namespace](build-namespace.md).

**Do not mount the runner's own token.** The runner's authentication token lives
in the config Secret the operator manages. Mounting it into job pods hands every
CI job the credential for the runner itself.

**This is not how you authenticate image pulls.** A registry credential mounted
as a volume is readable by tools inside the job (kaniko, docker login). The
kubelet pulling the job's own image needs `image_pull_secrets` instead. See
[pull from a private registry](private-registry.md).

## Related

- [Pull from a private registry](private-registry.md)
- [Rootless builds](rootless-builds.md), which is this mechanism plus an `items`
  rename
- [Persistent and ephemeral storage](storage.md), for volumes that hold data
  rather than credentials
