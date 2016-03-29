package main

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/golang/glog"
	"github.com/spf13/cobra"
)

type ListOptions struct {
	org  string
	repo string
}

var (
	listCmd = &cobra.Command{
		Use:   "list",
		Short: "List all GitHub hooks for OpenShift BuildConfig triggers",
		Long: `
The list command will list all the webhooks for a specific GitHub Organization (using the GitHub API)
that references OpenShift BuildConfigs.

It needs a GitHub Token to authenticate against the GitHub API.
Note that the token requires the "read:repo_hook" scope.
It can be set either with the --github-token option, or the GITHUB_ACCESS_TOKEN environment variable.`,
		Run: func(cmd *cobra.Command, args []string) {
			listHooks(cmd)
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if len(listOptions.org) == 0 {
				return fmt.Errorf("Empty organization!")
			}
			return nil
		},
	}

	listOptions = &ListOptions{}
)

func init() {
	listCmd.Flags().StringVar(&listOptions.org, "organization", "",
		"The name of the GitHub Organization for which we will list the repositories and webhooks. Mandatory.")
	listCmd.Flags().StringVar(&listOptions.repo, "repository", "",
		"The name of the GitHub Repository for which we will list the webhooks. Optional (default to retrieve all repositories from the organization).")
}

// listHooks prints the github hooks that references openshift buildconfigs
func listHooks(cmd *cobra.Command) {
	gh := NewGitHubHooksManager(githubToken)

	var reposAndHooks []RepositoryAndHook
	var err error
	if len(listOptions.repo) > 0 {
		reposAndHooks, err = gh.ListHooksForRepository(listOptions.org, listOptions.repo)
	} else {
		reposAndHooks, err = gh.ListHooksForOrganization(listOptions.org)
	}
	if err != nil {
		glog.Fatalf("Failed to list GitHub hooks: %v", err)
	}

	w := &tabwriter.Writer{}
	w.Init(os.Stdout, 10, 4, 3, ' ', 0)
	fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", "OWNER", "REPOSITORY", "NAMESPACE", "BUILDCONFIG", "WEBHOOK SECRET")

	for i := range reposAndHooks {
		repo := reposAndHooks[i].Repository
		hook := reposAndHooks[i].Hook
		hookUrl := ""
		if val, found := hook.Config["url"]; found {
			hookUrl = val.(string)
		}

		if !isOpenshiftHook(hookUrl, openshiftPublicUrl) {
			glog.V(3).Infof("Ignoring non-openshift hook %s for repository %s", hookUrl, *repo.FullName)
		} else {
			ns, bc, secret := explodeOpenshiftWebhookUrl(hookUrl)
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", *repo.Owner.Login, *repo.Name, ns, bc, secret)
		}
	}

	w.Flush()

}
