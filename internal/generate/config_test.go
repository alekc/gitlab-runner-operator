package generate

import (
	"log"
	"testing"

	"gitlab.k8s.alekc.dev/api/v1beta1"
	"k8s.io/utils/pointer"
)

func TestConfigText(t *testing.T) {
	runner := &v1beta1.Runner{
		Spec: v1beta1.RunnerSpec{
			RegistrationConfig: v1beta1.RegisterNewRunnerOptions{
				Token: pointer.StringPtr("1"),
			},
			GitlabInstanceURL: "https://gitlab.com",
		},
	}
	text, hash, err := ConfigText(runner)
	if err != nil {
		t.Error(err)
	}
	log.Println("Text", text)
	log.Println("Hash", hash)
}

func TestConfigTexts(t *testing.T) {
	runner := &v1beta1.Runner{
		Spec: v1beta1.RunnerSpec{
			Runners: []v1beta1.RunnerRunner{
				{
					RegistrationConfig: v1beta1.RegisterNewRunnerOptions{
						Token: pointer.StringPtr("1"),
					},
					GitlabInstanceURL: "https://gitlab.com",
				},
				{
					RegistrationConfig: v1beta1.RegisterNewRunnerOptions{
						Token: pointer.StringPtr("2"),
					},
					GitlabInstanceURL: "https://gitlab.corp",
				},
			},
		},
	}
	text, hash, err := ConfigText(runner)
	if err != nil {
		t.Error(err)
	}
	log.Println("Text", text)
	log.Println("Hash", hash)
}
