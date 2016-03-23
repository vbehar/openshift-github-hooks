package main

import (
	"regexp"

	buildapi "github.com/openshift/origin/pkg/build/api"
	"github.com/openshift/origin/pkg/client"

	"k8s.io/kubernetes/pkg/watch"
)

var (
	// GithubUriRegexp is a regexp that can extract the repository owner and name from its URI
	GithubUriRegexp = regexp.MustCompile(`github\.com[:/]([^/]+)/([^.]+)`)
)

// EventType represents the type of event
type EventType string

const (
	CreateOrUpdateEvent EventType = "create-or-update"
	DeleteEvent         EventType = "delete"
)

// Event represents a BuildConfig change event
type Event struct {
	Type                  EventType
	GithubRepositoryOwner string
	GithubRepositoryName  string
	HookUrl               string
}

// NewEvent instantiates a new Event
func NewEvent(bcNamespacer client.BuildConfigsNamespacer, watchEvent watch.Event) (*Event, error) {
	bc := watchEvent.Object.(*buildapi.BuildConfig)
	event := &Event{}

	switch watchEvent.Type {
	case watch.Added, watch.Modified:
		event.Type = CreateOrUpdateEvent
	case watch.Deleted:
		event.Type = DeleteEvent
	}

	for _, trigger := range bc.Spec.Triggers {
		switch trigger.Type {
		case buildapi.GitHubWebHookBuildTriggerType:
			url, err := bcNamespacer.BuildConfigs(bc.Namespace).WebHookURL(bc.Name, &trigger)
			if err != nil {
				return nil, err
			}
			event.HookUrl = url.String()
			break
		}
	}

	if bc.Spec.Source.Git != nil {
		event.GithubRepositoryOwner, event.GithubRepositoryName = extractRepositoryOwnerAndName(bc.Spec.Source.Git.URI)
	}

	return event, nil
}

// extractRepositoryOwnerAndName extracts the owner and name of a github repository URI
func extractRepositoryOwnerAndName(repositoryUri string) (owner, name string) {
	switch matches := GithubUriRegexp.FindStringSubmatch(repositoryUri); len(matches) {
	case 3:
		owner = matches[1]
		name = matches[2]
	}
	return
}
