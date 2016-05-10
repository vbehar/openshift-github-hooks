package cmd

import (
	"flag"

	"github.com/spf13/cobra"

	// init glog to get its flags
	_ "github.com/golang/glog"
)

var (
	// RootCmd is the main command
	RootCmd = &cobra.Command{
		Use:   "openshift-github-hooks",
		Short: "Manages GitHub hooks for OpenShift BuildConfig triggers",
		Long:  `openshift-github-hooks helps you manage your GitHub hooks for OpenShift BuildConfig triggers`,
		Run:   RunHelp,
	}
)

func init() {
	// add glog flags
	RootCmd.PersistentFlags().AddGoFlagSet(flag.CommandLine)
}
