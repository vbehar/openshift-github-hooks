package api

import (
	"fmt"
)

// ParseGithubRepository extracts the owner and name of a github repository URI
func ParseGithubRepository(repositoryURI string) (*GithubRepository, error) {
	switch matches := GithubURIRegexp.FindStringSubmatch(repositoryURI); len(matches) {
	case 3:
		return &GithubRepository{
			Owner: matches[1],
			Name:  matches[2],
		}, nil
	}
	return nil, fmt.Errorf("Failed to parse owner and name from URI %s", repositoryURI)
}
