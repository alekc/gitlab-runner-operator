---
description: >-
  Every field of the Runner and MultiRunner CRDs, generated from the Go
  types: authentication, executor config, volumes, resources and status.
---

!!! info "Generated page"

    Rendered from the Go types in `api/` by `make docs`. Edits here are
    reverted by the next run; change the types instead.

# API Reference

## Packages
- [gitlab.k8s.alekc.dev/v1beta2](#gitlabk8salekcdevv1beta2)


## gitlab.k8s.alekc.dev/v1beta2

Package v1beta2 contains API Schema definitions for the gitlab v1beta2 API group

### Resource Types
- [MultiRunner](#multirunner)
- [Runner](#runner)



#### CAKeyRef



CAKeyRef points at a single key inside a Secret or ConfigMap.



_Appears in:_
- [CASource](#casource)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name of the Secret or ConfigMap. |  | MinLength: 1 <br /> |
| `key` _string_ | Key holding the PEM CA bundle. Defaults to "ca.crt" when empty. |  | Optional: \{\} <br /> |


#### CASource



CASource provides a PEM-encoded CA bundle used to verify the GitLab endpoint,
both for the operator's own API calls and for the runner's connection. Set at
most one of Value, SecretKeyRef, or ConfigMapKeyRef.



_Appears in:_
- [MultiRunnerSpec](#multirunnerspec)
- [RunnerSpec](#runnerspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `value` _string_ | Value is an inline PEM CA bundle, supplied directly in the manifest.<br />Convenient for small bundles; prefer a Secret or ConfigMap ref when the<br />bundle is large or rotated independently of the runner spec. |  | Optional: \{\} <br /> |
| `secretKeyRef` _[CAKeyRef](#cakeyref)_ | SecretKeyRef selects a key in a Secret holding the PEM CA bundle. |  | Optional: \{\} <br /> |
| `configMapKeyRef` _[CAKeyRef](#cakeyref)_ | ConfigMapKeyRef selects a key in a ConfigMap holding the PEM CA bundle. |  | Optional: \{\} <br /> |


#### ConcurrencyLimits



ConcurrencyLimits are the per-entry budgets, embedded inline in both kinds so
the two cannot drift apart. Neither field is defaulted: left unset the key is
omitted from config.toml and gitlab-runner applies its own default, so the
operator never invents a ceiling the spec does not state.



_Appears in:_
- [MultiRunnerEntry](#multirunnerentry)
- [RunnerSpec](#runnerspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `limit` _integer_ | Limit caps the jobs this entry runs at once. Zero omits the key, and<br />upstream acquireBuild only enforces a limit when it is above zero, so the<br />entry is bounded by Concurrent alone. Both apply, lower wins. |  | Minimum: 0 <br /> |
| `request_concurrency` _integer_ | RequestConcurrency caps job requests in flight to GitLab, not jobs<br />running. Zero omits the key, and upstream GetRequestConcurrency returns<br />max(1, x), so absent means 1 and a raised Limit then fills slowly. |  | Minimum: 0 <br /> |


#### GitlabAuth



GitlabAuth configures how a runner authenticates to GitLab. GitLab removed
the legacy registration-token workflow (deprecated in 16.0, disabled by
default from 18.0); runners now authenticate with a runner authentication
token (the "glrt-" token). Exactly one of two modes must be provided:

  - Bring-your-own token: set Token to a runner authentication token created
    in the GitLab UI or via the API. The operator performs no GitLab API
    calls and writes the token straight into the runner config.

  - Managed: set AccessToken to a personal, group, or project access token
    holding the "create_runner" scope, together with a CreateOptions block.
    The operator creates the runner through POST /user/runners, stores the
    returned token, and deletes the runner from GitLab when the object is
    removed.

Each credential is a TokenSource, so it may be supplied inline (value) or
from a Secret key (secret_key_ref, with a configurable key defaulting to
"token").



_Appears in:_
- [MultiRunnerEntry](#multirunnerentry)
- [RunnerSpec](#runnerspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `token` _[TokenSource](#tokensource)_ | Token is the pre-created runner authentication token ("glrt-...") used in<br />bring-your-own mode. Mutually exclusive with the managed CreateOptions. |  | Optional: \{\} <br /> |
| `access_token` _[TokenSource](#tokensource)_ | AccessToken is a personal, group, or project access token with the<br />"create_runner" scope. Required for the managed mode. |  | Optional: \{\} <br /> |
| `create_options` _[RunnerCreateOptions](#runnercreateoptions)_ | CreateOptions describes the runner to create. When set, the operator runs<br />in managed mode and owns the runner's lifecycle on GitLab. |  | Optional: \{\} <br /> |


#### KubernetesAffinity







_Appears in:_
- [KubernetesConfig](#kubernetesconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `node_affinity` _[KubernetesNodeAffinity](#kubernetesnodeaffinity)_ |  |  |  |
| `pod_affinity` _[KubernetesPodAffinity](#kubernetespodaffinity)_ |  |  |  |
| `pod_anti_affinity` _[KubernetesPodAntiAffinity](#kubernetespodantiaffinity)_ |  |  |  |


#### KubernetesAppArmorProfile



KubernetesAppArmorProfile selects the AppArmor profile for pod containers.
Requires Kubernetes 1.30 or newer. The runner silently drops a profile it
cannot use, so the constraints here are validated at admission instead.



_Appears in:_
- [KubernetesContainerSecurityContext](#kubernetescontainersecuritycontext)
- [KubernetesPodSecurityContext](#kubernetespodsecuritycontext)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `type` _string_ |  |  | Enum: [RuntimeDefault Localhost Unconfined] <br /> |
| `localhost_profile` _string_ |  |  |  |


#### KubernetesCSI







_Appears in:_
- [KubernetesVolumes](#kubernetesvolumes)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ |  |  |  |
| `mount_path` _string_ |  |  |  |
| `sub_path` _string_ |  |  |  |
| `driver` _string_ |  |  |  |
| `fs_type` _string_ |  |  |  |
| `read_only` _boolean_ |  |  |  |
| `volume_attributes` _object (keys:string, values:string)_ |  |  |  |


#### KubernetesConfig



KubernetesConfig is the kubernetes executor configuration for a runner unit.



_Appears in:_
- [MultiRunnerEntry](#multirunnerentry)
- [RunnerSpec](#runnerspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `host` _string_ |  |  |  |
| `cert_file` _string_ |  |  |  |
| `key_file` _string_ |  |  |  |
| `ca_file` _string_ |  |  |  |
| `bearer_token_overwrite_allowed` _boolean_ |  |  |  |
| `bearer_token` _string_ |  |  |  |
| `image` _string_ |  |  |  |
| `namespace` _string_ |  |  |  |
| `namespace_overwrite_allowed` _string_ |  |  |  |
| `privileged` _boolean_ |  |  |  |
| `runtime_class_name` _string_ |  |  |  |
| `allow_privilege_escalation` _boolean_ |  |  |  |
| `cpu_limit` _string_ |  |  |  |
| `cpu_limit_overwrite_max_allowed` _string_ |  |  |  |
| `cpu_request` _string_ |  |  |  |
| `cpu_request_overwrite_max_allowed` _string_ |  |  |  |
| `memory_limit` _string_ |  |  |  |
| `memory_limit_overwrite_max_allowed` _string_ |  |  |  |
| `memory_request` _string_ |  |  |  |
| `memory_request_overwrite_max_allowed` _string_ |  |  |  |
| `ephemeral_storage_limit` _string_ |  |  |  |
| `ephemeral_storage_limit_overwrite_max_allowed` _string_ |  |  |  |
| `ephemeral_storage_request` _string_ |  |  |  |
| `ephemeral_storage_request_overwrite_max_allowed` _string_ |  |  |  |
| `service_cpu_limit` _string_ |  |  |  |
| `service_cpu_limit_overwrite_max_allowed` _string_ |  |  |  |
| `service_cpu_request` _string_ |  |  |  |
| `service_cpu_request_overwrite_max_allowed` _string_ |  |  |  |
| `service_memory_limit` _string_ |  |  |  |
| `service_memory_limit_overwrite_max_allowed` _string_ |  |  |  |
| `service_memory_request` _string_ |  |  |  |
| `service_memory_request_overwrite_max_allowed` _string_ |  |  |  |
| `service_ephemeral_storage_limit` _string_ |  |  |  |
| `service_ephemeral_storage_limit_overwrite_max_allowed` _string_ |  |  |  |
| `service_ephemeral_storage_request` _string_ |  |  |  |
| `service_ephemeral_storage_request_overwrite_max_allowed` _string_ |  |  |  |
| `helper_cpu_limit` _string_ |  |  |  |
| `helper_cpu_limit_overwrite_max_allowed` _string_ |  |  |  |
| `helper_cpu_request` _string_ |  |  |  |
| `helper_cpu_request_overwrite_max_allowed` _string_ |  |  |  |
| `helper_memory_limit` _string_ |  |  |  |
| `helper_memory_limit_overwrite_max_allowed` _string_ |  |  |  |
| `helper_memory_request` _string_ |  |  |  |
| `helper_memory_request_overwrite_max_allowed` _string_ |  |  |  |
| `helper_ephemeral_storage_limit` _string_ |  |  |  |
| `helper_ephemeral_storage_limit_overwrite_max_allowed` _string_ |  |  |  |
| `helper_ephemeral_storage_request` _string_ |  |  |  |
| `helper_ephemeral_storage_request_overwrite_max_allowed` _string_ |  |  |  |
| `allowed_images` _string array_ |  |  |  |
| `allowed_pull_policies` _string array_ |  |  |  |
| `allowed_services` _string array_ |  |  |  |
| `pull_policy` _string array_ |  |  |  |
| `node_selector` _object (keys:string, values:string)_ |  |  |  |
| `node_selector_overwrite_allowed` _string_ |  |  |  |
| `node_tolerations` _object (keys:string, values:string)_ |  |  |  |
| `affinity` _[KubernetesAffinity](#kubernetesaffinity)_ |  |  |  |
| `image_pull_secrets` _string array_ |  |  |  |
| `helper_image` _string_ |  |  |  |
| `helper_image_flavor` _string_ | HelperImageFlavor selects the OS base for the helper image. Upstream<br />interpolates it into the image tag rather than validating it, so an<br />unrecognised value fails as an ImagePullBackOff on the build pod rather<br />than at admission. Empty means alpine, or concrete under FF_CONCRETE. |  |  |
| `terminationGracePeriodSeconds` _integer_ | Deprecated: no effect since gitlab-runner v17.0.0, which removed the key.<br />A value set here is accepted, rendered into config.toml, then silently<br />dropped by the runner. Set pod_termination_grace_period_seconds and<br />cleanup_grace_period_seconds instead. |  |  |
| `pod_termination_grace_period_seconds` _integer_ | PodTerminationGracePeriodSeconds is the build pod's grace period. Unset<br />means the Kubernetes default of 30s. Before gitlab-runner v17.0.0 an<br />unset grace period meant 0s, so a runner upgraded across that boundary<br />waits 30s where it used to terminate at once. |  |  |
| `cleanup_grace_period_seconds` _integer_ | CleanupGracePeriodSeconds is the DeleteOptions grace period used when<br />tearing down the build pod and its credentials secret after a job. Unset<br />defers to each object's own grace period, which for the pod is<br />pod_termination_grace_period_seconds. |  |  |
| `poll_interval` _integer_ |  |  |  |
| `poll_timeout` _integer_ |  |  |  |
| `resource_availability_check_max_attempts` _integer_ |  |  |  |
| `pod_labels` _object (keys:string, values:string)_ |  |  |  |
| `pod_labels_overwrite_allowed` _string_ |  |  |  |
| `scheduler_name` _string_ |  |  |  |
| `service_account` _string_ |  |  |  |
| `service_account_overwrite_allowed` _string_ |  |  |  |
| `pod_annotations` _object (keys:string, values:string)_ |  |  |  |
| `pod_annotations_overwrite_allowed` _string_ |  |  |  |
| `pod_security_context` _[KubernetesPodSecurityContext](#kubernetespodsecuritycontext)_ |  |  |  |
| `init_permissions_container_security_context` _[KubernetesContainerSecurityContext](#kubernetescontainersecuritycontext)_ |  |  |  |
| `build_container_security_context` _[KubernetesContainerSecurityContext](#kubernetescontainersecuritycontext)_ |  |  |  |
| `helper_container_security_context` _[KubernetesContainerSecurityContext](#kubernetescontainersecuritycontext)_ |  |  |  |
| `service_container_security_context` _[KubernetesContainerSecurityContext](#kubernetescontainersecuritycontext)_ |  |  |  |
| `volumes` _[KubernetesVolumes](#kubernetesvolumes)_ |  |  |  |
| `host_aliases` _[KubernetesHostAliases](#kuberneteshostaliases) array_ |  |  |  |
| `services` _[Service](#service) array_ |  |  |  |
| `cap_add` _string array_ |  |  |  |
| `cap_drop` _string array_ |  |  |  |
| `dns_policy` _string_ |  |  |  |
| `dns_config` _[KubernetesDNSConfig](#kubernetesdnsconfig)_ |  |  |  |
| `container_lifecycle` _[KubernetesContainerLifecyle](#kubernetescontainerlifecyle)_ |  |  |  |
| `priority_class_name` _string_ |  |  |  |
| `context` _string_ |  |  |  |
| `namespace_per_job` _boolean_ |  |  |  |
| `pod_cpu_limit` _string_ |  |  |  |
| `pod_cpu_limit_overwrite_max_allowed` _string_ |  |  |  |
| `pod_cpu_request` _string_ |  |  |  |
| `pod_cpu_request_overwrite_max_allowed` _string_ |  |  |  |
| `pod_memory_limit` _string_ |  |  |  |
| `pod_memory_limit_overwrite_max_allowed` _string_ |  |  |  |
| `pod_memory_request` _string_ |  |  |  |
| `pod_memory_request_overwrite_max_allowed` _string_ |  |  |  |
| `node_tolerations_overwrite_allowed` _string_ |  |  |  |
| `helper_image_autoset_arch_and_os` _boolean_ |  |  |  |
| `logs_base_dir` _string_ |  |  |  |
| `scripts_base_dir` _string_ |  |  |  |
| `pod_spec` _[KubernetesPodSpec](#kubernetespodspec) array_ |  |  |  |
| `allowed_users` _string array_ |  |  |  |
| `allowed_groups` _string array_ |  |  |  |
| `automount_service_account_token` _boolean_ |  |  |  |
| `pod_disruption_budget` _boolean_ |  |  |  |
| `print_pod_warning_events` _boolean_ |  |  |  |
| `use_service_account_image_pull_secrets` _boolean_ |  |  |  |
| `cleanup_resources_timeout` _string_ | A Go duration string such as 5m. The runner's toml decoder parses it into<br />time.Duration, so no bespoke CRD type is needed. |  | MaxLength: 32 <br />Pattern: `^\+?(0\|(([0-9]+(\.[0-9]*)?\|\.[0-9]+)(ns\|us\|µs\|μs\|ms\|s\|m\|h))+)$` <br /> |
| `retry_limit` _integer_ |  |  |  |
| `retry_limits` _object (keys:string, values:integer)_ |  |  |  |
| `retry_backoff_max` _integer_ |  |  |  |


#### KubernetesConfigMap







_Appears in:_
- [KubernetesVolumes](#kubernetesvolumes)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ |  |  |  |
| `mount_path` _string_ |  |  |  |
| `sub_path` _string_ |  |  |  |
| `read_only` _boolean_ |  |  |  |
| `items` _object (keys:string, values:string)_ |  |  |  |


#### KubernetesContainerCapabilities







_Appears in:_
- [KubernetesContainerSecurityContext](#kubernetescontainersecuritycontext)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `add` _[Capability](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.36/#capability-v1-core) array_ |  |  |  |
| `drop` _[Capability](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.36/#capability-v1-core) array_ |  |  |  |


#### KubernetesContainerLifecyle



KubernetesContainerLifecyle exposes PostStart and PreStop only. PostStart is
not ordered against the image ENTRYPOINT, and PreStop does not run when the
container crashes or exits on its own.



_Appears in:_
- [KubernetesConfig](#kubernetesconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `post_start` _[KubernetesLifecycleHandler](#kuberneteslifecyclehandler)_ |  |  |  |
| `pre_stop` _[KubernetesLifecycleHandler](#kuberneteslifecyclehandler)_ |  |  |  |


#### KubernetesContainerSecurityContext







_Appears in:_
- [KubernetesConfig](#kubernetesconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `capabilities` _[KubernetesContainerCapabilities](#kubernetescontainercapabilities)_ |  |  |  |
| `privileged` _boolean_ |  |  |  |
| `run_as_user` _integer_ |  |  |  |
| `run_as_group` _integer_ |  |  |  |
| `run_as_non_root` _boolean_ |  |  |  |
| `read_only_root_filesystem` _boolean_ |  |  |  |
| `allow_privilege_escalation` _boolean_ |  |  |  |
| `proc_mount` _string_ |  |  |  |
| `selinux_type` _string_ |  |  |  |
| `seccomp_profile` _[KubernetesSeccompProfile](#kubernetesseccompprofile)_ |  |  |  |
| `app_armor_profile` _[KubernetesAppArmorProfile](#kubernetesapparmorprofile)_ |  |  |  |


#### KubernetesDNSConfig







_Appears in:_
- [KubernetesConfig](#kubernetesconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `nameservers` _string array_ |  |  |  |
| `options` _[KubernetesDNSConfigOption](#kubernetesdnsconfigoption) array_ |  |  |  |
| `searches` _string array_ |  |  |  |


#### KubernetesDNSConfigOption







_Appears in:_
- [KubernetesDNSConfig](#kubernetesdnsconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ |  |  |  |
| `value` _string_ |  |  |  |


#### KubernetesEmptyDir







_Appears in:_
- [KubernetesVolumes](#kubernetesvolumes)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ |  |  |  |
| `mount_path` _string_ |  |  |  |
| `sub_path` _string_ |  |  |  |
| `medium` _string_ |  |  |  |
| `size_limit` _string_ |  |  |  |
| `mount_propagation` _string_ |  |  |  |


#### KubernetesHostAliases







_Appears in:_
- [KubernetesConfig](#kubernetesconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `ip` _string_ |  |  |  |
| `hostnames` _string array_ |  |  |  |


#### KubernetesHostPath







_Appears in:_
- [KubernetesVolumes](#kubernetesvolumes)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ |  |  |  |
| `mount_path` _string_ |  |  |  |
| `sub_path` _string_ |  |  |  |
| `read_only` _boolean_ |  |  |  |
| `host_path` _string_ |  |  |  |
| `mount_propagation` _string_ |  |  |  |


#### KubernetesLifecycleExecAction







_Appears in:_
- [KubernetesLifecycleHandler](#kuberneteslifecyclehandler)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `command` _string array_ |  |  |  |


#### KubernetesLifecycleHTTPGet







_Appears in:_
- [KubernetesLifecycleHandler](#kuberneteslifecyclehandler)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `host` _string_ |  |  |  |
| `http_headers` _[KubernetesLifecycleHTTPGetHeader](#kuberneteslifecyclehttpgetheader) array_ |  |  |  |
| `path` _string_ |  |  |  |
| `port` _integer_ |  |  |  |
| `scheme` _string_ |  |  |  |


#### KubernetesLifecycleHTTPGetHeader







_Appears in:_
- [KubernetesLifecycleHTTPGet](#kuberneteslifecyclehttpget)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ |  |  |  |
| `value` _string_ |  |  |  |


#### KubernetesLifecycleHandler







_Appears in:_
- [KubernetesContainerLifecyle](#kubernetescontainerlifecyle)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `exec` _[KubernetesLifecycleExecAction](#kuberneteslifecycleexecaction)_ |  |  |  |
| `http_get` _[KubernetesLifecycleHTTPGet](#kuberneteslifecyclehttpget)_ |  |  |  |
| `tcp_socket` _[KubernetesLifecycleTCPSocket](#kuberneteslifecycletcpsocket)_ |  |  |  |


#### KubernetesLifecycleTCPSocket







_Appears in:_
- [KubernetesLifecycleHandler](#kuberneteslifecyclehandler)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `host` _string_ |  |  |  |
| `port` _integer_ |  |  |  |


#### KubernetesNFS



KubernetesNFS is an NFS share mounted into the build pod. Upstream's
UnmarshalTOML rejects the volume unless name, mount_path, server and path are
all set, so those four are required here rather than optional.



_Appears in:_
- [KubernetesVolumes](#kubernetesvolumes)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ |  |  | MinLength: 1 <br /> |
| `mount_path` _string_ |  |  | MinLength: 1 <br /> |
| `sub_path` _string_ |  |  |  |
| `server` _string_ |  |  | MinLength: 1 <br /> |
| `path` _string_ |  |  | MinLength: 1 <br /> |
| `read_only` _boolean_ |  |  |  |


#### KubernetesNodeAffinity







_Appears in:_
- [KubernetesAffinity](#kubernetesaffinity)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `required_during_scheduling_ignored_during_execution` _[NodeSelector](#nodeselector)_ |  |  |  |
| `preferred_during_scheduling_ignored_during_execution` _[PreferredSchedulingTerm](#preferredschedulingterm) array_ |  |  |  |


#### KubernetesPVC







_Appears in:_
- [KubernetesVolumes](#kubernetesvolumes)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ |  |  |  |
| `mount_path` _string_ |  |  |  |
| `sub_path` _string_ |  |  |  |
| `read_only` _boolean_ |  |  |  |
| `mount_propagation` _string_ |  |  |  |


#### KubernetesPodAffinity







_Appears in:_
- [KubernetesAffinity](#kubernetesaffinity)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `required_during_scheduling_ignored_during_execution` _[PodAffinityTerm](#podaffinityterm) array_ |  |  |  |
| `preferred_during_scheduling_ignored_during_execution` _[WeightedPodAffinityTerm](#weightedpodaffinityterm) array_ |  |  |  |


#### KubernetesPodAntiAffinity







_Appears in:_
- [KubernetesAffinity](#kubernetesaffinity)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `required_during_scheduling_ignored_during_execution` _[PodAffinityTerm](#podaffinityterm) array_ |  |  |  |
| `preferred_during_scheduling_ignored_during_execution` _[WeightedPodAffinityTerm](#weightedpodaffinityterm) array_ |  |  |  |


#### KubernetesPodSecurityContext







_Appears in:_
- [KubernetesConfig](#kubernetesconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `fs_group` _integer_ |  |  |  |
| `run_as_group` _integer_ |  |  |  |
| `run_as_non_root` _boolean_ |  |  |  |
| `run_as_user` _integer_ |  |  |  |
| `supplemental_groups` _integer array_ |  |  |  |
| `selinux_type` _string_ |  |  |  |
| `app_armor_profile` _[KubernetesAppArmorProfile](#kubernetesapparmorprofile)_ |  |  |  |
| `seccomp_profile` _[KubernetesSeccompProfile](#kubernetesseccompprofile)_ |  |  |  |


#### KubernetesPodSpec



KubernetesPodSpec is an experimental gitlab-runner option that patches the
generated build pod spec. PatchType is one of merge, json, or strategic.



_Appears in:_
- [KubernetesConfig](#kubernetesconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ |  |  |  |
| `patch_path` _string_ |  |  |  |
| `patch` _string_ |  |  |  |
| `patch_type` _string_ |  |  |  |


#### KubernetesSeccompProfile



KubernetesSeccompProfile selects the seccomp profile for pod containers. The
runner silently drops a profile it cannot use, so the constraints here are
validated at admission instead.



_Appears in:_
- [KubernetesContainerSecurityContext](#kubernetescontainersecuritycontext)
- [KubernetesPodSecurityContext](#kubernetespodsecuritycontext)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `type` _string_ |  |  | Enum: [RuntimeDefault Localhost Unconfined] <br /> |
| `localhost_profile` _string_ |  |  |  |


#### KubernetesSecret







_Appears in:_
- [KubernetesVolumes](#kubernetesvolumes)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ |  |  |  |
| `mount_path` _string_ |  |  |  |
| `sub_path` _string_ |  |  |  |
| `read_only` _boolean_ |  |  |  |
| `items` _object (keys:string, values:string)_ |  |  |  |


#### KubernetesVolumes







_Appears in:_
- [KubernetesConfig](#kubernetesconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `host_path` _[KubernetesHostPath](#kuberneteshostpath) array_ |  |  |  |
| `pvc` _[KubernetesPVC](#kubernetespvc) array_ |  |  |  |
| `config_map` _[KubernetesConfigMap](#kubernetesconfigmap) array_ |  |  |  |
| `secret` _[KubernetesSecret](#kubernetessecret) array_ |  |  |  |
| `empty_dir` _[KubernetesEmptyDir](#kubernetesemptydir) array_ |  |  |  |
| `csi` _[KubernetesCSI](#kubernetescsi) array_ |  |  |  |
| `nfs` _[KubernetesNFS](#kubernetesnfs) array_ |  |  |  |


#### LabelSelector







_Appears in:_
- [PodAffinityTerm](#podaffinityterm)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `match_labels` _object (keys:string, values:string)_ |  |  |  |
| `match_expressions` _[NodeSelectorRequirement](#nodeselectorrequirement) array_ |  |  |  |


#### MultiRunner



MultiRunner is the Schema for the multirunners API





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `gitlab.k8s.alekc.dev/v1beta2` | | |
| `kind` _string_ | `MultiRunner` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.36/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[MultiRunnerSpec](#multirunnerspec)_ |  |  |  |
| `status` _[MultiRunnerStatus](#multirunnerstatus)_ |  |  |  |


#### MultiRunnerEntry







_Appears in:_
- [MultiRunnerSpec](#multirunnerspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ |  |  | MinLength: 1 <br /> |
| `authentication` _[GitlabAuth](#gitlabauth)_ |  |  |  |
| `executor_config` _[KubernetesConfig](#kubernetesconfig)_ |  |  |  |
| `environment` _string array_ |  |  |  |
| `limit` _integer_ | Limit caps the jobs this entry runs at once. Zero omits the key, and<br />upstream acquireBuild only enforces a limit when it is above zero, so the<br />entry is bounded by Concurrent alone. Both apply, lower wins. |  | Minimum: 0 <br /> |
| `request_concurrency` _integer_ | RequestConcurrency caps job requests in flight to GitLab, not jobs<br />running. Zero omits the key, and upstream GetRequestConcurrency returns<br />max(1, x), so absent means 1 and a raised Limit then fills slowly. |  | Minimum: 0 <br /> |


#### MultiRunnerSpec



MultiRunnerSpec defines the desired state of MultiRunner



_Appears in:_
- [MultiRunner](#multirunner)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `concurrent` _integer_ |  |  | Minimum: 1 <br /> |
| `log_level` _string_ |  |  | Enum: [panic fatal error warning info debug] <br /> |
| `log_format` _string_ |  |  | Enum: [runner text json] <br /> |
| `check_interval` _integer_ |  |  | Minimum: 3 <br /> |
| `sentry_dsn` _string_ | SentryDsn Enables tracking of all system level errors to Sentry. |  |  |
| `gitlab_instance_url` _string_ |  | https://gitlab.com/ |  |
| `runner_image` _string_ | RunnerImage overrides the gitlab-runner container image. Defaults to<br />DefaultRunnerImage when empty. |  | Optional: \{\} <br /> |
| `runner_resources` _[ResourceRequirements](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.36/#resourcerequirements-v1-core)_ | RunnerResources overrides the resource requests/limits of the runner<br />manager container. |  | Optional: \{\} <br /> |
| `runner_image_pull_policy` _[PullPolicy](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.36/#pullpolicy-v1-core)_ | RunnerImagePullPolicy overrides the runner container image pull policy. |  | Enum: [Always Never IfNotPresent] <br />Optional: \{\} <br /> |
| `runner_security_context` _[SecurityContext](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.36/#securitycontext-v1-core)_ | RunnerSecurityContext overrides the runner manager container security<br />context. |  | Optional: \{\} <br /> |
| `runner_node_selector` _object (keys:string, values:string)_ | RunnerNodeSelector constrains the runner manager pod to nodes carrying<br />these labels. Shapes the manager only; executor_config.node_selector<br />places job pods. |  | Optional: \{\} <br /> |
| `runner_tolerations` _[Toleration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.36/#toleration-v1-core) array_ | RunnerTolerations lets the runner manager pod schedule onto tainted<br />nodes. Native Kubernetes list shape, not the "key=value": "effect" map<br />that executor_config.node_tolerations takes. |  | Optional: \{\} <br /> |
| `runner_affinity` _[Affinity](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.36/#affinity-v1-core)_ | RunnerAffinity sets affinity on the runner manager pod. Prefer<br />RunnerNodeSelector for equality matching and reach for this only for<br />In / NotIn / Exists or a soft preference. |  | Optional: \{\} <br /> |
| `runner_priority_class_name` _string_ | RunnerPriorityClassName protects the runner manager pod from preemption.<br />A manager killed mid-job loses the jobs it was tracking, so it wants a<br />higher priority than the workloads it shares a node with. |  | Optional: \{\} <br /> |
| `caCertificate` _[CASource](#casource)_ | CACertificate, when set, provides a PEM CA bundle used to verify the<br />GitLab endpoint for both the operator's API calls and every runner<br />entry's own connection. Supply it inline (value) or from a Secret or<br />ConfigMap key. |  | Optional: \{\} <br /> |
| `entries` _[MultiRunnerEntry](#multirunnerentry) array_ |  |  | MaxItems: 100 <br />MinItems: 1 <br /> |


#### MultiRunnerStatus



MultiRunnerStatus defines the observed state of MultiRunner. The maps are
keyed by entry name.



_Appears in:_
- [MultiRunner](#multirunner)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `error` _string_ |  |  |  |
| `runner_ids` _object (keys:string, values:integer)_ | RunnerIDs holds the GitLab numeric id per entry name for managed runners. |  |  |
| `registration_hashes` _object (keys:string, values:string)_ | RegistrationHashes holds the create-options hash per entry name. |  |  |
| `token_expires_at` _object (keys:string, values:[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.36/#time-v1-meta))_ | TokenExpiresAt holds the managed runner token expiry per entry name. |  |  |
| `observed_generation` _integer_ | ObservedGeneration is the spec generation the controller last acted on. |  | Optional: \{\} <br /> |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.36/#condition-v1-meta) array_ | Conditions holds the latest observations of the runner state. |  | Optional: \{\} <br /> |
| `ready` _boolean_ |  |  |  |
| `config_map_version` _string_ |  |  |  |


#### NodeSelector







_Appears in:_
- [KubernetesNodeAffinity](#kubernetesnodeaffinity)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `node_selector_terms` _[NodeSelectorTerm](#nodeselectorterm) array_ |  |  |  |


#### NodeSelectorRequirement







_Appears in:_
- [LabelSelector](#labelselector)
- [NodeSelectorTerm](#nodeselectorterm)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `key` _string_ |  |  |  |
| `operator` _string_ |  |  |  |
| `values` _string array_ |  |  |  |


#### NodeSelectorTerm







_Appears in:_
- [NodeSelector](#nodeselector)
- [PreferredSchedulingTerm](#preferredschedulingterm)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `match_expressions` _[NodeSelectorRequirement](#nodeselectorrequirement) array_ |  |  |  |
| `match_fields` _[NodeSelectorRequirement](#nodeselectorrequirement) array_ |  |  |  |


#### PodAffinityTerm







_Appears in:_
- [KubernetesPodAffinity](#kubernetespodaffinity)
- [KubernetesPodAntiAffinity](#kubernetespodantiaffinity)
- [WeightedPodAffinityTerm](#weightedpodaffinityterm)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `label_selector` _[LabelSelector](#labelselector)_ |  |  |  |
| `namespaces` _string array_ |  |  |  |
| `topology_key` _string_ |  |  |  |
| `namespace_selector` _[LabelSelector](#labelselector)_ |  |  |  |
| `match_label_keys` _string array_ |  |  |  |
| `mismatch_label_keys` _string array_ |  |  |  |


#### PreferredSchedulingTerm







_Appears in:_
- [KubernetesNodeAffinity](#kubernetesnodeaffinity)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `weight` _integer_ |  |  |  |
| `preference` _[NodeSelectorTerm](#nodeselectorterm)_ |  |  |  |


#### Runner



Runner is the Schema for the runners API





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `gitlab.k8s.alekc.dev/v1beta2` | | |
| `kind` _string_ | `Runner` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.36/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[RunnerSpec](#runnerspec)_ |  |  |  |
| `status` _[RunnerStatus](#runnerstatus)_ |  |  |  |


#### RunnerCreateOptions



RunnerCreateOptions mirrors the POST /user/runners request body. It is only
used in managed mode.



_Appears in:_
- [GitlabAuth](#gitlabauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `runner_type` _string_ | RunnerType selects the scope of the runner to create. |  | Enum: [instance_type group_type project_type] <br /> |
| `group_id` _integer_ | GroupID is required when RunnerType is group_type. |  | Optional: \{\} <br /> |
| `project_id` _integer_ | ProjectID is required when RunnerType is project_type. |  | Optional: \{\} <br /> |
| `description` _string_ |  |  | Optional: \{\} <br /> |
| `paused` _boolean_ |  |  | Optional: \{\} <br /> |
| `locked` _boolean_ |  |  | Optional: \{\} <br /> |
| `run_untagged` _boolean_ |  |  | Optional: \{\} <br /> |
| `tag_list` _string array_ |  |  | Optional: \{\} <br /> |
| `access_level` _string_ |  |  | Optional: \{\} <br /> |
| `maximum_timeout` _integer_ |  |  | Optional: \{\} <br /> |
| `maintenance_note` _string_ |  |  | Optional: \{\} <br /> |


#### RunnerSpec



RunnerSpec defines the desired state of Runner



_Appears in:_
- [Runner](#runner)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `authentication` _[GitlabAuth](#gitlabauth)_ | Authentication configures how the runner authenticates to GitLab. |  |  |
| `gitlab_instance_url` _string_ |  | https://gitlab.com/ | Optional: \{\} <br /> |
| `log_level` _string_ |  |  | Enum: [panic fatal error warning info debug] <br /> |
| `concurrent` _integer_ |  |  | Minimum: 1 <br /> |
| `limit` _integer_ | Limit caps the jobs this entry runs at once. Zero omits the key, and<br />upstream acquireBuild only enforces a limit when it is above zero, so the<br />entry is bounded by Concurrent alone. Both apply, lower wins. |  | Minimum: 0 <br /> |
| `request_concurrency` _integer_ | RequestConcurrency caps job requests in flight to GitLab, not jobs<br />running. Zero omits the key, and upstream GetRequestConcurrency returns<br />max(1, x), so absent means 1 and a raised Limit then fills slowly. |  | Minimum: 0 <br /> |
| `check_interval` _integer_ |  |  | Minimum: 3 <br /> |
| `log_format` _string_ |  |  | Enum: [runner text json] <br /> |
| `executor_config` _[KubernetesConfig](#kubernetesconfig)_ |  |  |  |
| `environment` _string array_ | Environment contains custom environment variables injected to build environment |  | Optional: \{\} <br /> |
| `runner_image` _string_ | RunnerImage overrides the gitlab-runner container image. Defaults to<br />DefaultRunnerImage when empty. |  | Optional: \{\} <br /> |
| `runner_resources` _[ResourceRequirements](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.36/#resourcerequirements-v1-core)_ | RunnerResources overrides the resource requests/limits of the runner<br />manager container. |  | Optional: \{\} <br /> |
| `runner_image_pull_policy` _[PullPolicy](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.36/#pullpolicy-v1-core)_ | RunnerImagePullPolicy overrides the runner container image pull policy. |  | Enum: [Always Never IfNotPresent] <br />Optional: \{\} <br /> |
| `runner_security_context` _[SecurityContext](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.36/#securitycontext-v1-core)_ | RunnerSecurityContext overrides the runner manager container security<br />context. |  | Optional: \{\} <br /> |
| `runner_node_selector` _object (keys:string, values:string)_ | RunnerNodeSelector constrains the runner manager pod to nodes carrying<br />these labels. Shapes the manager only; executor_config.node_selector<br />places job pods. |  | Optional: \{\} <br /> |
| `runner_tolerations` _[Toleration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.36/#toleration-v1-core) array_ | RunnerTolerations lets the runner manager pod schedule onto tainted<br />nodes. Native Kubernetes list shape, not the "key=value": "effect" map<br />that executor_config.node_tolerations takes. |  | Optional: \{\} <br /> |
| `runner_affinity` _[Affinity](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.36/#affinity-v1-core)_ | RunnerAffinity sets affinity on the runner manager pod. Prefer<br />RunnerNodeSelector for equality matching and reach for this only for<br />In / NotIn / Exists or a soft preference. |  | Optional: \{\} <br /> |
| `runner_priority_class_name` _string_ | RunnerPriorityClassName protects the runner manager pod from preemption.<br />A manager killed mid-job loses the jobs it was tracking, so it wants a<br />higher priority than the workloads it shares a node with. |  | Optional: \{\} <br /> |
| `caCertificate` _[CASource](#casource)_ | CACertificate, when set, provides a PEM CA bundle used to verify the<br />GitLab endpoint for both the operator's API calls and the runner's own<br />connection. Supply it inline (value) or from a Secret/ConfigMap key. |  | Optional: \{\} <br /> |


#### RunnerStatus



RunnerStatus defines the observed state of Runner



_Appears in:_
- [Runner](#runner)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `error` _string_ |  |  |  |
| `runner_id` _integer_ | RunnerID is the numeric id GitLab assigned to a managed runner created<br />through the access-token path. Zero for bring-your-own-token runners. |  |  |
| `token_expires_at` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.36/#time-v1-meta)_ | TokenExpiresAt is GitLab's expiry for a managed runner token, if any. |  | Optional: \{\} <br /> |
| `registration_hash` _string_ | RegistrationHash captures the create options that produced the current<br />managed runner; a change forces a recreate. |  |  |
| `config_map_version` _string_ |  |  |  |
| `observed_generation` _integer_ | ObservedGeneration is the spec generation the controller last acted on. |  | Optional: \{\} <br /> |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.36/#condition-v1-meta) array_ | Conditions holds the latest observations of the runner state. |  | Optional: \{\} <br /> |
| `ready` _boolean_ | Ready indicates that all runner operations have completed and the object<br />is ready to serve. |  |  |


#### SecretKeySelector



SecretKeySelector points at a single key inside a Secret in the runner's
namespace. It mirrors corev1.SecretKeySelector but makes Key optional so it
can default to "token"; the upstream type marks Key required, which would
force every reference to spell it out.



_Appears in:_
- [TokenSource](#tokensource)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name of the Secret in the runner's namespace. |  | MinLength: 1 <br /> |
| `key` _string_ | Key holding the token. Defaults to "token" when omitted. |  | Optional: \{\} <br /> |
| `optional` _boolean_ | Optional, when true, lets a missing secret or key resolve to an empty<br />token instead of failing. |  | Optional: \{\} <br /> |


#### Service







_Appears in:_
- [KubernetesConfig](#kubernetesconfig)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ |  |  |  |
| `alias` _string_ |  |  |  |
| `command` _string array_ |  |  |  |
| `entrypoint` _string array_ |  |  |  |
| `environment` _string array_ |  |  |  |


#### TokenSource



TokenSource supplies a credential in one of two mutually exclusive ways: an
inline literal value, or a reference to a key inside a Kubernetes Secret in
the runner's namespace. Exactly one of Value / SecretKeyRef may be set.



_Appears in:_
- [GitlabAuth](#gitlabauth)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `value` _string_ | Value is the literal token. Convenient for testing; prefer SecretKeyRef<br />in production so the token is not stored in the object spec. |  | Optional: \{\} <br /> |
| `secret_key_ref` _[SecretKeySelector](#secretkeyselector)_ | SecretKeyRef reads the token from a Secret in the runner's namespace. The<br />referenced key defaults to "token" when Key is omitted. Optional is<br />honoured: when true, a missing secret or key resolves to an empty token<br />instead of failing. |  | Optional: \{\} <br /> |


#### WeightedPodAffinityTerm







_Appears in:_
- [KubernetesPodAffinity](#kubernetespodaffinity)
- [KubernetesPodAntiAffinity](#kubernetespodantiaffinity)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `weight` _integer_ |  |  |  |
| `pod_affinity_term` _[PodAffinityTerm](#podaffinityterm)_ |  |  |  |


