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

package controllers

import (
	"crypto/md5"
	"encoding/hex"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xanzy/go-gitlab"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	api2 "gitlab.k8s.alekc.dev/internal/api"

	gitlabv1beta1 "gitlab.k8s.alekc.dev/api/v1beta1"
	// +kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecsWithDefaultAndCustomReporters(t,
		"Controller Suite",
		[]Reporter{printer.NewlineReporter{}})
}

var _ = BeforeSuite(func() {

	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	// Expect(os.Setenv("USE_EXISTING_CLUSTER", "true")).To(Succeed())

	// read crds
	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
	}

	// start test environment cluster
	cfg, err := testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	// ensure that the schema is the latest up and running
	err = gitlabv1beta1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = gitlabv1beta1.AddToScheme(scheme.Scheme)
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
		Client: k8sManager.GetClient(),
		Log: controllerruntime.Log.
			WithName("controllers").
			WithName("GitlabRunner"),
		GitlabApiClient: &api2.MockedGitlabClient{
			OnRegister: func(config gitlabv1beta1.RegisterNewRunnerOptions) (string, error) {
				// here we create a unique hash representing a combination of registration token
				// and runner's tags, since any changes to these fields will cause the reregistration of the runner
				hash := md5.Sum([]byte(*config.Token + strings.Join(config.TagList, ",")))
				return hex.EncodeToString(hash[:]), nil
			},
			OnDeleteByTokens: func(token string) (*gitlab.Response, error) {
				return &gitlab.Response{}, nil
			},
		},
	}).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	// run the reconciler in a separate go routine
	go func() {
		err = k8sManager.Start(controllerruntime.SetupSignalHandler())
		Expect(err).ToNot(HaveOccurred())
	}()
}, 60)

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})
