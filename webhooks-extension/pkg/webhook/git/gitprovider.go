// /*
// Copyright 2019 The Tekton Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// 		http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */

package git

// import (
// 	"errors"
// 	"fmt"
// 	"net/url"
// 	"os"
// 	"strings"

// 	"github.com/tektoncd/experimental/webhooks-extension/pkg/model"
// )

// // GitProvider defines a
// type GitProvider interface {
// 	AddWebhook(gitURL *url.URL, accessToken, secretToken string) error
// 	DeleteWebhook(gitURL *url.URL, accessToken, secretToken string) error
// }

// // AddWebhook : attempts to add a webhook
// func (g Group) AddWebhook(hook model.Webhook, org, repo string) (err error) {
// 	return addOrRemoveWebhook(hook, org, repo, "add", g)
// }

// // RemoveWebhook : attempts to remove a webhook from the project
// func (g Group) RemoveWebhook(hook webhook, org, repo string) (err error) {
// 	return addOrRemoveWebhook(hook, org, repo, "remove", g)
// }

// func addOrRemoveWebhook(hook webhook, org, repo, action string, g Group) (err error) {
// 	// Configure the Git Provider
// 	gitProvider, err := g.createGitProviderForWebhook(hook, org, repo)
// 	if err != nil {
// 		return err
// 	}

// 	// Get webhook
// 	webhook, err := getWebhook(gitProvider)
// 	if err != nil {
// 		return err
// 	}

// 	if webhook == nil && action == "remove" {
// 		// Return without error because there is no webhook to be deleted
// 		// Error?
// 		return nil
// 	} else if webhook == nil && action == "add" {
// 		// Add the Webhook
// 		return gitProvider.AddWebhook(hook)
// 	} else if webhook != nil && action == "remove" {
// 		// Remove the Webhook
// 		return gitProvider.DeleteWebhook(webhook)
// 	} else if webhook != nil && action == "add" {
// 		// Return without error because the webhook already exists, so no need to create the webhook
// 		// Error?
// 		return nil
// 	}
// 	return errors.New("Unsupported action in call to AddOrRemoveWebhook")
// }

// // Create the GitProvider for the webhookData
// func (g Group) createGitProviderForWebhook(hook webhook, org, reponame string) (GitProvider, error) {
// 	gitURL, err := url.ParseRequestURI(hook.GitRepositoryURL)
// 	if err != nil {
// 		return nil, err
// 	}

// 	// Get extra git option to skip ssl verification
// 	sslVerify := true
// 	ssl := os.Getenv("SSL_VERIFICATION_ENABLED")
// 	if strings.ToLower(ssl) == "false" {
// 		sslVerify = false
// 	}

// 	if err != nil {
// 		return nil, err
// 	}

// 	// Determine which GitProvider to use
// 	switch {
// 	// PUBLIC GITHUB
// 	case strings.Contains(gitURL.Host, "github.com"):
// 		apiURL := "https://api.github.com/"
// 		return g.initGitHub(sslVerify, apiURL, hook.AccessTokenRef, org, reponame)
// 	// GHE
// 	case strings.Contains(gitURL.Host, "github"):
// 		apiURL := gitURL.Scheme + "://" + gitURL.Host + "/api/v3/"
// 		return g.initGitHub(sslVerify, apiURL, hook.AccessTokenRef, org, reponame)
// 	// NOT RECOGNIZED/SUPPORTED
// 	default:
// 		msg := fmt.Sprintf("Git Provider for project URL: %s not recognized", gitURL)
// 		return nil, errors.New(msg)
// 	}
// }

// // Get the webhook (returns nil, nil if no webhook is found)
// func getWebhook(gitProvider GitProvider) (GitWebhook, error) {
// 	hooks, err := gitProvider.GetAllWebhooks()
// 	if err != nil {
// 		return nil, err
// 	}
// 	for _, hook := range hooks {
// 		if os.Getenv("WEBHOOK_CALLBACK_URL") == hook.GetURL() {
// 			return hook, nil
// 		}
// 	}
// 	return nil, nil
// }
