# Gitlab Runner Operator
Runner operator based on gitlab runner kubernetes executor

## What is this operator for
This operator permits you to run multiple runners with their own, unique configuration written in yaml (no more Toml, yay), following IAC approach

## Why
The official gitlab way consist in 2 steps registration, first you register your runner, and then you append certain features to it. 

This breaks the IAC way, and doesn't permit you to be flexible in your configuration. 

This operator aims to provide you with all configuration which are provided by [kubernetes executor](https://docs.gitlab.com/runner/executors/kubernetes.html), see below for some examples. 

## Status & Contributions
This operator is still in alpha stages, so there is a possibility of breaking changes (which will be announced if any). Please open issues if you find any bugs

## Installation 
TODO

## Configuration
The CRD Runner is composed by following fields (all of them are optional, except for registration tokens, see examples below):
```yaml
apiVersion: gitlab.k8s.alekc.dev/v1alpha1
kind: Runner
metadata:
  name: runner-sample
spec:
  concurrent: 1
  executor_config: {}
  log_level: info
  gitlab_instance_url: https://gitlab.com
  registration_config: {}    
```
|Key  |Description  |
|--|--|
| concurrent | limits how many jobs globally can be run concurrently. The most upper limit of jobs using all defined runners. Minimum value is 1 |
| executor_config | contains config values for kubernetes executor see [this page](https://docs.gitlab.com/runner/executors/kubernetes.html#the-keywords) for detailed explanation of different values.|
| log_level | log level of the executor. Can be one of following: panic;fatal;error;warning;info;debug|
| gitlab_instance_url| in case you are not using the official gitlab you will need to set this value to your instance url |
| registration_config | see [this link](https://docs.gitlab.com/ee/api/runners.html#register-a-new-runner) for detailed explanation

### Examples
In order to run your runner you will need to obtain a registration token ([see the official documentation](https://docs.gitlab.com/runner/register/))

#### Minimum configuration

In this config you only need to specify the name of your runner (`runner-sample`), registration token (`xxx`), and it's tag (`test-gitlab-runner`)  
```yaml
apiVersion: gitlab.k8s.alekc.dev/v1alpha1
kind: Runner
metadata:
  name: runner-sample
spec:
  registration_config:
    token: "xxx"
    tag_list:
      - test-gitlab-runner
```

#### Using registration token inside a secret
If you prefer not to expose your registration token in the crd, you can specify the secret name. 
Note that the token **MUST** be contained in the key `token`

```yaml
apiVersion: v1
data:
  token: c2VjcmV0LXRva2Vu
kind: Secret
metadata:
  name: gitlab-runner-token
type: Opaque
---
apiVersion: gitlab.k8s.alekc.dev/v1alpha1
kind: Runner
metadata:
  name: runner-sample
spec:
  registration_config:
    token: "gitlab-runner-token"
    tag_list:
      - test-gitlab-runner
```

#### Mounting secrets or config maps as volumes
If you need to mount some config maps or secrets as volumes in your runner pods, you can easily achieve it with following config
```yaml
---
apiVersion: v1
kind: Secret
metadata:
  name: test-secret
stringData:
  foo: "this is foo"
  bar: "this is bar"
---
apiVersion: v1
kind: Secret
metadata:
  name: test-secret-2
stringData:
  zar: "this is zar"
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: test-config
data:
  foo.txt: |
    contents of foo
  bar.txt: |
    zzz
---
apiVersion: gitlab.k8s.alekc.dev/v1alpha1
kind: Runner
metadata:
  name: runner-sample
spec:
  log_level: debug
  executor_config:
    image: "debian:slim"
    memory_limit: "150Mi"
    memory_request: "150Mi"
    volumes:
      config_map:
        - mount_path: /cm/
          name: test-config
      secret:
        - mount_path: /secrets/1/
          name: test-secret
        - mount_path: /secrets/2/
          name: test-secret-2
  registration_config:
    token: "xxx"
    tag_list:
      - test-gitlab-runner
```
