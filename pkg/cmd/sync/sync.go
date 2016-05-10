package sync

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/vbehar/openshift-github-hooks/pkg/api"
	"github.com/vbehar/openshift-github-hooks/pkg/github"
	"github.com/vbehar/openshift-github-hooks/pkg/openshift"

	"k8s.io/kubernetes/pkg/client/cache"

	"github.com/golang/glog"
)

// syncHooks runs the main sync loop to watch for all BC
// and handle the matching events to the github hooks manager
func syncHooks(options *Options) {
	if options.DryRun {
		glog.Info("Starting openshift-github-hooks sync in DRY-RUN mode...")
	} else {
		glog.Info("Starting openshift-github-hooks sync...")
	}

	stopChan := make(chan struct{})

	hooksManager := github.NewHooksManager(options.Token)

	oclient, _, err := openshift.Factory.Clients()
	if err != nil {
		glog.Fatalf("Failed to get OpenShift client: %v", err)
	}

	keyFunc := func(obj interface{}) (string, error) {
		hook, ok := obj.(api.Hook)
		if !ok {
			return "", fmt.Errorf("Invalid object type %T (expected a Hook)", obj)
		}
		if !openshift.IsOpenshiftHook(hook.TargetURL, options.OpenshiftPublicURL) {
			return "", fmt.Errorf("Hook %s does not target an OpenShift endpoint", hook.TargetURL)
		}
		ns, bc, _ := openshift.ExplodeOpenshiftWebhookURL(hook.TargetURL)
		if len(ns) == 0 || len(bc) == 0 {
			return "", fmt.Errorf("Hook %s does not target a valid OpenShift endpoint", hook.TargetURL)
		}
		return fmt.Sprintf("%s/%s", ns, bc), nil
	}

	// store used as a cache for hooks from github
	// (to avoid too many requests on github.com)
	store := cache.NewTTLStore(keyFunc, 2*time.Minute)

	(&openshift.BuildConfigsController{
		OpenshiftPublicURL:     options.OpenshiftPublicURL,
		ResyncPeriod:           options.ResyncPeriod,
		BuildConfigsNamespacer: oclient,
		HookHandlerFunc: func(hook api.Hook) error {
			if strings.ToLower(hook.GithubRepository.Owner) != strings.ToLower(options.OrganizationName) {
				glog.V(4).Infof("Ignoring hook for external repository '%s' owned by '%s' (instead of '%s')", hook.GithubRepository.Name, hook.GithubRepository.Owner, options.OrganizationName)
				return nil
			}

			if hook.Enabled {
				if options.DryRun {
					glog.Infof("DRY_RUN_MODE: would have registered hook on %s with target URL: %s", hook.GithubRepository, hook.TargetURL)
					return nil
				}
				_, err := hooksManager.RegisterHook(hook)
				return err
			}

			if options.DryRun {
				glog.Infof("DRY_RUN_MODE: would have deleted hook from %s with target URL: %s", hook.GithubRepository, hook.TargetURL)
				return nil
			}
			_, err := hooksManager.DeleteHook(hook)
			return err
		},
		KeyListFunc: func() []string {
			hooks, err := hooksManager.ListHooksForOrganization(options.OrganizationName)
			if err != nil {
				glog.Fatalf("Failed to list github hooks for org %s: %v", options.OrganizationName, err)
			}

			keys := []string{}
			for _, hook := range hooks {
				if openshift.IsOpenshiftHook(hook.TargetURL, options.OpenshiftPublicURL) {
					key, err := keyFunc(hook)
					if err != nil {
						glog.Errorf("Failed to retrieve key from hook %+v: %v", hook, err)
						continue
					}
					keys = append(keys, key)
					if err := store.Add(hook); err != nil {
						glog.Errorf("Failed to cache hook %+v: %v", hook, err)
						continue
					}
				} else {
					glog.V(5).Infof("Ignoring non-openshift hook %s for repository %s", hook.TargetURL, hook.GithubRepository)
				}
			}
			return keys
		},
		KeyGetFunc: func(key string) (interface{}, bool, error) {
			item, exists, err := store.GetByKey(key)
			if exists && err == nil {
				return item, true, nil
			}
			if err != nil {
				glog.Warning("Failed to retrieve object from cache using key '%s': %v", key, err)
			}

			hooks, err := hooksManager.ListHooksForOrganization(options.OrganizationName)
			if err != nil {
				return "", false, err
			}

			for _, hook := range hooks {
				if openshift.IsOpenshiftHook(hook.TargetURL, options.OpenshiftPublicURL) {
					localKey, err := keyFunc(hook)
					if err != nil {
						glog.Errorf("Failed to retrieve key from hook %+v: %v", hook, err)
						continue
					}

					if localKey == key {
						return hook, true, nil
					}
				}

			}
			return "", false, nil
		},
	}).RunUntil(stopChan)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill, syscall.SIGTERM)
	select {
	case <-c:
		glog.Infof("Interrupted by user (or killed) !")
		close(stopChan)
	}

	glog.Info("Shutting down openshift-github-hooks sync")
}
