package main

import (
	"fmt"
	"strconv"
	"strings"

	buildapi "github.com/openshift/origin/pkg/build/api"
	"github.com/openshift/origin/pkg/cmd/util/clientcmd"

	"k8s.io/kubernetes/pkg/kubectl/resource"
	"k8s.io/kubernetes/pkg/watch"

	"github.com/golang/glog"
)

const (
	// IgnoreAnnotation is an annotation whose boolean value
	// is used to ignore a buildconfig
	IgnoreAnnotation = "openshift-github-hooks-sync/ignore"
)

// BuildConfigsWatcher provides an easy way to watch buildconfigs
type BuildConfigsWatcher struct {
	factory clientcmd.Factory
}

// Watch watches all buildconfigs and send the matching ones
// (the ones with a github build trigger)
// to the given channel
func (watcher *BuildConfigsWatcher) Watch(c chan<- Event) error {
	for {
		w, err := watcher.internalWatcher()
		if err != nil {
			return err
		}

		client, _, err := watcher.factory.Clients()
		if err != nil {
			return err
		}

		glog.V(1).Infof("Starting watch loop on buildconfigs for all namespaces")
		for {
			watchEvent, open := <-w.ResultChan()
			if !open {
				glog.Warningf("Watch channel has been closed!")
				break
			}

			glog.V(3).Infof("Got event %v for %T", watchEvent.Type, watchEvent.Object)
			if watcher.shouldAcceptEvent(watchEvent) {
				event, err := NewEvent(client, watchEvent)
				if err != nil {
					glog.Errorf("Failed to parse event %+v: %v", watchEvent, err)
				}
				glog.V(2).Infof("Handling event %+v", event)
				c <- *event
			}

		}
		glog.V(1).Infof("End of watch loop")
	}
}

// shouldAcceptEvent checks if the given event is acceptable or not
// an acceptable event is one that contains a buildconfig with a github trigger
func (watcher *BuildConfigsWatcher) shouldAcceptEvent(event watch.Event) bool {
	// filter out error events
	switch event.Type {
	case watch.Error:
		glog.V(3).Infof("Ignoring error event %+v", event)
		return false
	}

	// filter out non-bc objects
	bc, ok := event.Object.(*buildapi.BuildConfig)
	if !ok {
		glog.V(3).Infof("Ignoring non-BC object %T", event.Object)
		return false
	}

	// filter out non-git sources
	if bc.Spec.Source.Git == nil {
		glog.V(3).Infof("Ignoring BC %s/%s with non-git sources", bc.Namespace, bc.Name)
		return false
	}
	// filter our non-github sources
	if !strings.Contains(bc.Spec.Source.Git.URI, "github") {
		glog.V(3).Infof("Ignoring BC %s/%s with non-github sources", bc.Namespace, bc.Name)
		return false
	}

	// filter out bc without github trigger
	githubTriggerFound := false
	for _, trigger := range bc.Spec.Triggers {
		switch trigger.Type {
		case buildapi.GitHubWebHookBuildTriggerType:
			githubTriggerFound = true
		}
	}
	if !githubTriggerFound {
		glog.V(3).Infof("Ignoring BC %s/%s with no github trigger", bc.Namespace, bc.Name)
		return false
	}

	// filter out bc because of "ignore" annotation
	if ignoreStr, found := bc.Annotations[IgnoreAnnotation]; found {
		ignore, err := strconv.ParseBool(ignoreStr)
		if err != nil {
			glog.Errorf("Failed to parse annotation value '%v' for %s on BC %s/%s: %v", ignoreStr, IgnoreAnnotation, bc.Namespace, bc.Name, err)
		}
		if ignore {
			glog.V(3).Infof("Ignoring BC %s/%s because of annotation %s (%v)", bc.Namespace, bc.Name, IgnoreAnnotation, ignoreStr)
			return false
		}
	}

	glog.V(3).Infof("Accepting BC %s/%s", bc.Namespace, bc.Name)
	return true
}

// internalWatcher creates a k8s watcher for all buildconfigs (or an error)
func (watcher *BuildConfigsWatcher) internalWatcher() (watch.Interface, error) {
	mapper, typer := watcher.factory.Object()
	clientMapper := watcher.factory.ClientMapperForCommand()

	builder := resource.NewBuilder(mapper, typer, clientMapper).
		AllNamespaces(true).
		ResourceTypeOrNameArgs(true, "buildconfig").
		SingleResourceType().
		Latest()
	r := builder.Do()
	err := r.Err()
	if err != nil {
		return nil, err
	}

	infos, err := r.Infos()
	if err != nil {
		return nil, err
	}
	if len(infos) != 1 {
		return nil, fmt.Errorf("watch is only supported on individual resources and resource collections - %d resources were found", len(infos))
	}
	info := infos[0]
	mapping := info.ResourceMapping()

	obj, err := r.Object()
	if err != nil {
		return nil, err
	}
	rv, err := mapping.MetadataAccessor.ResourceVersion(obj)
	if err != nil {
		return nil, err
	}

	w, err := r.Watch(rv)
	if err != nil {
		return nil, err
	}

	return w, nil
}
