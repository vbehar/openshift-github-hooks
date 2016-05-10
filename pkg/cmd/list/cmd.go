package list

import (
	"fmt"
	"os"

	"github.com/vbehar/openshift-github-hooks/pkg/cmd"
	"github.com/vbehar/openshift-github-hooks/pkg/openshift"

	"github.com/spf13/cobra"
)

// Options represents the command's options
type Options struct {
	OrganizationName   string
	RepositoryName     string
	Token              string
	OpenshiftPublicURL string
}

var (
	listCmdExample = `
	# List all github webhooks of all the repositories in the "my-org" organization
	$ %[1]s --organization=my-org --github-token=...

	# List all github webhooks of the "my-org/some-repository" repository
	$ %[1]s --organization=my-org --repository=some-repository --github-token=...`

	listCmd = &cobra.Command{
		Use:   "list",
		Short: "List GitHub hooks targeting OpenShift BuildConfigs",
		Long: `
The list command will list GitHub hooks that targets OpenShift BuildConfigs (for a specific OpenShift instance).
It can either list webhooks of all the repositories in a specific GitHub Organization, or webhooks of a single repository.

As it use the GitHub API to list the hooks, it needs a GitHub Token to authenticate against the GitHub API.
Note that the token requires the "repo" and "read:repo_hook" scopes.
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
			listHooks(options)
		},
	}

	options = &Options{}
)

func init() {
	cmd.RootCmd.AddCommand(listCmd)

	listCmd.Example = fmt.Sprintf(listCmdExample, cmd.FullName(listCmd))

	listCmd.Flags().AddFlagSet(openshift.Flags)

	listCmd.Flags().StringVar(&options.Token, "github-token", os.Getenv("GITHUB_ACCESS_TOKEN"),
		"The GitHub Access Token - could also be defined by the GITHUB_ACCESS_TOKEN env var. See https://github.com/settings/tokens to get one.")
	listCmd.Flags().StringVar(&options.OrganizationName, "organization", os.Getenv("GITHUB_ORGANIZATION"),
		"The name of the GitHub Organization for which we will list the repositories and webhooks - could also be defined by the GITHUB_ORGANIZATION env var.")
	listCmd.Flags().StringVar(&options.RepositoryName, "repository", "",
		"The name of the GitHub Repository for which we will list the webhooks. Optional (default to retrieve all repositories from the organization).")
	listCmd.Flags().StringVar(&options.OpenshiftPublicURL, "openshift-public-url", openshift.DefaultOpenshiftPublicURL(),
		"The public URL of your OpenShift Master, used to generate the Webhooks URLs.")
}
