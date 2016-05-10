package github

import (
	"github.com/vbehar/openshift-github-hooks/pkg/api"

	"github.com/google/go-github/github"
)

// NewGithubHook returns a GitHub representation of a hook
// GitHub hook refenrence: https://developer.github.com/v3/repos/hooks/#parameters
func NewGithubHook(hook api.Hook) *github.Hook {
	return &github.Hook{
		Name:   func(name string) *string { return &name }("web"),
		Active: func(active bool) *bool { return &active }(true),
		Events: []string{"*"},
		Config: map[string]interface{}{
			"url":          hook.TargetURL,
			"content_type": "json",
			"insecure_ssl": "true",
		},
	}
}

// HooksMatches checks if the given GitHub hook is the same as the current OpenShift hook
func HooksMatches(hook api.Hook, githubHook github.Hook) bool {
	return hook.TargetURL == githubHook.Config["url"]
}
