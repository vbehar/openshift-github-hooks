package main

import (
	"strconv"
	"sync"

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
	hooks, err := gh.listHooks(repoOwner, repoName)
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

	hooks, err := gh.listHooks(repoOwner, repoName)
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

// ListHooksForOrganization returns an array of (repository, webhook) tuples
// for the given github organization
func (gh *GitHubHooksManager) ListHooksForOrganization(org string) ([]RepositoryAndHook, error) {
	glog.V(2).Infof("Listing hooks for organization %s ...", org)
	repos, err := gh.getOrganizationRepositories(org)
	if err != nil {
		return []RepositoryAndHook{}, nil
	}
	return gh.listHooksForRepositories(repos)
}

// ListHooksForRepository returns an array of (repository, webhook) tuples
// for the given github repository, identified by its owner and name
func (gh *GitHubHooksManager) ListHooksForRepository(owner string, repo string) ([]RepositoryAndHook, error) {
	glog.V(2).Infof("Listing hooks for repository %s/%s ...", owner, repo)
	repository, _, err := gh.client.Repositories.Get(owner, repo)
	if err != nil {
		return []RepositoryAndHook{}, nil
	}
	return gh.listHooksForRepositories([]github.Repository{*repository})
}

// listHooksForRepositories returns an array of (repository, webhook) tuples
// for the given list of github repositories.
func (gh *GitHubHooksManager) listHooksForRepositories(repositories []github.Repository) ([]RepositoryAndHook, error) {
	repositoriesAndHooks := []RepositoryAndHook{}
	wg := &sync.WaitGroup{}
	c := make(chan RepositoryAndHook)
	// this "limiter" is used to limit the number of parallel requests to github
	limiter := make(chan struct{}, 10)

	// single goroutine that writes results to the repositoriesAndHooks array
	go func(c <-chan RepositoryAndHook) {
		for {
			repositoryAndHook, open := <-c
			if !open {
				break
			}
			repositoriesAndHooks = append(repositoriesAndHooks, repositoryAndHook)
		}
	}(c)

	// fetch the hooks in parallel, but restricted by the limiter
	for r := range repositories {
		limiter <- struct{}{}
		wg.Add(1)
		go func(c chan<- RepositoryAndHook, repository github.Repository) {
			defer wg.Done()
			defer func() {
				<-limiter
			}()
			hooks, err := gh.listHooks(*repository.Owner.Login, *repository.Name)
			if err != nil {
				glog.Errorf("Failed to list hooks for repository %s: %v", *repository.FullName, err)
				return
			}
			for h := range hooks {
				c <- RepositoryAndHook{
					Repository: &repository,
					Hook:       &hooks[h],
				}
			}
		}(c, repositories[r])
	}

	wg.Wait()
	return repositoriesAndHooks, nil
}

// listHooks lists the hooks from the github api for the given repository
func (gh *GitHubHooksManager) listHooks(repoOwner, repoName string) (hooks []github.Hook, err error) {
	glog.V(3).Infof("Listing hooks for repository %s/%s ...", repoOwner, repoName)
	page := 1
	for {
		opts := &github.ListOptions{
			PerPage: 100,
			Page:    page,
		}
		objs, resp, err := gh.client.Repositories.ListHooks(repoOwner, repoName, opts)
		if err != nil {
			return hooks, err
		}
		hooks = append(hooks, objs...)
		page = resp.NextPage
		if resp.NextPage == 0 {
			break
		}
	}

	glog.V(3).Infof("Found %d hooks for repository %s/%s", len(hooks), repoOwner, repoName)
	return hooks, nil
}

// getOrganizationRepositories returns the repositories for the given github organization
func (gh *GitHubHooksManager) getOrganizationRepositories(org string) (repositories []github.Repository, err error) {
	glog.V(3).Infof("Listing repositories for organization %s ...", org)
	page := 1
	for {
		opts := &github.RepositoryListByOrgOptions{
			Type: "all",
			ListOptions: github.ListOptions{
				PerPage: 100,
				Page:    page,
			},
		}
		repos, resp, err := gh.client.Repositories.ListByOrg(org, opts)
		if err != nil {
			return repositories, err
		}
		repositories = append(repositories, repos...)
		page = resp.NextPage
		if resp.NextPage == 0 {
			break
		}
	}

	glog.V(3).Infof("Found %d repositories for organization %s", len(repositories), org)
	return repositories, nil
}

type RepositoryAndHook struct {
	Repository *github.Repository
	Hook       *github.Hook
}
