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

package restapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/tektoncd/experimental/webhooks-extension/pkg/model"
	"github.com/tektoncd/experimental/webhooks-extension/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_createCredential(t *testing.T) {
	tests := []struct {
		name string
		cred model.CredentialRequest
		// Whether the credential should be exist before the request is made
		seed            bool
		statusCode      int
		contentLocation string
	}{
		// Correct
		{
			name: "Regular Credential",
			cred: model.CredentialRequest{
				Name:        "cred",
				AccessToken: "accessToken",
			},
			seed:            false,
			statusCode:      201,
			contentLocation: "/webhooks/credentials/cred",
		},
		// Incorrect
		{
			name: "Already Exists Credential",
			cred: model.CredentialRequest{
				Name:        "cred",
				AccessToken: "accessToken",
			},
			seed:       true,
			statusCode: http.StatusBadRequest,
		},
		{
			name: "No Name",
			cred: model.CredentialRequest{
				AccessToken: "accessToken",
			},
			seed:       false,
			statusCode: http.StatusBadRequest,
		},
		{
			name: "No Access Token",
			cred: model.CredentialRequest{
				Name: "cred",
			},
			seed:       false,
			statusCode: http.StatusBadRequest,
		},
	}
	for i := range tests {
		t.Run(tests[i].name, func(t *testing.T) {
			cg := DummyGroup()
			server := DummyServer(cg)
			// Seed secret
			if tests[i].seed {
				secret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      tests[i].cred.Name,
						Namespace: cg.Defaults.Namespace,
					},
				}
				if _, err := cg.K8sClient.CoreV1().Secrets(cg.Defaults.Namespace).Create(secret); err != nil {
					t.Fatalf("Error seeding resource: %s", err)
				}
			}
			// Intialize request
			jsonBytes, err := json.Marshal(tests[i].cred)
			if err != nil {
				t.Fatalf("Error marshalling response body: %s", err)
			}
			httpReq, err := DummyHTTPRequest("POST", fmt.Sprintf("%s/webhooks/credentials", server.URL), bytes.NewBuffer(jsonBytes))
			if err != nil {
				t.Fatal(err)
			}
			// Make request
			response, err := http.DefaultClient.Do(httpReq)
			if err != nil {
				t.Fatalf("Error on request: %s", err)
			}
			// Compare
			if diff := cmp.Diff(tests[i].statusCode, response.StatusCode); diff != "" {
				t.Errorf("Status code mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tests[i].contentLocation, response.Header.Get("Content-Location")); diff != "" {
				t.Errorf("Content location mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func Test_deleteCredential(t *testing.T) {
	tests := []struct {
		name string
		// Specifying the url allows us to specify an empty path parameter
		url      string
		credName string
		// Whether the credential should be exist before the request is made
		seed       bool
		statusCode int
	}{
		// Correct
		{
			name:       "Regular Path",
			url:        "/webhooks/credentials/cred",
			credName:   "cred",
			seed:       true,
			statusCode: 204,
		},
		// Incorrect
		{
			name:       "No secret",
			url:        "/webhooks/credentials/cred",
			credName:   "cred",
			seed:       false,
			statusCode: http.StatusNotFound,
		},
		{
			name:       "Bad path",
			url:        "/webhooks/credentials/",
			credName:   "cred",
			seed:       true,
			statusCode: http.StatusBadRequest,
		},
	}
	for i := range tests {
		t.Run(tests[i].name, func(t *testing.T) {
			cg := DummyGroup()
			server := DummyServer(cg)
			// Seed secret
			if tests[i].seed {
				secret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      tests[i].credName,
						Namespace: cg.Defaults.Namespace,
					},
				}
				if _, err := cg.K8sClient.CoreV1().Secrets(cg.Defaults.Namespace).Create(secret); err != nil {
					t.Fatalf("Error seeding resource: %s", err)
				}
			}
			httpReq, err := DummyHTTPRequest("DELETE", fmt.Sprintf("%s%s", server.URL, tests[i].url), nil)
			if err != nil {
				t.Fatal(err)
			}
			// Make request
			response, err := http.DefaultClient.Do(httpReq)
			if err != nil {
				t.Fatalf("Error on request: %s", err)
			}
			// Compare
			if diff := cmp.Diff(tests[i].statusCode, response.StatusCode); diff != "" {
				t.Errorf("Status code mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func Test_getAllCredentials(t *testing.T) {
	seededSecretToken := "Ze7gKS3PSbsRMjIFYHmz"
	tests := []struct {
		name        string
		credentials []model.CredentialResponse
		statusCode  int
	}{
		{
			name:        "No Credential",
			credentials: []model.CredentialResponse{},
			statusCode:  http.StatusOK,
		},
		{
			name: "One Credential",
			credentials: []model.CredentialResponse{
				model.CredentialResponse{
					CredentialRequest: model.CredentialRequest{
						Name:        "cred1",
						AccessToken: "accessToken",
					},
					SecretToken: seededSecretToken,
				},
			},
			statusCode: http.StatusOK,
		},
		{
			name: "Two Credentials",
			credentials: []model.CredentialResponse{
				model.CredentialResponse{
					CredentialRequest: model.CredentialRequest{
						Name:        "cred1",
						AccessToken: "accessToken",
					},
					SecretToken: seededSecretToken,
				},
				model.CredentialResponse{
					CredentialRequest: model.CredentialRequest{
						Name:        "cred2",
						AccessToken: "accessToken",
					},
					SecretToken: seededSecretToken,
				},
			},
			statusCode: http.StatusOK,
		},
	}
	for i := range tests {
		t.Run(tests[i].name, func(t *testing.T) {
			cg := DummyGroup()
			server := DummyServer(cg)
			// Seed secret
			for _, cred := range tests[i].credentials {
				secret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      cred.Name,
						Namespace: cg.Defaults.Namespace,
					},
					Type: corev1.SecretTypeOpaque,
					Data: map[string][]byte{
						AccessToken: []byte(cred.AccessToken),
						SecretToken: []byte(cred.SecretToken),
					},
				}
				if _, err := cg.K8sClient.CoreV1().Secrets(cg.Defaults.Namespace).Create(secret); err != nil {
					t.Fatalf("Error seeding resource: %s", err)
				}
			}
			// Intialize request
			httpReq, err := DummyHTTPRequest("GET", fmt.Sprintf("%s/webhooks/credentials", server.URL), nil)
			if err != nil {
				t.Fatal(err)
			}
			// Make request
			response, err := http.DefaultClient.Do(httpReq)
			if err != nil {
				t.Fatalf("Error on request: %s", err)
			}
			// Read request
			bodyBytes, err := ioutil.ReadAll(response.Body)
			if err != nil {
				t.Fatalf("Failed to read body: %s", err)
			}
			var credentials []model.CredentialResponse
			if err := json.Unmarshal(bodyBytes, &credentials); err != nil {
				t.Fatalf("Failed to unmarshal body: %s", err)
			}
			// Compare
			if diff := cmp.Diff(tests[i].credentials, credentials); diff != "" {
				t.Errorf("Credentials mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tests[i].statusCode, response.StatusCode); diff != "" {
				t.Errorf("Status code mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func Test_credentialRequestToSecret(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		cred      model.CredentialRequest
		secret    *corev1.Secret
	}{
		{
			name:      "Cred 1",
			namespace: "ns1",
			cred: model.CredentialRequest{
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
					SecretToken: []byte("sJyQs22cRR81AZcI3qh2"),
				},
			},
		},
		{
			name:      "Cred 2",
			namespace: "ns2",
			cred: model.CredentialRequest{
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
					SecretToken: []byte("sJyQs22cRR81AZcI3qh2"),
				},
			},
		},
	}
	for i := range tests {
		t.Run(tests[i].name, func(t *testing.T) {
			src = rand.NewSource(0)
			secret := credentialRequestToSecret(tests[i].cred, tests[i].namespace)
			if diff := cmp.Diff(tests[i].secret, secret); diff != "" {
				t.Errorf("Secret mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func Test_secretToCredentialResponse(t *testing.T) {
	randomToken := util.GetRandomToken(src)
	tests := []struct {
		name   string
		secret *corev1.Secret
		cred   model.CredentialResponse
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
			cred: model.CredentialResponse{
				CredentialRequest: model.CredentialRequest{
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
			cred: model.CredentialResponse{
				CredentialRequest: model.CredentialRequest{
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
