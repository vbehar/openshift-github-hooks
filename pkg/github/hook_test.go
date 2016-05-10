package github

import (
	"testing"

	"github.com/vbehar/openshift-github-hooks/pkg/api"

	"github.com/google/go-github/github"
)

func TestNewGithubHook(t *testing.T) {
	hooks := []api.Hook{
		{
			TargetURL: "",
		},
		{
			TargetURL: "https://openshift.org/",
		},
	}

	for count, hook := range hooks {
		githubHook := NewGithubHook(hook)
		if githubHook.Config["url"] != hook.TargetURL {
			t.Errorf("Test[%d] Failed: Expected '%s' URL but got '%s'", count, hook.TargetURL, githubHook.Config["url"])
		}
		if githubHook.Config["insecure_ssl"] != "true" {
			t.Errorf("Test[%d] Failed: Expected '%v' insecure SSL but got '%v'", count, "true", githubHook.Config["insecure_ssl"])
		}
		if githubHook.Config["content_type"] != "json" {
			t.Errorf("Test[%d] Failed: Expected '%s' content type but got '%s'", count, "json", githubHook.Config["content_type"])
		}
		if *githubHook.Name != "web" {
			t.Errorf("Test[%d] Failed: Expected '%s' name but got '%s'", count, "web", *githubHook.Name)
		}
		if !*githubHook.Active {
			t.Errorf("Test[%d] Failed: Expected hook to be active", count)
		}
	}
}

func TestHooksMatches(t *testing.T) {
	tests := []struct {
		hook           api.Hook
		githubHook     github.Hook
		expectedResult bool
	}{
		{
			hook: api.Hook{
				TargetURL: "https://openshift.org/",
			},
			githubHook: github.Hook{
				Config: map[string]interface{}{
					"url": "https://openshift.org/",
				},
			},
			expectedResult: true,
		},
		{
			hook: api.Hook{
				TargetURL: "https://openshift.org/",
			},
			githubHook: github.Hook{
				Config: map[string]interface{}{
					"url": "https://openshift.org",
				},
			},
			expectedResult: false,
		},
	}

	for count, test := range tests {
		result := HooksMatches(test.hook, test.githubHook)
		if result != test.expectedResult {
			t.Errorf("Test[%d] Failed: Expected '%v' but got '%v'", count, test.expectedResult, result)
		}
	}
}
