package main

import (
	"testing"

	buildapi "github.com/openshift/origin/pkg/build/api"

	kapi "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/watch"
)

func TestBuildConfigsWatcherShouldAcceptEvent(t *testing.T) {
	tests := []struct {
		event          watch.Event
		expectedResult bool
	}{
		// should ignore error events
		{
			event: watch.Event{
				Type: watch.Error,
			},
			expectedResult: false,
		},
		// should ignore non-bc objects
		{
			event: watch.Event{
				Object: &buildapi.Build{},
			},
			expectedResult: false,
		},
		// should ignore non-git sources
		{
			event: watch.Event{
				Object: &buildapi.BuildConfig{},
			},
			expectedResult: false,
		},
		// should ignore non-github sources
		{
			event: watch.Event{
				Object: &buildapi.BuildConfig{
					Spec: buildapi.BuildConfigSpec{
						BuildSpec: buildapi.BuildSpec{
							Source: buildapi.BuildSource{
								Git: &buildapi.GitBuildSource{
									URI: "git@bitbucket.org:owner/name.git",
								},
							},
						},
					},
				},
			},
			expectedResult: false,
		},
		// should ignore bc without github trigger
		{
			event: watch.Event{
				Object: &buildapi.BuildConfig{
					Spec: buildapi.BuildConfigSpec{
						BuildSpec: buildapi.BuildSpec{
							Source: buildapi.BuildSource{
								Git: &buildapi.GitBuildSource{
									URI: "git@github.com:owner/name.git",
								},
							},
						},
					},
				},
			},
			expectedResult: false,
		},
		// should ignore bc because of "ignore" annotation
		{
			event: watch.Event{
				Object: &buildapi.BuildConfig{
					ObjectMeta: kapi.ObjectMeta{
						Annotations: map[string]string{
							IgnoreAnnotation: "true",
						},
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
									Secret: "secret",
								},
							},
						},
					},
				},
			},
			expectedResult: false,
		},
		// should accept a BC with a github source and a valid github trigger
		// and an invalid value for the "ignore" annotation
		{
			event: watch.Event{
				Object: &buildapi.BuildConfig{
					ObjectMeta: kapi.ObjectMeta{
						Annotations: map[string]string{
							IgnoreAnnotation: "whatever",
						},
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
									Secret: "secret",
								},
							},
						},
					},
				},
			},
			expectedResult: true,
		},
		// should accept a BC with a github source and a valid github trigger
		{
			event: watch.Event{
				Object: &buildapi.BuildConfig{
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
									Secret: "secret",
								},
							},
						},
					},
				},
			},
			expectedResult: true,
		},
	}

	watcher := &BuildConfigsWatcher{}
	for count, test := range tests {
		result := watcher.shouldAcceptEvent(test.event)
		if result != test.expectedResult {
			t.Errorf("Test[%d] Failed: Expected '%v' but got '%v'", count, test.expectedResult, result)
		}
	}
}
