/*
Copyright 2019 The Tekton Authors
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
		http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package endpoints

import (
	"math/rand"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/tektoncd/experimental/webhooks-extension/pkg/models"
	"github.com/tektoncd/experimental/webhooks-extension/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func init() {
	src = rand.NewSource(0)
}

func Test_credentialRequestToSecret(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		cred      models.CredentialRequest
		secret    *corev1.Secret
	}{
		{
			name:      "Cred 1",
			namespace: "ns1",
			cred: models.CredentialRequest{
				Name:        "cred1",
				AccessToken: "token1",
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cred1",
					Namespace: "ns1",
				},
				Type: corev1.SecretTypeOpaque,
				Data: map[string][]byte{
					AccessToken: []byte("token1"),
					SecretToken: utils.GetRandomToken(src),
				},
			},
		},
		{
			name:      "Cred 2",
			namespace: "ns2",
			cred: models.CredentialRequest{
				Name:        "cred2",
				AccessToken: "token2",
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cred2",
					Namespace: "ns2",
				},
				Type: corev1.SecretTypeOpaque,
				Data: map[string][]byte{
					AccessToken: []byte("token2"),
					SecretToken: utils.GetRandomToken(src),
				},
			},
		},
	}
	for i := range tests {
		t.Run(tests[i].name, func(t *testing.T) {
			secret := credentialRequestToSecret(tests[i].cred, tests[i].namespace)
			if diff := cmp.Diff(tests[i].secret, secret); diff != "" {
				t.Errorf("Secret mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func Test_secretToCredentialResponse(t *testing.T) {
	randomToken := utils.GetRandomToken(src)
	tests := []struct {
		name   string
		secret *corev1.Secret
		cred   models.CredentialResponse
	}{
		{
			name: "Cred 1",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cred1",
					Namespace: "ns1",
				},
				Type: corev1.SecretTypeOpaque,
				Data: map[string][]byte{
					AccessToken: []byte("token1"),
					SecretToken: randomToken,
				},
			},
			cred: models.CredentialResponse{
				CredentialRequest: models.CredentialRequest{
					Name:        "cred1",
					AccessToken: "token1",
				},
				SecretToken: string(randomToken),
			},
		},
		{
			name: "Cred 2",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cred2",
					Namespace: "ns2",
				},
				Type: corev1.SecretTypeOpaque,
				Data: map[string][]byte{
					AccessToken: []byte("token2"),
					SecretToken: randomToken,
				},
			},
			cred: models.CredentialResponse{
				CredentialRequest: models.CredentialRequest{
					Name:        "cred2",
					AccessToken: "token2",
				},
				SecretToken: string(randomToken),
			},
		},
	}
	for i := range tests {
		t.Run(tests[i].name, func(t *testing.T) {
			cred := secretToCredentialResponse(*tests[i].secret)
			if diff := cmp.Diff(tests[i].cred, cred); diff != "" {
				t.Errorf("Credential mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func Test_isCredential(t *testing.T) {
	tests := []struct {
		name   string
		secret corev1.Secret
		isCred bool
	}{
		// Correct
		{
			name: "AccessToken And SecretToken",
			secret: corev1.Secret{
				Data: map[string][]byte{
					AccessToken: []byte("accessToken"),
					SecretToken: []byte("secretToken"),
				},
			},
			isCred: true,
		},
		// Incorrect
		{
			name: "AccessToken Only",
			secret: corev1.Secret{
				Data: map[string][]byte{
					AccessToken: []byte("accessToken"),
				},
			},
			isCred: false,
		},
		{
			name: "SecretToken Only",
			secret: corev1.Secret{
				Data: map[string][]byte{
					SecretToken: []byte("secretToken"),
				},
			},
			isCred: false,
		},
		{
			name:   "Empty Secret",
			secret: corev1.Secret{},
			isCred: false,
		},
	}
	for i := range tests {
		t.Run(tests[i].name, func(t *testing.T) {
			isCred := isCredential(tests[i].secret)
			if diff := cmp.Diff(tests[i].isCred, isCred); diff != "" {
				t.Errorf("isCredential() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
