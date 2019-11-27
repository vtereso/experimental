package models

// CredentialResponse is the intended structure of an http response when getting
// a credential, which is a K8s secret of type access token
type CredentialResponse struct {
	CredentialRequest `json:",inline"`
	SecretToken       string `json:"secrettoken,omitempty"`
}
