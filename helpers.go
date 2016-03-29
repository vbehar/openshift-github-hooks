package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"

	"github.com/emicklei/go-restful/swagger"
	"github.com/golang/glog"
	"github.com/spf13/pflag"
)

// defaultOpenshiftPublicUrl returns the openshift public URL as defined
// in the master config - and exposed in the swagger API.
// Internally, this method will retrieve the swagger API to extract the public URL.
// If it can't be retrieved, it will either return an empty string,
// or the host of the server as defined by the client config.
func defaultOpenshiftPublicUrl() string {
	factory := getFactory(pflag.NewFlagSet("", pflag.ExitOnError))
	config, err := factory.OpenShiftClientConfig.ClientConfig()
	if err != nil {
		glog.Warningf("Failed to get Openshift Config: %v", err)
		return ""
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/swaggerapi/api/v1", config.Host), nil)
	if err != nil {
		glog.Warningf("Failed to build a request for the swagger API: %v", err)
		return config.Host
	}

	resp, err := client.Do(req)
	if err != nil {
		glog.Warningf("Failed to request the swagger API: %v", err)
		return config.Host
	}

	defer resp.Body.Close()
	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		glog.Warningf("Failed to read the swagger API response: %v", err)
		return config.Host
	}

	swagger := &swagger.ApiDeclaration{}
	if err = json.Unmarshal(bytes, swagger); err != nil {
		glog.Warningf("Failed to unmarshall the swagger API response: %v", err)
		return config.Host
	}

	return swagger.BasePath
}

// openshiftWebhookRegexp is a regexp that can extract the namespace, buildconfig and secret from an Openshift Webhook URI
var openshiftWebhookRegexp = regexp.MustCompile(`oapi/v1/namespaces/([^/]+)/buildconfigs/([^/]+)/webhooks/([^/]+)/github`)

// explodeOpenshiftWebhookUrl extracts the given openshift webhook url
// and returns the namespace, buildconfig and webhook secret
func explodeOpenshiftWebhookUrl(url string) (namespace, buildconfig, secret string) {
	switch matches := openshiftWebhookRegexp.FindStringSubmatch(url); len(matches) {
	case 4:
		namespace = matches[1]
		buildconfig = matches[2]
		secret = matches[3]
	}
	return
}

// isOpenshiftHook returns true if the given hook URL is an Openshift hook URL
// and targets the given openshift instance (identified by its public URL)
func isOpenshiftHook(hookUrl string, openshiftPublicUrl string) bool {
	if !strings.Contains(hookUrl, openshiftPublicUrl) {
		return false
	}
	if !strings.HasSuffix(hookUrl, "github") {
		return false
	}
	return true
}
