package main

import (
	"strconv"
	"testing"

	"github.com/google/go-github/github"
)

func TestHookToGithubHook(t *testing.T) {
	hooks := []Hook{
		{
			Url:         "",
			InsecureSsl: true,
		},
		{
			Url:         "https://openshift.org/",
			InsecureSsl: true,
		},
		{
			Url:         "https://openshift.org/",
			InsecureSsl: false,
		},
	}

	for count, hook := range hooks {
		githubHook := hook.ToGithubHook()
		if githubHook.Config["url"] != hook.Url {
			t.Errorf("Test[%d] Failed: Expected '%s' URL but got '%s'", count, hook.Url, githubHook.Config["url"])
		}
		insecureSslStr := githubHook.Config["insecure_ssl"].(string)
		insecureSsl, err := strconv.ParseBool(insecureSslStr)
		if insecureSsl != hook.InsecureSsl {
			t.Errorf("Test[%d] Failed: Expected '%v' insecure SSL but got '%v' (err is: %v)", count, hook.InsecureSsl, insecureSsl, err)
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

func TestHookMatchesGithubHook(t *testing.T) {
	tests := []struct {
		hook           Hook
		githubHook     github.Hook
		expectedResult bool
	}{
		{
			hook: Hook{
				Url: "https://openshift.org/",
			},
			githubHook: github.Hook{
				Config: map[string]interface{}{
					"url": "https://openshift.org/",
				},
			},
			expectedResult: true,
		},
		{
			hook: Hook{
				Url: "https://openshift.org/",
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
		result := test.hook.MatchesGithubHook(test.githubHook)
		if result != test.expectedResult {
			t.Errorf("Test[%d] Failed: Expected '%v' but got '%v'", count, test.expectedResult, result)
		}
	}
}
