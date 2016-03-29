package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/golang/glog"
	"github.com/spf13/cobra"
)

var (
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
			syncHooks(cmd)
		},
	}
)

// syncHooks runs the main sync loop to watch for all BC
// and handle the matching events to the github hooks manager
func syncHooks(cmd *cobra.Command) {
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
