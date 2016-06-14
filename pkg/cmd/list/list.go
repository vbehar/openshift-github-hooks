package list

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/vbehar/openshift-github-hooks/pkg/api"
	"github.com/vbehar/openshift-github-hooks/pkg/github"
	"github.com/vbehar/openshift-github-hooks/pkg/openshift"

	"github.com/golang/glog"
)

// listHooks prints the github hooks that references openshift buildconfigs
func listHooks(options *Options) {
	hooksManager, err := github.NewHooksManager(options.GithubBaseURL, options.Token, options.GithubInsecureSkipVerify)
	if err != nil {
		glog.Fatalf("Failed to connect to GitHub: %v", err)
	}

	var hooks []api.Hook
	if len(options.RepositoryName) > 0 {
		repository := api.GithubRepository{
			Owner: options.OrganizationName,
			Name:  options.RepositoryName,
		}
		hooks, err = hooksManager.ListHooksForRepository(repository)
	} else {
		hooks, err = hooksManager.ListHooksForOrganization(options.OrganizationName)
	}
	if err != nil {
		glog.Fatalf("Failed to list GitHub hooks: %v", err)
	}

	w := &tabwriter.Writer{}
	w.Init(os.Stdout, 10, 4, 3, ' ', 0)
	fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", "OWNER", "REPOSITORY", "NAMESPACE", "BUILDCONFIG", "WEBHOOK SECRET")

	for _, hook := range hooks {
		if !openshift.IsOpenshiftHook(hook.TargetURL, options.OpenshiftPublicURL) {
			glog.V(4).Infof("Ignoring non-openshift hook %s for repository %s", hook.TargetURL, hook.GithubRepository)
		} else {
			ns, bc, secret := openshift.ExplodeOpenshiftWebhookURL(hook.TargetURL)
			if len(ns) > 0 && len(bc) > 0 {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", hook.GithubRepository.Owner, hook.GithubRepository.Name, ns, bc, secret)
			}
		}
	}

	w.Flush()
}
