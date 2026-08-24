package validate

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"gitlab.k8s.alekc.dev/api/v1beta2"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func placementRunner(spec v1beta2.RunnerSpec) *v1beta2.Runner {
	return &v1beta2.Runner{
		TypeMeta:   metav1.TypeMeta{APIVersion: "gitlab.k8s.alekc.dev/v1beta2", Kind: "Runner"},
		ObjectMeta: metav1.ObjectMeta{Name: "r1", Namespace: "ns", UID: "uid-1"},
		Spec:       spec,
	}
}

func fullPlacement() v1beta2.RunnerSpec {
	return v1beta2.RunnerSpec{
		RunnerNodeSelector: map[string]string{"node-pool": "ci", "kubernetes.io/arch": "arm64"},
		RunnerTolerations: []corev1.Toleration{{
			Key:      "dedicated",
			Operator: corev1.TolerationOpEqual,
			Value:    "ci",
			Effect:   corev1.TaintEffectNoSchedule,
		}},
		RunnerAffinity: &corev1.Affinity{
			NodeAffinity: &corev1.NodeAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
					NodeSelectorTerms: []corev1.NodeSelectorTerm{{
						MatchExpressions: []corev1.NodeSelectorRequirement{{
							Key:      "node-lifecycle",
							Operator: corev1.NodeSelectorOpNotIn,
							Values:   []string{"spot"},
						}},
					}},
				},
			},
		},
		RunnerPriorityClassName: "system-cluster-critical",
	}
}

// reconcile runs Deployment against a client already holding the runner, and
// reports whether it wrote anything. A nil result means it found no change.
func reconcile(t *testing.T, cl client.Client, runner *v1beta2.Runner) (rolled bool) {
	t.Helper()
	res, err := Deployment(context.Background(), cl, runner, logr.Discard())
	if err != nil {
		t.Fatalf("Deployment: %v", err)
	}
	return res != nil
}

// The four placement fields have to reach the pod template, which is the whole
// point of #83: executor_config places job pods, these place the manager.
func TestDeployment_PlacementProjection(t *testing.T) {
	runner := placementRunner(fullPlacement())
	cl := fake.NewClientBuilder().WithScheme(systemIDScheme(t)).Build()
	if !reconcile(t, cl, runner) {
		t.Fatal("first reconcile must create the deployment")
	}

	got := getDeployment(t, cl, runner).Spec.Template.Spec
	if !apiequality.Semantic.DeepEqual(got.NodeSelector, runner.Spec.RunnerNodeSelector) {
		t.Errorf("nodeSelector: got %v, want %v", got.NodeSelector, runner.Spec.RunnerNodeSelector)
	}
	if !apiequality.Semantic.DeepEqual(got.Tolerations, runner.Spec.RunnerTolerations) {
		t.Errorf("tolerations: got %v, want %v", got.Tolerations, runner.Spec.RunnerTolerations)
	}
	if !apiequality.Semantic.DeepEqual(got.Affinity, runner.Spec.RunnerAffinity) {
		t.Errorf("affinity: got %v, want %v", got.Affinity, runner.Spec.RunnerAffinity)
	}
	if got.PriorityClassName != runner.Spec.RunnerPriorityClassName {
		t.Errorf("priorityClassName: got %q, want %q", got.PriorityClassName, runner.Spec.RunnerPriorityClassName)
	}
}

// Unset must project nothing rather than an empty map or list, so a cluster
// with no placement configured keeps the scheduler's own defaults.
func TestDeployment_PlacementUnsetProjectsNothing(t *testing.T) {
	runner := placementRunner(v1beta2.RunnerSpec{})
	cl := fake.NewClientBuilder().WithScheme(systemIDScheme(t)).Build()
	reconcile(t, cl, runner)

	deployment := getDeployment(t, cl, runner)
	got := deployment.Spec.Template.Spec
	if got.NodeSelector != nil || got.Tolerations != nil || got.Affinity != nil || got.PriorityClassName != "" {
		t.Fatalf("expected no placement, got %+v", ManagerPodShape(&deployment.Spec.Template))
	}
}

// The reconcile must converge. Before #83 the comparison covered config, image
// and system_id only; a spec whose placement had just been written would have
// re-applied forever if the compare saw defaulted fields.
func TestDeployment_ReconcileConvergesOnSecondPass(t *testing.T) {
	for name, spec := range map[string]v1beta2.RunnerSpec{
		"unset":             {},
		"full placement":    fullPlacement(),
		"empty collections": {RunnerNodeSelector: map[string]string{}, RunnerTolerations: []corev1.Toleration{}},
	} {
		t.Run(name, func(t *testing.T) {
			runner := placementRunner(spec)
			cl := fake.NewClientBuilder().WithScheme(systemIDScheme(t)).Build()
			if !reconcile(t, cl, runner) {
				t.Fatal("first reconcile must create the deployment")
			}
			if rolled := reconcile(t, cl, runner); rolled {
				t.Fatal("second reconcile rolled an unchanged deployment")
			}
		})
	}
}

// Each field on its own must roll the pod. Accepting a placement field and not
// acting on it is the failure mode this replaces.
func TestDeployment_PlacementChangeRolls(t *testing.T) {
	for name, mutate := range map[string]func(*v1beta2.RunnerSpec){
		"node selector":  func(s *v1beta2.RunnerSpec) { s.RunnerNodeSelector["node-pool"] = "build" },
		"tolerations":    func(s *v1beta2.RunnerSpec) { s.RunnerTolerations[0].Value = "build" },
		"affinity":       func(s *v1beta2.RunnerSpec) { s.RunnerAffinity.NodeAffinity = nil },
		"priority class": func(s *v1beta2.RunnerSpec) { s.RunnerPriorityClassName = "" },
	} {
		t.Run(name, func(t *testing.T) {
			runner := placementRunner(fullPlacement())
			cl := fake.NewClientBuilder().WithScheme(systemIDScheme(t)).Build()
			reconcile(t, cl, runner)
			if reconcile(t, cl, runner) {
				t.Fatal("precondition: reconcile must be settled before the change")
			}

			mutate(&runner.Spec)
			if !reconcile(t, cl, runner) {
				t.Fatalf("a %s change did not roll the deployment", name)
			}
			if rolled := reconcile(t, cl, runner); rolled {
				t.Fatalf("a %s change did not settle after rolling", name)
			}
		})
	}
}

// runner_resources, runner_security_context and runner_env were silently inert
// before this change: none is in config.toml, so none moved the config hash.
func TestDeployment_ContainerFieldChangeRolls(t *testing.T) {
	runner := placementRunner(v1beta2.RunnerSpec{})
	cl := fake.NewClientBuilder().WithScheme(systemIDScheme(t)).Build()
	reconcile(t, cl, runner)

	runner.Spec.RunnerSecurityContext = &corev1.SecurityContext{}
	if !reconcile(t, cl, runner) {
		t.Fatal("a runner_security_context change did not roll the deployment")
	}
	if reconcile(t, cl, runner) {
		t.Fatal("precondition: reconcile must be settled before the runner_env change")
	}

	runner.Spec.RunnerEnv = []corev1.EnvVar{{Name: "HTTP_PROXY", Value: "http://proxy:3128"}}
	if !reconcile(t, cl, runner) {
		t.Fatal("a runner_env change did not roll the deployment")
	}
}

// A sidecar injected ahead of the runner container must not look like a change.
// Comparing Containers[0] would compare the sidecar image to the runner image
// and re-apply on every reconcile, fighting the injecting webhook.
func TestDeployment_InjectedSidecarDoesNotRoll(t *testing.T) {
	runner := placementRunner(fullPlacement())
	cl := fake.NewClientBuilder().WithScheme(systemIDScheme(t)).Build()
	reconcile(t, cl, runner)

	deployment := getDeployment(t, cl, runner)
	deployment.Spec.Template.Spec.Containers = append(
		[]corev1.Container{{Name: "istio-proxy", Image: "proxy:1"}},
		deployment.Spec.Template.Spec.Containers...,
	)
	if err := cl.Update(context.Background(), &deployment); err != nil {
		t.Fatalf("inject sidecar: %v", err)
	}

	if reconcile(t, cl, runner) {
		t.Fatal("an injected sidecar rolled the deployment")
	}
}

// mutatingClient stands in for a policy webhook that rewrites the manager pod
// on every write. The operator must not fight it: each reconcile would roll the
// pod and lose the jobs it was tracking.
type mutatingClient struct {
	client.Client
}

func (m mutatingClient) mutate(obj client.Object) {
	if deployment, ok := obj.(*appsv1.Deployment); ok {
		deployment.Spec.Template.Spec.NodeSelector = map[string]string{"injected-by": "policy"}
	}
}

// Both writes are intercepted. Overriding Update alone leaves the create path
// storing our own spec, so the shapes match on the next pass and the test
// passes without ever exercising the webhook.
func (m mutatingClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	m.mutate(obj)
	return m.Client.Create(ctx, obj, opts...)
}

func (m mutatingClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	m.mutate(obj)
	return m.Client.Update(ctx, obj, opts...)
}

func TestDeployment_SettlesAgainstAMutatingWebhook(t *testing.T) {
	runner := placementRunner(fullPlacement())
	cl := mutatingClient{Client: fake.NewClientBuilder().WithScheme(systemIDScheme(t)).Build()}
	reconcile(t, cl, runner)

	// The webhook overwrote our node selector, so the shapes will never match.
	// Settling on that is the point, and it has to hold on every later pass, not
	// just the second.
	for pass := 2; pass <= 4; pass++ {
		if reconcile(t, cl, runner) {
			t.Fatalf("pass %d kept trying to overwrite a mutating webhook instead of settling", pass)
		}
	}
}

// MultiRunner reaches the same pod template through types.RunnerInfo, but its
// four accessors are a separate copy. A slip returning nil, or reading the wrong
// field, compiles and would otherwise pass the whole suite.
func TestDeployment_MultiRunnerPlacementProjection(t *testing.T) {
	spec := fullPlacement()
	multi := &v1beta2.MultiRunner{
		TypeMeta:   metav1.TypeMeta{APIVersion: "gitlab.k8s.alekc.dev/v1beta2", Kind: "MultiRunner"},
		ObjectMeta: metav1.ObjectMeta{Name: "m1", Namespace: "ns", UID: "uid-m1"},
		Spec: v1beta2.MultiRunnerSpec{
			Entries:                 []v1beta2.MultiRunnerEntry{{Name: "e1"}},
			RunnerNodeSelector:      spec.RunnerNodeSelector,
			RunnerTolerations:       spec.RunnerTolerations,
			RunnerAffinity:          spec.RunnerAffinity,
			RunnerPriorityClassName: spec.RunnerPriorityClassName,
			RunnerEnv:               []corev1.EnvVar{{Name: "HTTP_PROXY", Value: "http://proxy:3128"}},
		},
	}

	cl := fake.NewClientBuilder().WithScheme(systemIDScheme(t)).Build()
	res, err := Deployment(context.Background(), cl, multi, logr.Discard())
	if err != nil {
		t.Fatalf("Deployment: %v", err)
	}
	if res == nil {
		t.Fatal("first reconcile must create the deployment")
	}

	var deployment appsv1.Deployment
	key := client.ObjectKey{Namespace: multi.GetNamespace(), Name: multi.ChildName()}
	if err := cl.Get(context.Background(), key, &deployment); err != nil {
		t.Fatalf("deployment not found: %v", err)
	}

	got := ManagerPodShape(&deployment.Spec.Template)
	want := PodShape{
		Image:             multi.RunnerImage(),
		ImagePullPolicy:   multi.RunnerImagePullPolicy(),
		Resources:         multi.RunnerResources(),
		SecurityContext:   multi.RunnerSecurityContext(),
		Env:               multi.RunnerEnv(),
		NodeSelector:      spec.RunnerNodeSelector,
		Tolerations:       spec.RunnerTolerations,
		Affinity:          spec.RunnerAffinity,
		PriorityClassName: spec.RunnerPriorityClassName,
	}
	if !apiequality.Semantic.DeepEqual(got, want) {
		t.Fatalf("MultiRunner manager pod shape:\n got: %+v\nwant: %+v", got, want)
	}
}
