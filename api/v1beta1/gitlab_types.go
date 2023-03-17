package v1beta1

import api "k8s.io/api/core/v1"

type KubernetesConfig struct {
	Host                                              string                              `toml:"host" json:"host,omitempty" long:"host" env:"KUBERNETES_HOST" description:"Optional Kubernetes master host URL (auto-discovery attempted if not specified)"`
	CertFile                                          string                              `toml:"cert_file,omitempty" json:"cert_file,omitempty" long:"cert-file" env:"KUBERNETES_CERT_FILE" description:"Optional Kubernetes master auth certificate"`
	KeyFile                                           string                              `toml:"key_file,omitempty" json:"key_file,omitempty" long:"key-file" env:"KUBERNETES_KEY_FILE" description:"Optional Kubernetes master auth private key"`
	CAFile                                            string                              `toml:"ca_file,omitempty" json:"ca_file,omitempty" long:"ca-file" env:"KUBERNETES_CA_FILE" description:"Optional Kubernetes master auth ca certificate"`
	BearerTokenOverwriteAllowed                       *bool                               `toml:"bearer_token_overwrite_allowed,omitempty" json:"bearer_token_overwrite_allowed,omitempty" long:"bearer_token_overwrite_allowed" env:"KUBERNETES_BEARER_TOKEN_OVERWRITE_ALLOWED" description:"Bool to authorize builds to specify their own bearer token for creation."`
	BearerToken                                       string                              `toml:"bearer_token,omitempty" json:"bearer_token,omitempty" long:"bearer_token" env:"KUBERNETES_BEARER_TOKEN" description:"Optional Kubernetes service account token used to start build pods."`
	Image                                             string                              `toml:"image,omitempty" json:"image,omitempty" long:"image" env:"KUBERNETES_IMAGE" description:"Default docker image to use for builds when none is specified"`
	Namespace                                         string                              `toml:"namespace,omitempty" json:"namespace,omitempty" long:"namespace" env:"KUBERNETES_NAMESPACE" description:"Namespace to run Kubernetes jobs in"`
	NamespaceOverwriteAllowed                         string                              `toml:"namespace_overwrite_allowed,omitempty" json:"namespace_overwrite_allowed,omitempty" long:"namespace_overwrite_allowed" env:"KUBERNETES_NAMESPACE_OVERWRITE_ALLOWED" description:"Regex to validate 'KUBERNETES_NAMESPACE_OVERWRITE' value"`
	Privileged                                        *bool                               `toml:"privileged,omitzero" json:"privileged,omitempty" long:"privileged" env:"KUBERNETES_PRIVILEGED" description:"Run all containers with the privileged flag enabled"`
	RuntimeClassName                                  *string                             `toml:"runtime_class_name,omitempty" json:"runtime_class_name,omitempty" long:"runtime-class-name" env:"KUBERNETES_RUNTIME_CLASS_NAME" description:"A Runtime Class to use for all created pods, errors if the feature is unsupported by the cluster"`
	AllowPrivilegeEscalation                          *bool                               `toml:"allow_privilege_escalation,omitzero" json:"allow_privilege_escalation,omitempty" long:"allow-privilege-escalation" env:"KUBERNETES_ALLOW_PRIVILEGE_ESCALATION" description:"Run all containers with the security context allowPrivilegeEscalation flag enabled. When empty, it does not define the allowPrivilegeEscalation flag in the container SecurityContext and allows Kubernetes to use the default privilege escalation behavior."`
	CPULimit                                          string                              `toml:"cpu_limit,omitempty" json:"cpu_limit,omitempty" long:"cpu-limit" env:"KUBERNETES_CPU_LIMIT" description:"The CPU allocation given to build containers"`
	CPULimitOverwriteMaxAllowed                       string                              `toml:"cpu_limit_overwrite_max_allowed,omitempty" json:"cpu_limit_overwrite_max_allowed,omitempty" long:"cpu-limit-overwrite-max-allowed" env:"KUBERNETES_CPU_LIMIT_OVERWRITE_MAX_ALLOWED" description:"If set, the max amount the cpu limit can be set to. Used with the KUBERNETES_CPU_LIMIT variable in the build."`
	CPURequest                                        string                              `toml:"cpu_request,omitempty" json:"cpu_request,omitempty" long:"cpu-request" env:"KUBERNETES_CPU_REQUEST" description:"The CPU allocation requested for build containers"`
	CPURequestOverwriteMaxAllowed                     string                              `toml:"cpu_request_overwrite_max_allowed,omitempty" json:"cpu_request_overwrite_max_allowed,omitempty" long:"cpu-request-overwrite-max-allowed" env:"KUBERNETES_CPU_REQUEST_OVERWRITE_MAX_ALLOWED" description:"If set, the max amount the cpu request can be set to. Used with the KUBERNETES_CPU_REQUEST variable in the build."`
	MemoryLimit                                       string                              `toml:"memory_limit,omitempty" json:"memory_limit,omitempty" long:"memory-limit" env:"KUBERNETES_MEMORY_LIMIT" description:"The amount of memory allocated to build containers"`
	MemoryLimitOverwriteMaxAllowed                    string                              `toml:"memory_limit_overwrite_max_allowed,omitempty" json:"memory_limit_overwrite_max_allowed,omitempty" long:"memory-limit-overwrite-max-allowed" env:"KUBERNETES_MEMORY_LIMIT_OVERWRITE_MAX_ALLOWED" description:"If set, the max amount the memory limit can be set to. Used with the KUBERNETES_MEMORY_LIMIT variable in the build."`
	MemoryRequest                                     string                              `toml:"memory_request,omitempty" json:"memory_request,omitempty" long:"memory-request" env:"KUBERNETES_MEMORY_REQUEST" description:"The amount of memory requested from build containers"`
	MemoryRequestOverwriteMaxAllowed                  string                              `toml:"memory_request_overwrite_max_allowed,omitempty" json:"memory_request_overwrite_max_allowed,omitempty" long:"memory-request-overwrite-max-allowed" env:"KUBERNETES_MEMORY_REQUEST_OVERWRITE_MAX_ALLOWED" description:"If set, the max amount the memory request can be set to. Used with the KUBERNETES_MEMORY_REQUEST variable in the build."`
	EphemeralStorageLimit                             string                              `toml:"ephemeral_storage_limit,omitempty" json:"ephemeral_storage_limit,omitempty" long:"ephemeral-storage-limit" env:"KUBERNETES_EPHEMERAL_STORAGE_LIMIT" description:"The amount of ephemeral storage allocated to build containers"`
	EphemeralStorageLimitOverwriteMaxAllowed          string                              `toml:"ephemeral_storage_limit_overwrite_max_allowed,omitempty" json:"ephemeral_storage_limit_overwrite_max_allowed,omitempty" long:"ephemeral-storage-limit-overwrite-max-allowed" env:"KUBERNETES_EPHEMERAL_STORAGE_LIMIT_OVERWRITE_MAX_ALLOWED" description:"If set, the max amount the ephemeral limit can be set to. Used with the KUBERNETES_EPHEMERAL_STORAGE_LIMIT variable in the build."`
	EphemeralStorageRequest                           string                              `toml:"ephemeral_storage_request,omitempty" json:"ephemeral_storage_request,omitempty" long:"ephemeral-storage-request" env:"KUBERNETES_EPHEMERAL_STORAGE_REQUEST" description:"The amount of ephemeral storage requested from build containers"`
	EphemeralStorageRequestOverwriteMaxAllowed        string                              `toml:"ephemeral_storage_request_overwrite_max_allowed,omitempty" json:"ephemeral_storage_request_overwrite_max_allowed,omitempty" long:"ephemeral-storage-request-overwrite-max-allowed" env:"KUBERNETES_EPHEMERAL_STORAGE_REQUEST_OVERWRITE_MAX_ALLOWED" description:"If set, the max amount the ephemeral storage request can be set to. Used with the KUBERNETES_EPHEMERAL_STORAGE_REQUEST variable in the build."`
	ServiceCPULimit                                   string                              `toml:"service_cpu_limit,omitempty" json:"service_cpu_limit,omitempty" long:"service-cpu-limit" env:"KUBERNETES_SERVICE_CPU_LIMIT" description:"The CPU allocation given to build service containers"`
	ServiceCPULimitOverwriteMaxAllowed                string                              `toml:"service_cpu_limit_overwrite_max_allowed,omitempty" json:"service_cpu_limit_overwrite_max_allowed,omitempty" long:"service-cpu-limit-overwrite-max-allowed" env:"KUBERNETES_SERVICE_CPU_LIMIT_OVERWRITE_MAX_ALLOWED" description:"If set, the max amount the service cpu limit can be set to. Used with the KUBERNETES_SERVICE_CPU_LIMIT variable in the build."`
	ServiceCPURequest                                 string                              `toml:"service_cpu_request,omitempty" json:"service_cpu_request,omitempty" long:"service-cpu-request" env:"KUBERNETES_SERVICE_CPU_REQUEST" description:"The CPU allocation requested for build service containers"`
	ServiceCPURequestOverwriteMaxAllowed              string                              `toml:"service_cpu_request_overwrite_max_allowed,omitempty" json:"service_cpu_request_overwrite_max_allowed,omitempty" long:"service-cpu-request-overwrite-max-allowed" env:"KUBERNETES_SERVICE_CPU_REQUEST_OVERWRITE_MAX_ALLOWED" description:"If set, the max amount the service cpu request can be set to. Used with the KUBERNETES_SERVICE_CPU_REQUEST variable in the build."`
	ServiceMemoryLimit                                string                              `toml:"service_memory_limit,omitempty" json:"service_memory_limit,omitempty" long:"service-memory-limit" env:"KUBERNETES_SERVICE_MEMORY_LIMIT" description:"The amount of memory allocated to build service containers"`
	ServiceMemoryLimitOverwriteMaxAllowed             string                              `toml:"service_memory_limit_overwrite_max_allowed,omitempty" json:"service_memory_limit_overwrite_max_allowed,omitempty" long:"service-memory-limit-overwrite-max-allowed" env:"KUBERNETES_SERVICE_MEMORY_LIMIT_OVERWRITE_MAX_ALLOWED" description:"If set, the max amount the service memory limit can be set to. Used with the KUBERNETES_SERVICE_MEMORY_LIMIT variable in the build."`
	ServiceMemoryRequest                              string                              `toml:"service_memory_request,omitempty" json:"service_memory_request,omitempty" long:"service-memory-request" env:"KUBERNETES_SERVICE_MEMORY_REQUEST" description:"The amount of memory requested for build service containers"`
	ServiceMemoryRequestOverwriteMaxAllowed           string                              `toml:"service_memory_request_overwrite_max_allowed,omitempty" json:"service_memory_request_overwrite_max_allowed,omitempty" long:"service-memory-request-overwrite-max-allowed" env:"KUBERNETES_SERVICE_MEMORY_REQUEST_OVERWRITE_MAX_ALLOWED" description:"If set, the max amount the service memory request can be set to. Used with the KUBERNETES_SERVICE_MEMORY_REQUEST variable in the build."`
	ServiceEphemeralStorageLimit                      string                              `toml:"service_ephemeral_storage_limit,omitempty" json:"service_ephemeral_storage_limit,omitempty" long:"service-ephemeral_storage-limit" env:"KUBERNETES_SERVICE_EPHEMERAL_STORAGE_LIMIT" description:"The amount of ephemeral storage allocated to build service containers"`
	ServiceEphemeralStorageLimitOverwriteMaxAllowed   string                              `toml:"service_ephemeral_storage_limit_overwrite_max_allowed,omitempty" json:"service_ephemeral_storage_limit_overwrite_max_allowed,omitempty" long:"service-ephemeral_storage-limit-overwrite-max-allowed" env:"KUBERNETES_SERVICE_EPHEMERAL_STORAGE_LIMIT_OVERWRITE_MAX_ALLOWED" description:"If set, the max amount the service ephemeral storage limit can be set to. Used with the KUBERNETES_SERVICE_EPHEMERAL_STORAGE_LIMIT variable in the build."`
	ServiceEphemeralStorageRequest                    string                              `toml:"service_ephemeral_storage_request,omitempty" json:"service_ephemeral_storage_request,omitempty" long:"service-ephemeral_storage-request" env:"KUBERNETES_SERVICE_EPHEMERAL_STORAGE_REQUEST" description:"The amount of ephemeral storage requested for build service containers"`
	ServiceEphemeralStorageRequestOverwriteMaxAllowed string                              `toml:"service_ephemeral_storage_request_overwrite_max_allowed,omitempty" json:"service_ephemeral_storage_request_overwrite_max_allowed,omitempty" long:"service-ephemeral_storage-request-overwrite-max-allowed" env:"KUBERNETES_SERVICE_EPHEMERAL_STORAGE_REQUEST_OVERWRITE_MAX_ALLOWED" description:"If set, the max amount the service ephemeral storage request can be set to. Used with the KUBERNETES_SERVICE_EPHEMERAL_STORAGE_REQUEST variable in the build."`
	HelperCPULimit                                    string                              `toml:"helper_cpu_limit,omitempty" json:"helper_cpu_limit,omitempty" long:"helper-cpu-limit" env:"KUBERNETES_HELPER_CPU_LIMIT" description:"The CPU allocation given to build helper containers"`
	HelperCPULimitOverwriteMaxAllowed                 string                              `toml:"helper_cpu_limit_overwrite_max_allowed,omitempty" json:"helper_cpu_limit_overwrite_max_allowed,omitempty" long:"helper-cpu-limit-overwrite-max-allowed" env:"KUBERNETES_HELPER_CPU_LIMIT_OVERWRITE_MAX_ALLOWED" description:"If set, the max amount the helper cpu limit can be set to. Used with the KUBERNETES_HELPER_CPU_LIMIT variable in the build."`
	HelperCPURequest                                  string                              `toml:"helper_cpu_request,omitempty" json:"helper_cpu_request,omitempty" long:"helper-cpu-request" env:"KUBERNETES_HELPER_CPU_REQUEST" description:"The CPU allocation requested for build helper containers"`
	HelperCPURequestOverwriteMaxAllowed               string                              `toml:"helper_cpu_request_overwrite_max_allowed,omitempty" json:"helper_cpu_request_overwrite_max_allowed,omitempty" long:"helper-cpu-request-overwrite-max-allowed" env:"KUBERNETES_HELPER_CPU_REQUEST_OVERWRITE_MAX_ALLOWED" description:"If set, the max amount the helper cpu request can be set to. Used with the KUBERNETES_HELPER_CPU_REQUEST variable in the build."`
	HelperMemoryLimit                                 string                              `toml:"helper_memory_limit,omitempty" json:"helper_memory_limit,omitempty" long:"helper-memory-limit" env:"KUBERNETES_HELPER_MEMORY_LIMIT" description:"The amount of memory allocated to build helper containers"`
	HelperMemoryLimitOverwriteMaxAllowed              string                              `toml:"helper_memory_limit_overwrite_max_allowed,omitempty" json:"helper_memory_limit_overwrite_max_allowed,omitempty" long:"helper-memory-limit-overwrite-max-allowed" env:"KUBERNETES_HELPER_MEMORY_LIMIT_OVERWRITE_MAX_ALLOWED" description:"If set, the max amount the helper memory limit can be set to. Used with the KUBERNETES_HELPER_MEMORY_LIMIT variable in the build."`
	HelperMemoryRequest                               string                              `toml:"helper_memory_request,omitempty" json:"helper_memory_request,omitempty" long:"helper-memory-request" env:"KUBERNETES_HELPER_MEMORY_REQUEST" description:"The amount of memory requested for build helper containers"`
	HelperMemoryRequestOverwriteMaxAllowed            string                              `toml:"helper_memory_request_overwrite_max_allowed,omitempty" json:"helper_memory_request_overwrite_max_allowed,omitempty" long:"helper-memory-request-overwrite-max-allowed" env:"KUBERNETES_HELPER_MEMORY_REQUEST_OVERWRITE_MAX_ALLOWED" description:"If set, the max amount the helper memory request can be set to. Used with the KUBERNETES_HELPER_MEMORY_REQUEST variable in the build."`
	HelperEphemeralStorageLimit                       string                              `toml:"helper_ephemeral_storage_limit,omitempty" json:"helper_ephemeral_storage_limit,omitempty" long:"helper-ephemeral_storage-limit" env:"KUBERNETES_HELPER_EPHEMERAL_STORAGE_LIMIT" description:"The amount of ephemeral storage allocated to build helper containers"`
	HelperEphemeralStorageLimitOverwriteMaxAllowed    string                              `toml:"helper_ephemeral_storage_limit_overwrite_max_allowed,omitempty" json:"helper_ephemeral_storage_limit_overwrite_max_allowed,omitempty" long:"helper-ephemeral_storage-limit-overwrite-max-allowed" env:"KUBERNETES_HELPER_EPHEMERAL_STORAGE_LIMIT_OVERWRITE_MAX_ALLOWED" description:"If set, the max amount the helper ephemeral storage limit can be set to. Used with the KUBERNETES_HELPER_EPHEMERAL_STORAGE_LIMIT variable in the build."`
	HelperEphemeralStorageRequest                     string                              `toml:"helper_ephemeral_storage_request,omitempty" json:"helper_ephemeral_storage_request,omitempty" long:"helper-ephemeral_storage-request" env:"KUBERNETES_HELPER_EPHEMERAL_STORAGE_REQUEST" description:"The amount of ephemeral storage requested for build helper containers"`
	HelperEphemeralStorageRequestOverwriteMaxAllowed  string                              `toml:"helper_ephemeral_storage_request_overwrite_max_allowed,omitempty" json:"helper_ephemeral_storage_request_overwrite_max_allowed,omitempty" long:"helper-ephemeral_storage-request-overwrite-max-allowed" env:"KUBERNETES_HELPER_EPHEMERAL_STORAGE_REQUEST_OVERWRITE_MAX_ALLOWED" description:"If set, the max amount the helper ephemeral storage request can be set to. Used with the KUBERNETES_HELPER_EPHEMERAL_STORAGE_REQUEST variable in the build."`
	AllowedImages                                     []string                            `toml:"allowed_images,omitempty" json:"allowed_images,omitempty" long:"allowed-images" env:"KUBERNETES_ALLOWED_IMAGES" description:"Image allowlist"`
	AllowedPullPolicies                               []string                            `toml:"allowed_pull_policies,omitempty" json:"allowed_pull_policies,omitempty" long:"allowed-pull-policies" env:"KUBERNETES_ALLOWED_PULL_POLICIES" description:"Pull policy allowlist"`
	AllowedServices                                   []string                            `toml:"allowed_services,omitempty" json:"allowed_services,omitempty" long:"allowed-services" env:"KUBERNETES_ALLOWED_SERVICES" description:"Service allowlist"`
	PullPolicy                                        []string                            `toml:"pull_policy,omitempty" json:"pull_policy,omitempty" long:"pull-policy" env:"KUBERNETES_PULL_POLICY" description:"Policy for if/when to pull a container image (never, if-not-present, always). The cluster default will be used if not set"`
	NodeSelector                                      map[string]string                   `toml:"node_selector,omitempty" json:"node_selector,omitempty" long:"node-selector" env:"KUBERNETES_NODE_SELECTOR" description:"A toml table/json object of key:value. Value is expected to be a string. When set this will create pods on k8s nodes that match all the key:value pairs. Only one selector is supported through environment variable configuration."`
	NodeSelectorOverwriteAllowed                      string                              `toml:"node_selector_overwrite_allowed" json:"node_selector_overwrite_allowed,omitempty" long:"node_selector_overwrite_allowed" env:"KUBERNETES_NODE_SELECTOR_OVERWRITE_ALLOWED" description:"Regex to validate 'KUBERNETES_NODE_SELECTOR_*' values"`
	NodeTolerations                                   map[string]string                   `toml:"node_tolerations,omitempty" json:"node_tolerations,omitempty" long:"node-tolerations" env:"KUBERNETES_NODE_TOLERATIONS" description:"A toml table/json object of key=value:effect. Value and effect are expected to be strings. When set, pods will tolerate the given taints. Only one toleration is supported through environment variable configuration."`
	Affinity                                          *KubernetesAffinity                 `toml:"affinity,omitempty" json:"affinity,omitempty" long:"affinity" description:"Kubernetes Affinity setting that is used to select the node that spawns a pod"`
	ImagePullSecrets                                  []string                            `toml:"image_pull_secrets,omitempty" json:"image_pull_secrets,omitempty" long:"image-pull-secrets" env:"KUBERNETES_IMAGE_PULL_SECRETS" description:"A list of image pull secrets that are used for pulling docker image"`
	HelperImage                                       string                              `toml:"helper_image,omitempty" json:"helper_image,omitempty" long:"helper-image" env:"KUBERNETES_HELPER_IMAGE" description:"[ADVANCED] Override the default helper image used to clone repos and upload artifacts"`
	HelperImageFlavor                                 string                              `toml:"helper_image_flavor,omitempty" json:"helper_image_flavor,omitempty" long:"helper-image-flavor" env:"KUBERNETES_HELPER_IMAGE_FLAVOR" description:"Set helper image flavor (alpine, ubuntu), defaults to alpine"`
	TerminationGracePeriodSeconds                     *int64                              `toml:"terminationGracePeriodSeconds,omitzero" json:"terminationGracePeriodSeconds,omitempty" long:"terminationGracePeriodSeconds" env:"KUBERNETES_TERMINATIONGRACEPERIODSECONDS" description:"Duration after the processes running in the pod are sent a termination signal and the time when the processes are forcibly halted with a kill signal.DEPRECATED: use KUBERNETES_POD_TERMINATION_GRACE_PERIOD_SECONDS and KUBERNETES_CLEANUP_GRACE_PERIOD_SECONDS instead."`
	PodTerminationGracePeriodSeconds                  *int64                              `toml:"pod_termination_grace_period_seconds,omitzero" json:"pod_termination_grace_period_seconds,omitempty" long:"pod_termination_grace_period_seconds" env:"KUBERNETES_POD_TERMINATION_GRACE_PERIOD_SECONDS" description:"Pod-level setting which determines the duration in seconds which the pod has to terminate gracefully. After this, the processes are forcibly halted with a kill signal. Ignored if KUBERNETES_TERMINATIONGRACEPERIODSECONDS is specified."`
	CleanupGracePeriodSeconds                         *int64                              `toml:"cleanup_grace_period_seconds,omitzero" json:"cleanup_grace_period_seconds,omitempty" long:"cleanup_grace_period_seconds" env:"KUBERNETES_CLEANUP_GRACE_PERIOD_SECONDS" description:"When cleaning up a pod on completion of a job, the duration in seconds which the pod has to terminate gracefully. After this, the processes are forcibly halted with a kill signal. Ignored if KUBERNETES_TERMINATIONGRACEPERIODSECONDS is specified."`
	PollInterval                                      int                                 `toml:"poll_interval,omitzero" json:"poll_interval,omitempty" long:"poll-interval" env:"KUBERNETES_POLL_INTERVAL" description:"How frequently, in seconds, the runner will poll the Kubernetes pod it has just created to check its status"`
	PollTimeout                                       int                                 `toml:"poll_timeout,omitzero" json:"poll_timeout,omitempty" long:"poll-timeout" env:"KUBERNETES_POLL_TIMEOUT" description:"The total amount of time, in seconds, that needs to pass before the runner will timeout attempting to connect to the pod it has just created (useful for queueing more builds that the cluster can handle at a time)"`
	ResourceAvailabilityCheckMaxAttempts              int                                 `toml:"resource_availability_check_max_attempts,omitzero" json:"resource_availability_check_max_attempts,omitempty" long:"resource-availability-check-max-attempts" env:"KUBERNETES_RESOURCE_AVAILABILITY_CHECK_MAX_ATTEMPTS" default:"5" description:"The maximum number of attempts to check if a resource (service account and/or pull secret) set is available before giving up. There is 5 seconds interval between each attempt"`
	PodLabels                                         map[string]string                   `toml:"pod_labels,omitempty" json:"pod_labels,omitempty" long:"pod-labels" description:"A toml table/json object of key-value. Value is expected to be a string. When set, this will create pods with the given pod labels. Environment variables will be substituted for values here."`
	PodLabelsOverwriteAllowed                         string                              `toml:"pod_labels_overwrite_allowed" json:"pod_labels_overwrite_allowed,omitempty" long:"pod_labels_overwrite_allowed" env:"KUBERNETES_POD_LABELS_OVERWRITE_ALLOWED" description:"Regex to validate 'KUBERNETES_POD_LABELS_*' values"`
	SchedulerName                                     string                              `toml:"scheduler_name,omitempty" json:"scheduler_name,omitempty" long:"scheduler-name,omitempty" env:"KUBERNETES_SCHEDULER_NAME" description:"Pods will be scheduled using this scheduler, if it exists"`
	ServiceAccount                                    string                              `toml:"service_account,omitempty" json:"service_account,omitempty" long:"service-account" env:"KUBERNETES_SERVICE_ACCOUNT" description:"Executor pods will use this Service Account to talk to kubernetes API"`
	ServiceAccountOverwriteAllowed                    string                              `toml:"service_account_overwrite_allowed,omitempty" json:"service_account_overwrite_allowed,omitempty" long:"service_account_overwrite_allowed" env:"KUBERNETES_SERVICE_ACCOUNT_OVERWRITE_ALLOWED" description:"Regex to validate 'KUBERNETES_SERVICE_ACCOUNT' value"`
	PodAnnotations                                    map[string]string                   `toml:"pod_annotations,omitempty" json:"pod_annotations,omitempty" long:"pod-annotations" description:"A toml table/json object of key-value. Value is expected to be a string. When set, this will create pods with the given annotations. Can be overwritten in build with KUBERNETES_POD_ANNOTATION_* variables"`
	PodAnnotationsOverwriteAllowed                    string                              `toml:"pod_annotations_overwrite_allowed,omitempty" json:"pod_annotations_overwrite_allowed,omitempty" long:"pod_annotations_overwrite_allowed" env:"KUBERNETES_POD_ANNOTATIONS_OVERWRITE_ALLOWED" description:"Regex to validate 'KUBERNETES_POD_ANNOTATIONS_*' values"`
	PodSecurityContext                                *KubernetesPodSecurityContext       `toml:"pod_security_context,omitempty" json:"pod_security_context,omitempty" namespace:"pod-security-context" description:"A security context attached to each build pod"`
	InitPermissionsContainerSecurityContext           *KubernetesContainerSecurityContext `toml:"init_permissions_container_security_context,omitempty" json:"init_permissions_container_security_context,omitempty" namespace:"init_permissions_container_security_context" description:"A security context attached to the init-permissions container inside the build pod"`
	BuildContainerSecurityContext                     *KubernetesContainerSecurityContext `toml:"build_container_security_context,omitempty" json:"build_container_security_context,omitempty" namespace:"build_container_security_context" description:"A security context attached to the build container inside the build pod"`
	HelperContainerSecurityContext                    *KubernetesContainerSecurityContext `toml:"helper_container_security_context,omitempty" json:"helper_container_security_context,omitempty" namespace:"helper_container_security_context" description:"A security context attached to the helper container inside the build pod"`
	ServiceContainerSecurityContext                   *KubernetesContainerSecurityContext `toml:"service_container_security_context,omitempty" json:"service_container_security_context,omitempty" namespace:"service_container_security_context" description:"A security context attached to the service containers inside the build pod"`
	Volumes                                           *KubernetesVolumes                  `toml:"volumes,omitempty" json:"volumes,omitempty" json:"volumes,omitempty"`
	HostAliases                                       []KubernetesHostAliases             `toml:"host_aliases,omitempty" json:"host_aliases,omitempty" long:"host_aliases" description:"Add a custom host-to-IP mapping"`
	Services                                          []Service                           `toml:"services,omitempty" json:"services,omitempty" description:"Add service that is started with container"`
	CapAdd                                            []string                            `toml:"cap_add,omitempty" json:"cap_add,omitempty" long:"cap-add" env:"KUBERNETES_CAP_ADD" description:"Add Linux capabilities"`
	CapDrop                                           []string                            `toml:"cap_drop,omitempty" json:"cap_drop,omitempty" long:"cap-drop" env:"KUBERNETES_CAP_DROP" description:"Drop Linux capabilities"`
	DNSPolicy                                         string                              `toml:"dns_policy,omitempty" json:"dns_policy,omitempty" long:"dns-policy" env:"KUBERNETES_DNS_POLICY" description:"How Kubernetes should try to resolve DNS from the created pods. If unset, Kubernetes will use the default 'ClusterFirst'. Valid values are: none, default, cluster-first, cluster-first-with-host-net"`
	DNSConfig                                         *KubernetesDNSConfig                `toml:"dns_config" json:"dns_config,omitempty" description:"Pod DNS config"`
	ContainerLifecycle                                *KubernetesContainerLifecyle        `toml:"container_lifecycle,omitempty" json:"container_lifecycle,omitempty" description:"Actions that the management system should take in response to container lifecycle events"`
	PriorityClassName                                 string                              `toml:"priority_class_name,omitempty" json:"priority_class_name,omitempty" long:"priority_class_name" env:"KUBERNETES_PRIORITY_CLASS_NAME" description:"If set, the Kubernetes Priority Class to be set to the Pods"`
}

type KubernetesDNSConfig struct {
	Nameservers []string                    `toml:"nameservers" json:"nameservers,omitempty" description:"A list of IP addresses that will be used as DNS servers for the Pod."`
	Options     []KubernetesDNSConfigOption `toml:"options" json:"options,omitempty" description:"An optional list of objects where each object may have a name property (required) and a value property (optional)."`
	Searches    []string                    `toml:"searches" json:"searches,omitempty" description:"A list of DNS search domains for hostname lookup in the Pod."`
}

type KubernetesDNSConfigOption struct {
	Name  string  `toml:"name" json:"name" json:"name"`
	Value *string `toml:"value,omitempty" json:"value,omitempty" json:"value,omitempty"`
}

type KubernetesVolumes struct {
	HostPaths  []KubernetesHostPath  `toml:"host_path" json:"host_path,omitempty" description:"The host paths which will be mounted"`
	PVCs       []KubernetesPVC       `toml:"pvc" json:"pvc,omitempty" description:"The persistent volume claims that will be mounted"`
	ConfigMaps []KubernetesConfigMap `toml:"config_map" json:"config_map,omitempty" description:"The config maps which will be mounted as volumes"`
	Secrets    []KubernetesSecret    `toml:"secret" json:"secret,omitempty"  description:"The secret maps which will be mounted"`
	EmptyDirs  []KubernetesEmptyDir  `toml:"empty_dir" json:"empty_dir,omitempty" description:"The empty dirs which will be mounted"`
	CSIs       []KubernetesCSI       `toml:"csi" json:"csi,omitempty" description:"The CSI volumes which will be mounted"`
}

type KubernetesConfigMap struct {
	Name      string            `toml:"name" json:"name" description:"The name of the volume and ConfigMap to use"`
	MountPath string            `toml:"mount_path" json:"mount_path" description:"Path where volume should be mounted inside of container"`
	SubPath   string            `toml:"sub_path,omitempty" json:"sub_path,omitempty" description:"The sub-path of the volume to mount (defaults to volume root)"`
	ReadOnly  bool              `toml:"read_only,omitempty" json:"read_only,omitempty" description:"If this volume should be mounted read only"`
	Items     map[string]string `toml:"items,omitempty" json:"items,omitempty" description:"Key-to-path mapping for keys from the config map that should be used."`
}

type KubernetesHostPath struct {
	Name      string `toml:"name" json:"name" description:"The name of the volume"`
	MountPath string `toml:"mount_path" json:"mount_path" description:"Path where volume should be mounted inside of container"`
	SubPath   string `toml:"sub_path,omitempty" json:"sub_path,omitempty" description:"The sub-path of the volume to mount (defaults to volume root)"`
	ReadOnly  bool   `toml:"read_only,omitempty" json:"read_only,omitempty" description:"If this volume should be mounted read only"`
	HostPath  string `toml:"host_path,omitempty" json:"host_path,omitempty" description:"Path from the host that should be mounted as a volume"`
}

type KubernetesPVC struct {
	Name      string `toml:"name" json:"name" description:"The name of the volume and PVC to use"`
	MountPath string `toml:"mount_path" json:"mount_path" description:"Path where volume should be mounted inside of container"`
	SubPath   string `toml:"sub_path,omitempty" json:"sub_path,omitempty" description:"The sub-path of the volume to mount (defaults to volume root)"`
	ReadOnly  bool   `toml:"read_only,omitempty" json:"read_only,omitempty" description:"If this volume should be mounted read only"`
}

type KubernetesSecret struct {
	Name      string            `toml:"name" json:"name" description:"The name of the volume and Secret to use"`
	MountPath string            `toml:"mount_path" json:"mount_path" description:"Path where volume should be mounted inside of container"`
	SubPath   string            `toml:"sub_path,omitempty" json:"sub_path,omitempty" description:"The sub-path of the volume to mount (defaults to volume root)"`
	ReadOnly  bool              `toml:"read_only,omitempty" json:"read_only,omitempty" description:"If this volume should be mounted read only"`
	Items     map[string]string `toml:"items,omitempty" json:"items,omitempty" description:"Key-to-path mapping for keys from the secret that should be used."`
}

type KubernetesEmptyDir struct {
	Name      string `toml:"name" json:"name" description:"The name of the volume and EmptyDir to use"`
	MountPath string `toml:"mount_path" json:"mount_path" description:"Path where volume should be mounted inside of container"`
	SubPath   string `toml:"sub_path,omitempty" json:"sub_path,omitempty" description:"The sub-path of the volume to mount (defaults to volume root)"`
	Medium    string `toml:"medium,omitempty" json:"medium,omitempty" description:"Set to 'Memory' to have a tmpfs"`
}

type KubernetesCSI struct {
	Name             string            `toml:"name" json:"name" description:"The name of the CSI volume and volumeMount to use"`
	MountPath        string            `toml:"mount_path" json:"mount_path" description:"Path where volume should be mounted inside of container"`
	SubPath          string            `toml:"sub_path,omitempty" json:"sub_path,omitempty" description:"The sub-path of the volume to mount (defaults to volume root)"`
	Driver           string            `toml:"driver" json:"driver" description:"A string value that specifies the name of the volume driver to use."`
	FSType           string            `toml:"fs_type" json:"fs_type" description:"Filesystem type to mount. If not provided, the empty value is passed to the associated CSI driver which will determine the default filesystem to apply."`
	ReadOnly         bool              `toml:"read_only,omitempty" json:"read_only,omitempty" description:"If this volume should be mounted read only"`
	VolumeAttributes map[string]string `toml:"volume_attributes,omitempty" json:"volume_attributes,omitempty" description:"Key-value pair mapping for attributes of the CSI volume."`
}

type KubernetesPodSecurityContext struct {
	FSGroup            *int64  `toml:"fs_group,omitempty" json:"fs_group,omitempty" long:"fs-group" env:"KUBERNETES_POD_SECURITY_CONTEXT_FS_GROUP" description:"A special supplemental group that applies to all containers in a pod"`
	RunAsGroup         *int64  `toml:"run_as_group,omitempty" json:"run_as_group,omitempty" long:"run-as-group" env:"KUBERNETES_POD_SECURITY_CONTEXT_RUN_AS_GROUP" description:"The GID to run the entrypoint of the container process"`
	RunAsNonRoot       *bool   `toml:"run_as_non_root,omitempty" json:"run_as_non_root,omitempty" long:"run-as-non-root" env:"KUBERNETES_POD_SECURITY_CONTEXT_RUN_AS_NON_ROOT" description:"Indicates that the container must run as a non-root user"`
	RunAsUser          *int64  `toml:"run_as_user,omitempty" json:"run_as_user,omitempty" long:"run-as-user" env:"KUBERNETES_POD_SECURITY_CONTEXT_RUN_AS_USER" description:"The UID to run the entrypoint of the container process"`
	SupplementalGroups []int64 `toml:"supplemental_groups,omitempty" json:"supplemental_groups,omitempty" long:"supplemental-groups" description:"A list of groups applied to the first process run in each container, in addition to the container's primary GID"`
}

type KubernetesContainerCapabilities struct {
	Add  []api.Capability `toml:"add" json:"add" long:"add" env:"@ADD" description:"List of capabilities to add to the build container"`
	Drop []api.Capability `toml:"drop" json:"drop" long:"drop" env:"@DROP" description:"List of capabilities to drop from the build container"`
}

type KubernetesContainerSecurityContext struct {
	Capabilities             *KubernetesContainerCapabilities `toml:"capabilities,omitempty" json:"capabilities,omitempty" namespace:"capabilities" description:"The capabilities to add/drop when running the container"`
	Privileged               *bool                            `toml:"privileged" json:"privileged,omitempty" long:"privileged" env:"@PRIVILEGED" description:"Run container in privileged mode"`
	RunAsUser                *int64                           `toml:"run_as_user,omitempty" json:"run_as_user,omitempty" long:"run-as-user" env:"@RUN_AS_USER" description:"The UID to run the entrypoint of the container process"`
	RunAsGroup               *int64                           `toml:"run_as_group,omitempty" json:"run_as_group,omitempty" long:"run-as-group" env:"@RUN_AS_GROUP" description:"The GID to run the entrypoint of the container process"`
	RunAsNonRoot             *bool                            `toml:"run_as_non_root,omitempty" json:"run_as_non_root,omitempty" long:"run-as-non-root" env:"@RUN_AS_NON_ROOT" description:"Indicates that the container must run as a non-root user"`
	ReadOnlyRootFilesystem   *bool                            `toml:"read_only_root_filesystem" json:"read_only_root_filesystem,omitempty" long:"read-only-root-filesystem" env:"@READ_ONLY_ROOT_FILESYSTEM" description:" Whether this container has a read-only root filesystem."`
	AllowPrivilegeEscalation *bool                            `toml:"allow_privilege_escalation" json:"allow_privilege_escalation,omitempty" long:"allow-privilege-escalation" env:"@ALLOW_PRIVILEGE_ESCALATION" description:"AllowPrivilegeEscalation controls whether a process can gain more privileges than its parent process"`
}

type KubernetesAffinity struct {
	NodeAffinity    *KubernetesNodeAffinity    `toml:"node_affinity,omitempty" json:"node_affinity,omitempty" long:"node-affinity" description:"Node affinity is conceptually similar to nodeSelector -- it allows you to constrain which nodes your pod is eligible to be scheduled on, based on labels on the node."`
	PodAffinity     *KubernetesPodAffinity     `toml:"pod_affinity,omitempty" json:"pod_affinity,omitempty" description:"Pod affinity allows to constrain which nodes your pod is eligible to be scheduled on based on the labels on other pods."`
	PodAntiAffinity *KubernetesPodAntiAffinity `toml:"pod_anti_affinity,omitempty" json:"pod_anti_affinity,omitempty" description:"Pod anti-affinity allows to constrain which nodes your pod is eligible to be scheduled on based on the labels on other pods."`
}

type KubernetesNodeAffinity struct {
	RequiredDuringSchedulingIgnoredDuringExecution  *NodeSelector             `toml:"required_during_scheduling_ignored_during_execution,omitempty" json:"required_during_scheduling_ignored_during_execution,omitempty" json:"required_during_scheduling_ignored_during_execution"`
	PreferredDuringSchedulingIgnoredDuringExecution []PreferredSchedulingTerm `toml:"preferred_during_scheduling_ignored_during_execution,omitempty" json:"preferred_during_scheduling_ignored_during_execution,omitempty" json:"preferred_during_scheduling_ignored_during_execution"`
}

type KubernetesPodAffinity struct {
	RequiredDuringSchedulingIgnoredDuringExecution  []PodAffinityTerm         `toml:"required_during_scheduling_ignored_during_execution,omitempty" json:"required_during_scheduling_ignored_during_execution,omitempty" json:"required_during_scheduling_ignored_during_execution"`
	PreferredDuringSchedulingIgnoredDuringExecution []WeightedPodAffinityTerm `toml:"preferred_during_scheduling_ignored_during_execution,omitempty" json:"preferred_during_scheduling_ignored_during_execution,omitempty" json:"preferred_during_scheduling_ignored_during_execution"`
}

type KubernetesPodAntiAffinity struct {
	RequiredDuringSchedulingIgnoredDuringExecution  []PodAffinityTerm         `toml:"required_during_scheduling_ignored_during_execution,omitempty" json:"required_during_scheduling_ignored_during_execution,omitempty" json:"required_during_scheduling_ignored_during_execution"`
	PreferredDuringSchedulingIgnoredDuringExecution []WeightedPodAffinityTerm `toml:"preferred_during_scheduling_ignored_during_execution,omitempty" json:"preferred_during_scheduling_ignored_during_execution,omitempty" json:"preferred_during_scheduling_ignored_during_execution"`
}

type KubernetesHostAliases struct {
	IP        string   `toml:"ip" json:"ip" long:"ip" description:"The IP address you want to attach hosts to"`
	Hostnames []string `toml:"hostnames" json:"hostnames" long:"hostnames" description:"A list of hostnames that will be attached to the IP"`
}

// https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.21/#lifecycle-v1-core
type KubernetesContainerLifecyle struct {
	PostStart *KubernetesLifecycleHandler `toml:"post_start,omitempty" json:"post_start,omitempty" description:"PostStart is called immediately after a container is created. If the handler fails, the container is terminated and restarted according to its restart policy. Other management of the container blocks until the hook completes"`
	PreStop   *KubernetesLifecycleHandler `toml:"pre_stop,omitempty" json:"pre_stop,omitempty" description:"PreStop is called immediately before a container is terminated due to an API request or management event such as liveness/startup probe failure, preemption, resource contention, etc. The handler is not called if the container crashes or exits. The reason for termination is passed to the handler. The Pod's termination grace period countdown begins before the PreStop hooked is executed. Regardless of the outcome of the handler, the container will eventually terminate within the Pod's termination grace period. Other management of the container blocks until the hook completes or until the termination grace period is reached"`
}

type KubernetesLifecycleHandler struct {
	Exec      *KubernetesLifecycleExecAction `toml:"exec" json:"exec"  description:"Exec specifies the action to take"`
	HTTPGet   *KubernetesLifecycleHTTPGet    `toml:"http_get" json:"http_get"  description:"HTTPGet specifies the http request to perform."`
	TCPSocket *KubernetesLifecycleTCPSocket  `toml:"tcp_socket" json:"tcp_socket"  description:"TCPSocket specifies an action involving a TCP port"`
}

type KubernetesLifecycleExecAction struct {
	Command []string `toml:"command" json:"command" description:"Command is the command line to execute inside the container, the working directory for the command  is root ('/') in the container's filesystem. The command is simply exec'd, it is not run inside a shell, so traditional shell instructions ('|', etc) won't work. To use a shell, you need to explicitly call out to that shell. Exit status of 0 is treated as live/healthy and non-zero is unhealthy"`
}

type KubernetesLifecycleHTTPGet struct {
	Host        string                             `toml:"host" json:"host" description:"Host name to connect to, defaults to the pod IP. You probably want to set \"Host\" in httpHeaders instead"`
	HTTPHeaders []KubernetesLifecycleHTTPGetHeader `toml:"http_headers" json:"http_headers" description:"Custom headers to set in the request. HTTP allows repeated headers"`
	Path        string                             `toml:"path" json:"path" description:"Path to access on the HTTP server"`
	Port        int                                `toml:"port" json:"port" description:"Number of the port to access on the container. Number must be in the range 1 to 65535"`
	Scheme      string                             `toml:"scheme" json:"scheme" description:"Scheme to use for connecting to the host. Defaults to HTTP"`
}

type KubernetesLifecycleHTTPGetHeader struct {
	Name  string `toml:"name" json:"name" description:"The header field name"`
	Value string `toml:"value" json:"value" description:"The header field value"`
}

type KubernetesLifecycleTCPSocket struct {
	Host string `toml:"host" json:"host" description:"Host name to connect to, defaults to the pod IP. You probably want to set \"Host\" in httpHeaders instead"`
	Port int    `toml:"port" json:"port" description:"Number of the port to access on the container. Number must be in the range 1 to 65535"`
}

type NodeSelector struct {
	NodeSelectorTerms []NodeSelectorTerm `toml:"node_selector_terms" json:"node_selector_terms" json:"node_selector_terms"`
}

type PreferredSchedulingTerm struct {
	Weight     int32            `toml:"weight" json:"weight" json:"weight"`
	Preference NodeSelectorTerm `toml:"preference" json:"preference" json:"preference"`
}

type WeightedPodAffinityTerm struct {
	Weight          int32           `toml:"weight" json:"weight" json:"weight"`
	PodAffinityTerm PodAffinityTerm `toml:"pod_affinity_term" json:"pod_affinity_term" json:"pod_affinity_term"`
}

type NodeSelectorTerm struct {
	MatchExpressions []NodeSelectorRequirement `toml:"match_expressions,omitempty" json:"match_expressions,omitempty" json:"match_expressions"`
	MatchFields      []NodeSelectorRequirement `toml:"match_fields,omitempty" json:"match_fields,omitempty" json:"match_fields"`
}

type NodeSelectorRequirement struct {
	Key      string   `toml:"key,omitempty" json:"key,omitempty" json:"key"`
	Operator string   `toml:"operator,omitempty" json:"operator,omitempty" json:"operator"`
	Values   []string `toml:"values,omitempty" json:"values,omitempty" json:"values"`
}

type PodAffinityTerm struct {
	LabelSelector     *LabelSelector `toml:"label_selector,omitempty" json:"label_selector,omitempty" json:"label_selector"`
	Namespaces        []string       `toml:"namespaces,omitempty" json:"namespaces,omitempty" json:"namespaces"`
	TopologyKey       string         `toml:"topology_key,omitempty" json:"topology_key,omitempty" json:"topology_key"`
	NamespaceSelector *LabelSelector `toml:"namespace_selector,omitempty" json:"namespace_selector,omitempty" json:"namespace_selector"`
}

type LabelSelector struct {
	MatchLabels      map[string]string         `toml:"match_labels,omitempty" json:"match_labels,omitempty" json:"match_labels"`
	MatchExpressions []NodeSelectorRequirement `toml:"match_expressions,omitempty" json:"match_expressions,omitempty" json:"match_expressions"`
}

type Service struct {
	Name       string   `toml:"name" json:"name" long:"name" description:"The image path for the service"`
	Alias      string   `toml:"alias,omitempty" json:"alias,omitempty" long:"alias" description:"The alias of the service"`
	Command    []string `toml:"command" json:"command" long:"command" description:"Command or script that should be used as the container’s command. Syntax is similar to https://docs.docker.com/engine/reference/builder/#cmd"`
	Entrypoint []string `toml:"entrypoint" json:"entrypoint" long:"entrypoint" description:"Command or script that should be executed as the container’s entrypoint. syntax is similar to https://docs.docker.com/engine/reference/builder/#entrypoint"`
}

// RegisterNewRunnerOptions represents the available RegisterNewRunner()
// options.
//
// GitLab API docs:
// https://docs.gitlab.com/ce/api/runners.html#register-a-new-runner
type RegisterNewRunnerOptions struct {
	Token       *string                       `url:"token" json:"token,omitempty"`
	Description *string                       `url:"description,omitempty" json:"description,omitempty"`
	Info        *RegisterNewRunnerInfoOptions `url:"info,omitempty" json:"info,omitempty"`
	// Active is deprecated. use paused instead
	Active          *bool    `url:"active,omitempty" json:"active,omitempty"`
	Paused          *bool    `url:"paused,omitempty" json:"paused,omitempty"`
	Locked          *bool    `url:"locked,omitempty" json:"locked,omitempty"`
	RunUntagged     *bool    `url:"run_untagged,omitempty" json:"run_untagged,omitempty"`
	TagList         []string `url:"tag_list[],omitempty" json:"tag_list,omitempty"`
	AccessLevel     *string  `url:"accessLevel,omitempty" json:"accessLevel,omitempty"`
	MaximumTimeout  *int     `url:"maximum_timeout,omitempty" json:"maximum_timeout,omitempty"`
	MaintenanceNote *string  `url:"maintenance_note,omitempty" json:"maintenance_note,omitempty"`
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
