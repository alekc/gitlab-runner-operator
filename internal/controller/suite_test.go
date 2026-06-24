/*
Copyright 2020 Alexander Chernov

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

package controller

import (
	"crypto/md5"
	"encoding/hex"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	api2 "gitlab.k8s.alekc.dev/internal/api"

	gitlabv1beta2 "gitlab.k8s.alekc.dev/api/v1beta2"
	// +kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func() {

	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	// Expect(os.Setenv("USE_EXISTING_CLUSTER", "true")).To(Succeed())

	// read crds
	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
	}

	// start test environment cluster
	cfg, err := testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	// ensure that the schema is the latest up and running
	err = gitlabv1beta2.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = gitlabv1beta2.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:scheme

	// create the test client
	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	// we also do need the manager
	k8sManager, err := controllerruntime.NewManager(cfg, controllerruntime.Options{
		Scheme: scheme.Scheme,
	})
	Expect(err).ToNot(HaveOccurred())

	err = (&RunnerReconciler{
		Client:    k8sManager.GetClient(),
		APIReader: k8sManager.GetAPIReader(),
		Log: controllerruntime.Log.
			WithName("controllers").
			WithName("GitlabRunner"),
		GitlabApiClient: &api2.MockedGitlabClient{
			OnCreateRunner: func(opts gitlabv1beta2.RunnerCreateOptions) (api2.CreatedRunner, error) {
				// deterministic token derived from the create options so that
				// changing them yields a new token and triggers a recreate
				hash := md5.Sum([]byte(opts.RunnerType + strings.Join(opts.TagList, ",")))
				return api2.CreatedRunner{ID: 1, Token: hex.EncodeToString(hash[:])}, nil
			},
			OnDeleteRunner: func(id int) error { return nil },
		},
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	err = (&MultiRunnerReconciler{
		Client:    k8sManager.GetClient(),
		APIReader: k8sManager.GetAPIReader(),
		GitlabApiClient: &api2.MockedGitlabClient{
			OnCreateRunner: func(opts gitlabv1beta2.RunnerCreateOptions) (api2.CreatedRunner, error) {
				hash := md5.Sum([]byte(opts.RunnerType + strings.Join(opts.TagList, ",")))
				return api2.CreatedRunner{ID: 2, Token: hex.EncodeToString(hash[:])}, nil
			},
			OnDeleteRunner: func(id int) error { return nil },
		},
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	// run the reconciler in a separate go routine
	go func() {
		err = k8sManager.Start(controllerruntime.SetupSignalHandler())
		Expect(err).ToNot(HaveOccurred())
	}()
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})
