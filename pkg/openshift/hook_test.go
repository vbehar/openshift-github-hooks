package openshift

import (
	"net/url"
	"testing"
)

func TestExplodeOpenshiftWebhookURL(t *testing.T) {
	tests := []struct {
		url                 string
		expectedNamespace   string
		expectedBuildConfig string
		expectedSecret      string
	}{
		{
			url:                 "https://my.openshift.master:8443/oapi/v1/namespaces/mynamespace/buildconfigs/mybc/webhooks/mysecret/github",
			expectedNamespace:   "mynamespace",
			expectedBuildConfig: "mybc",
			expectedSecret:      "mysecret",
		},
		{
			url:                 "https://my.openshift.master:8443/oapi/v1/namespaces/mynamespace/buildconfigs/mybc/webhooks/mysecret/generic",
			expectedNamespace:   "",
			expectedBuildConfig: "",
			expectedSecret:      "",
		},
	}

	for count, test := range tests {
		ns, bc, secret := ExplodeOpenshiftWebhookURL(test.url)
		if ns != test.expectedNamespace {
			t.Errorf("Test[%d] Failed: Expected namespace '%s' but got '%s'", count, test.expectedNamespace, ns)
		}
		if bc != test.expectedBuildConfig {
			t.Errorf("Test[%d] Failed: Expected buildconfig '%s' but got '%s'", count, test.expectedBuildConfig, bc)
		}
		if secret != test.expectedSecret {
			t.Errorf("Test[%d] Failed: Expected secret '%s' but got '%s'", count, test.expectedSecret, secret)
		}
	}
}

func TestIsOpenshiftHook(t *testing.T) {
	tests := []struct {
		hookURL            string
		openshiftPublicURL string
		expectedResult     bool
	}{
		{
			hookURL:            "",
			openshiftPublicURL: "",
			expectedResult:     false,
		},
		{
			hookURL:            "https://127.0.0.1:443/oapi/v1/namespaces/mynamespace/buildconfigs/mybc/webhooks/mysecret/github",
			openshiftPublicURL: "https://my.openshift.master:8443",
			expectedResult:     false,
		},
		{
			hookURL:            "https://my.openshift.master:8443/oapi/v1/namespaces/mynamespace/buildconfigs/mybc/webhooks/mysecret/generic",
			openshiftPublicURL: "https://my.openshift.master:8443",
			expectedResult:     false,
		},
		{
			hookURL:            "https://my.openshift.master:8443/oapi/v1/namespaces/mynamespace/buildconfigs/mybc/webhooks/mysecret/github",
			openshiftPublicURL: "https://my.openshift.master:8443",
			expectedResult:     true,
		},
	}

	for count, test := range tests {
		result := IsOpenshiftHook(test.hookURL, test.openshiftPublicURL)
		if result != test.expectedResult {
			t.Errorf("Test[%d] Failed: Expected '%v' but got '%v'", count, test.expectedResult, result)
		}
	}
}

func TestFixOpenshiftHookURL(t *testing.T) {
	tests := []struct {
		hookURL            string
		openshiftPublicURL string
		expectedResult     string
	}{
		{
			hookURL:            "https://somewhere.com/some/path",
			openshiftPublicURL: "",
			expectedResult:     "https://somewhere.com/some/path",
		},
		{
			hookURL:            "https://127.0.0.1:443/some/path",
			openshiftPublicURL: "https://127.0.0.1:443",
			expectedResult:     "https://127.0.0.1:443/some/path",
		},
		{
			hookURL:            "https://127.0.0.1:443/some/path",
			openshiftPublicURL: "https://my.openshift.master:8443",
			expectedResult:     "https://my.openshift.master:8443/some/path",
		},
		{
			hookURL:            "https://127.0.0.1:443/some/path",
			openshiftPublicURL: "http://my.openshift.master",
			expectedResult:     "http://my.openshift.master/some/path",
		},
	}

	for count, test := range tests {
		hookURL, err := url.Parse(test.hookURL)
		if err != nil {
			t.Errorf("Test[%d] Failed: got unexpected error %v", count, err)
		}
		result := fixOpenshiftHookURL(hookURL, test.openshiftPublicURL)
		if result != test.expectedResult {
			t.Errorf("Test[%d] Failed: Expected '%s' but got '%s'", count, test.expectedResult, result)
		}
	}
}
