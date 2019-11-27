package models

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestCredentialRequestValidate(t *testing.T) {
	tests := []struct {
		name   string
		c      CredentialRequest
		hasErr bool
	}{
		// Correct
		{
			name: "CredentialRequest All Fields",
			c: CredentialRequest{
				Name:        "cred",
				AccessToken: "accessToken",
			},
			hasErr: false,
		},
		// Incorrect
		{
			name: "CredentialRequest No Name",
			c: CredentialRequest{
				AccessToken: "accessToken",
			},
			hasErr: true,
		},
		{
			name: "CredentialRequest No Access Token",
			c: CredentialRequest{
				Name: "cred",
			},
			hasErr: true,
		},
	}
	for i := range tests {
		t.Run(tests[i].name, func(t *testing.T) {
			var hasErr bool
			if err := tests[i].c.Validate(); err != nil {
				hasErr = true
			}
			if diff := cmp.Diff(tests[i].hasErr, hasErr); diff != "" {
				t.Errorf("Validate error mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
