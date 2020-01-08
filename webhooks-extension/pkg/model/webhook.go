package model

import (
	"strings"

	"golang.org/x/xerrors"
)

// Webhook contains only the form payload structure used to create a webhook.
// This is defined within /src/components/WebhookCreate/WebhookCreate.js
type Webhook struct {
	// Name is the name of the webhook in the UI
	Name string `json:"name"`
	// Namespace is the namespace passed to the TriggerTemplate
	Namespace string `json:"namespace"`
	// ServiceAccount is the serviceAccount passed to the TriggerTemplate
	ServiceAccount string `json:"serviceaccount,omitempty"`
	// AccessTokenRef is the name of the git secret used. This is used for
	// validation
	AccessTokenRef string `json:"accesstoken"`
	// Pipeline is the pipeline that a webhook is being created for. The
	// pipeline must have corresponding triggers resources.
	Pipeline string `json:"pipeline"`
	// DockerRegistry is the registry used to upload images within the pipeline
	DockerRegistry string `json:"dockerregistry,omitempty"`
	// GitRepositoryURL is broken down into fields (server, org, and repo) and
	// passed to the TriggerTemplate. This is also used for validation.
	GitRepositoryURL string `json:"gitrepositoryurl"`
}

// ValidateWebhookName validates a webhook name
func ValidateWebhookName(name string) error {
	if len(name) == 0 {
		return xerrors.New("Name must cannot be empty")
	}
	if len(name) > 57 {
		return xerrors.New("Name must be less than 58 characters")
	}
	if strings.Contains(name, "-") {
		return xerrors.New("Name may not contains hyphens")
	}
	return nil
}

// Validate validates the webhook.
func (w *Webhook) Validate() error {
	if err := ValidateWebhookName(w.Name); err != nil {
		return err
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
	if w.Pipeline == "" {
		return xerrors.New("Pipeline cannot be empty")
	}
	if w.DockerRegistry == "" {
		return xerrors.New("Docker Registry cannot be empty")
	}
	if w.GitRepositoryURL == "" {
		return xerrors.New("GitRepositoryURL cannot be empty")
	}
	return nil
}
