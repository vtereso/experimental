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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func init() {
	src = rand.NewSource(0)
}

// func TestCreateCredential(t *testing.T) {

// }

// func TestDeleteCredential(t *testing.T) {

// }

// func TestGetAllCredentials(t *testing.T) {

// }

func Test_credentialRequestToSecret(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		cred      credentialRequest
		secret    *corev1.Secret
	}{
		{
			name:      "Cred 1",
			namespace: "ns1",
			cred: credentialRequest{
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
					accessToken: []byte("token1"),
					secretToken: getRandomToken(),
				},
			},
		},
		{
			name:      "Cred 2",
			namespace: "ns2",
			cred: credentialRequest{
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
					accessToken: []byte("token2"),
					secretToken: getRandomToken(),
				},
			},
		},
	}
	for i := range tests {
		t.Run(tests[i].name, func(t *testing.T) {
			secret := credentialRequestToSecret(tests[i].cred, tests[i].namespace)
			if diff := cmp.Diff(tests[i].secret, secret); diff != "" {
				t.Errorf("credentialToSecret() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func Test_secretToCredentialResponse(t *testing.T) {
	tests := []struct {
		name   string
		cred   credentialResponse
		secret *corev1.Secret
	}{
		{
			name:      "Cred 1",
			namespace: "ns1",
			cred: credentialResponse{
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
					accessToken: []byte("token1"),
					secretToken: getRandomToken(),
				},
			},
		},
		{
			name:      "Cred 2",
			namespace: "ns2",
			cred: credentialResponse{
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
					accessToken: []byte("token2"),
					secretToken: getRandomToken(),
				},
			},
		},
	}
	for i := range tests {
		t.Run(tests[i].name, func(t *testing.T) {
			secret := secretToCredentialResponse(tests[i].cred, tests[i].namespace)
			if diff := cmp.Diff(tests[i].secret, secret); diff != "" {
				t.Errorf("credentialToSecret() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func Test_getRandomToken(t *testing.T) {
	tests := []struct {
		name  string
		bytes []byte
	}{
		{
			name:  "Random Token",
			bytes: []byte("Ze7gKS3PSbsRMjIFYHmz"),
		},
		{
			name:  "Another Random Token",
			bytes: []byte("Ze7gKS3PSbsRMjIFYHmz"),
		},
	}
	for i := range tests {
		t.Run(tests[i].name, func(t *testing.T) {
			bytes := getRandomToken()
			if diff := cmp.Diff(tests[i].bytes, bytes); diff != "" {
				t.Errorf("getRandomToken() mismatch (-want +got):\n%s", diff)
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
					accessToken: []byte("accessToken"),
					secretToken: []byte("secretToken"),
				},
			},
			isCred: true,
		},
		// Incorrect
		{
			name: "AccessToken Only",
			secret: corev1.Secret{
				Data: map[string][]byte{
					accessToken: []byte("accessToken"),
				},
			},
			isCred: false,
		},
		{
			name: "SecretToken Only",
			secret: corev1.Secret{
				Data: map[string][]byte{
					secretToken: []byte("secretToken"),
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
