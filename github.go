package main

import (
	"strconv"

	"golang.org/x/oauth2"

	"github.com/golang/glog"
	"github.com/google/go-github/github"
)

// Hook represents a Webhook as seen by OpenShift
type Hook struct {
	Url         string
	InsecureSsl bool
}

// ToGithubHook returns a GitHub representation of a hook,
// from the OpenShift representation of a hook
// GitHub hook refenrence: https://developer.github.com/v3/repos/hooks/#parameters
func (hook *Hook) ToGithubHook() *github.Hook {
	return &github.Hook{
		Name:   func(name string) *string { return &name }("web"),
		Active: func(active bool) *bool { return &active }(true),
		Events: []string{"*"},
		Config: map[string]interface{}{
			"url":          hook.Url,
			"content_type": "json",
			"insecure_ssl": strconv.FormatBool(hook.InsecureSsl),
		},
	}
}

// MatchesGithubHook checks if the given GitHub hook is the same as the current OpenShift hook
func (hook *Hook) MatchesGithubHook(githubHook github.Hook) bool {
	return githubHook.Config["url"] == hook.Url
}

// GitHubHooksManager provides an easy way to manage GitHub hooks
type GitHubHooksManager struct {
	client *github.Client
}

// NewGitHubHooksManager instantiates a GitHubHooksManager
// using the given GitHub access token
func NewGitHubHooksManager(token string) *GitHubHooksManager {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(oauth2.NoContext, ts)

	return &GitHubHooksManager{
		client: github.NewClient(tc),
	}
}

// Run runs the manager using the given channel as input
// and process each event by creating or deleting a hook on GitHub
func (gh *GitHubHooksManager) Run(c <-chan Event) {
	for {
		event, open := <-c

		if !open {
			glog.Errorf("Channel has been closed!")
			break
		}

		hook := Hook{
			Url:         event.HookUrl,
			InsecureSsl: true,
		}

		switch event.Type {

		case CreateOrUpdateEvent:
			if _, err := gh.registerHook(event.GithubRepositoryOwner, event.GithubRepositoryName, hook); err != nil {
				glog.Errorf("Failed to register hook %s on github repo %s/%s: %v", event.HookUrl, event.GithubRepositoryOwner, event.GithubRepositoryName, err)
			}

		case DeleteEvent:
			if _, err := gh.deleteHook(event.GithubRepositoryOwner, event.GithubRepositoryName, hook); err != nil {
				glog.Errorf("Failed to delete hook %s on github repo %s/%s: %v", event.HookUrl, event.GithubRepositoryOwner, event.GithubRepositoryName, err)
			}

		default:
			glog.Warningf("Unknown event type %v", event.Type)
		}

	}
}

// registerHook registers a hook for the given repo
// only if the hook has not already been registered for this repo
// returns true if the hook has been created
func (gh *GitHubHooksManager) registerHook(repoOwner string, repoName string, hook Hook) (bool, error) {
	glog.V(2).Infof("Creating Hook %s on Github repository %s/%s ...", hook.Url, repoOwner, repoName)

	hasHook, err := gh.hasHook(repoOwner, repoName, hook)
	if err != nil {
		return false, err
	}
	if hasHook {
		glog.V(2).Infof("Hook %s already exists on Github repository %s/%s - nothing to do", hook.Url, repoOwner, repoName)
		return false, nil
	}

	githubHook := hook.ToGithubHook()
	_, _, err = gh.client.Repositories.CreateHook(repoOwner, repoName, githubHook)
	if err != nil {
		return false, err
	}

	glog.V(1).Infof("Hook %s created on Github repository %s/%s", hook.Url, repoOwner, repoName)
	return true, nil
}

// hasHook checks if the given hook (URL) has already been registered for this repo
func (gh *GitHubHooksManager) hasHook(repoOwner string, repoName string, hook Hook) (bool, error) {
	opts := &github.ListOptions{
		PerPage: 100,
		Page:    1,
	}
	hooks, _, err := gh.client.Repositories.ListHooks(repoOwner, repoName, opts)
	if err != nil {
		return false, err
	}

	for _, h := range hooks {
		if hook.MatchesGithubHook(h) {
			return true, nil
		}
	}
	return false, nil
}

// deleteHook deletes a hook from the given repo
// returns true if the hook has been deleted
func (gh *GitHubHooksManager) deleteHook(repoOwner string, repoName string, hook Hook) (bool, error) {
	glog.V(2).Infof("Deleting Hook %s on Github repository %s/%s ...", hook.Url, repoOwner, repoName)

	opts := &github.ListOptions{
		PerPage: 100,
		Page:    1,
	}
	hooks, _, err := gh.client.Repositories.ListHooks(repoOwner, repoName, opts)
	if err != nil {
		return false, err
	}

	for _, h := range hooks {
		if hook.MatchesGithubHook(h) {
			_, err := gh.client.Repositories.DeleteHook(repoOwner, repoName, *h.ID)
			if err != nil {
				return false, err
			}

			glog.V(1).Infof("Hook %s deleted on Github repository %s/%s", hook.Url, repoOwner, repoName)
			return true, nil
		}
	}

	glog.V(2).Infof("Hook %s not found on Github repository %s/%s - nothing to do", hook.Url, repoOwner, repoName)
	return false, nil
}
