package validate

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"gitlab.k8s.alekc.dev/api/v1beta2"
	"gitlab.k8s.alekc.dev/internal/generate"
	"gitlab.k8s.alekc.dev/internal/types"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func systemIDScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := v1beta2.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	return scheme
}

func systemIDRunner() *v1beta2.Runner {
	return &v1beta2.Runner{
		TypeMeta:   metav1.TypeMeta{APIVersion: "gitlab.k8s.alekc.dev/v1beta2", Kind: "Runner"},
		ObjectMeta: metav1.ObjectMeta{Name: "r1", Namespace: "ns", UID: "uid-1"},
	}
}

func getDeployment(t *testing.T, cl client.Client, runner *v1beta2.Runner) appsv1.Deployment {
	t.Helper()
	var d appsv1.Deployment
	key := client.ObjectKey{Namespace: runner.GetNamespace(), Name: runner.ChildName()}
	if err := cl.Get(context.Background(), key, &d); err != nil {
		t.Fatalf("deployment not found: %v", err)
	}
	return d
}

// configVolume returns the named volume, failing rather than asserting that it
// is the only one, so an unrelated volume added later does not break this.
func configVolume(t *testing.T, deployment appsv1.Deployment) corev1.Volume {
	t.Helper()
	for _, v := range deployment.Spec.Template.Spec.Volumes {
		if v.Name == "config" {
			return v
		}
	}
	t.Fatalf("no config volume in %+v", deployment.Spec.Template.Spec.Volumes)
	return corev1.Volume{}
}

// TestDeployment_SystemIDProjection covers the fix for #45: the derived
// system_id is projected into the config volume as .runner_system_id, so
// gitlab-runner finds a valid state file instead of minting a new identity.
func TestDeployment_SystemIDProjection(t *testing.T) {
	runner := systemIDRunner()
	want := generate.SystemID(runner.GetUID())

	cl := fake.NewClientBuilder().WithScheme(systemIDScheme(t)).Build()
	if _, err := Deployment(context.Background(), cl, runner, logr.Discard()); err != nil {
		t.Fatalf("Deployment: %v", err)
	}
	deployment := getDeployment(t, cl, runner)

	t.Run("annotated on the deployment and the pod template", func(t *testing.T) {
		if got := deployment.GetAnnotations()[types.SystemIDAnnotationKey]; got != want {
			t.Fatalf("deployment annotation = %q, want %q", got, want)
		}
		got := deployment.Spec.Template.GetAnnotations()[types.SystemIDAnnotationKey]
		if got != want {
			t.Fatalf("pod template annotation = %q, want %q", got, want)
		}
	})

	t.Run("config volume projects the secret and the system id", func(t *testing.T) {
		projected := configVolume(t, deployment).Projected
		if projected == nil {
			t.Fatal("config volume is not a projected volume")
		}

		var secretName, systemIDPath, fieldPath string
		for _, source := range projected.Sources {
			if source.Secret != nil {
				secretName = source.Secret.Name
			}
			if source.DownwardAPI != nil && len(source.DownwardAPI.Items) == 1 {
				systemIDPath = source.DownwardAPI.Items[0].Path
				if ref := source.DownwardAPI.Items[0].FieldRef; ref != nil {
					fieldPath = ref.FieldPath
				}
			}
		}

		if secretName != runner.ChildName() {
			t.Errorf("projected secret = %q, want %q", secretName, runner.ChildName())
		}
		// gitlab-runner hardcodes this filename next to config.toml.
		if systemIDPath != ".runner_system_id" {
			t.Errorf("system id path = %q, want .runner_system_id", systemIDPath)
		}
		if wantField := "metadata.annotations['system-id']"; fieldPath != wantField {
			t.Errorf("fieldRef = %q, want %q", fieldPath, wantField)
		}
	})

	// The projection only works if config.toml and the state file end up in the
	// same directory, which is what gitlab-runner derives the state path from.
	t.Run("config volume still mounts at the runner config dir", func(t *testing.T) {
		containers := deployment.Spec.Template.Spec.Containers
		if len(containers) == 0 {
			t.Fatal("deployment has no containers")
		}
		var mount corev1.VolumeMount
		for _, m := range containers[0].VolumeMounts {
			if m.Name == "config" {
				mount = m
			}
		}
		if mount.MountPath != "/etc/gitlab-runner/" {
			t.Fatalf("config mount = %+v, want /etc/gitlab-runner/", mount)
		}
		if mount.SubPath != "" {
			t.Fatalf("mount uses subPath %q, which would break secret auto-update", mount.SubPath)
		}
	})
}

// staleDeployment builds a deployment matching on config version and image, so
// only the system_id can trigger a roll. deploymentID and templateID seed the
// two annotation copies independently.
func staleDeployment(runner *v1beta2.Runner, deploymentID, templateID string) *appsv1.Deployment {
	annotate := func(systemID string) map[string]string {
		out := map[string]string{types.ConfigVersionAnnotationKey: runner.ConfigMapVersion()}
		if systemID != "" {
			out[types.SystemIDAnnotationKey] = systemID
		}
		return out
	}
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        runner.ChildName(),
			Namespace:   runner.GetNamespace(),
			Annotations: annotate(deploymentID),
		},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Annotations: annotate(templateID)},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{Name: "runner", Image: runner.RunnerImage()}},
				},
			},
		},
	}
}

// TestDeployment_RollsOnMissingSystemID covers the upgrade and the rollback
// path. Without the system_id check a deployment from an older operator would
// never roll, and reading the deployment-level copy rather than the pod
// template would miss a rollback that dropped only the template one.
func TestDeployment_RollsOnMissingSystemID(t *testing.T) {
	runner := systemIDRunner()
	want := generate.SystemID(runner.GetUID())

	cases := map[string]*appsv1.Deployment{
		"created by an older operator":   staleDeployment(runner, "", ""),
		"rolled back to an old template": staleDeployment(runner, want, ""),
	}

	for name, stale := range cases {
		t.Run(name, func(t *testing.T) {
			cl := fake.NewClientBuilder().WithScheme(systemIDScheme(t)).WithObjects(stale).Build()
			res, err := Deployment(context.Background(), cl, runner, logr.Discard())
			if err != nil {
				t.Fatalf("Deployment: %v", err)
			}
			if res == nil {
				t.Fatal("expected a requeue after rolling the deployment, got none")
			}

			rolled := getDeployment(t, cl, runner)
			if got := rolled.Spec.Template.GetAnnotations()[types.SystemIDAnnotationKey]; got != want {
				t.Fatalf("pod template annotation = %q, want %q", got, want)
			}
			if configVolume(t, rolled).Projected == nil {
				t.Fatal("rolled deployment did not get the projected config volume")
			}
		})
	}
}

// TestDeployment_StableSystemIDDoesNotRoll guards the other side: once the
// annotation is in place, reconciling again must be a no-op rather than an
// endless roll.
func TestDeployment_StableSystemIDDoesNotRoll(t *testing.T) {
	runner := systemIDRunner()
	cl := fake.NewClientBuilder().WithScheme(systemIDScheme(t)).Build()

	if _, err := Deployment(context.Background(), cl, runner, logr.Discard()); err != nil {
		t.Fatalf("first Deployment: %v", err)
	}
	first := getDeployment(t, cl, runner)

	res, err := Deployment(context.Background(), cl, runner, logr.Discard())
	if err != nil {
		t.Fatalf("second Deployment: %v", err)
	}
	if res != nil {
		t.Fatal("second reconcile rolled the deployment, expected a no-op")
	}

	second := getDeployment(t, cl, runner)
	if first.ResourceVersion != second.ResourceVersion {
		t.Fatalf("deployment was rewritten: %q then %q", first.ResourceVersion, second.ResourceVersion)
	}
}
