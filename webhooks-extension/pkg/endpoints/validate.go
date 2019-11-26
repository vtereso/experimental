package endpoints

import (
	"net/url"
	"strings"

	"golang.org/x/xerrors"
)

// checkCredentialRequest returns an error if there any empty values within the
// credentialRequest
func checkCredentialRequest(cred credentialRequest) error {
	if cred.Name == "" {
		return xerrors.New("Name cannot be empty")
	}
	if cred.AccessToken == "" {
		return xerrors.New("AccessToken cannot be empty")
	}
	return nil
}

// checkWebhook returns an error if there are any empty values within the
// webhook
func checkWebhook(w webhook) error {
	if w.Name == "" {
		return xerrors.New("Name must cannot be empty")
	}
	if w.Namespace == "" {
		return xerrors.New("Namespace cannot be empty")
	}
	if w.ServiceAccount == "" {
		return xerrors.New("ServiceAccount cannot be emptyd")
	}
	if w.AccessTokenRef == "" {
		return xerrors.New("AccessTokenRef cannot be empty")
	}
	if w.DockerRegistry == "" {
		return xerrors.New("Docker Registry cannot be empty")
	}
	if w.GitRepositoryURL == "" {
		return xerrors.New("GitRepositoryURL cannot be empty")
	}
	return nil
}

// sanitizeGitURL returns a URL for the specified rawurl string, where
// the .git suffix is removed. The rawurl must have the following format:
// `http(s)://<git-site>.com/<some-org>/<some-repo>(.git)`
func sanitizeGitURL(rawurl string) (*url.URL, error) {
	url, err := url.ParseRequestURI(strings.TrimSuffix(rawurl, ".git"))
	if err != nil {
		return nil, err
	}
	if !strings.HasSuffix(url.Hostname(), ".com") {
		return nil, xerrors.Errorf("URL hostname '%s' is invalid", url.Hostname())
	}
	if url.Scheme != "http" || url.Scheme != "https" {
		return nil, xerrors.Errorf("URL scheme '%s' is invalid", url.Scheme)
	}
	// Does not allow trailing slashes
	// Expects a path in the format: /<some-org>/<some-repo>
	s := strings.Split(url.Path, "/")
	if len(s) != 3 || s[1] == "" || s[2] == "" {
		return nil, xerrors.Errorf("URL path '%s' is invalid", url.Path)
	}
	return url, nil
}
