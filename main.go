package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:   "openshift-github-hooks",
		Short: "Manages GitHub hooks for OpenShift BuildConfig triggers",
		Long:  `openshift-github-hooks helps you manage your GitHub hooks for OpenShift BuildConfig triggers`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := cmd.Help(); err != nil {
				fmt.Printf("Failed to print help message! %v", err)
			}
		},
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Use != "openshift-github-hooks" && len(githubToken) == 0 {
				return fmt.Errorf("Empty GitHub token! Please set the token either with the --github-token option, or the GITHUB_ACCESS_TOKEN environment variable.")
			}
			return nil
		},
	}

	openshiftPublicUrl string
	githubToken        string
)

func main() {
	rootCmd.AddCommand(syncCmd)
	rootCmd.AddCommand(listCmd)

	rootCmd.PersistentFlags().AddGoFlagSet(flag.CommandLine)
	rootCmd.PersistentFlags().StringVar(&openshiftPublicUrl, "openshift-public-url", defaultOpenshiftPublicUrl(),
		"The public URL of your OpenShift Master, used to generate the Webhooks URLs")
	rootCmd.PersistentFlags().StringVar(&githubToken, "github-token", os.Getenv("GITHUB_ACCESS_TOKEN"),
		"The GitHub Access Token - could also be defined by the GITHUB_ACCESS_TOKEN env var. See https://github.com/settings/tokens to get one.")

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}
