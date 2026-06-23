package api

import (
	"io"

	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.k8s.alekc.dev/api/v1beta2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreatedRunner is the result of creating a managed runner on GitLab.
type CreatedRunner struct {
	ID             int
	Token          string
	TokenExpiresAt *metav1.Time
}

// GitlabClient drives the managed-runner lifecycle through the GitLab API. It
// is only used for managed runners (the access-token path);
// bring-your-own-token runners never call it.
type GitlabClient interface {
	// CreateRunner creates a runner via POST /user/runners and returns its
	// numeric id and authentication token.
	CreateRunner(opts v1beta2.RunnerCreateOptions) (CreatedRunner, error)
	// DeleteRunner removes a runner by its numeric id.
	DeleteRunner(id int) error
	// VerifyToken reports whether the given runner authentication token is
	// still accepted by GitLab. A false (with nil error) means the token was
	// rejected (revoked/expired); a non-nil error means the check itself failed.
	VerifyToken(token string) (bool, error)
	// RefreshToken resets a managed runner's authentication token in place
	// (without recreating the runner) and returns the new token and expiry.
	RefreshToken(id int) (CreatedRunner, error)
	// RunnerExists reports whether a runner with the given id still exists on
	// GitLab.
	RunnerExists(id int) (bool, error)
}

type gitlabApi struct {
	gitlabApiClient *gitlab.Client
}

func (g *gitlabApi) CreateRunner(opts v1beta2.RunnerCreateOptions) (CreatedRunner, error) {
	sdkOpts := &gitlab.CreateUserRunnerOptions{
		RunnerType:  gitlab.Ptr(opts.RunnerType),
		Paused:      opts.Paused,
		Locked:      opts.Locked,
		RunUntagged: opts.RunUntagged,
	}
	if opts.GroupID != nil {
		sdkOpts.GroupID = gitlab.Ptr(int64(*opts.GroupID))
	}
	if opts.ProjectID != nil {
		sdkOpts.ProjectID = gitlab.Ptr(int64(*opts.ProjectID))
	}
	if opts.Description != "" {
		sdkOpts.Description = gitlab.Ptr(opts.Description)
	}
	if len(opts.TagList) > 0 {
		tags := opts.TagList
		sdkOpts.TagList = &tags
	}
	if opts.AccessLevel != "" {
		sdkOpts.AccessLevel = gitlab.Ptr(opts.AccessLevel)
	}
	if opts.MaximumTimeout != nil {
		sdkOpts.MaximumTimeout = gitlab.Ptr(int64(*opts.MaximumTimeout))
	}
	if opts.MaintenanceNote != "" {
		sdkOpts.MaintenanceNote = gitlab.Ptr(opts.MaintenanceNote)
	}

	runner, resp, err := g.gitlabApiClient.Users.CreateUserRunner(sdkOpts)
	if err != nil {
		return CreatedRunner{}, err
	}
	defer closeBody(resp)

	created := CreatedRunner{ID: int(runner.ID), Token: runner.Token}
	if runner.TokenExpiresAt != nil {
		t := metav1.NewTime(*runner.TokenExpiresAt)
		created.TokenExpiresAt = &t
	}
	return created, nil
}

func (g *gitlabApi) DeleteRunner(id int) error {
	_, err := g.gitlabApiClient.Runners.RemoveRunner(id)
	return err
}

func (g *gitlabApi) VerifyToken(token string) (bool, error) {
	resp, err := g.gitlabApiClient.Runners.VerifyRegisteredRunner(&gitlab.VerifyRegisteredRunnerOptions{
		Token: gitlab.Ptr(token),
	})
	if err == nil {
		return true, nil
	}
	// A 401/403 means GitLab rejected the token (revoked/expired); that is a
	// definitive "not valid", not a transport failure.
	if resp != nil && (resp.StatusCode == 401 || resp.StatusCode == 403) {
		return false, nil
	}
	return false, err
}

func (g *gitlabApi) RefreshToken(id int) (CreatedRunner, error) {
	tok, resp, err := g.gitlabApiClient.Runners.ResetRunnerAuthenticationToken(int64(id))
	if err != nil {
		return CreatedRunner{}, err
	}
	defer closeBody(resp)

	refreshed := CreatedRunner{ID: id}
	if tok.Token != nil {
		refreshed.Token = *tok.Token
	}
	if tok.TokenExpiresAt != nil {
		t := metav1.NewTime(*tok.TokenExpiresAt)
		refreshed.TokenExpiresAt = &t
	}
	return refreshed, nil
}

func (g *gitlabApi) RunnerExists(id int) (bool, error) {
	_, resp, err := g.gitlabApiClient.Runners.GetRunnerDetails(id)
	if err == nil {
		return true, nil
	}
	if resp != nil && resp.StatusCode == 404 {
		return false, nil
	}
	return false, err
}

func closeBody(resp *gitlab.Response) {
	if resp == nil || resp.Body == nil {
		return
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
}

// NewGitlabClient builds a GitLab API client authenticated with the given
// access token. An empty url defaults to the public gitlab.com instance.
func NewGitlabClient(token, url string) (GitlabClient, error) {
	if url == "" {
		url = "https://gitlab.com/"
	}
	obj := &gitlabApi{}
	var err error
	obj.gitlabApiClient, err = gitlab.NewClient(token, gitlab.WithBaseURL(url))
	return obj, err
}
