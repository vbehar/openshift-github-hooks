package openshift

import (
	"strconv"
	"strings"
	"time"

	"github.com/vbehar/openshift-github-hooks/pkg/api"

	"github.com/golang/glog"
	buildapi "github.com/openshift/origin/pkg/build/api"
	"github.com/openshift/origin/pkg/client"
	"github.com/openshift/origin/pkg/controller"

	kapi "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/client/cache"
	"k8s.io/kubernetes/pkg/runtime"
	kutil "k8s.io/kubernetes/pkg/util"
	"k8s.io/kubernetes/pkg/watch"
)

// BuildConfigsController represents a controller that will react to BC changes,
// and handle only the BC with a github hook trigger.
type BuildConfigsController struct {

	// BuildConfigsNamespacer is used to list/watch the BCs
	// and build the BC's hook URL.
	BuildConfigsNamespacer client.BuildConfigsNamespacer

	// HookHandlerFunc is the function that will handle the Hook
	HookHandlerFunc func(api.Hook) error

	// KeyListFunc is a function that returns the list of keys ("namespace/name" format)
	// that we "know about" (to get a 2-way sync)
	KeyListFunc func() []string

	// KeyGetFunc is a function that returns the object that we "know about"
	// for the given key ("namespace/name" format) - and a boolean if it exists
	KeyGetFunc func(key string) (interface{}, bool, error)

	// ResyncPeriod is the interval of time at which the controller
	// will perform of full resync (list) of the BuildConfigs
	ResyncPeriod time.Duration

	// OpenshiftPublicURL is the public URL of the OpenShift instance
	// used to make sure the hook URL does not use an internal hostname ;-)
	OpenshiftPublicURL string
}

// RunUntil runs the controller in a goroutine
// until stopChan is closed
func (c *BuildConfigsController) RunUntil(stopChan <-chan struct{}) {
	queue := cache.NewDeltaFIFO(cache.MetaNamespaceKeyFunc, nil, c)
	cache.NewReflector(c, &buildapi.BuildConfig{}, queue, c.ResyncPeriod).RunUntil(stopChan)

	retryController := &controller.RetryController{
		Handle: c.handle,
		Queue:  queue,
		RetryManager: controller.NewQueueRetryManager(
			queue,
			cache.MetaNamespaceKeyFunc,
			c.retry,
			kutil.NewTokenBucketRateLimiter(1, 10)),
	}

	retryController.RunUntil(stopChan)
}

// handle handles a BuildConfig change
// by filtering it first (excluding BC without github hook triggers)
// and then converting it to a Hook to could be handled by the HookHandlerFunc function
func (c *BuildConfigsController) handle(obj interface{}) error {
	deltas := obj.(cache.Deltas)
	for _, delta := range deltas {

		if bc, ok := delta.Object.(*buildapi.BuildConfig); ok {
			glog.V(5).Infof("Handling %v for BC %s/%s", delta.Type, bc.Namespace, bc.Name)

			if c.acceptBuildConfig(bc) {
				glog.V(3).Infof("Accepting BC %s/%s", bc.Namespace, bc.Name)
				hook, err := c.newHook(bc, delta.Type)
				if err != nil {
					return err
				}

				if err = c.HookHandlerFunc(*hook); err != nil {
					return err
				}
			}

			continue
		}

		if deletedObject, ok := delta.Object.(cache.DeletedFinalStateUnknown); ok {
			glog.V(5).Infof("Handling %v DeletedFinalStateUnknown for %s: %+v", delta.Type, deletedObject.Key, deletedObject.Obj)

			if hook, ok := deletedObject.Obj.(api.Hook); ok {
				hook.Enabled = false // make sure the hook is marked has not enabled, so that it will be deleted
				glog.V(3).Infof("Processing hook %+v for key %s", hook, deletedObject.Key)
				if err := c.HookHandlerFunc(hook); err != nil {
					return err
				}
				continue
			}

			glog.Warningf("Un-handled %v DeletedFinalStateUnknown for %s: %+v", delta.Type, deletedObject.Key, deletedObject.Obj)
			continue
		}

		glog.Warningf("Un-handled delta type %T (%s)", delta.Object, delta.Type)

	}

	return nil
}

// acceptBuildConfig checks if the given BC is acceptable or not
// an acceptable BC is one that has a valid github trigger
func (c *BuildConfigsController) acceptBuildConfig(bc *buildapi.BuildConfig) bool {
	// filter out invalid BC
	if bc == nil {
		glog.V(4).Infof("Ignoring empty BC")
		return false
	}

	// filter out non-git sources
	if bc.Spec.Source.Git == nil {
		glog.V(4).Infof("Ignoring BC %s/%s with non-git sources", bc.Namespace, bc.Name)
		return false
	}
	// filter out non-github sources
	if !strings.Contains(bc.Spec.Source.Git.URI, "github") {
		glog.V(4).Infof("Ignoring BC %s/%s with non-github sources", bc.Namespace, bc.Name)
		return false
	}

	// filter out BC without github trigger
	githubTriggerFound := false
	for _, trigger := range bc.Spec.Triggers {
		switch trigger.Type {
		case buildapi.GitHubWebHookBuildTriggerType:
			githubTriggerFound = true
		}
	}
	if !githubTriggerFound {
		glog.V(4).Infof("Ignoring BC %s/%s with no github trigger", bc.Namespace, bc.Name)
		return false
	}

	// filter out BC because of "ignore" annotation
	if ignoreStr, found := bc.Annotations[api.IgnoreAnnotation]; found {
		ignore, err := strconv.ParseBool(ignoreStr)
		if err != nil {
			glog.Errorf("Failed to parse annotation value '%v' for %s on BC %s/%s: %v", ignoreStr, api.IgnoreAnnotation, bc.Namespace, bc.Name, err)
		}
		if ignore {
			glog.V(4).Infof("Ignoring BC %s/%s because of annotation %s (%v)", bc.Namespace, bc.Name, api.IgnoreAnnotation, ignoreStr)
			return false
		}
	}

	return true
}

// newHook instantiates a new Hook object for the given BC
func (c *BuildConfigsController) newHook(bc *buildapi.BuildConfig, changeType cache.DeltaType) (*api.Hook, error) {
	hook := &api.Hook{}

	switch changeType {
	case cache.Deleted:
		hook.Enabled = false
	default:
		hook.Enabled = true
	}

	for _, trigger := range bc.Spec.Triggers {
		switch trigger.Type {
		case buildapi.GitHubWebHookBuildTriggerType:
			hookURL, err := c.BuildConfigsNamespacer.BuildConfigs(bc.Namespace).WebHookURL(bc.Name, &trigger)
			if err != nil {
				return nil, err
			}
			hook.TargetURL = fixOpenshiftHookURL(hookURL, c.OpenshiftPublicURL)
			break
		}
	}

	if bc.Spec.Source.Git != nil {
		repo, err := api.ParseGithubRepository(bc.Spec.Source.Git.URI)
		if err != nil {
			return nil, err
		}
		hook.GithubRepository = *repo
	}

	return hook, nil
}

// retry is a controller.RetryFunc that should return true if the given object and error
// should be retried after the provided number of times.
func (c *BuildConfigsController) retry(obj interface{}, err error, retries controller.Retry) bool {
	// let's retry a few times...
	return retries.Count < 5
}

// List is for the cache.ListerWatcher implementation
// List should return a list type object; the Items field will be extracted, and the
// ResourceVersion field will be used to start the watch in the right place.
func (c *BuildConfigsController) List(options kapi.ListOptions) (runtime.Object, error) {
	glog.V(3).Infof("Listing BuildConfigs with options %+v", options)
	return c.BuildConfigsNamespacer.BuildConfigs(kapi.NamespaceAll).List(options)
}

// Watch is for the cache.ListerWatcher implementation
// Watch should begin a watch at the specified version.
func (c *BuildConfigsController) Watch(options kapi.ListOptions) (watch.Interface, error) {
	glog.V(3).Infof("Watching BuildConfigs with options %+v", options)
	return c.BuildConfigsNamespacer.BuildConfigs(kapi.NamespaceAll).Watch(options)
}

// ListKeys implements the cache.KeyLister interface
// It is a function that returns the list of keys ("namespace/name" format)
// that we "know about" (to get a 2-way sync)
func (c *BuildConfigsController) ListKeys() []string {
	return c.KeyListFunc()
}

// GetByKey implements the cache.KeyGetter interface
// It is a function that returns the object that we "know about"
// for the given key ("namespace/name" format) - and a boolean if it exists
func (c *BuildConfigsController) GetByKey(key string) (interface{}, bool, error) {
	return c.KeyGetFunc(key)
}
