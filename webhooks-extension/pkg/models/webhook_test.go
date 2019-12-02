package models

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestWebhookValidate(t *testing.T) {
	tests := []struct {
		name   string
		w      Webhook
		hasErr bool
	}{
		// Correct
		{
			name: "Webhook All Fields",
			w: Webhook{
				Name:             "webhook",
				Namespace:        "namespace",
				ServiceAccount:   "serviceAccount",
				AccessTokenRef:   "tokenRef",
				Pipeline:         "pipeline",
				DockerRegistry:   "dockerRegistry",
				GitRepositoryURL: "gitURL",
			},
			hasErr: false,
		},
		// Incorrect
		{
			name: "Webhook No Name",
			w: Webhook{
				Namespace:        "namespace",
				ServiceAccount:   "serviceAccount",
				AccessTokenRef:   "tokenRef",
				Pipeline:         "pipeline",
				DockerRegistry:   "dockerRegistry",
				GitRepositoryURL: "gitURL",
			},
			hasErr: true,
		},
		{
			name: "Webhook Name Too Long",
			w: Webhook{
				// 58 Characters
				Name:             "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				Namespace:        "namespace",
				ServiceAccount:   "serviceAccount",
				AccessTokenRef:   "tokenRef",
				Pipeline:         "pipeline",
				DockerRegistry:   "dockerRegistry",
				GitRepositoryURL: "gitURL",
			},
			hasErr: true,
		},
		{
			name: "Webhook No Namespace",
			w: Webhook{
				Name:             "webhook",
				ServiceAccount:   "serviceAccount",
				AccessTokenRef:   "tokenRef",
				Pipeline:         "pipeline",
				DockerRegistry:   "dockerRegistry",
				GitRepositoryURL: "gitURL",
			},
			hasErr: true,
		},
		{
			name: "Webhook No ServiceAccount",
			w: Webhook{
				Name:             "webhook",
				Namespace:        "namespace",
				AccessTokenRef:   "tokenRef",
				Pipeline:         "pipeline",
				DockerRegistry:   "dockerRegistry",
				GitRepositoryURL: "gitURL",
			},
			hasErr: true,
		},
		{
			name: "Webhook No AccessTokenRef",
			w: Webhook{
				Name:             "webhook",
				Namespace:        "namespace",
				ServiceAccount:   "serviceAccount",
				Pipeline:         "pipeline",
				DockerRegistry:   "dockerRegistry",
				GitRepositoryURL: "gitURL",
			},
			hasErr: true,
		},
		{
			name: "Webhook No Pipeline",
			w: Webhook{
				Name:             "webhook",
				Namespace:        "namespace",
				ServiceAccount:   "serviceAccount",
				AccessTokenRef:   "tokenRef",
				DockerRegistry:   "dockerRegistry",
				GitRepositoryURL: "gitURL",
			},
			hasErr: true,
		},
		{
			name: "Webhook No DockerRegistry",
			w: Webhook{
				Name:             "webhook",
				Namespace:        "namespace",
				ServiceAccount:   "serviceAccount",
				AccessTokenRef:   "tokenRef",
				Pipeline:         "pipeline",
				GitRepositoryURL: "gitURL",
			},
			hasErr: true,
		},
		{
			name: "Webhook No GitRepositoryURL",
			w: Webhook{
				Name:           "webhook",
				Namespace:      "namespace",
				ServiceAccount: "serviceAccount",
				AccessTokenRef: "tokenRef",
				Pipeline:       "pipeline",
				DockerRegistry: "dockerRegistry",
			},
			hasErr: true,
		},
	}
	for i := range tests {
		t.Run(tests[i].name, func(t *testing.T) {
			var hasErr bool
			if err := tests[i].w.Validate(); err != nil {
				hasErr = true
			}
			if diff := cmp.Diff(tests[i].hasErr, hasErr); diff != "" {
				t.Errorf("Validate error mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestValidateWebhookName(t *testing.T) {
	tests := []struct {
		name        string
		webhookName string
		hasErr      bool
	}{
		// Correct
		{
			name:        "Valid Webhook Name",
			webhookName: "webhook",
			hasErr:      false,
		},
		// Incorrect
		{
			name:        "Empty Webhook Name",
			webhookName: "",
			hasErr:      true,
		},
		{
			name:        "Too Long Webhook Name",
			webhookName: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			hasErr:      true,
		},
		{
			name:        "Webhook Name With Hyphens",
			webhookName: "a-webhook",
			hasErr:      true,
		},
	}
	for i := range tests {
		t.Run(tests[i].name, func(t *testing.T) {
			var hasErr bool
			if err := ValidateWebhookName(tests[i].webhookName); err != nil {
				hasErr = true
			}
			if diff := cmp.Diff(tests[i].hasErr, hasErr); diff != "" {
				t.Errorf("Validate error mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
