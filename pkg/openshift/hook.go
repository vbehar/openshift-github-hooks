package openshift

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// openshiftWebhookRegexp is a regexp that can extract the namespace, buildconfig and secret from an Openshift Webhook URI
var openshiftWebhookRegexp = regexp.MustCompile(`oapi/v1/namespaces/([^/]+)/buildconfigs/([^/]+)/webhooks/([^/]+)/github`)

// ExplodeOpenshiftWebhookURL explodes the given openshift webhook url
// and returns the namespace, buildconfig and webhook secret
func ExplodeOpenshiftWebhookURL(url string) (namespace, buildconfig, secret string) {
	switch matches := openshiftWebhookRegexp.FindStringSubmatch(url); len(matches) {
	case 4:
		namespace = matches[1]
		buildconfig = matches[2]
		secret = matches[3]
	}
	return
}

// IsOpenshiftHook returns true if the given hook URL is an Openshift hook URL
// that targets the given openshift instance (identified by its public URL)
func IsOpenshiftHook(hookURL string, openshiftPublicURL string) bool {
	if !strings.Contains(hookURL, openshiftPublicURL) {
		return false
	}
	if !strings.HasSuffix(hookURL, "github") {
		return false
	}
	return true
}

// fixOpenshiftHookURL tranforms the hook URL to make sure it is available through the given public (host) URL
func fixOpenshiftHookURL(hookURL *url.URL, openshiftPublicURL string) string {
	if len(openshiftPublicURL) == 0 {
		return hookURL.String()
	}

	return strings.Replace(hookURL.String(), fmt.Sprintf("%s://%s", hookURL.Scheme, hookURL.Host), openshiftPublicURL, 1)
}
