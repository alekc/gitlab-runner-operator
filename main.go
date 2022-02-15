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

package main

import (
	"flag"
	"gitlab.k8s.alekc.dev/internal/file"
	certutil "k8s.io/client-go/util/cert"
	"os"
	"path/filepath"
	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	gitlabv1beta1 "gitlab.k8s.alekc.dev/api/v1beta1"
	"gitlab.k8s.alekc.dev/controllers"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(gitlabv1beta1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	err := injectTLS()
	if err != nil {
		os.Exit(1)
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "e47991d1.k8s.alekc.dev",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&controllers.RunnerReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Runner")
		os.Exit(1)
	}
	if err = (&gitlabv1beta1.Runner{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "Runner")
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
func injectTLS() error {
	certPath := filepath.Join(os.TempDir(), "k8s-webhook-server", "serving-certs")
	if file.Exist(certPath + "/tls.crt") {
		// tls certificate already exists, there is nothing to do for us
		return nil
	}

	setupLog.Info("Generating self signed cert as no cert is provided")
	host, err := os.Hostname()
	if err != nil {
		setupLog.Error(err, "Failed to retrieve hostname for self-signed cert")
	}
	err = os.MkdirAll(certPath, 0755)
	if err != nil {
		setupLog.Error(err, "could not create folder for certs")
		return err
	}
	certBytes, keyBytes, err := certutil.GenerateSelfSignedCertKey(host, nil, nil)
	if err != nil {
		setupLog.Error(err, "could not generate sign key")
		return err
	}
	if err := os.WriteFile(certPath+"/tls.key", keyBytes, 0644); err != nil {
		setupLog.Error(err, "cannot write tls.key")
		return err
	}
	if err := os.WriteFile(certPath+"/tls.crt", certBytes, 0644); err != nil {
		setupLog.Error(err, "cannot write tls.crt")
		return err
	}
	setupLog.Info("certs has been generated")
	return nil

	// setupLog.Info("no certs has been found, generating self-signed certs")
	// // generate certs
	// serviceName := os.Getenv("SERVICE_NAME")
	// if serviceName == "" {
	// 	serviceName = "gitlab-runner-operator"
	// }
	// namespace, err := discovery.GetNamespace()
	// if err != nil {
	// 	setupLog.Error(err, "could not discover current namespace")
	// 	return err
	// }
	//
	// // CA config
	// ca := &x509.Certificate{
	// 	SerialNumber:          big.NewInt(2020),
	// 	NotBefore:             time.Now(),
	// 	NotAfter:              time.Now().AddDate(1, 0, 0),
	// 	IsCA:                  true,
	// 	ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
	// 	KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
	// 	BasicConstraintsValid: true,
	// }
	// // CA private key
	// caPrivKey, err := rsa.GenerateKey(cryptorand.Reader, 4096)
	// if err != nil {
	// 	setupLog.Error(err, "could not generate CA private key")
	// 	return err
	// }
	// // Self signed CA certificate
	// caBytes, err := x509.CreateCertificate(cryptorand.Reader, ca, ca, &caPrivKey.PublicKey, caPrivKey)
	// if err != nil {
	// 	setupLog.Error(err, "could not create CA cert")
	// 	return err
	// }
	// // PEM encode CA cert
	// caPEM := new(bytes.Buffer)
	// _ = pem.Encode(caPEM, &pem.Block{
	// 	Type:  "CERTIFICATE",
	// 	Bytes: caBytes,
	// })
	// dnsNames := []string{
	// 	serviceName,
	// 	fmt.Sprintf("%s.%s", namespace, serviceName),
	// 	fmt.Sprintf("%s.%s.svc", namespace, serviceName),
	// }
	// commonName := dnsNames[2]
	//
	// // server cert config
	// cert := &x509.Certificate{
	// 	DNSNames:     dnsNames,
	// 	SerialNumber: big.NewInt(1658),
	// 	Subject: pkix.Name{
	// 		CommonName: commonName,
	// 	},
	// 	NotBefore:    time.Now(),
	// 	NotAfter:     time.Now().AddDate(1, 0, 0),
	// 	SubjectKeyId: []byte{1, 2, 3, 4, 6},
	// 	ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
	// 	KeyUsage:     x509.KeyUsageDigitalSignature,
	// }
	// // server private key
	// serverPrivKey, err := rsa.GenerateKey(cryptorand.Reader, 4096)
	// if err != nil {
	// 	setupLog.Error(err, "could not create server private key")
	// 	return err
	// }
	// // sign the server cert
	// serverCertBytes, err := x509.CreateCertificate(cryptorand.Reader, cert, ca, &serverPrivKey.PublicKey, caPrivKey)
	// if err != nil {
	// 	setupLog.Error(err, "could not create server cert")
	// 	return err
	// }
	//
	// // PEM encode the  server cert and key
	// serverCertPEM := new(bytes.Buffer)
	// _ = pem.Encode(serverCertPEM, &pem.Block{
	// 	Type:  "CERTIFICATE",
	// 	Bytes: serverCertBytes,
	// })
	//
	// serverPrivKeyPEM := new(bytes.Buffer)
	// _ = pem.Encode(serverPrivKeyPEM, &pem.Block{
	// 	Type:  "RSA PRIVATE KEY",
	// 	Bytes: x509.MarshalPKCS1PrivateKey(serverPrivKey),
	// })
	// err = os.MkdirAll(certPath, 0755)
	// if err != nil {
	// 	setupLog.Error(err, "could not create folder for certs")
	// 	return err
	// }
	// if err := os.WriteFile(certPath+"/tls.crt", serverCertPEM.Bytes(), 0644); err != nil {
	// 	setupLog.Error(err, "cannot write tls.crt")
	// 	return err
	// }
	// if err := os.WriteFile(certPath+"/tls.crt", serverPrivKeyPEM.Bytes(), 0644); err != nil {
	// 	setupLog.Error(err, "cannot write tls.key")
	// 	return err
	// }
	// return nil
}
