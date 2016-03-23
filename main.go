package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/golang/glog"
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
	}

	syncCmd = &cobra.Command{
		Use:   "sync",
		Short: "Automatically create or delete GitHub hooks based on OpenShift BuildConfig triggers",
		Long: `
The sync command will keep your GitHub hooks in sync with your OpenShift BuildConfig triggers,
by watching for all BuildConfig events in the OpenShift cluster, and automatically creating (or deleting)
GitHub hooks for the BuildConfig who have a GitHub Trigger defined.

It needs a GitHub Token to authenticate against the GitHub Hooks API.
Note that the token requires the "admin:repo_hook" scope.
It can be set either with the --github-token option, or the GITHUB_ACCESS_TOKEN environment variable.`,
		Run: func(cmd *cobra.Command, args []string) {
			sync(cmd)
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if len(githubToken) == 0 {
				return fmt.Errorf("Empty GitHub token! Please set the token either with the --github-token option, or the GITHUB_ACCESS_TOKEN environment variable.")
			}
			return nil
		},
	}

	githubToken string
)

func main() {
	rootCmd.AddCommand(syncCmd)
	rootCmd.PersistentFlags().AddGoFlagSet(flag.CommandLine)
	syncCmd.Flags().StringVar(&githubToken, "github-token", os.Getenv("GITHUB_ACCESS_TOKEN"),
		"The GitHub Access Token - could also be defined by the GITHUB_ACCESS_TOKEN env var. See https://github.com/settings/tokens to get one.")

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

// sync runs the main sync loop to watch for all BC
// and handle the matching events to the github hooks manager
func sync(cmd *cobra.Command) {
	glog.Info("Starting openshift-github-hooks sync...")

	events := make(chan Event)

	githubManager := NewGitHubHooksManager(githubToken)
	go githubManager.Run(events)

	factory := getFactory(cmd.Flags())
	watcher := &BuildConfigsWatcher{
		factory: *factory,
	}
	go func(events chan<- Event) {
		if err := watcher.Watch(events); err != nil {
			glog.Fatalf("Failed to watch: %v", err)
		}
	}(events)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill, syscall.SIGTERM)
	select {
	case <-c:
		glog.Infof("Interrupted by user (or killed) !")
		close(events)
	}

	glog.Info("Shutting down openshift-github-hooks sync")
}
