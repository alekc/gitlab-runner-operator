# Gitlab Runner Operator

Kubernetes operator that manages GitLab CI runners using the kubernetes executor.

It lets you run one or many GitLab runners, each with its own configuration
expressed in YAML (no more hand-written `config.toml`), following an
infrastructure-as-code approach. The supported
[kubernetes executor](https://docs.gitlab.com/runner/executors/kubernetes.html)
options are configurable through the CRD; the gaps are listed under
[limitations](https://gitlab-runner-operator.alekc.dev/latest/reference/limitations/).

**Documentation: <https://gitlab-runner-operator.alekc.dev/>**

## Status

Alpha. Breaking changes are possible and will be called out in the release
notes. Please open an issue if you hit a bug.

The current API group version is `gitlab.k8s.alekc.dev/v1beta2`. It replaces
the older `v1beta1` / `v1alpha1` versions. The change is breaking, and existing
objects are not converted automatically: see
[upgrading from v1beta1](https://gitlab-runner-operator.alekc.dev/latest/install/#upgrading-from-v1beta1).

## Install

```bash
helm repo add alekc https://charts.alekc.dev/
helm repo update
helm install gitlab-runner-operator alekc/gitlab-runner-operator
```

## Minimal runner

Create the runner in GitLab, then hand its `glrt-` token to a `Runner` object:

```yaml
apiVersion: gitlab.k8s.alekc.dev/v1beta2
kind: Runner
metadata:
  name: runner-sample
spec:
  authentication:
    token:
      value: "glrt-XXXXXXXXXXXXXXXXXXXX"
```

Reading the token from a Secret, letting the operator create the runner for you
with an access token, private CAs, build namespaces and the full field
reference are all in the docs.

## Documentation

| Page | |
| --- | --- |
| [Install](https://gitlab-runner-operator.alekc.dev/latest/install/) | Helm, kustomize, version matrix, upgrades |
| [Authentication](https://gitlab-runner-operator.alekc.dev/latest/authentication/) | Tokens, operator-managed runners, custom CA |
| [RBAC and namespaces](https://gitlab-runner-operator.alekc.dev/latest/operations/rbac-and-namespaces/) | What each runner is granted, and where jobs may run |
| [Uninstalling](https://gitlab-runner-operator.alekc.dev/latest/operations/uninstall/) | Deletion order, and what gets left behind |
| [CRD API reference](https://gitlab-runner-operator.alekc.dev/latest/reference/api/) | Every field, generated from the types |
| [Limitations](https://gitlab-runner-operator.alekc.dev/latest/reference/limitations/) | What it deliberately does not do |
| [Contributing](https://gitlab-runner-operator.alekc.dev/latest/contributing/) | Local cluster, tests, regenerating docs |

The source lives in [`docs/`](docs/) if you would rather read it here.

## License

Apache License 2.0. See [LICENSE](LICENSE).
