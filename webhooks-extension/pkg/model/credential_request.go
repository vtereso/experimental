package model

import (
	"golang.org/x/xerrors"
)

// CredentialRequest is the intended structure of an http request for creating a
// credential, which is a K8s secret of type access token
type CredentialRequest struct {
	Name        string `json:"name"`
	AccessToken string `json:"accesstoken"`
}

// Validate validates the credentialRequest. If there are any empty values, an
// error is returned
func (c *CredentialRequest) Validate() error {
	if c.Name == "" {
		return xerrors.New("Name cannot be empty")
	}
	if c.AccessToken == "" {
		return xerrors.New("AccessToken cannot be empty")
	}
	return nil
}
