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
	"math/rand"
	"net/http"
	"time"

	restful "github.com/emicklei/go-restful"
	logging "github.com/tektoncd/experimental/webhooks-extension/pkg/logging"
	"github.com/tektoncd/experimental/webhooks-extension/pkg/model"
	"github.com/tektoncd/experimental/webhooks-extension/pkg/util"
	"golang.org/x/xerrors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var src = rand.NewSource(time.Now().UnixNano())

const (
	// AccessToken is a key within a K8s secret Data field. This value of this
	// key should be a git access token
	AccessToken = "accessToken"
	// SecretToken is a key within a K8s secret Data field. This value of this
	// key should be used to validate payloads (e.g. webhooks).
	SecretToken = "secretToken"
)

// CreateCredential creates a secret of type access token, which should store
// a Git access token. The created secret also generates a secret string, which
// should be used to verify against payloads (e.g. webhooks).
func (cg *Group) createCredential(request *restful.Request, response *restful.Response) {
	logging.Log.Debug("In CreateCredential")
	credReq := model.CredentialRequest{}

	if err := request.ReadEntity(&credReq); err != nil {
		err = xerrors.Errorf("Error parsing request body: %s", err)
		util.RespondError(response, err, http.StatusBadRequest)
		return
	}

	if err := credReq.Validate(); err != nil {
		err = xerrors.Errorf("Invalid credential request value: %s", err)
		util.RespondError(response, err, http.StatusBadRequest)
		return
	}
	secret := credentialRequestToSecret(credReq, cg.Defaults.Namespace)
	logging.Log.Debugf("Creating credential %s in namespace %s", credReq.Name, cg.Defaults.Namespace)

	if _, err := cg.K8sClient.CoreV1().Secrets(cg.Defaults.Namespace).Create(secret); err != nil {
		util.RespondError(response, err, http.StatusBadRequest)
		return
	}
	util.WriteResponseLocation(request.Request, response, credReq.Name)
}

// DeleteCredential deletes the specified credential
func (cg *Group) deleteCredential(request *restful.Request, response *restful.Response) {
	logging.Log.Debug("In DeleteCredential")
	credName := request.PathParameter("name")
	if credName == "" {
		err := xerrors.New("Secret name for deletion was not specified")
		util.RespondError(response, err, http.StatusBadRequest)
		return
	}
	logging.Log.Debugf("Deleting secret: %s", credName)
	// Assumes whatever secret name specified would be a valid credential
	err := cg.K8sClient.CoreV1().Secrets(cg.Defaults.Namespace).Delete(credName, &metav1.DeleteOptions{})
	if err != nil {
		var errorCode int
		switch {
		case k8serrors.IsNotFound(err):
			errorCode = http.StatusNotFound
		default:
			errorCode = http.StatusInternalServerError
		}
		util.RespondError(response, err, errorCode)
		return
	}
	response.WriteHeader(204)
}

// GetAllCredentials returns all the credentials specified within the default
// namespace
func (cg *Group) getAllCredentials(request *restful.Request, response *restful.Response) {
	// Get secrets from the resource K8sClient
	secrets, err := cg.K8sClient.CoreV1().Secrets(cg.Defaults.Namespace).List(metav1.ListOptions{})
	if err != nil {
		util.RespondError(response, err, http.StatusInternalServerError)
		return
	}

	// Parse K8s secrets to credentials
	creds := []model.CredentialResponse{}
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
func credentialRequestToSecret(cred model.CredentialRequest, namespace string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cred.Name,
			Namespace: namespace,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			AccessToken: []byte(cred.AccessToken),
			SecretToken: util.GetRandomToken(src),
		},
	}
}

// secretToCredential converts a K8s secret into a credentialResponse
func secretToCredentialResponse(s corev1.Secret) model.CredentialResponse {
	return model.CredentialResponse{
		CredentialRequest: model.CredentialRequest{
			Name:        s.Name,
			AccessToken: string(s.Data[AccessToken]),
		},
		SecretToken: string(s.Data[SecretToken]),
	}
}

// isCredential returns whether the specified secret is a credential. This is a
// simple check against whether the specified keys exist.
func isCredential(secret corev1.Secret) bool {
	return secret.Data[AccessToken] != nil && secret.Data[SecretToken] != nil
}
