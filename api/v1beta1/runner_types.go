/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1beta1

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// RunnerSpec defines the desired state of Runner
type RunnerSpec struct {
	RegistrationConfig RegisterNewRunnerOptions `json:"registration_config"`

	// +kubebuilder:validation:Optional
	GitlabInstanceURL string `json:"gitlab_instance_url,omitempty"`

	// +kubebuilder:validation:Enum=panic;fatal;error;warning;info;debug
	LogLevel string `json:"log_level,omitempty"`

	// +kubebuilder:validation:Minimum=1
	Concurrent int `json:"concurrent,omitempty"`

	// +kubebuilder:validation:Minimum=3
	// +kubebuilder:default:3
	CheckInterval int `json:"check_interval"`

	ExecutorConfig KubernetesConfig `json:"executor_config,omitempty"`
}

// RunnerStatus defines the observed state of Runner
type RunnerStatus struct {
	Error string `json:"error"`

	// LastRegistrationToken is the last token used for a successful authentication
	LastRegistrationToken string `json:"last_registration_token"`

	// LastRegistrationTags are last tags used in successful registration
	LastRegistrationTags []string `json:"last_registration_tags"`

	// AuthenticationToken obtained from the gitlab which can be used in runner configuration for authentication
	AuthenticationToken string `json:"authentication_token"`
	ConfigMapVersion    string `json:"config_map_version"`

	// Ready indicates that all runner operation has been completed and final object is ready to serve
	Ready bool `json:"ready"`
}

// Runner is the Schema for the runners API
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type Runner struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RunnerSpec   `json:"spec,omitempty"`
	Status RunnerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RunnerList contains a list of Runner
type RunnerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Runner `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Runner{}, &RunnerList{})
}

//nolint:lll
type KubernetesConfig struct {
	Host                        string `toml:"host,omitempty" json:"host,omitempty" long:"host" env:"KUBERNETES_HOST" description:"Optional Kubernetes master host URL (auto-discovery attempted if not specified)"`
	CertFile                    string `toml:"cert_file,omitempty" json:"cert_file,omitempty"  long:"cert-file" env:"KUBERNETES_CERT_FILE" description:"Optional Kubernetes master auth certificate"`
	KeyFile                     string `toml:"key_file,omitempty" json:"key_file,omitempty"  long:"key-file" env:"KUBERNETES_KEY_FILE" description:"Optional Kubernetes master auth private key"`
	CAFile                      string `toml:"ca_file,omitempty" json:"ca_file,omitempty"  long:"ca-file" env:"KUBERNETES_CA_FILE" description:"Optional Kubernetes master auth ca certificate"`
	BearerTokenOverwriteAllowed bool   `toml:"bearer_token_overwrite_allowed" json:"bearer_token_overwrite_allowed,omitempty"  long:"bearer_token_overwrite_allowed" env:"KUBERNETES_BEARER_TOKEN_OVERWRITE_ALLOWED" description:"Bool to authorize builds to specify their own bearer token for creation."`
	BearerToken                 string `toml:"bearer_token,omitempty" json:"bearer_token,omitempty"  long:"bearer_token" env:"KUBERNETES_BEARER_TOKEN" description:"Optional Kubernetes service account token used to start build pods."`
	Image                       string `toml:"image,omitempty" json:"image,omitempty"  long:"image" env:"KUBERNETES_IMAGE" description:"Default docker image to use for builds when none is specified"`
	Namespace                   string `toml:"namespace" json:"namespace,omitempty"  long:"namespace" env:"KUBERNETES_NAMESPACE" description:"Namespace to run Kubernetes jobs in"`
	NamespaceOverwriteAllowed   string `toml:"namespace_overwrite_allowed" json:"namespace_overwrite_allowed,omitempty"  long:"namespace_overwrite_allowed" env:"KUBERNETES_NAMESPACE_OVERWRITE_ALLOWED" description:"Regex to validate 'KUBERNETES_NAMESPACE_OVERWRITE' value"`
	Privileged                  bool   `toml:"privileged,omitzero" json:"privileged,omitempty"  long:"privileged" env:"KUBERNETES_PRIVILEGED" description:"Run all containers with the privileged flag enabled"`
	CPULimit                    string `toml:"cpu_limit,omitempty" json:"cpu_limit,omitempty"  long:"cpu-limit" env:"KUBERNETES_CPU_LIMIT" description:"The CPU allocation given to build containers"`
	CPULimitOverwriteMaxAllowed string `toml:"cpu_limit_overwrite_max_allowed,omitempty" json:"cpu_limit_overwrite_max_allowed,omitempty"  long:"cpu-limit-overwrite-max-allowed" env:"KUBERNETES_CPU_LIMIT_OVERWRITE_MAX_ALLOWED" description:"If set, the max amount the cpu limit can be set to. Used with the KUBERNETES_CPU_LIMIT variable in the build."`

	MemoryLimit                      string                       `toml:"memory_limit,omitempty" json:"memory_limit,omitempty"  long:"memory-limit" env:"KUBERNETES_MEMORY_LIMIT" description:"The amount of memory allocated to build containers"`
	MemoryLimitOverwriteMaxAllowed   string                       `toml:"memory_limit_overwrite_max_allowed,omitempty" json:"memory_limit_overwrite_max_allowed,omitempty"  long:"memory-limit-overwrite-max-allowed" env:"KUBERNETES_MEMORY_LIMIT_OVERWRITE_MAX_ALLOWED" description:"If set, the max amount the memory limit can be set to. Used with the KUBERNETES_MEMORY_LIMIT variable in the build."`
	ServiceCPULimit                  string                       `toml:"service_cpu_limit,omitempty" json:"service_cpu_limit,omitempty"  long:"service-cpu-limit" env:"KUBERNETES_SERVICE_CPU_LIMIT" description:"The CPU allocation given to build service containers"`
	ServiceMemoryLimit               string                       `toml:"service_memory_limit,omitempty" json:"service_memory_limit,omitempty"  long:"service-memory-limit" env:"KUBERNETES_SERVICE_MEMORY_LIMIT" description:"The amount of memory allocated to build service containers"`
	HelperCPULimit                   string                       `toml:"helper_cpu_limit,omitempty" json:"helper_cpu_limit,omitempty"  long:"helper-cpu-limit" env:"KUBERNETES_HELPER_CPU_LIMIT" description:"The CPU allocation given to build helper containers"`
	HelperMemoryLimit                string                       `toml:"helper_memory_limit,omitempty" json:"helper_memory_limit,omitempty"  long:"helper-memory-limit" env:"KUBERNETES_HELPER_MEMORY_LIMIT" description:"The amount of memory allocated to build helper containers"`
	CPURequest                       string                       `toml:"cpu_request,omitempty" json:"cpu_request,omitempty"  long:"cpu-request" env:"KUBERNETES_CPU_REQUEST" description:"The CPU allocation requested for build containers"`
	CPURequestOverwriteMaxAllowed    string                       `toml:"cpu_request_overwrite_max_allowed,omitempty" json:"cpu_request_overwrite_max_allowed,omitempty"  long:"cpu-request-overwrite-max-allowed" env:"KUBERNETES_CPU_REQUEST_OVERWRITE_MAX_ALLOWED" description:"If set, the max amount the cpu request can be set to. Used with the KUBERNETES_CPU_REQUEST variable in the build."`
	MemoryRequest                    string                       `toml:"memory_request,omitempty" json:"memory_request,omitempty"  long:"memory-request" env:"KUBERNETES_MEMORY_REQUEST" description:"The amount of memory requested from build containers"`
	MemoryRequestOverwriteMaxAllowed string                       `toml:"memory_request_overwrite_max_allowed,omitempty" json:"memory_request_overwrite_max_allowed,omitempty"  long:"memory-request-overwrite-max-allowed" env:"KUBERNETES_MEMORY_REQUEST_OVERWRITE_MAX_ALLOWED" description:"If set, the max amount the memory request can be set to. Used with the KUBERNETES_MEMORY_REQUEST variable in the build."`
	ServiceCPURequest                string                       `toml:"service_cpu_request,omitempty" json:"service_cpu_request,omitempty"  long:"service-cpu-request" env:"KUBERNETES_SERVICE_CPU_REQUEST" description:"The CPU allocation requested for build service containers"`
	ServiceMemoryRequest             string                       `toml:"service_memory_request,omitempty" json:"service_memory_request,omitempty"  long:"service-memory-request" env:"KUBERNETES_SERVICE_MEMORY_REQUEST" description:"The amount of memory requested for build service containers"`
	HelperCPURequest                 string                       `toml:"helper_cpu_request,omitempty" json:"helper_cpu_request,omitempty"  long:"helper-cpu-request" env:"KUBERNETES_HELPER_CPU_REQUEST" description:"The CPU allocation requested for build helper containers"`
	HelperMemoryRequest              string                       `toml:"helper_memory_request,omitempty" json:"helper_memory_request,omitempty"  long:"helper-memory-request" env:"KUBERNETES_HELPER_MEMORY_REQUEST" description:"The amount of memory requested for build helper containers"`
	PullPolicy                       string                       `toml:"pull_policy,omitempty" json:"pull_policy,omitempty"  long:"pull-policy" env:"KUBERNETES_PULL_POLICY" description:"Policy for if/when to pull a container image (never, if-not-present, always). The cluster default will be used if not set"`
	NodeSelector                     map[string]string            `toml:"node_selector,omitempty" json:"node_selector,omitempty"  long:"node-selector" env:"KUBERNETES_NODE_SELECTOR" description:"A toml table/json object of key=value. Value is expected to be a string. When set this will create pods on k8s nodes that match all the key=value pairs."`
	NodeTolerations                  map[string]string            `toml:"node_tolerations,omitempty" json:"node_tolerations,omitempty"  long:"node-tolerations" env:"KUBERNETES_NODE_TOLERATIONS" description:"A toml table/json object of key=value:effect. Value and effect are expected to be strings. When set, pods will tolerate the given taints. Only one toleration is supported through environment variable configuration."`
	Affinity                         KubernetesAffinity           `toml:"affinity,omitempty" json:"affinity,omitempty" long:"affinity" description:"Kubernetes Affinity setting that is used to select the node that spawns a pod"`
	ImagePullSecrets                 []string                     `toml:"image_pull_secrets,omitempty" json:"image_pull_secrets,omitempty"  long:"image-pull-secrets" env:"KUBERNETES_IMAGE_PULL_SECRETS" description:"A list of image pull secrets that are used for pulling docker image"`
	HelperImage                      string                       `toml:"helper_image,omitempty" json:"helper_image,omitempty"  long:"helper-image,omitempty" env:"KUBERNETES_HELPER_IMAGE" description:"[ADVANCED] Override the default helper image used to clone repos and upload artifacts"`
	TerminationGracePeriodSeconds    int64                        `toml:"terminationGracePeriodSeconds,omitzero" json:"terminationGracePeriodSeconds,omitempty"  long:"terminationGracePeriodSeconds" env:"KUBERNETES_TERMINATIONGRACEPERIODSECONDS" description:"Duration after the processes running in the pod are sent a termination signal and the time when the processes are forcibly halted with a kill signal."`
	PollInterval                     int                          `toml:"poll_interval,omitzero" json:"poll_interval,omitempty"  long:"poll-interval" env:"KUBERNETES_POLL_INTERVAL" description:"How frequently, in seconds, the runner will poll the Kubernetes pod it has just created to check its status"`
	PollTimeout                      int                          `toml:"poll_timeout,omitzero" json:"poll_timeout,omitempty"  long:"poll-timeout" env:"KUBERNETES_POLL_TIMEOUT" description:"The total amount of time, in seconds, that needs to pass before the runner will timeout attempting to connect to the pod it has just created (useful for queueing more builds that the cluster can handle at a time)"`
	PodLabels                        map[string]string            `toml:"pod_labels,omitempty" json:"pod_labels,omitempty"  long:"pod-labels" description:"A toml table/json object of key-value. Value is expected to be a string. When set, this will create pods with the given pod labels. Environment variables will be substituted for values here."`
	ServiceAccount                   string                       `toml:"service_account,omitempty" json:"service_account,omitempty"  long:"service-account" env:"KUBERNETES_SERVICE_ACCOUNT" description:"Executor pods will use this Service Account to talk to kubernetes API"`
	ServiceAccountOverwriteAllowed   string                       `toml:"service_account_overwrite_allowed" json:"service_account_overwrite_allowed,omitempty"  long:"service_account_overwrite_allowed" env:"KUBERNETES_SERVICE_ACCOUNT_OVERWRITE_ALLOWED" description:"Regex to validate 'KUBERNETES_SERVICE_ACCOUNT' value"`
	PodAnnotations                   map[string]string            `toml:"pod_annotations,omitempty" json:"pod_annotations,omitempty"  long:"pod-annotations" description:"A toml table/json object of key-value. Value is expected to be a string. When set, this will create pods with the given annotations. Can be overwritten in build with KUBERNETES_POD_ANNOTATION_* variables"`
	PodAnnotationsOverwriteAllowed   string                       `toml:"pod_annotations_overwrite_allowed" json:"pod_annotations_overwrite_allowed,omitempty"  long:"pod_annotations_overwrite_allowed" env:"KUBERNETES_POD_ANNOTATIONS_OVERWRITE_ALLOWED" description:"Regex to validate 'KUBERNETES_POD_ANNOTATIONS_*' values"`
	PodSecurityContext               KubernetesPodSecurityContext `toml:"pod_security_context,omitempty" json:"pod_security_context,omitempty"  namespace:"pod-security-context" description:"A security context attached to each build pod"`
	Volumes                          *KubernetesVolumes           `toml:"volumes" json:"volumes,omitempty"`
	Services                         []Service                    `toml:"services,omitempty" json:"services,omitempty"  description:"Add service that is started with container"`
}

type KubernetesVolumes struct {
	HostPaths  []KubernetesHostPath  `toml:"host_path" json:"host_path,omitempty"  description:"The host paths which will be mounted"`
	PVCs       []KubernetesPVC       `toml:"pvc" json:"pvc,omitempty" description:"The persistent volume claims that will be mounted"`
	ConfigMaps []KubernetesConfigMap `toml:"config_map" json:"config_map,omitempty"  description:"The config maps which will be mounted as volumes"`
	Secrets    []KubernetesSecret    `toml:"secret" json:"secret,omitempty"  description:"The secret maps which will be mounted"`
	EmptyDirs  []KubernetesEmptyDir  `toml:"empty_dir" json:"empty_dir,omitempty"  description:"The empty dirs which will be mounted"`
}

//nolint:lll
type KubernetesConfigMap struct {
	Name      string            `toml:"name" json:"name"  description:"The name of the volume and ConfigMap to use"`
	MountPath string            `toml:"mount_path" json:"mount_path"  description:"Path where volume should be mounted inside of container"`
	ReadOnly  bool              `toml:"read_only,omitempty" json:"read_only,omitempty"  description:"If this volume should be mounted read only"`
	Items     map[string]string `toml:"items,omitempty" json:"items,omitempty"  description:"Key-to-path mapping for keys from the config map that should be used."`
}

type KubernetesHostPath struct {
	Name      string `toml:"name" json:"name"  description:"The name of the volume"`
	MountPath string `toml:"mount_path" json:"mount_path"  description:"Path where volume should be mounted inside of container"`
	ReadOnly  bool   `toml:"read_only,omitempty" json:"read_only,omitempty"  description:"If this volume should be mounted read only"`
	HostPath  string `toml:"host_path,omitempty" json:"host_path,omitempty"  description:"Path from the host that should be mounted as a volume"`
}

type KubernetesPVC struct {
	Name      string `toml:"name" json:"name"  description:"The name of the volume and PVC to use"`
	MountPath string `toml:"mount_path" json:"mount_path"  description:"Path where volume should be mounted inside of container"`
	ReadOnly  bool   `toml:"read_only,omitempty" json:"read_only,omitempty"  description:"If this volume should be mounted read only"`
}

//nolint:lll
type KubernetesSecret struct {
	Name      string            `toml:"name" json:"name"  description:"The name of the volume and Secret to use"`
	MountPath string            `toml:"mount_path" json:"mount_path"  description:"Path where volume should be mounted inside of container"`
	ReadOnly  bool              `toml:"read_only,omitempty" json:"read_only,omitempty"  description:"If this volume should be mounted read only"`
	Items     map[string]string `toml:"items,omitempty" json:"items,omitempty"  description:"Key-to-path mapping for keys from the secret that should be used."`
}

type KubernetesEmptyDir struct {
	Name      string `toml:"name" json:"name"  description:"The name of the volume and EmptyDir to use"`
	MountPath string `toml:"mount_path" json:"mount_path"  description:"Path where volume should be mounted inside of container"`
	Medium    string `toml:"medium,omitempty" json:"medium,omitempty"  description:"Set to 'Memory' to have a tmpfs"`
}

//nolint:lll
type KubernetesPodSecurityContext struct {
	FSGroup            *int64  `toml:"fs_group,omitempty" json:"fs_group,omitempty"  long:"fs-group" env:"KUBERNETES_POD_SECURITY_CONTEXT_FS_GROUP" description:"A special supplemental group that applies to all containers in a pod"`
	RunAsGroup         *int64  `toml:"run_as_group,omitempty" json:"run_as_group,omitempty"  long:"run-as-group" env:"KUBERNETES_POD_SECURITY_CONTEXT_RUN_AS_GROUP" description:"The GID to run the entrypoint of the container process"`
	RunAsNonRoot       *bool   `toml:"run_as_non_root,omitempty" json:"run_as_non_root,omitempty"  long:"run-as-non-root" env:"KUBERNETES_POD_SECURITY_CONTEXT_RUN_AS_NON_ROOT" description:"Indicates that the container must run as a non-root user"`
	RunAsUser          *int64  `toml:"run_as_user,omitempty" json:"run_as_user,omitempty"  long:"run-as-user" env:"KUBERNETES_POD_SECURITY_CONTEXT_RUN_AS_USER" description:"The UID to run the entrypoint of the container process"`
	SupplementalGroups []int64 `toml:"supplemental_groups,omitempty" json:"supplemental_groups,omitempty"  long:"supplemental-groups" description:"A list of groups applied to the first process run in each container, in addition to the container's primary GID"`
}

type Service struct {
	Name  string `toml:"name" json:"name"  long:"name" description:"The image path for the service"`
	Alias string `toml:"alias,omitempty" json:"alias,omitempty"  long:"alias" description:"The alias of the service"`
}

// RegisterNewRunnerOptions represents the available RegisterNewRunner()
// options.
//
// GitLab API docs:
// https://docs.gitlab.com/ce/api/runners.html#register-a-new-runner
type RegisterNewRunnerOptions struct {
	Token          *string                       `url:"token" json:"token,omitempty"`
	TokenSecret    string                        `json:"token_secret,omitempty"`
	Description    *string                       `url:"description,omitempty" json:"description,omitempty"`
	Info           *RegisterNewRunnerInfoOptions `url:"info,omitempty" json:"info,omitempty"`
	Active         *bool                         `url:"active,omitempty" json:"active,omitempty"`
	Locked         *bool                         `url:"locked,omitempty" json:"locked,omitempty"`
	RunUntagged    *bool                         `url:"run_untagged,omitempty" json:"run_untagged,omitempty"`
	TagList        []string                      `url:"tag_list[],omitempty" json:"tag_list,omitempty"`
	MaximumTimeout *int                          `url:"maximum_timeout,omitempty" json:"maximum_timeout,omitempty"`
}

// RegisterNewRunnerInfoOptions represents the info hashmap parameter in
// RegisterNewRunnerOptions.
//
// GitLab API docs:
// https://docs.gitlab.com/ce/api/runners.html#register-a-new-runner
type RegisterNewRunnerInfoOptions struct {
	Name         *string `url:"name,omitempty" json:"name,omitempty"`
	Version      *string `url:"version,omitempty" json:"version,omitempty"`
	Revision     *string `url:"revision,omitempty" json:"revision,omitempty"`
	Platform     *string `url:"platform,omitempty" json:"platform,omitempty"`
	Architecture *string `url:"architecture,omitempty" json:"architecture,omitempty"`
}

//nolint:lll
type KubernetesAffinity struct {
	NodeAffinity *KubernetesNodeAffinity `toml:"node_affinity,omitempty" json:"node_affinity,omitempty" long:"node-affinity" description:"Node affinity is conceptually similar to nodeSelector -- it allows you to constrain which nodes your pod is eligible to be scheduled on, based on labels on the node."`
}

//nolint:lll
type KubernetesNodeAffinity struct {
	RequiredDuringSchedulingIgnoredDuringExecution  *NodeSelector             `toml:"required_during_scheduling_ignored_during_execution,omitempty" json:"required_during_scheduling_ignored_during_execution"`
	PreferredDuringSchedulingIgnoredDuringExecution []PreferredSchedulingTerm `toml:"preferred_during_scheduling_ignored_during_execution,omitempty" json:"preferred_during_scheduling_ignored_during_execution"`
}
type NodeSelector struct {
	NodeSelectorTerms []NodeSelectorTerm `toml:"node_selector_terms" json:"node_selector_terms"`
}

type PreferredSchedulingTerm struct {
	Weight     int32            `toml:"weight" json:"weight"`
	Preference NodeSelectorTerm `toml:"preference" json:"preference"`
}

type NodeSelectorTerm struct {
	MatchExpressions []NodeSelectorRequirement `toml:"match_expressions,omitempty" json:"match_expressions"`
	MatchFields      []NodeSelectorRequirement `toml:"match_fields,omitempty" json:"match_fields"`
}

//nolint:lll
type NodeSelectorRequirement struct {
	Key      string   `toml:"key,omitempty" json:"key"`
	Operator string   `toml:"operator,omitempty" json:"operator"`
	Values   []string `toml:"values,omitempty" json:"values"`
}

// GetAnnotation returns the annotation value for a given key.
// If the key doesn't exist, an empty string is returned
func (r *Runner) GetAnnotation(key string) string {
	annotations := r.GetAnnotations()
	if annotations == nil {
		return ""
	}
	if val, ok := annotations[key]; ok {
		return val
	}
	return ""
}

// GenerateOwnerReference returns a generated owner reference which can be used later to for any child object
func (r *Runner) GenerateOwnerReference() []metav1.OwnerReference {
	return []metav1.OwnerReference{{
		APIVersion:         GroupVersion.String(), // due to https://github.com/kubernetes/client-go/issues/541 type meta is empty
		Kind:               "Runner",
		Name:               r.Name,
		UID:                r.UID,
		Controller:         pointer.BoolPtr(true),
		BlockOwnerDeletion: nil,
	}}
}

// ChildName returns a name which should be used for all child objects generated by the runner
func (r *Runner) ChildName() string {
	return fmt.Sprintf("gitlab-runner-%s", r.Name)
}
