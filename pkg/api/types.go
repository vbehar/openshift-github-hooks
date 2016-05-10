package api

import (
	"fmt"
	"regexp"
)

// Hook is a very basic representation of a WebHook
// that links a Github repository to an OpenShift BuildConfig
// through the hook's TargetURL (OpenShift endpoint used to trigger a new build)
type Hook struct {
	Enabled          bool
	TargetURL        string
	GithubRepository GithubRepository
}

// GithubRepository is a very basic representation of a GitHub repository
type GithubRepository struct {
	Owner string
	Name  string
}

func (r GithubRepository) String() string {
	return fmt.Sprintf("%s/%s", r.Owner, r.Name)
}

const (
	// IgnoreAnnotation is an annotation whose boolean value
	// is used to ignore a buildconfig
	IgnoreAnnotation = "openshift-github-hooks-sync/ignore"
)

var (
	// GithubURIRegexp is a regexp that can extract the repository owner and name from its URI
	GithubURIRegexp = regexp.MustCompile(`github\.com[:/]([^/]+)/([^.]+)`)
)
