package api

import "testing"

func TestParseGithubRepository(t *testing.T) {
	tests := []struct {
		repositoryURI      string
		expectedRepository *GithubRepository
		expectedError      bool
	}{
		{
			repositoryURI:      "",
			expectedRepository: nil,
			expectedError:      true,
		},
		{
			repositoryURI:      "owner/name",
			expectedRepository: nil,
			expectedError:      true,
		},
		{
			repositoryURI: "https://github.com/owner/name",
			expectedRepository: &GithubRepository{
				Owner: "owner",
				Name:  "name",
			},
			expectedError: false,
		},
		{
			repositoryURI: "https://github.com/owner/name.git",
			expectedRepository: &GithubRepository{
				Owner: "owner",
				Name:  "name",
			},
			expectedError: false,
		},
		{
			repositoryURI: "git@github.com:owner/name.git",
			expectedRepository: &GithubRepository{
				Owner: "owner",
				Name:  "name",
			},
			expectedError: false,
		},
		{
			repositoryURI:      "git@bitbucket.org:owner/name.git",
			expectedRepository: nil,
			expectedError:      true,
		},
		{
			repositoryURI: "https://www.github.com/owner/name",
			expectedRepository: &GithubRepository{
				Owner: "owner",
				Name:  "name",
			},
			expectedError: false,
		},
		{
			repositoryURI:      "https://github.com/owner",
			expectedRepository: nil,
			expectedError:      true,
		},
	}

	for count, test := range tests {
		repository, err := ParseGithubRepository(test.repositoryURI)
		if err != nil {
			if !test.expectedError {
				t.Errorf("Test[%d] Failed: Got an unexpected error: %v", count, err)
			}
			continue
		}
		if test.expectedError {
			t.Errorf("Test[%d] Failed: Expected an error but got none", count)
		}
		if repository.String() != test.expectedRepository.String() {
			t.Errorf("Test[%d] Failed: Expected %s but got %s", count, test.expectedRepository, repository)
		}
	}
}
