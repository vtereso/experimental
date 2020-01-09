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

package restapi

import (
	"fmt"
	"net/url"
	"strings"
	"testing"
	"time"

	"gopkg.in/h2non/gock.v1"

	"github.com/google/go-cmp/cmp"
	routesv1 "github.com/openshift/api/route/v1"
	"github.com/tektoncd/experimental/webhooks-extension/pkg/model"
	pipelinesv1alpha1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	triggersv1alpha1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// func Test_createWebhook(t *testing.T) {}

// func Test_deleteWebhook(t *testing.T) {}

// func Test_getAllWebhooks(t *testing.T) {}

func Test_deletePipelineRuns(t *testing.T) {
	tests := []struct {
		name         string
		repoURL      *url.URL
		pipelineName string
		seed         bool
		hasErr       bool
	}{
		{
			name: "Delete PipelineRun",
			repoURL: &url.URL{
				Host: "website.com",
				Path: "/org/repo",
			},
			pipelineName: "pipeline",
			seed:         true,
			hasErr:       false,
		},
		{
			name: "No PipelineRuns",
			repoURL: &url.URL{
				Host: "website.com",
				Path: "/org/repo",
			},
			pipelineName: "pipeline",
			seed:         false,
			hasErr:       false,
		},
	}
	for i := range tests {
		t.Run(tests[i].name, func(t *testing.T) {
			cg := DummyGroup()

			if tests[i].seed {
				plr := &pipelinesv1alpha1.PipelineRun{
					ObjectMeta: metav1.ObjectMeta{
						Name:   tests[i].pipelineName,
						Labels: makePipelineRunSelectorSet(tests[i].repoURL),
					},
				}
				_, err := cg.TektonClient.TektonV1alpha1().PipelineRuns(cg.Defaults.Namespace).Create(plr)
				if err != nil {
					t.Fatalf("Unexpected error:\n%s", err)
				}
			}
			var hasErr bool
			err := cg.deletePipelineRuns(tests[i].repoURL, cg.Defaults.Namespace, tests[i].pipelineName)
			if err != nil {
				hasErr = true
			}
			if diff := cmp.Diff(tests[i].hasErr, hasErr); diff != "" {
				t.Fatalf("Error mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func Test_makePipelineRunSelectorSet(t *testing.T) {
	tests := []struct {
		name     string
		url      *url.URL
		selector map[string]string
	}{
		{
			name: "Repo Selector",
			url: &url.URL{
				Host: "website.com",
				Path: "/org/repo",
			},
			selector: map[string]string{
				pipelineRunServerName: "website.com",
				pipelineRunOrgName:    "org",
				pipelineRunRepoName:   "repo",
			},
		},
	}
	for i := range tests {
		t.Run(tests[i].name, func(t *testing.T) {
			selector := makePipelineRunSelectorSet(tests[i].url)
			if diff := cmp.Diff(tests[i].selector, selector); diff != "" {
				t.Fatalf("Selector mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func Test_createOpenshiftRoute(t *testing.T) {
	cg := DummyGroup()
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
					Name:      "route",
					Namespace: cg.Defaults.Namespace,
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
			var hasErr bool
			if err := cg.createOpenshiftRoute(tests[i].serviceName); err != nil {
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

func Test_createIngress(t *testing.T) {
	tests := []struct {
		name        string
		serviceName string
		seedIngress bool
		hasErr      bool
	}{
		// Correct
		{
			name:        "Unseeded Ingress",
			serviceName: "service1",
			seedIngress: false,
			hasErr:      false,
		},
		// Incorrect
		{
			name:        "Seeded Ingress",
			serviceName: "service2",
			seedIngress: true,
			hasErr:      true,
		},
	}
	for i := range tests {
		t.Run(tests[i].name, func(t *testing.T) {
			cg := DummyGroup()
			if tests[i].seedIngress {
				ingress := &v1beta1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name: tests[i].serviceName,
					},
				}
				_, err := cg.K8sClient.ExtensionsV1beta1().Ingresses(cg.Defaults.Namespace).Create(ingress)
				if err != nil {
					t.Fatal(err)
				}
			}

			var hasErr bool
			if err := cg.createIngress(tests[i].serviceName); err != nil {
				hasErr = true
			}
			if diff := cmp.Diff(tests[i].hasErr, hasErr); diff != "" {
				t.Fatalf("Error mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func Test_deleteIngress(t *testing.T) {
	tests := []struct {
		name        string
		serviceName string
		seedIngress bool
		hasErr      bool
	}{
		// Correct
		{
			name:        "Seeded Ingress",
			serviceName: "service1",
			seedIngress: true,
			hasErr:      false,
		},
		// Incorrect
		{
			name:        "Unseeded Ingress",
			serviceName: "service2",
			seedIngress: false,
			hasErr:      true,
		},
	}
	for i := range tests {
		t.Run(tests[i].name, func(t *testing.T) {
			cg := DummyGroup()
			if tests[i].seedIngress {
				ingress := &v1beta1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name: tests[i].serviceName,
					},
				}
				_, err := cg.K8sClient.ExtensionsV1beta1().Ingresses(cg.Defaults.Namespace).Create(ingress)
				if err != nil {
					t.Fatal(err)
				}
			}

			var hasErr bool
			if err := cg.deleteIngress(tests[i].serviceName); err != nil {
				hasErr = true
			}
			if diff := cmp.Diff(tests[i].hasErr, hasErr); diff != "" {
				t.Fatalf("Error mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func Test_addWebhookTriggers(t *testing.T) {
	cg := DummyGroup()

	tests := []struct {
		name          string
		eventListener *triggersv1alpha1.EventListener
		webhook       model.Webhook
		newTriggers   []triggersv1alpha1.EventListenerTrigger
	}{
		{
			name:          "EventListener No Triggers",
			eventListener: &triggersv1alpha1.EventListener{},
			webhook: model.Webhook{
				Name:             "name",
				Namespace:        "ns",
				ServiceAccount:   "sa",
				AccessTokenRef:   "atr",
				Pipeline:         "pl",
				DockerRegistry:   "dr",
				GitRepositoryURL: "https://vcs.com/org/repo",
			},
			newTriggers: []triggersv1alpha1.EventListenerTrigger{
				triggersv1alpha1.EventListenerTrigger{
					Name: fmt.Sprintf("%s-%s", "name", pushTriggerBindingPostfix),
					Binding: triggersv1alpha1.EventListenerBinding{
						Name:       fmt.Sprintf("%s-%s", "pl", pushTriggerBindingPostfix),
						APIVersion: "v1alpha1",
					},
					Params: []pipelinesv1alpha1.Param{
						{Name: wextTargetNamespace, Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: "ns"}},
						{Name: wextServiceAccount, Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: "sa"}},
						{Name: wextDockerRegistry, Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: "dr"}},
						{Name: wextGitServer, Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: "vcs.com"}},
						{Name: wextGitOrg, Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: "org"}},
						{Name: wextGitRepo, Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: "repo"}},
					},
					Template: triggersv1alpha1.EventListenerTemplate{
						Name:       fmt.Sprintf("%s-%s", "pl", triggerTemplatePostfix),
						APIVersion: "v1alpha1",
					},
					Interceptor: &triggersv1alpha1.EventInterceptor{
						Header: []pipelinesv1alpha1.Param{
							{Name: WextInterceptorTriggerName, Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: fmt.Sprintf("%s-%s", "name", pushTriggerBindingPostfix)}},
							{Name: WextInterceptorRepoURL, Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: "https://vcs.com/org/repo"}},
							{Name: WextInterceptorEvent, Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: "push"}},
							{Name: WextInterceptorSecretName, Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: "atr"}}},
						ObjectRef: &corev1.ObjectReference{
							APIVersion: "v1",
							Kind:       "Service",
							Name:       wextValidator,
							Namespace:  cg.Defaults.Namespace,
						},
					},
				},
				triggersv1alpha1.EventListenerTrigger{
					Name: fmt.Sprintf("%s-%s", "name", pullTriggerBindingPostfix),
					Binding: triggersv1alpha1.EventListenerBinding{
						Name:       fmt.Sprintf("%s-%s", "pl", pullTriggerBindingPostfix),
						APIVersion: "v1alpha1",
					},
					Params: []pipelinesv1alpha1.Param{
						{Name: wextTargetNamespace, Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: "ns"}},
						{Name: wextServiceAccount, Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: "sa"}},
						{Name: wextDockerRegistry, Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: "dr"}},
						{Name: wextGitServer, Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: "vcs.com"}},
						{Name: wextGitOrg, Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: "org"}},
						{Name: wextGitRepo, Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: "repo"}},
					},
					Template: triggersv1alpha1.EventListenerTemplate{
						Name:       fmt.Sprintf("%s-%s", "pl", triggerTemplatePostfix),
						APIVersion: "v1alpha1",
					},
					Interceptor: &triggersv1alpha1.EventInterceptor{
						Header: []pipelinesv1alpha1.Param{
							{Name: WextInterceptorTriggerName, Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: fmt.Sprintf("%s-%s", "name", pullTriggerBindingPostfix)}},
							{Name: WextInterceptorRepoURL, Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: "https://vcs.com/org/repo"}},
							{Name: WextInterceptorEvent, Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: "pull_request"}},
							{Name: WextInterceptorSecretName, Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: "atr"}}},
						ObjectRef: &corev1.ObjectReference{
							APIVersion: "v1",
							Kind:       "Service",
							Name:       wextValidator,
							Namespace:  cg.Defaults.Namespace,
						},
					},
				},
				triggersv1alpha1.EventListenerTrigger{
					Name: fmt.Sprintf("%s-%s", "name", monitorTaskName),
					Binding: triggersv1alpha1.EventListenerBinding{
						Name:       fmt.Sprintf("%s-%s", "pl", monitorTriggerBindingPostfix),
						APIVersion: "v1alpha1",
					},
					Params: []pipelinesv1alpha1.Param{
						{Name: wextMonitorSecretName, Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: "atr"}},
						{Name: wextMonitorSecretKey, Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: AccessToken}},
						{Name: wextMonitorDashboardURL, Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: cg.getDashboardURL()}},
					},
					Template: triggersv1alpha1.EventListenerTemplate{
						Name:       fmt.Sprintf("%s-%s", "pl", triggerTemplatePostfix),
						APIVersion: "v1alpha1",
					},
					Interceptor: &triggersv1alpha1.EventInterceptor{
						Header: []pipelinesv1alpha1.Param{
							{Name: WextInterceptorTriggerName, Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: fmt.Sprintf("%s-%s", "name", monitorTaskName)}},
							{Name: WextInterceptorRepoURL, Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: "https://vcs.com/org/repo"}},
							{Name: WextInterceptorEvent, Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: "pull_request"}},
							{Name: WextInterceptorSecretName, Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: "atr"}}},
						ObjectRef: &corev1.ObjectReference{
							APIVersion: "v1",
							Kind:       "Service",
							Name:       wextValidator,
							Namespace:  cg.Defaults.Namespace,
						},
					},
				},
			},
		},
	}
	for i := range tests {
		t.Run(tests[i].name, func(t *testing.T) {
			cg := DummyGroup()
			// expected triggers
			triggers := append(tests[i].eventListener.Spec.DeepCopy().Triggers, tests[i].newTriggers...)
			addWebhookTriggers(cg, tests[i].eventListener, tests[i].webhook)
			if diff := cmp.Diff(triggers, tests[i].eventListener.Spec.Triggers); diff != "" {
				t.Fatalf("EventListenerTriggers mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func Test_removeWebhookTriggers(t *testing.T) {
	tests := []struct {
		name          string
		eventListener *triggersv1alpha1.EventListener
		webhookName   string
	}{
		{
			name:          "No Triggers",
			eventListener: &triggersv1alpha1.EventListener{},
			webhookName:   "webhook",
		},
		{
			name: "Remove Triggers",
			eventListener: &triggersv1alpha1.EventListener{
				Spec: triggersv1alpha1.EventListenerSpec{
					Triggers: []triggersv1alpha1.EventListenerTrigger{
						triggersv1alpha1.EventListenerTrigger{
							Name: "red-trigger",
						},
						triggersv1alpha1.EventListenerTrigger{
							Name: "webhook-trigger",
						},
					},
				},
			},
			webhookName: "webhook",
		},
	}
	for i := range tests {
		t.Run(tests[i].name, func(t *testing.T) {
			removeWebhookTriggers(tests[i].eventListener, tests[i].webhookName)
			for _, trigger := range tests[i].eventListener.Spec.Triggers {
				if strings.HasPrefix(trigger.Name, tests[i].webhookName) {
					t.Errorf("Trigger %s should have been deleted", trigger.Name)
				}
			}
		})
	}
}

func Test_newTrigger(t *testing.T) {
	tests := []struct {
		name                 string
		triggerName          string
		bindingName          string
		templateName         string
		interceptorNamespace string
		repoURL              string
		eventType            string
		secretName           string
		params               []pipelinesv1alpha1.Param
		trigger              triggersv1alpha1.EventListenerTrigger
	}{
		{
			name:                 "New Trigger Params",
			triggerName:          "trigger",
			bindingName:          "binding",
			templateName:         "template",
			interceptorNamespace: "interceptor-namespace",
			repoURL:              "repoURL",
			eventType:            "event",
			secretName:           "secretName",
			params: []pipelinesv1alpha1.Param{
				pipelinesv1alpha1.Param{
					Name: "param",
					Value: pipelinesv1alpha1.ArrayOrString{
						Type:      pipelinesv1alpha1.ParamTypeString,
						StringVal: "value",
					},
				},
			},
			trigger: triggersv1alpha1.EventListenerTrigger{
				Name: "trigger",
				Binding: triggersv1alpha1.EventListenerBinding{
					Name:       "binding",
					APIVersion: "v1alpha1",
				},
				Params: []pipelinesv1alpha1.Param{
					pipelinesv1alpha1.Param{
						Name: "param",
						Value: pipelinesv1alpha1.ArrayOrString{
							Type:      pipelinesv1alpha1.ParamTypeString,
							StringVal: "value",
						},
					},
				},
				Template: triggersv1alpha1.EventListenerTemplate{
					Name:       "template",
					APIVersion: "v1alpha1",
				},
				Interceptor: &triggersv1alpha1.EventInterceptor{
					Header: []pipelinesv1alpha1.Param{
						{Name: WextInterceptorTriggerName, Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: "trigger"}},
						{Name: WextInterceptorRepoURL, Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: "repoURL"}},
						{Name: WextInterceptorEvent, Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: "event"}},
						{Name: WextInterceptorSecretName, Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: "secretName"}}},
					ObjectRef: &corev1.ObjectReference{
						APIVersion: "v1",
						Kind:       "Service",
						Name:       wextValidator,
						Namespace:  "interceptor-namespace",
					},
				},
			},
		},
		{
			name:                 "New Trigger No Params",
			triggerName:          "trigger",
			bindingName:          "binding",
			templateName:         "template",
			interceptorNamespace: "interceptor-namespace",
			repoURL:              "repoURL",
			eventType:            "event",
			secretName:           "secretName",
			params:               nil,
			trigger: triggersv1alpha1.EventListenerTrigger{
				Name: "trigger",
				Binding: triggersv1alpha1.EventListenerBinding{
					Name:       "binding",
					APIVersion: "v1alpha1",
				},
				Params: nil,
				Template: triggersv1alpha1.EventListenerTemplate{
					Name:       "template",
					APIVersion: "v1alpha1",
				},
				Interceptor: &triggersv1alpha1.EventInterceptor{
					Header: []pipelinesv1alpha1.Param{
						{Name: WextInterceptorTriggerName, Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: "trigger"}},
						{Name: WextInterceptorRepoURL, Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: "repoURL"}},
						{Name: WextInterceptorEvent, Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: "event"}},
						{Name: WextInterceptorSecretName, Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: "secretName"}}},
					ObjectRef: &corev1.ObjectReference{
						APIVersion: "v1",
						Kind:       "Service",
						Name:       wextValidator,
						Namespace:  "interceptor-namespace",
					},
				},
			},
		},
	}
	for i := range tests {
		t.Run(tests[i].name, func(t *testing.T) {
			trigger := newTrigger(
				tests[i].triggerName,
				tests[i].bindingName,
				tests[i].templateName,
				tests[i].interceptorNamespace,
				tests[i].repoURL,
				tests[i].eventType,
				tests[i].secretName,
				tests[i].params,
			)
			if diff := cmp.Diff(tests[i].trigger, trigger); diff != "" {
				t.Errorf("Trigger mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func Test_getMonitorTriggerParams(t *testing.T) {
	tests := []struct {
		name    string
		webhook model.Webhook
		params  []pipelinesv1alpha1.Param
	}{
		{
			name: "Valid Webhook",
			webhook: model.Webhook{
				AccessTokenRef: "secretName",
			},
			params: []pipelinesv1alpha1.Param{
				{Name: wextMonitorSecretName, Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: "secretName"}},
				{Name: wextMonitorSecretKey, Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: AccessToken}},
				{Name: wextMonitorDashboardURL, Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: DummyGroup().getDashboardURL()}},
			},
		},
	}
	for i := range tests {
		t.Run(tests[i].name, func(t *testing.T) {
			params := getMonitorTriggerParams(DummyGroup(), tests[i].webhook)
			if diff := cmp.Diff(tests[i].params, params); diff != "" {
				t.Errorf("Params mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func Test_getPipelineTriggerParams(t *testing.T) {
	tests := []struct {
		name    string
		webhook model.Webhook
		params  []pipelinesv1alpha1.Param
	}{
		{
			name: "Valid Webhook",
			webhook: model.Webhook{
				Namespace:        "namespace",
				ServiceAccount:   "serviceAccount",
				DockerRegistry:   "dockerRegistry",
				GitRepositoryURL: "https://github.com/org/repo",
			},
			params: []pipelinesv1alpha1.Param{
				{Name: wextTargetNamespace, Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: "namespace"}},
				{Name: wextServiceAccount, Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: "serviceAccount"}},
				{Name: wextDockerRegistry, Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: "dockerRegistry"}},
				{Name: wextGitServer, Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: "github.com"}},
				{Name: wextGitOrg, Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: "org"}},
				{Name: wextGitRepo, Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: "repo"}},
			},
		},
	}
	for i := range tests {
		t.Run(tests[i].name, func(t *testing.T) {
			params := getPipelineTriggerParams(tests[i].webhook)
			if diff := cmp.Diff(tests[i].params, params); diff != "" {
				t.Errorf("Params mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func Test_triggerToWebhook(t *testing.T) {
	tests := []struct {
		name    string
		trigger triggersv1alpha1.EventListenerTrigger
		webhook *model.Webhook
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
			webhook: &model.Webhook{
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
				t.Fatalf("Error mismatch (-want +got):\n%s", diff)
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
		webhooks []model.Webhook
		repoURL  string
		size     int
	}{
		{
			name: "No matches",
			webhooks: []model.Webhook{
				model.Webhook{GitRepositoryURL: "repo1"},
				model.Webhook{GitRepositoryURL: "repo2"},
				model.Webhook{GitRepositoryURL: "repo3"},
			},
			repoURL: "repo0",
			size:    0,
		},
		{
			name: "One matches",
			webhooks: []model.Webhook{
				model.Webhook{GitRepositoryURL: "repo1"},
				model.Webhook{GitRepositoryURL: "repo2"},
				model.Webhook{GitRepositoryURL: "repo3"},
			},
			repoURL: "repo1",
			size:    1,
		},
		{
			name: "All match",
			webhooks: []model.Webhook{
				model.Webhook{GitRepositoryURL: "repo1"},
				model.Webhook{GitRepositoryURL: "repo1"},
				model.Webhook{GitRepositoryURL: "repo1"},
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
		webhooks    []model.Webhook
		webhookName string
		hasErr      bool
	}{
		{
			name: "Existing webhook",
			webhooks: []model.Webhook{
				model.Webhook{Name: "webhook1"},
				model.Webhook{Name: "webhook2"},
			},
			webhookName: "webhook1",
			hasErr:      false,
		},
		{
			name: "Nonexisting webhook",
			webhooks: []model.Webhook{
				model.Webhook{Name: "webhook1"},
				model.Webhook{Name: "webhook2"},
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
			cg := DummyGroup()
			if tests[i].seedEventListener != nil {
				_, err := cg.TriggersClient.TektonV1alpha1().EventListeners(cg.Defaults.Namespace).Create(tests[i].seedEventListener)
				if err != nil {
					t.Fatal(err)
				}
			}
			var hasErr bool
			if _, err := cg.getWebhookEventListener(); err != nil {
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
			cg := DummyGroup()
			var hasErr bool
			if err := cg.createEventListener(tests[i].eventListener); err != nil {
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
			cg := DummyGroup()
			_, err := cg.TriggersClient.TektonV1alpha1().EventListeners(cg.Defaults.Namespace).Update(tests[i].eventListener)
			if err != nil {
				t.Fatal(err)
			}
			var hasErr bool
			if err := cg.updateEventListener(tests[i].eventListener); err != nil {
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
			cg := DummyGroup()
			if tests[i].seedEventListener != nil {
				_, err := cg.TriggersClient.TektonV1alpha1().EventListeners(cg.Defaults.Namespace).Create(tests[i].seedEventListener)
				if err != nil {
					t.Fatal(err)
				}
			}
			var hasErr bool
			if err := cg.deleteEventListener(); err != nil {
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
		webhooks []model.Webhook
	}{
		{
			name: "One Webhook",
			webhooks: []model.Webhook{
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
			webhooks: []model.Webhook{
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
			cg := DummyGroup()
			el := getBaseEventListener(cg.Defaults.Namespace)
			for _, webhook := range tests[i].webhooks {
				addWebhookTriggers(cg, el, webhook)
			}
			webhooks := getWebhooksFromEventListener(el)
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
			cg := DummyGroup()
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
			_, err := cg.waitForEventListenerStatus()
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
			accessToken: "",
			secretToken: "",
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
			accessToken: "",
			secretToken: "",
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
			accessToken: "",
			secretToken: "",
			hasErr:      true,
		},
	}
	for i := range tests {
		t.Run(tests[i].name, func(t *testing.T) {
			cg := DummyGroup()
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
		mock         bool
	}{
		{
			name:         "No Dashboard Service",
			dashboardURL: "http://localhost:9097/",
			seedPlatform: "vanilla",
		},
		{
			name:         "Mocked Dashboard Service",
			dashboardURL: "https://tekton-dashboard.nip.io",
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
			mock:         true,
		},
		{
			name:         "Unmocked Dashboard Service",
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
			name:         "Mocked OpenShift Dashboard Service",
			dashboardURL: "https://tekton-dashboard.nip.io",
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
			mock:         true,
		},
		{
			name:         "Unmocked OpenShift Dashboard Service",
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
			cg := DummyGroup()
			cg.Defaults.Platform = tests[i].seedPlatform
			if tests[i].seedService != nil {
				_, err := cg.K8sClient.CoreV1().Services(cg.Defaults.Namespace).Create(tests[i].seedService)
				if err != nil {
					t.Fatal(err)
				}
				if tests[i].mock {
					name := tests[i].seedService.Name
					scheme := tests[i].seedService.Spec.Ports[0].Name
					port := tests[i].seedService.Spec.Ports[0].Port
					dashboardURL := fmt.Sprintf("%s://%s:%d/v1/namespaces/%s/endpoints", scheme, name, port, cg.Defaults.Namespace)

					defer gock.Disable()
					gock.New(dashboardURL).
						Get("/").
						Reply(200).
						JSON(fmt.Sprintf(`[{"url": "%s"}]`, tests[i].dashboardURL))
				}
			}
			dashboardURL := cg.getDashboardURL()
			if diff := cmp.Diff(tests[i].dashboardURL, dashboardURL); diff != "" {
				t.Errorf("Dashboard URL mismatch (-want +got):\n%s", diff)
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
			cg := DummyGroup()
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
			if err := cg.deleteOpenshiftRoute(tests[i].routeName); err != nil {
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
