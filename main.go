package main

import (
	"fmt"
	"os"

	"github.com/vbehar/openshift-github-hooks/pkg/cmd"

	// init all the commands
	_ "github.com/vbehar/openshift-github-hooks/pkg/cmd/list"
	_ "github.com/vbehar/openshift-github-hooks/pkg/cmd/sync"
)

func main() {
	if err := cmd.RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}
