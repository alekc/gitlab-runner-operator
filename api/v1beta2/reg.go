package v1beta2

type GitlabRegInfo struct {
	RegisterNewRunnerOptions
	AuthToken string `json:"auth_token"`
	GitlabUrl string
}
