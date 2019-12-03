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

package endpoints

import (
	"fmt"
	"net/url"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	routesv1 "github.com/openshift/api/route/v1"
	"github.com/tektoncd/experimental/webhooks-extension/pkg/client/fake"
	"github.com/tektoncd/experimental/webhooks-extension/pkg/models"
	pipelinesv1alpha1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	triggersv1alpha1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func Test_deletePipelineRuns(t *testing.T) {}

// func deletePipelineRuns(cg *client.Group, repoURL *url.URL, namespace, pipeline string) error {
// }

func Test_makePipelineRunSelectorSet(t *testing.T) {}

// func makePipelineRunSelectorSet(repoURL *url.URL) map[string]string {
// }

func Test_createOpenshiftRoute(t *testing.T) {
	tests := []struct {
		name        string
		serviceName string
		route       *routesv1.Route
		hasErr      bool
	}{
		{
			name:        "OpenShift Route",
			serviceName: "route",
			route: &routesv1.Route{
				ObjectMeta: metav1.ObjectMeta{
					Name: "route",
				},
				Spec: routesv1.RouteSpec{
					To: routesv1.RouteTargetReference{
						Kind: "Service",
						Name: "route",
					},
				},
			},
			hasErr: false,
		},
	}
	for i := range tests {
		t.Run(tests[i].name, func(t *testing.T) {
			cg := fake.DummyGroup()
			var hasErr bool
			if err := createOpenshiftRoute(cg, tests[i].serviceName); err != nil {
				hasErr = true
			}
			if diff := cmp.Diff(tests[i].hasErr, hasErr); diff != "" {
				t.Fatalf("Error mismatch (-want +got):\n%s", diff)
			}
			route, _ := cg.RoutesClient.RouteV1().Routes(cg.Defaults.Namespace).Get(tests[i].serviceName, metav1.GetOptions{})
			if diff := cmp.Diff(tests[i].route, route); diff != "" {
				t.Errorf("Route mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func Test_deleteOpenshiftRoute(t *testing.T) {
	tests := []struct {
		name      string
		routeName string
		hasErr    bool
	}{
		{
			name:      "OpenShift Route",
			routeName: "route",
			hasErr:    false,
		},
	}
	for i := range tests {
		t.Run(tests[i].name, func(t *testing.T) {
			cg := fake.DummyGroup()
			// Seed route for deletion
			route := &routesv1.Route{
				ObjectMeta: metav1.ObjectMeta{
					Name: tests[i].routeName,
				},
			}
			if _, err := cg.RoutesClient.RouteV1().Routes(cg.Defaults.Namespace).Create(route); err != nil {
				t.Fatal(err)
			}
			// Delete
			var hasErr bool
			if err := deleteOpenshiftRoute(cg, tests[i].routeName); err != nil {
				hasErr = true
			}
			if diff := cmp.Diff(tests[i].hasErr, hasErr); diff != "" {
				t.Fatalf("Error mismatch (-want +got):\n%s", diff)
			}
			_, err := cg.RoutesClient.RouteV1().Routes(cg.Defaults.Namespace).Get(tests[i].routeName, metav1.GetOptions{})
			if err == nil {
				t.Errorf("Route not expected")
			}
		})
	}
}

func Test_createIngress(t *testing.T) {}

// func createIngress(cg *client.Group, serviceName string) error {
// }

func Test_deleteIngress(t *testing.T) {}

// func deleteIngress(cg *client.Group, ingressName string) error {
// }

func Test_addWebhookTriggers(t *testing.T) {}

// func addWebhookTriggers(cg *client.Group, eventListener *triggersv1alpha1.EventListener, webhook models.Webhook) {
// }

func Test_removeWebhookTriggers(t *testing.T) {}

// func removeWebhookTriggers(cg *client.Group, eventListener *triggersv1alpha1.EventListener, webhookName string) {
// }

func Test_newTrigger(t *testing.T) {}

// func newTrigger(triggerName, bindingName, templateName, interceptorNamespace, repoURL, eventType, secretName string, params []pipelinesv1alpha1.Param) triggersv1alpha1.EventListenerTrigger {
// }

func Test_getMonitorTriggerParams(t *testing.T) {}

// func getMonitorTriggerParams(cg *client.Group, w models.Webhook) {
// }

func Test_getPipelineTriggerParams(t *testing.T) {}

// func getPipelineTriggerParams(w models.Webhook) []pipelinesv1alpha1.Param {
// }

func Test_triggerToWebhook(t *testing.T) {
	tests := []struct {
		name    string
		trigger triggersv1alpha1.EventListenerTrigger
		webhook *models.Webhook
		hasErr  bool
	}{
		// Correct
		{
			name: "Valid Trigger",
			trigger: triggersv1alpha1.EventListenerTrigger{
				Name: "trigger-some-prefix",
				Template: triggersv1alpha1.EventListenerTemplate{
					Name: "pipeline-some-prefix",
				},
				Params: []pipelinesv1alpha1.Param{
					pipelinesv1alpha1.Param{
						Name: wextTargetNamespace,
						Value: pipelinesv1alpha1.ArrayOrString{
							Type:      pipelinesv1alpha1.ParamTypeString,
							StringVal: "namespace",
						},
					},
					pipelinesv1alpha1.Param{
						Name: wextServiceAccount,
						Value: pipelinesv1alpha1.ArrayOrString{
							Type:      pipelinesv1alpha1.ParamTypeString,
							StringVal: "serviceAccount",
						},
					},
					pipelinesv1alpha1.Param{
						Name: wextDockerRegistry,
						Value: pipelinesv1alpha1.ArrayOrString{
							Type:      pipelinesv1alpha1.ParamTypeString,
							StringVal: "dockerRegistry",
						},
					},
				},
				Interceptor: &triggersv1alpha1.EventInterceptor{
					Header: []pipelinesv1alpha1.Param{
						pipelinesv1alpha1.Param{
							Name: WextInterceptorSecretName,
							Value: pipelinesv1alpha1.ArrayOrString{
								Type:      pipelinesv1alpha1.ParamTypeString,
								StringVal: "secretName",
							},
						},
						pipelinesv1alpha1.Param{
							Name: WextInterceptorRepoURL,
							Value: pipelinesv1alpha1.ArrayOrString{
								Type:      pipelinesv1alpha1.ParamTypeString,
								StringVal: "repoURL",
							},
						},
					},
				},
			},
			webhook: &models.Webhook{
				Name:             "trigger",
				Namespace:        "namespace",
				ServiceAccount:   "serviceAccount",
				AccessTokenRef:   "secretName",
				Pipeline:         "pipeline",
				DockerRegistry:   "dockerRegistry",
				GitRepositoryURL: "repoURL",
			},
			hasErr: false,
		},
		// Incorrect
		{
			name: "Missing Params",
			trigger: triggersv1alpha1.EventListenerTrigger{
				Name: "trigger-some-prefix",
				Template: triggersv1alpha1.EventListenerTemplate{
					Name: "pipeline-some-prefix",
				},
				Params: []pipelinesv1alpha1.Param{},
				Interceptor: &triggersv1alpha1.EventInterceptor{
					Header: []pipelinesv1alpha1.Param{},
				},
			},
			webhook: nil,
			hasErr:  true,
		},
	}
	for i := range tests {
		t.Run(tests[i].name, func(t *testing.T) {
			var hasErr bool
			webhook, err := triggerToWebhook(tests[i].trigger)
			if err != nil {
				hasErr = true
			}
			if diff := cmp.Diff(tests[i].hasErr, hasErr); diff != "" {
				t.Fatal("Error mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tests[i].webhook, webhook); diff != "" {
				t.Errorf("Webhook mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func Test_filterWebhooksByRepo(t *testing.T) {
	tests := []struct {
		name     string
		webhooks []models.Webhook
		repoURL  string
		size     int
	}{
		{
			name: "No matches",
			webhooks: []models.Webhook{
				models.Webhook{GitRepositoryURL: "repo1"},
				models.Webhook{GitRepositoryURL: "repo2"},
				models.Webhook{GitRepositoryURL: "repo3"},
			},
			repoURL: "repo0",
			size:    0,
		},
		{
			name: "One matches",
			webhooks: []models.Webhook{
				models.Webhook{GitRepositoryURL: "repo1"},
				models.Webhook{GitRepositoryURL: "repo2"},
				models.Webhook{GitRepositoryURL: "repo3"},
			},
			repoURL: "repo1",
			size:    1,
		},
		{
			name: "All match",
			webhooks: []models.Webhook{
				models.Webhook{GitRepositoryURL: "repo1"},
				models.Webhook{GitRepositoryURL: "repo1"},
				models.Webhook{GitRepositoryURL: "repo1"},
			},
			repoURL: "repo1",
			size:    3,
		},
	}
	for i := range tests {
		t.Run(tests[i].name, func(t *testing.T) {
			webhooks := filterWebhooksByRepo(tests[i].webhooks, tests[i].repoURL)
			if diff := cmp.Diff(tests[i].size, len(webhooks)); diff != "" {
				t.Errorf("Webhook list length mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func Test_findWebhookByName(t *testing.T) {
	tests := []struct {
		name        string
		webhooks    []models.Webhook
		webhookName string
		hasErr      bool
	}{
		{
			name: "Existing webhook",
			webhooks: []models.Webhook{
				models.Webhook{Name: "webhook1"},
				models.Webhook{Name: "webhook2"},
			},
			webhookName: "webhook1",
			hasErr:      false,
		},
		{
			name: "Nonexisting webhook",
			webhooks: []models.Webhook{
				models.Webhook{Name: "webhook1"},
				models.Webhook{Name: "webhook2"},
			},
			webhookName: "webhook3",
			hasErr:      true,
		},
	}
	for i := range tests {
		t.Run(tests[i].name, func(t *testing.T) {
			var hasErr bool
			_, err := findWebhookByName(tests[i].webhooks, tests[i].webhookName)
			if err != nil {
				hasErr = true
			}
			if diff := cmp.Diff(tests[i].hasErr, hasErr); diff != "" {
				t.Errorf("Error mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func Test_getPipelineNameFromTrigger(t *testing.T) {
	tests := []struct {
		name        string
		trigger     triggersv1alpha1.EventListenerTrigger
		webhookName string
	}{
		{
			name: "Get Name",
			trigger: triggersv1alpha1.EventListenerTrigger{
				Template: triggersv1alpha1.EventListenerTemplate{
					Name: fmt.Sprintf("%s-%s", "webhook", triggerTemplatePostfix),
				},
			},
			webhookName: "webhook",
		},
	}
	for i := range tests {
		t.Run(tests[i].name, func(t *testing.T) {
			webhookName := getPipelineNameFromTrigger(tests[i].trigger)
			if diff := cmp.Diff(tests[i].webhookName, webhookName); diff != "" {
				t.Errorf("Webhook name mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func Test_getWebhookNameFromTrigger(t *testing.T) {
	tests := []struct {
		name        string
		trigger     triggersv1alpha1.EventListenerTrigger
		webhookName string
	}{
		{
			name: "Get Name",
			trigger: triggersv1alpha1.EventListenerTrigger{
				Name: fmt.Sprintf("%s-%s", "webhook", "postfix"),
			},
			webhookName: "webhook",
		},
	}
	for i := range tests {
		t.Run(tests[i].name, func(t *testing.T) {
			webhookName := getWebhookNameFromTrigger(tests[i].trigger)
			if diff := cmp.Diff(tests[i].webhookName, webhookName); diff != "" {
				t.Errorf("Webhook name mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func Test_getBaseEventListener(t *testing.T) {
	tests := []struct {
		name          string
		namespace     string
		eventListener *triggersv1alpha1.EventListener
	}{
		{
			name:      "Namespace1",
			namespace: "namespace1",
			eventListener: &triggersv1alpha1.EventListener{
				ObjectMeta: metav1.ObjectMeta{
					Name:      eventListenerName,
					Namespace: "namespace1",
				},
				Spec: triggersv1alpha1.EventListenerSpec{
					ServiceAccountName: eventListenerSA,
				},
			},
		},
		{
			name:      "Namespace2",
			namespace: "namespace2",
			eventListener: &triggersv1alpha1.EventListener{
				ObjectMeta: metav1.ObjectMeta{
					Name:      eventListenerName,
					Namespace: "namespace2",
				},
				Spec: triggersv1alpha1.EventListenerSpec{
					ServiceAccountName: eventListenerSA,
				},
			},
		},
	}
	for i := range tests {
		t.Run(tests[i].name, func(t *testing.T) {
			eventListener := getBaseEventListener(tests[i].namespace)
			if diff := cmp.Diff(tests[i].eventListener, eventListener); diff != "" {
				t.Errorf("EventListener mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func Test_getWebhookEventListener(t *testing.T) {
	tests := []struct {
		name              string
		seedEventListener *triggersv1alpha1.EventListener
		hasErr            bool
	}{
		{
			name: "Existing EventListener",
			seedEventListener: &triggersv1alpha1.EventListener{
				ObjectMeta: metav1.ObjectMeta{
					Name: eventListenerName,
				},
			},
			hasErr: false,
		},
		{
			name:              "Nonexisting EventListener",
			seedEventListener: nil,
			hasErr:            true,
		},
	}
	for i := range tests {
		t.Run(tests[i].name, func(t *testing.T) {
			cg := fake.DummyGroup()
			if tests[i].seedEventListener != nil {
				_, err := cg.TriggersClient.TektonV1alpha1().EventListeners(cg.Defaults.Namespace).Create(tests[i].seedEventListener)
				if err != nil {
					t.Fatal(err)
				}
			}
			var hasErr bool
			if _, err := getWebhookEventListener(cg); err != nil {
				hasErr = true
			}
			if diff := cmp.Diff(tests[i].hasErr, hasErr); diff != "" {
				t.Errorf("Error mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func Test_createEventListener(t *testing.T) {
	tests := []struct {
		name          string
		eventListener *triggersv1alpha1.EventListener
		hasErr        bool
	}{
		{
			name: "Create EventListener",
			eventListener: &triggersv1alpha1.EventListener{
				ObjectMeta: metav1.ObjectMeta{
					Name: eventListenerName,
				},
			},
			hasErr: false,
		},
	}
	for i := range tests {
		t.Run(tests[i].name, func(t *testing.T) {
			cg := fake.DummyGroup()
			var hasErr bool
			if err := createEventListener(cg, tests[i].eventListener); err != nil {
				hasErr = true
			}
			if diff := cmp.Diff(tests[i].hasErr, hasErr); diff != "" {
				t.Errorf("Error mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func Test_updateEventListener(t *testing.T) {
	tests := []struct {
		name          string
		eventListener *triggersv1alpha1.EventListener
		hasErr        bool
	}{
		{
			name: "Update EventListener",
			eventListener: &triggersv1alpha1.EventListener{
				ObjectMeta: metav1.ObjectMeta{
					Name: eventListenerName,
				},
			},
			hasErr: false,
		},
	}
	for i := range tests {
		t.Run(tests[i].name, func(t *testing.T) {
			cg := fake.DummyGroup()
			_, err := cg.TriggersClient.TektonV1alpha1().EventListeners(cg.Defaults.Namespace).Update(tests[i].eventListener)
			if err != nil {
				t.Fatal(err)
			}
			var hasErr bool
			if err := updateEventListener(cg, tests[i].eventListener); err != nil {
				hasErr = true
			}
			if diff := cmp.Diff(tests[i].hasErr, hasErr); diff != "" {
				t.Errorf("Error mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func Test_deleteEventListener(t *testing.T) {
	tests := []struct {
		name              string
		seedEventListener *triggersv1alpha1.EventListener
		hasErr            bool
	}{
		// Correct
		{
			name: "Seeded EventListener",
			seedEventListener: &triggersv1alpha1.EventListener{
				ObjectMeta: metav1.ObjectMeta{
					Name: eventListenerName,
				},
			},
			hasErr: false,
		},
		// Incorrect
		{
			name:              "Unseeded EventListener",
			seedEventListener: nil,
			hasErr:            true,
		},
	}
	for i := range tests {
		t.Run(tests[i].name, func(t *testing.T) {
			cg := fake.DummyGroup()
			if tests[i].seedEventListener != nil {
				_, err := cg.TriggersClient.TektonV1alpha1().EventListeners(cg.Defaults.Namespace).Create(tests[i].seedEventListener)
				if err != nil {
					t.Fatal(err)
				}
			}
			var hasErr bool
			if err := deleteEventListener(cg); err != nil {
				hasErr = true
			}
			if diff := cmp.Diff(tests[i].hasErr, hasErr); diff != "" {
				t.Errorf("Error mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func Test_getWebhooksFromEventListener(t *testing.T) {
	tests := []struct {
		name     string
		webhooks []models.Webhook
	}{
		{
			name: "One Webhook",
			webhooks: []models.Webhook{
				{
					Name:             "name",
					Namespace:        "namespace",
					ServiceAccount:   "serviceAccount",
					AccessTokenRef:   "accessTokenRef",
					Pipeline:         "pipeline",
					DockerRegistry:   "dockerRegistry",
					GitRepositoryURL: "https://gitpalace.com/org/repo",
				},
			},
		},
		{
			name: "Two Webhooks",
			webhooks: []models.Webhook{
				{
					Name:             "name1",
					Namespace:        "namespace1",
					ServiceAccount:   "serviceAccount1",
					AccessTokenRef:   "accessTokenRef1",
					Pipeline:         "pipeline1",
					DockerRegistry:   "dockerRegistry",
					GitRepositoryURL: "https://gitpalace.com/org/repo",
				},
				{
					Name:             "name2",
					Namespace:        "namespace2",
					ServiceAccount:   "serviceAccount2",
					AccessTokenRef:   "accessTokenRef2",
					Pipeline:         "pipeline2",
					DockerRegistry:   "dockerRegistry",
					GitRepositoryURL: "https://gitpalace.com/org/repo",
				},
			},
		},
	}
	for i := range tests {
		t.Run(tests[i].name, func(t *testing.T) {
			cg := fake.DummyGroup()
			el := getBaseEventListener(cg.Defaults.Namespace)
			t.Log("Trigger spec:", el.Spec.Triggers)
			for _, webhook := range tests[i].webhooks {
				addWebhookTriggers(cg, el, webhook)
			}
			webhooks := getWebhooksFromEventListener(*el)
			if diff := cmp.Diff(tests[i].webhooks, webhooks); diff != "" {
				t.Errorf("Webhooks mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func Test_waitForEventListenerStatus(t *testing.T) {
	tests := []struct {
		name              string
		seedEventListener *triggersv1alpha1.EventListener
		hasErr            bool
	}{
		// Correct
		{
			name: "EventListener With Status",
			seedEventListener: &triggersv1alpha1.EventListener{
				ObjectMeta: metav1.ObjectMeta{
					Name: eventListenerName,
				},
				Status: triggersv1alpha1.EventListenerStatus{
					Configuration: triggersv1alpha1.EventListenerConfig{
						GeneratedResourceName: "generatedName",
					},
				},
			},
			hasErr: false,
		},
		{
			name: "EventListener Without Status",
			seedEventListener: &triggersv1alpha1.EventListener{
				ObjectMeta: metav1.ObjectMeta{
					Name: eventListenerName,
				},
			},
			hasErr: false,
		},
		// Incorrect
		{
			name:   "No EventListener",
			hasErr: true,
		},
	}
	for i := range tests {
		t.Run(tests[i].name, func(t *testing.T) {
			cg := fake.DummyGroup()
			if tests[i].seedEventListener != nil {
				// Simulate async update to the EventListener status
				if tests[i].seedEventListener.Status.Configuration.GeneratedResourceName == "" {
					go func() {
						// Ensure first check fails
						time.Sleep(time.Millisecond * 100)
						tests[i].seedEventListener.Status.Configuration.GeneratedResourceName = "generatedNamed"
						_, err := cg.TriggersClient.TektonV1alpha1().EventListeners(cg.Defaults.Namespace).Update(tests[i].seedEventListener)
						if err != nil {
							t.Fatal(err)
						}
					}()
				}
				_, err := cg.TriggersClient.TektonV1alpha1().EventListeners(cg.Defaults.Namespace).Create(tests[i].seedEventListener)
				if err != nil {
					t.Fatal(err)
				}
			}
			var hasErr bool
			_, err := waitForEventListenerStatus(cg)
			if err != nil {
				hasErr = true
			}
			if diff := cmp.Diff(tests[i].hasErr, hasErr); diff != "" {
				t.Errorf("Error mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func Test_getGitValues(t *testing.T) {
	tests := []struct {
		name   string
		url    url.URL
		server string
		org    string
		repo   string
	}{
		{
			name: "GitHub",
			url: url.URL{
				Host: "github.com",
				Path: "/org/repo",
			},
			server: "github.com",
			org:    "org",
			repo:   "repo",
		},
		{
			name: "GitLab",
			url: url.URL{
				Host: "gitlab.com",
				Path: "/org/repo",
			},
			server: "gitlab.com",
			org:    "org",
			repo:   "repo",
		},
	}
	for i := range tests {
		t.Run(tests[i].name, func(t *testing.T) {
			server, org, repo := getGitValues(tests[i].url)
			if diff := cmp.Diff(tests[i].server, server); diff != "" {
				t.Errorf("Server mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tests[i].org, org); diff != "" {
				t.Errorf("Org mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tests[i].repo, repo); diff != "" {
				t.Errorf("Repo mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func Test_getWebhookSecretTokens(t *testing.T) {
	tests := []struct {
		name        string
		seedSecret  *corev1.Secret
		secretName  string
		accessToken string
		secretToken string
		hasErr      bool
	}{
		// Correct
		{
			name: "Seeded Secret With Tokens",
			seedSecret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "secret",
				},
				Type: corev1.SecretTypeOpaque,
				Data: map[string][]byte{
					AccessToken: []byte("accessToken"),
					SecretToken: []byte("secretToken"),
				},
			},
			secretName:  "secret",
			accessToken: "accessToken",
			secretToken: "secretToken",
			hasErr:      false,
		},
		// Invalid
		{
			name:        "No Seeded Secret",
			secretName:  "secret",
			accessToken: "accessToken",
			secretToken: "secretToken",
			hasErr:      true,
		},
		{
			name: "Seeded Secret No AccessToken",
			seedSecret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "secret",
				},
				Type: corev1.SecretTypeOpaque,
				Data: map[string][]byte{
					SecretToken: []byte("secretToken"),
				},
			},
			secretName:  "secret",
			accessToken: "accessToken",
			secretToken: "secretToken",
			hasErr:      true,
		},
		{
			name: "Seeded Secret No SecretToken",
			seedSecret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "secret",
				},
				Type: corev1.SecretTypeOpaque,
				Data: map[string][]byte{
					AccessToken: []byte("accessToken"),
				},
			},
			secretName:  "secret",
			accessToken: "accessToken",
			secretToken: "secretToken",
			hasErr:      true,
		},
	}
	for i := range tests {
		t.Run(tests[i].name, func(t *testing.T) {
			cg := fake.DummyGroup()
			if tests[i].seedSecret != nil {
				_, err := cg.K8sClient.CoreV1().Secrets(cg.Defaults.Namespace).Create(tests[i].seedSecret)
				if err != nil {
					t.Fatal(err)
				}
			}
			var hasErr bool
			accessToken, secretToken, err := getWebhookSecretTokens(cg, tests[i].secretName)
			if err != nil {
				hasErr = true
			}
			if diff := cmp.Diff(tests[i].hasErr, hasErr); diff != "" {
				t.Fatalf("Error mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tests[i].accessToken, accessToken); diff != "" {
				t.Errorf("Access token mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tests[i].secretToken, secretToken); diff != "" {
				t.Errorf("Secret token mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func Test_sanitizeGitURL(t *testing.T) {
	tests := []struct {
		name   string
		gitURL string
		hasErr bool
	}{
		// Correct
		{
			name:   "HTTPS Git Repo With Suffix",
			gitURL: "https://gitpalace.com/org/repo.git",
			hasErr: false,
		},
		{
			name:   "HTTPS Git Repo",
			gitURL: "https://gitpalace.com/org/repo",
			hasErr: false,
		},
		{
			name:   "HTTP Git Repo With Suffix",
			gitURL: "http://gitpalace.com/org/repo.git",
			hasErr: false,
		},
		{
			name:   "HTTP Git Repo",
			gitURL: "http://gitpalace.com/org/repo",
			hasErr: false,
		},
		// Incorrect
		{
			name:   "Invalid Scheme",
			gitURL: "abcd://gitpalace.com/org/repo.git",
			hasErr: true,
		},
		{
			name:   "Not Com TopLevelDomain",
			gitURL: "https://gitpalace.io/org/repo",
			hasErr: true,
		},
		{
			name:   "Empty Org",
			gitURL: "https://gitpalace.com//repo.git",
			hasErr: true,
		},
		{
			name:   "Empty Repo",
			gitURL: "https://gitpalace.com/org/",
			hasErr: true,
		},
		{
			name:   "Empty Hostname",
			gitURL: "https:///org/repo",
			hasErr: true,
		},
		{
			name:   "Empty Second Level Domain",
			gitURL: "https://.com/org/repo",
			hasErr: true,
		},
	}
	for i := range tests {
		t.Run(tests[i].name, func(t *testing.T) {
			var hasErr bool
			_, err := sanitizeGitURL(tests[i].gitURL)
			if err != nil {
				hasErr = true
			}
			if diff := cmp.Diff(tests[i].hasErr, hasErr); diff != "" {
				t.Errorf("Error mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func Test_getDashboardURL(t *testing.T) {
	tests := []struct {
		name         string
		dashboardURL string
		seedService  *corev1.Service
		seedPlatform string
	}{
		{
			name:         "No Dashboard Service",
			dashboardURL: "http://localhost:9097/",
			seedPlatform: "vanilla",
		},
		{
			name:         "Dashboard Service",
			dashboardURL: "http://fake-dashboard:1234/v1/namespaces/default/endpoints",
			seedService: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name: "fake-dashboard",
					Labels: map[string]string{
						"app": "tekton-dashboard",
					},
				},
				Spec: corev1.ServiceSpec{
					Ports: []corev1.ServicePort{
						corev1.ServicePort{
							Name:       "http",
							Protocol:   "TCP",
							Port:       1234,
							NodePort:   5678,
							TargetPort: intstr.FromInt(91011),
						},
					},
				},
			},
			seedPlatform: "vanilla",
		},
		{
			name:         "OpenShift Dashboard Service",
			dashboardURL: "http://fake-openshift-dashboard:1234/v1/namespaces/default/endpoints",
			seedService: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name: "fake-openshift-dashboard",
					Labels: map[string]string{
						"app": "tekton-dashboard-internal",
					},
				},
				Spec: corev1.ServiceSpec{
					Ports: []corev1.ServicePort{
						corev1.ServicePort{
							Name:       "http",
							Protocol:   "TCP",
							Port:       1234,
							NodePort:   5678,
							TargetPort: intstr.FromInt(91011),
						},
					},
				},
			},
			seedPlatform: "openshift",
		},
	}
	for i := range tests {
		t.Run(tests[i].name, func(t *testing.T) {
			cg := fake.DummyGroup()
			cg.Defaults.Platform = tests[i].seedPlatform
			if tests[i].seedService != nil {
				_, err := cg.K8sClient.CoreV1().Services(cg.Defaults.Namespace).Create(tests[i].seedService)
				if err != nil {
					t.Fatal(err)
				}
			}
			dashboardURL := getDashboardURL(cg)
			if diff := cmp.Diff(tests[i].dashboardURL, dashboardURL); diff != "" {
				t.Errorf("Dashboard URL mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
