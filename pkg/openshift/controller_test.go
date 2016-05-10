package openshift

import (
	"testing"

	"github.com/vbehar/openshift-github-hooks/pkg/api"

	buildapi "github.com/openshift/origin/pkg/build/api"

	kapi "k8s.io/kubernetes/pkg/api"
)

func TestBuildConfigsControllerAcceptBuildConfig(t *testing.T) {
	tests := []struct {
		bc             *buildapi.BuildConfig
		expectedResult bool
	}{
		// should ignore nil BC
		{
			bc:             nil,
			expectedResult: false,
		},
		// should ignore non-git sources
		{
			bc:             &buildapi.BuildConfig{},
			expectedResult: false,
		},
		// should ignore non-github sources
		{
			bc: &buildapi.BuildConfig{
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
			expectedResult: false,
		},
		// should ignore bc without github trigger
		{
			bc: &buildapi.BuildConfig{
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
			expectedResult: false,
		},
		// should ignore bc because of "ignore" annotation
		{
			bc: &buildapi.BuildConfig{
				ObjectMeta: kapi.ObjectMeta{
					Annotations: map[string]string{
						api.IgnoreAnnotation: "true",
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
			expectedResult: false,
		},
		// should accept a BC with a github source and a valid github trigger
		// and an invalid value for the "ignore" annotation
		{
			bc: &buildapi.BuildConfig{
				ObjectMeta: kapi.ObjectMeta{
					Annotations: map[string]string{
						api.IgnoreAnnotation: "whatever",
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
			expectedResult: true,
		},
		// should accept a BC with a github source and a valid github trigger
		{
			bc: &buildapi.BuildConfig{
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
			expectedResult: true,
		},
	}

	controller := &BuildConfigsController{}
	for count, test := range tests {
		result := controller.acceptBuildConfig(test.bc)
		if result != test.expectedResult {
			t.Errorf("Test[%d] Failed: Expected '%v' but got '%v'", count, test.expectedResult, result)
		}
	}
}
