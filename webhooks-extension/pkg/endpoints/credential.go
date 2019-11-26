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
	"net/http"
	"time"

	restful "github.com/emicklei/go-restful"
	logging "github.com/tektoncd/experimental/webhooks-extension/pkg/logging"
	"github.com/tektoncd/experimental/webhooks-extension/pkg/utils"
	"golang.org/x/xerrors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var src = rand.NewSource(time.Now().UnixNano())

const (
	// accessToken is a key within a K8s secret Data field. This value of this
	// key should be a git access token
	accessToken = "accessToken"
	// secretToken is a key within a K8s secret Data field. This value of this
	// key should be used to validate payloads (e.g. webhooks).
	secretToken   = "secretToken"
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
	letterBytes   = "123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
)

// credentialRequest is the intended structure of an http request for creating a
// credential, which is a K8s secret of type access token
type credentialRequest struct {
	Name        string `json:"name"`
	AccessToken string `json:"accesstoken"`
}

// credentialRequest is the intended structure of an http response when getting
// a credential, which is a K8s secret of type access token
type credentialResponse struct {
	credentialRequest `json:",inline"`
	SecretToken       string `json:"secrettoken,omitempty"`
}

// CreateCredential creates a secret of type access token, which should store
// a Git access token. The created secret also generates a secret string, which
// should be used to verify against payloads (e.g. webhooks).
func (r Resource) CreateCredential(request *restful.Request, response *restful.Response) {
	logging.Log.Debug("In CreateCredential")
	cred := credentialRequest{}

	if err := request.ReadEntity(&cred); err != nil {
		err = xerrors.Errorf("Error parsing request body: %s", err)
		utils.RespondError(response, err, http.StatusBadRequest)
		return
	}

	if err := checkCredentialRequest(cred); err != nil {
		err = xerrors.Errorf("Invalid credential value: %s", err)
		utils.RespondError(response, err, http.StatusBadRequest)
		return
	}
	secret := credentialRequestToSecret(cred, r.Defaults.Namespace)
	logging.Log.Debugf("Creating credential %s in namespace %s", cred.Name, r.Defaults.Namespace)

	if _, err := r.K8sClient.CoreV1().Secrets(r.Defaults.Namespace).Create(secret); err != nil {
		utils.RespondError(response, err, http.StatusBadRequest)
		return
	}
	utils.WriteResponseLocation(request, response, cred.Name)
}

// DeleteCredential deletes the specified credential
func (r Resource) DeleteCredential(request *restful.Request, response *restful.Response) {
	logging.Log.Debug("In DeleteCredential")
	credName := request.PathParameter("name")
	if credName == "" {
		err := xerrors.New("Secret name for deletion was not specified")
		utils.RespondError(response, err, http.StatusBadRequest)
		return
	}
	logging.Log.Debugf("Deleting secret: %s", credName)
	// Assumes whatever secret name specified would be a valid credential
	err := r.K8sClient.CoreV1().Secrets(r.Defaults.Namespace).Delete(credName, &metav1.DeleteOptions{})
	if err != nil {
		var errorCode int
		switch {
		case k8serrors.IsNotFound(err):
			errorCode = http.StatusNotFound
		default:
			errorCode = http.StatusInternalServerError
		}
		utils.RespondError(response, err, errorCode)
		return
	}
	response.WriteHeader(204)
}

// GetAllCredentials returns all the credentials specified within the default
// namespace
func (r Resource) GetAllCredentials(request *restful.Request, response *restful.Response) {
	// Get secrets from the resource K8sClient
	secrets, err := r.K8sClient.CoreV1().Secrets(r.Defaults.Namespace).List(metav1.ListOptions{})
	if err != nil {
		utils.RespondError(response, err, http.StatusInternalServerError)
		return
	}

	// Parse K8s secrets to credentials
	creds := []credentialResponse{}
	for _, secret := range secrets.Items {
		if isCredential(secret) {
			// Return only the names
			creds = append(creds, secretToCredentialResponse(secret))
		}
	}
	logging.Log.Infof("getAllCredentials returning +%v", creds)

	// Write the response
	response.AddHeader("Content-Type", "application/json")
	response.WriteEntity(creds)
}

// credentialToSecret converts a credentialRequest into a K8s secret
func credentialRequestToSecret(cred credentialRequest, namespace string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cred.Name,
			Namespace: namespace,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			accessToken: []byte(cred.AccessToken),
			secretToken: getRandomToken(),
		},
	}
}

// secretToCredential converts a K8s secret into a credentialResponse
func secretToCredentialResponse(s corev1.Secret) credentialResponse {
	return credentialResponse{
		credentialRequest: credentialRequest{
			Name:        s.Name,
			AccessToken: string(s.Data[accessToken]),
		},
		SecretToken: string(s.Data[secretToken]),
	}
}

// getRandomToken generates a random 20-character secret.
// Source: https://stackoverflow.com/questions/22892120/how-to-generate-a-random-string-of-a-fixed-length-in-go
func getRandomToken() []byte {
	n := 20
	b := make([]byte, n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}
	return b
}

// isCredential returns whether the specified secret is a credential. This is a
// simple check against whether the specified keys exist.
func isCredential(secret corev1.Secret) bool {
	return secret.Data[accessToken] != nil && secret.Data[secretToken] != nil
}
