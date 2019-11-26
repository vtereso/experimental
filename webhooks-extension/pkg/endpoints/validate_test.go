package endpoints

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func Test_sanitizeGitURL(t *testing.T) {
	tests := []struct {
		name   string
		url    string
		hasErr bool
	}{
		// Correct
		{
			name:   "HTTPS Git URL",
			url:    "https://gitpalace.com/some/repo",
			hasErr: false,
		},
		{
			name:   "HTTP Git URL",
			url:    "http://gitpalace.com/some/repo",
			hasErr: false,
		},
		{
			name:   "HTTPS Git URL with GitSuffix",
			url:    "https://gitpalace.com/some/repo.git",
			hasErr: false,
		},
		{
			name:   "HTTP Git URL with GitSuffix",
			url:    "http://gitpalace.com/some/repo.git",
			hasErr: false,
		},
		{
			name:   "HTTPS Git Enterprise URL",
			url:    "https://gitpalace.enterprise.com/some/repo",
			hasErr: false,
		},
		{
			name:   "HTTP Git Enterprise URL",
			url:    "http://gitpalace.enterprise.com/some/repo",
			hasErr: false,
		},
		{
			name:   "HTTPS Git Enterprise URL with GitSuffix",
			url:    "https://gitpalace.enterprise.com/some/repo.git",
			hasErr: false,
		},
		{
			name:   "HTTP Git Enterprise URL with GitSuffix",
			url:    "http://gitpalace.enterprise.com/some/repo.git",
			hasErr: false,
		},
		// Incorrect
		{
			name:   "Bad scheme URL",
			url:    "abced://gitpalace.com/some/repo",
			hasErr: true,
		},
		{
			name:   "No scheme URL",
			url:    "gitpalacecom/some/repo",
			hasErr: true,
		},
		{
			name:   "Trailing slash URL",
			url:    "https://gitpalace.com/some/repo/",
			hasErr: true,
		},
	}
	for i := range tests {
		t.Run(tests[i].name, func(t *testing.T) {
			var hasErr bool
			err := sanitizeGitURL(tests[i].url)
			if err != nil {
				hasErr = true
			}
			if diff := cmp.Diff(tests[i].hasErr, hasErr); diff != "" {
				t.Errorf("sanitizeGitURL() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func Test_checkWebhook(t *testing.T) {
	tests := []struct {
		name   string
		w      webhook
		hasErr bool
	}{
		// Correct
		{
			name: "Webhook All Fields",
			w: webhook{
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
			w: webhook{
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
			w: webhook{
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
			w: webhook{
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
			w: webhook{
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
			w: webhook{
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
			w: webhook{
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
			w: webhook{
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
			err := checkWebhook(tests[i].url)
			if err != nil {
				hasErr = true
			}
			if diff := cmp.Diff(tests[i].hasErr, hasErr); diff != "" {
				t.Errorf("isCredential() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}


func Test_check(t *testing.T) {
	tests := []struct {
		name   string
		w      webhook
		hasErr bool
	}{
		// Correct
		{
			name: "Webhook All Fields",
			w: webhook{
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
			w: webhook{
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
			w: webhook{
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
			w: webhook{
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
			w: webhook{
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
			w: webhook{
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
			w: webhook{
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
			w: webhook{
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
			err := checkWebhook(tests[i].url)
			if err != nil {
				hasErr = true
			}
			if diff := cmp.Diff(tests[i].hasErr, hasErr); diff != "" {
				t.Errorf("isCredential() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}