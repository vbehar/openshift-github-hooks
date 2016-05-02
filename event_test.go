package main

import (
	"io"
	"net/url"
	"reflect"
	"testing"

	buildapi "github.com/openshift/origin/pkg/build/api"
	"github.com/openshift/origin/pkg/client"

	kapi "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/watch"
)

type fakeBuildConfigsNamespacer struct {
	err error
}

func (f fakeBuildConfigsNamespacer) BuildConfigs(namespace string) client.BuildConfigInterface {
	return fakeBuildConfigInterface{
		namespace: namespace,
		err:       f.err,
	}
}

type fakeBuildConfigInterface struct {
	namespace string
	err       error
}

func (f fakeBuildConfigInterface) List(options kapi.ListOptions) (*buildapi.BuildConfigList, error) {
	return nil, nil
}
func (f fakeBuildConfigInterface) Get(name string) (*buildapi.BuildConfig, error) {
	return nil, nil
}
func (f fakeBuildConfigInterface) Create(config *buildapi.BuildConfig) (*buildapi.BuildConfig, error) {
	return nil, nil
}
func (f fakeBuildConfigInterface) Update(config *buildapi.BuildConfig) (*buildapi.BuildConfig, error) {
	return nil, nil
}
func (f fakeBuildConfigInterface) Delete(name string) error {
	return nil
}
func (f fakeBuildConfigInterface) Watch(options kapi.ListOptions) (watch.Interface, error) {
	return nil, nil
}
func (f fakeBuildConfigInterface) Instantiate(request *buildapi.BuildRequest) (result *buildapi.Build, err error) {
	return nil, nil
}
func (f fakeBuildConfigInterface) InstantiateBinary(request *buildapi.BinaryBuildRequestOptions, r io.Reader) (result *buildapi.Build, err error) {
	return nil, nil
}
func (f fakeBuildConfigInterface) WebHookURL(name string, trigger *buildapi.BuildTriggerPolicy) (*url.URL, error) {
	if f.err != nil {
		return nil, f.err
	}
	uri, err := url.Parse("https://openshift.org/")
	if err != nil {
		return nil, err
	}
	return uri, nil
}

func TestNewEvent(t *testing.T) {
	tests := []struct {
		watchEvent          watch.Event
		expectedEventResult *Event
		expectedErrorResult error
	}{
		{
			watchEvent: watch.Event{
				Type: watch.Added,
				Object: &buildapi.BuildConfig{
					ObjectMeta: kapi.ObjectMeta{
						Namespace: "mynamespace",
						Name:      "mybc",
					},
					Spec: buildapi.BuildConfigSpec{
						BuildSpec: buildapi.BuildSpec{
							Source: buildapi.BuildSource{
								Git: &buildapi.GitBuildSource{
									URI: "git@github.com:owner/name.git",
								},
							},
						},
						Triggers: []buildapi.BuildTriggerPolicy{
							{
								Type: buildapi.GitHubWebHookBuildTriggerType,
								GitHubWebHook: &buildapi.WebHookTrigger{
									Secret: "mysecret",
								},
							},
						},
					},
				},
			},
			expectedEventResult: &Event{
				Type: CreateOrUpdateEvent,
				GithubRepositoryOwner: "owner",
				GithubRepositoryName:  "name",
				HookUrl:               "https://openshift.org/",
			},
			expectedErrorResult: nil,
		},
		{
			watchEvent: watch.Event{
				Type: watch.Deleted,
				Object: &buildapi.BuildConfig{
					ObjectMeta: kapi.ObjectMeta{
						Namespace: "mynamespace",
						Name:      "mybc",
					},
					Spec: buildapi.BuildConfigSpec{
						BuildSpec: buildapi.BuildSpec{
							Source: buildapi.BuildSource{
								Git: &buildapi.GitBuildSource{
									URI: "git@github.com:owner/name.git",
								},
							},
						},
						Triggers: []buildapi.BuildTriggerPolicy{
							{
								Type: buildapi.GitHubWebHookBuildTriggerType,
								GitHubWebHook: &buildapi.WebHookTrigger{
									Secret: "mysecret",
								},
							},
						},
					},
				},
			},
			expectedEventResult: &Event{
				Type: DeleteEvent,
				GithubRepositoryOwner: "owner",
				GithubRepositoryName:  "name",
				HookUrl:               "https://openshift.org/",
			},
			expectedErrorResult: nil,
		},
	}

	bcNamespacer := fakeBuildConfigsNamespacer{}
	for count, test := range tests {
		event, err := NewEvent(bcNamespacer, test.watchEvent)
		if !reflect.DeepEqual(event, test.expectedEventResult) || !reflect.DeepEqual(err, test.expectedErrorResult) {
			t.Errorf("Test[%d] Failed: Expected %+v and %+v but got %+v and %+v", count, test.expectedEventResult, test.expectedErrorResult, event, err)
		}
	}
}

func TestExtractRepositoryOwnerAndName(t *testing.T) {
	tests := []struct {
		repositoryUri           string
		expectedRepositoryOwner string
		expectedRepositoryName  string
	}{
		{
			repositoryUri:           "",
			expectedRepositoryOwner: "",
			expectedRepositoryName:  "",
		},
		{
			repositoryUri:           "owner/name",
			expectedRepositoryOwner: "",
			expectedRepositoryName:  "",
		},
		{
			repositoryUri:           "https://github.com/owner/name",
			expectedRepositoryOwner: "owner",
			expectedRepositoryName:  "name",
		},
		{
			repositoryUri:           "https://github.com/owner/name.git",
			expectedRepositoryOwner: "owner",
			expectedRepositoryName:  "name",
		},
		{
			repositoryUri:           "git@github.com:owner/name.git",
			expectedRepositoryOwner: "owner",
			expectedRepositoryName:  "name",
		},
		{
			repositoryUri:           "git@bitbucket.org:owner/name.git",
			expectedRepositoryOwner: "",
			expectedRepositoryName:  "",
		},
		{
			repositoryUri:           "https://www.github.com/owner/name",
			expectedRepositoryOwner: "owner",
			expectedRepositoryName:  "name",
		},
		{
			repositoryUri:           "https://github.com/owner",
			expectedRepositoryOwner: "",
			expectedRepositoryName:  "",
		},
	}

	for count, test := range tests {
		owner, name := extractRepositoryOwnerAndName(test.repositoryUri)
		if owner != test.expectedRepositoryOwner || name != test.expectedRepositoryName {
			t.Errorf("Test[%d] Failed: Expected '%s/%s' but got '%s/%s'", count, test.expectedRepositoryOwner, test.expectedRepositoryName, owner, name)
		}
	}
}

func TestFixOpenshiftHookUrl(t *testing.T) {
	tests := []struct {
		hookUrl            string
		openshiftPublicUrl string
		expectedResult     string
	}{
		{
			hookUrl:            "https://somewhere.com/some/path",
			openshiftPublicUrl: "",
			expectedResult:     "https://somewhere.com/some/path",
		},
		{
			hookUrl:            "https://127.0.0.1:443/some/path",
			openshiftPublicUrl: "https://127.0.0.1:443",
			expectedResult:     "https://127.0.0.1:443/some/path",
		},
		{
			hookUrl:            "https://127.0.0.1:443/some/path",
			openshiftPublicUrl: "https://my.openshift.master:8443",
			expectedResult:     "https://my.openshift.master:8443/some/path",
		},
		{
			hookUrl:            "https://127.0.0.1:443/some/path",
			openshiftPublicUrl: "http://my.openshift.master",
			expectedResult:     "http://my.openshift.master/some/path",
		},
	}

	for count, test := range tests {
		hookUrl, err := url.Parse(test.hookUrl)
		if err != nil {
			t.Errorf("Test[%d] Failed: got unexpected error %v", count, err)
		}
		result := fixOpenshiftHookUrl(hookUrl, test.openshiftPublicUrl)
		if result != test.expectedResult {
			t.Errorf("Test[%d] Failed: Expected '%s' but got '%s'", count, test.expectedResult, result)
		}
	}
}
