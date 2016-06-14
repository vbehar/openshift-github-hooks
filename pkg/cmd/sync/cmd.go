package sync

import (
	"fmt"
	"os"
	"time"

	"github.com/vbehar/openshift-github-hooks/pkg/cmd"
	"github.com/vbehar/openshift-github-hooks/pkg/openshift"

	"github.com/spf13/cobra"
)

// Options represents the command's options
type Options struct {
	GithubBaseURL            string
	GithubInsecureSkipVerify bool
	OrganizationName         string
	Token                    string
	OpenshiftPublicURL       string
	ResyncPeriod             time.Duration
	DryRun                   bool
}

var (
	syncCmdExample = `
	# Start the sync daemon for all the repositories in the "my-org" organization, in dry-run mode
	# (it will not create/delete hooks, but just print which hook would have been created/deleted)
	$ %[1]s --organization=my-org --github-token=... --dry-run

	# Start the sync daemon for all the repositories in the "my-org" organization
	$ %[1]s --organization=my-org --github-token=...

	# Start the sync daemon, and log each hook that has been created or deleted
	$ %[1]s --organization=my-org --github-token=... --v=1`

	syncCmd = &cobra.Command{
		Use:   "sync",
		Short: "Automatically create or delete GitHub hooks based on OpenShift BuildConfig triggers",
		Long: `
The sync command will keep your GitHub hooks in sync with your OpenShift BuildConfig triggers,
by watching for all BuildConfig events in the OpenShift cluster, and automatically creating (or deleting)
GitHub hooks for the BuildConfig who have a GitHub Trigger defined.

It will only try to create/delete hooks for repositories in a specific GitHub Organization,
specified by the --organization flag (or by the GITHUB_ORGANIZATION environment variable).

As it use the GitHub API to create/delete the hooks, it needs a GitHub Token to authenticate against the GitHub Hooks API.
Note that the token requires the "repo" and "admin:repo_hook" scopes.
It can be set either with the --github-token flag, or the GITHUB_ACCESS_TOKEN environment variable.`,
		PreRunE: func(command *cobra.Command, args []string) error {
			if len(options.Token) == 0 {
				return fmt.Errorf("Empty GitHub Access Token. Please provide one either with the --github-token flag or the GITHUB_ACCESS_TOKEN environment variable.")
			}
			if len(options.OrganizationName) == 0 {
				return fmt.Errorf("Empty GitHub Organization Name. Please provide one either with the --organization flag or the GITHUB_ORGANIZATION environment variable.")
			}
			return nil
		},
		Run: func(command *cobra.Command, args []string) {
			syncHooks(options)
		},
	}

	options = &Options{}
)

func init() {
	cmd.RootCmd.AddCommand(syncCmd)

	syncCmd.Example = fmt.Sprintf(syncCmdExample, cmd.FullName(syncCmd))

	syncCmd.Flags().AddFlagSet(openshift.Flags)

	syncCmd.Flags().StringVar(&options.GithubBaseURL, "github-base-url", "https://api.github.com/",
		"The GitHub Base URL - if you use GitHub Enterprise. Format: https://github.domain.tld/api/v3/")
	syncCmd.Flags().BoolVar(&options.GithubInsecureSkipVerify, "github-insecure-skip-tls-verify", false,
		"If true, the github server's certificate will not be checked for validity. This will make your HTTPS connections insecure.")
	syncCmd.Flags().StringVar(&options.Token, "github-token", os.Getenv("GITHUB_ACCESS_TOKEN"),
		"The GitHub Access Token - could also be defined by the GITHUB_ACCESS_TOKEN env var. See https://github.com/settings/tokens to get one.")
	syncCmd.Flags().DurationVar(&options.ResyncPeriod, "resync-period", 1*time.Hour,
		"If not zero, defines the interval of time to perform a full resync of all the webhooks.")
	syncCmd.Flags().StringVar(&options.OrganizationName, "organization", os.Getenv("GITHUB_ORGANIZATION"),
		"The name of the GitHub Organization for which we will sync the webhooks - could also be defined by the GITHUB_ORGANIZATION env var.")
	syncCmd.Flags().BoolVar(&options.DryRun, "dry-run", false,
		"Run in dry-run mode (does not really create/delete hooks on github).")
	syncCmd.Flags().StringVar(&options.OpenshiftPublicURL, "openshift-public-url", openshift.DefaultOpenshiftPublicURL(),
		"The public URL of your OpenShift Master, used to generate the Webhooks URLs.")
}
