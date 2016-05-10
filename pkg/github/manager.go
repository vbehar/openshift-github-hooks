package github

import (
	"sync"

	"github.com/vbehar/openshift-github-hooks/pkg/api"

	"github.com/golang/glog"
	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

// HooksManager provides an easy way to manage GitHub hooks
type HooksManager struct {
	client *github.Client
}

// NewHooksManager instantiates a HooksManager
// using the given GitHub access token
func NewHooksManager(token string) *HooksManager {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(oauth2.NoContext, ts)

	return &HooksManager{
		client: github.NewClient(tc),
	}
}

// RegisterHook registers the given hook (only if the hook does not already exists)
// returns true if the hook has been created
func (gh *HooksManager) RegisterHook(hook api.Hook) (bool, error) {
	glog.V(2).Infof("Creating Hook %s on Github repository %s ...", hook.TargetURL, hook.GithubRepository)

	hookExists, err := gh.hookExists(hook)
	if err != nil {
		return false, err
	}
	if hookExists {
		glog.V(2).Infof("Hook %s already exists on Github repository %s - nothing to do", hook.TargetURL, hook.GithubRepository)
		return false, nil
	}

	githubHook := NewGithubHook(hook)
	_, _, err = gh.client.Repositories.CreateHook(hook.GithubRepository.Owner, hook.GithubRepository.Name, githubHook)
	if err != nil {
		return false, err
	}

	glog.V(1).Infof("Hook %s created on Github repository %s", hook.TargetURL, hook.GithubRepository)
	return true, nil
}

// hookExists checks if the given hook (URL) exists
func (gh *HooksManager) hookExists(hook api.Hook) (bool, error) {
	hooks, err := gh.ListHooksForRepository(hook.GithubRepository)
	if err != nil {
		return false, err
	}

	for _, h := range hooks {
		if hook.TargetURL == h.TargetURL {
			return true, nil
		}
	}
	return false, nil
}

// DeleteHook deletes the given hook
// returns true if the hook has been deleted
func (gh *HooksManager) DeleteHook(hook api.Hook) (bool, error) {
	glog.V(2).Infof("Deleting Hook %s from Github repository %s ...", hook.TargetURL, hook.GithubRepository)

	hooks, err := gh.listHooks(hook.GithubRepository)
	if err != nil {
		return false, err
	}

	for _, h := range hooks {
		if HooksMatches(hook, h) {
			_, err = gh.client.Repositories.DeleteHook(hook.GithubRepository.Owner, hook.GithubRepository.Name, *h.ID)
			if err != nil {
				return false, err
			}

			glog.V(1).Infof("Hook %s deleted on Github repository %s", hook.TargetURL, hook.GithubRepository)
			return true, nil
		}
	}

	glog.V(2).Infof("Hook %s not found on Github repository %s - nothing to do", hook.TargetURL, hook.GithubRepository)
	return false, nil
}

// ListHooksForOrganization returns all the hooks for all the repositories in given github organization
func (gh *HooksManager) ListHooksForOrganization(org string) ([]api.Hook, error) {
	glog.V(2).Infof("Listing hooks for organization %s ...", org)
	githubRepositories, err := gh.getOrganizationRepositories(org)
	if err != nil {
		return []api.Hook{}, err
	}
	repositories := []api.GithubRepository{}
	for i := range githubRepositories {
		r := api.GithubRepository{
			Owner: *githubRepositories[i].Owner.Login,
			Name:  *githubRepositories[i].Name,
		}
		repositories = append(repositories, r)
	}
	return gh.listHooksForRepositories(repositories)
}

// ListHooksForRepository returns all the hooks for the given github repository
func (gh *HooksManager) ListHooksForRepository(repository api.GithubRepository) ([]api.Hook, error) {
	glog.V(2).Infof("Listing hooks for repository %s ...", repository)
	if _, _, err := gh.client.Repositories.Get(repository.Owner, repository.Name); err != nil {
		return []api.Hook{}, err
	}
	return gh.listHooksForRepositories([]api.GithubRepository{repository})
}

// listHooksForRepositories returns all the non-empty hooks for the given list of github repositories.
func (gh *HooksManager) listHooksForRepositories(repositories []api.GithubRepository) ([]api.Hook, error) {
	hooks := []api.Hook{}
	wg := &sync.WaitGroup{}
	c := make(chan api.Hook)
	// this "limiter" is used to limit the number of parallel requests to github
	limiter := make(chan struct{}, 5)

	// single goroutine that writes results to the repositoriesAndHooks array
	go func(c <-chan api.Hook) {
		for {
			hook, open := <-c
			if !open {
				break
			}
			hooks = append(hooks, hook)
		}
	}(c)

	// fetch the hooks in parallel, but restricted by the limiter
	for r := range repositories {
		limiter <- struct{}{}
		wg.Add(1)
		go func(c chan<- api.Hook, repository api.GithubRepository) {
			defer wg.Done()
			defer func() {
				<-limiter
			}()
			githubHooks, err := gh.listHooks(repository)
			if err != nil {
				glog.Errorf("Failed to list hooks for repository %s: %v", repository, err)
				return
			}
			for h := range githubHooks {
				hookURL := ""
				if val, found := githubHooks[h].Config["url"]; found {
					hookURL = val.(string)
				}
				if len(hookURL) > 0 {
					c <- api.Hook{
						Enabled:          true,
						TargetURL:        hookURL,
						GithubRepository: repository,
					}
				} else {
					glog.V(5).Infof("Ignoring empty hook on repository %s", repository)
				}
			}
		}(c, repositories[r])
	}

	wg.Wait()
	return hooks, nil
}

// listHooks lists the hooks from the github api for the given repository
func (gh *HooksManager) listHooks(repository api.GithubRepository) ([]github.Hook, error) {
	glog.V(3).Infof("Listing hooks for repository %s ...", repository)
	hooks := []github.Hook{}
	page := 1
	for {
		opts := &github.ListOptions{
			PerPage: 100,
			Page:    page,
		}
		objs, resp, err := gh.client.Repositories.ListHooks(repository.Owner, repository.Name, opts)
		if err != nil {
			return []github.Hook{}, err
		}
		hooks = append(hooks, objs...)
		page = resp.NextPage
		if resp.NextPage == 0 {
			break
		}
	}

	glog.V(3).Infof("Found %d hooks for repository %s", len(hooks), repository)
	return hooks, nil
}

// getOrganizationRepositories returns the repositories for the given github organization
func (gh *HooksManager) getOrganizationRepositories(org string) (repositories []github.Repository, err error) {
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
