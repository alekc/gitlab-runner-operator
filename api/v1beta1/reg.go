package v1beta1

type GitlabRegInfo struct {
	RegisterNewRunnerOptions
	AuthToken string `json:"auth_token"`
	GitlabUrl string
}
