package main

import "testing"

func TestExplodeOpenshiftWebhookUrl(t *testing.T) {
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
		ns, bc, secret := explodeOpenshiftWebhookUrl(test.url)
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
		hookUrl            string
		openshiftPublicUrl string
		expectedResult     bool
	}{
		{
			hookUrl:            "",
			openshiftPublicUrl: "",
			expectedResult:     false,
		},
		{
			hookUrl:            "https://127.0.0.1:443/oapi/v1/namespaces/mynamespace/buildconfigs/mybc/webhooks/mysecret/github",
			openshiftPublicUrl: "https://my.openshift.master:8443",
			expectedResult:     false,
		},
		{
			hookUrl:            "https://my.openshift.master:8443/oapi/v1/namespaces/mynamespace/buildconfigs/mybc/webhooks/mysecret/generic",
			openshiftPublicUrl: "https://my.openshift.master:8443",
			expectedResult:     false,
		},
		{
			hookUrl:            "https://my.openshift.master:8443/oapi/v1/namespaces/mynamespace/buildconfigs/mybc/webhooks/mysecret/github",
			openshiftPublicUrl: "https://my.openshift.master:8443",
			expectedResult:     true,
		},
	}

	for count, test := range tests {
		result := isOpenshiftHook(test.hookUrl, test.openshiftPublicUrl)
		if result != test.expectedResult {
			t.Errorf("Test[%d] Failed: Expected '%v' but got '%v'", count, test.expectedResult, result)
		}
	}
}
