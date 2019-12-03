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
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/xerrors"

	restful "github.com/emicklei/go-restful"
	routesv1 "github.com/openshift/api/route/v1"
	"github.com/tektoncd/experimental/webhooks-extension/pkg/client"
	logging "github.com/tektoncd/experimental/webhooks-extension/pkg/logging"
	"github.com/tektoncd/experimental/webhooks-extension/pkg/models"
	"github.com/tektoncd/experimental/webhooks-extension/pkg/utils"
	githook "github.com/tektoncd/experimental/webhooks-extension/pkg/webhook"
	pipelinesv1alpha1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	triggersv1alpha1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var (
	// eventListenerLock is the lock that must be acquired within functions that
	// modify the EventListener
	eventListenerLock sync.Mutex
	// pullRequestActions is a pipeline parameter with the set of actions to run
	// against for pull requests
	pullRequestActions = pipelinesv1alpha1.Param{
		Name: "Wext-Incoming-Actions",
		Value: pipelinesv1alpha1.ArrayOrString{
			Type:      pipelinesv1alpha1.ParamTypeString,
			StringVal: "opened,reopened,synchronize",
		},
	}
)

const (
	// monitorTaskName is the name of the monitor task that will be used in an
	// EventListenerTrigger to monitor PipelineRuns created by the same event
	monitorTaskName = "monitor-task"
	// eventListenrName is the name of the EventListener that is the singleton
	// source of truth for Triggers/events
	eventListenerName = "tekton-webhooks-eventlistener"
	// eventListenerSA is the name of the serviceAccount that should the
	// EventListener (eventListenerName) should be configured with
	eventListenerSA = "tekton-webhooks-extension-eventlistener"

	// triggerTemplatePostfix is the additional name postfix for a
	// TriggerTemplate. The webhook pipeline name is used for the base
	triggerTemplatePostfix = "template"
	// pushTriggerBindingPostfix is the additional name postfix for a
	// TriggerBinding. The webhook pipeline name is used for the base
	pushTriggerBindingPostfix = "push-binding"
	// pullTriggerBindingPostfix is the additional name postfix for a
	// TriggerBinding. The webhook pipeline name is used for the base
	pullTriggerBindingPostfix = "pullrequest-binding"
	// monitorTriggerBindingPostfix is the additional name postfix for a
	// TriggerBinding. The webhook pipeline name is used for the base
	monitorTriggerBindingPostfix = "binding"

	// wextMonitorSecretName is the name of the EventListenerTrigger parameter
	// for the secret being used by the monitor within a TriggerTemplate
	wextMonitorSecretName = "gitsecretname"
	// wextMonitorSecretKey is the name of the EventListenerTrigger parameter
	// for the secret being used by the monitor within a TriggerTemplate
	wextMonitorSecretKey = "gitsecretkeyname"
	// wextMonitorDashboardURL is the name of the EventListenerTrigger parameter
	// for the dashboard url used by the monitor within a TriggerTemplate
	wextMonitorDashboardURL = "dashboardurl"

	// wextTargetNamespace is the name of the EventListenerTrigger parameter for
	// the namespace used within a TriggerTemplate
	wextTargetNamespace = "Wext-Target-Namespace"
	// wextServiceAccount is the name of the EventListenerTrigger parameter for
	// the service account used within a TriggerTemplate
	wextServiceAccount = "Wext-Service-Account"
	// wextDockerRegistry is the name of the EventListenerTrigger parameter for
	// the docker registry used within a TriggerTemplate
	wextDockerRegistry = "Wext-Docker-Registry"
	// wextGitServer is the name of the EventListenerTrigger parameter for
	// the git server used within a TriggerTemplate
	wextGitServer = "Wext-Git-server"
	// wextGitOrg is the name of the EventListenerTrigger parameter for
	// the git organization used within a TriggerTemplate
	wextGitOrg = "Wext-Git-Org"
	// wextGitRepo is the name of the EventListenerTrigger parameter for
	// the git repo used within a TriggerTemplate
	wextGitRepo = "Wext-Git-Repo"

	// WextInterceptorTriggerName is the name of the EventListenerTrigger
	// Interceptor parameter used by the Webhook extension interceptor
	WextInterceptorTriggerName = "Wext-Trigger-Name"
	// WextInterceptorRepoURL is the name of the EventListenerTrigger
	// Interceptor parameter used by the Webhook extension interceptor
	WextInterceptorRepoURL = "Wext-Repository-Url"
	// WextInterceptorEvent is the name of the EventListenerTrigger Interceptor
	// parameter used by the Webhook extension interceptor
	WextInterceptorEvent = "Wext-Incoming-Event"
	// WextInterceptorSecretName is the name of the EventListenerTrigger
	// Interceptor parameter used by the Webhook extension interceptor
	WextInterceptorSecretName = "Wext-Secret-Name"
	// wextValidator is the name of the Webhook extension interceptor
	wextValidator = "tekton-webhooks-extension-validator"

	// pipelineRunServerName is the label key applied to PipelineRuns for
	// the git server
	pipelineRunServerName = "webhooks.tekton.dev/gitServer"
	// pipelineRunOrgName is the label key applied to PipelineRuns for
	// the git server
	pipelineRunOrgName = "webhooks.tekton.dev/gitOrg"
	// pipelineRunRepoName is the label key applied to PipelineRuns for
	// the git server
	pipelineRunRepoName = "webhooks.tekton.dev/gitRepo"
)

// CreateWebhook creates a webhook for a given repository and creates/updates
// the EventListener
func CreateWebhook(request *restful.Request, response *restful.Response, cg *client.Group) {
	logging.Log.Debug("CreateWebhook()")
	eventListenerLock.Lock()
	defer eventListenerLock.Unlock()

	logging.Log.Infof("Webhook creation request received with request: %+v.", request)
	// Read and validate webhook payload
	webhook := models.Webhook{}
	if err := request.ReadEntity(&webhook); err != nil {
		err = xerrors.Errorf("Error trying to read request entity as webhook %s", err)
		utils.RespondError(response, err, http.StatusBadRequest)
		return
	}
	if err := webhook.Validate(); err != nil {
		utils.RespondError(response, err, http.StatusBadRequest)
		return
	}
	// Validate Git URL
	gitURL, err := sanitizeGitURL(webhook.GitRepositoryURL)
	if err != nil {
		err = xerrors.Errorf("Invalid value webhook URL: %s", err)
		utils.RespondError(response, err, http.StatusBadRequest)
		return
	}

	// Check for Triggers resources
	_, templateErr := cg.TriggersClient.TektonV1alpha1().TriggerTemplates(cg.Defaults.Namespace).Get(fmt.Sprintf("%s-%s", webhook.Pipeline, triggerTemplatePostfix), metav1.GetOptions{})
	_, pushErr := cg.TriggersClient.TektonV1alpha1().TriggerBindings(cg.Defaults.Namespace).Get(fmt.Sprintf("%s-%s", webhook.Pipeline, pushTriggerBindingPostfix), metav1.GetOptions{})
	_, pullrequestErr := cg.TriggersClient.TektonV1alpha1().TriggerBindings(cg.Defaults.Namespace).Get(fmt.Sprintf("%s-%s", webhook.Pipeline, pullTriggerBindingPostfix), metav1.GetOptions{})
	if templateErr != nil || pushErr != nil || pullrequestErr != nil {
		err := xerrors.Errorf("Expected Trigger resources for '%s' pipeline not found", webhook.Pipeline)
		utils.RespondError(response, err, http.StatusBadRequest)
		return
	}

	// Get or initialize EventListener
	el, err := getWebhookEventListener(cg)
	// Errors other than IsNotFound
	if err != nil && !k8serrors.IsNotFound(err) {
		utils.RespondError(response, err, http.StatusInternalServerError)
		return
	}
	eventListenerExists := (err == nil)
	existingRepoWebhook := false
	if eventListenerExists {
		existingHooks := getWebhooksFromEventListener(*el)
		// Check if webhook exists already
		for _, existingHook := range existingHooks {
			if webhook.Name == existingHook.Name {
				err := xerrors.Errorf("Webhook already exists with name %s", webhook.Name)
				utils.RespondError(response, err, http.StatusBadRequest)
				return
			}
			if webhook.GitRepositoryURL == existingHook.GitRepositoryURL {
				existingRepoWebhook = true
				if webhook.Pipeline == existingHook.Pipeline {
					err := xerrors.Errorf("Webhook on URL %s already exists with Pipeline %s", webhook.GitRepositoryURL, webhook.Pipeline)
					utils.RespondError(response, err, http.StatusBadRequest)
					return
				}
			}
		}
	} else {
		el = getBaseEventListener(cg.Defaults.Namespace)
	}

	// Attempt to create webhook if not found
	if !existingRepoWebhook {
		accessToken, secretToken, err := getWebhookSecretTokens(cg, webhook.AccessTokenRef)
		if err != nil {
			utils.RespondError(response, err, http.StatusInternalServerError)
			return
		}
		err = githook.DoGitHubWebhookRequest(gitURL, cg.Defaults.CallbackURL, accessToken, secretToken, githook.Subscribe, []string{"push", "pull_request"})
		if err != nil {
			utils.RespondError(response, err, http.StatusInternalServerError)
			return
		}
		logging.Log.Debug("Webhook creation succeeded")
	}

	// Add new EventListenerTriggers for webhook request
	addWebhookTriggers(cg, el, webhook)

	// Update or create EventListener
	if eventListenerExists {
		if err := updateEventListener(cg, el); err != nil {
			utils.RespondError(response, err, http.StatusInternalServerError)
			return
		}
	} else {
		if err := createEventListener(cg, el); err != nil {
			utils.RespondError(response, err, http.StatusInternalServerError)
			return
		}
		// Await EventListenerStatus to be populated
		el, err = waitForEventListenerStatus(cg)
		if err != nil {
			utils.RespondError(response, err, http.StatusInternalServerError)
			return
		}
		// Create Route or Ingress
		if strings.Contains(strings.ToLower(cg.Defaults.Platform), "openshift") {
			if err := createOpenshiftRoute(cg, el.Status.Configuration.GeneratedResourceName); err != nil {
				logging.Log.Debug("Failed to create Route, deleting EventListener...")
				if err = deleteEventListener(cg); err != nil {
					logging.Log.Debug("Failed to delete EventListener")
				}
				utils.RespondError(response, xerrors.New("Failed to create Route for webhook"), http.StatusInternalServerError)
				return
			}
		} else {
			if err := createIngress(cg, el.Status.Configuration.GeneratedResourceName); err != nil {
				logging.Log.Debug("Failed to create Ingress, deleting EventListener...")
				if err = deleteEventListener(cg); err != nil {
					logging.Log.Debug("Failed to delete EventListener")
				}
				utils.RespondError(response, xerrors.New("Failed to create Ingress for webhook"), http.StatusInternalServerError)
				return
			}
		}
	}
	response.WriteHeader(http.StatusCreated)
}

// DeleteWebhook attempts to remove a webhook and the corresponding triggers on
// the EventListener
func DeleteWebhook(request *restful.Request, response *restful.Response, cg *client.Group) {
	logging.Log.Debug("DeleteWebhook()")
	eventListenerLock.Lock()
	defer eventListenerLock.Unlock()

	// Necessary path parameter
	name := request.PathParameter("name")
	if err := models.ValidateWebhookName(name); err != nil {
		utils.RespondError(response, err, http.StatusBadRequest)
		return
	}
	// Expected query parameters
	repo := request.QueryParameter("repository")
	if repo == "" {
		err := xerrors.New("Repository query parameter must be provided and non-empty")
		utils.RespondError(response, err, http.StatusBadRequest)
		return
	}
	// Validate Git URL
	gitURL, err := sanitizeGitURL(repo)
	if err != nil {
		err = xerrors.Errorf("Invalid value webhook URL: %s", err)
		utils.RespondError(response, err, http.StatusBadRequest)
		return
	}
	// Optional query parameter
	var deleteWebhookPipelineRuns bool
	deletePipelineRunsQueryParam := request.QueryParameter("deletepipelineruns")
	if deletePipelineRunsQueryParam != "" {
		deleteWebhookPipelineRuns, err = strconv.ParseBool(deletePipelineRunsQueryParam)
		if err != nil {
			err := xerrors.New("Bad request: 'deletepipelineruns' query params should be set to 'true', 'false', or not be provided")
			utils.RespondError(response, err, http.StatusInternalServerError)
			return
		}
	}
	logging.Log.Debugf("DeleteWebhook() name: %s, repo: %s, deleteRuns: %b", name, repo, deleteWebhookPipelineRuns)

	// Get webhooks
	el, err := getWebhookEventListener(cg)
	if err != nil {
		utils.RespondError(response, err, http.StatusInternalServerError)
		return
	}
	webhooks := getWebhooksFromEventListener(*el)
	// List of webhooks on repository
	webhooks = filterWebhooksByRepo(webhooks, repo)
	deleteWebhook, err := findWebhookByName(webhooks, name)
	if err != nil {
		utils.RespondError(response, err, http.StatusInternalServerError)
		return
	}

	switch len(webhooks) {
	case 0:
		err = xerrors.Errorf("No webhooks found for repo: %s", repo)
		utils.RespondError(response, err, http.StatusBadRequest)
		return
	case 1:
		accessToken, secretToken, err := getWebhookSecretTokens(cg, deleteWebhook.AccessTokenRef)
		if err != nil {
			utils.RespondError(response, err, http.StatusInternalServerError)
			return
		}
		// Attempt to remove webhook
		err = githook.DoGitHubWebhookRequest(gitURL, cg.Defaults.CallbackURL, accessToken, secretToken, githook.Unsubscribe, []string{"push", "pull_request"})
		if err != nil {
			utils.RespondError(response, err, http.StatusInternalServerError)
			return
		}
	}

	// Update the EventListenerTriggers
	removeWebhookTriggers(cg, el, name)
	switch len(el.Spec.Triggers) {
	// The EventListener cannot have no Triggers or it will fail validation
	case 0:
		if err := deleteEventListener(cg); err != nil {
			utils.RespondError(response, err, http.StatusInternalServerError)
			return
		}
	default:
		if err := updateEventListener(cg, el); err != nil {
			utils.RespondError(response, err, http.StatusInternalServerError)
			return
		}
	}

	// Remove PipelineRuns
	if deleteWebhookPipelineRuns {
		if err := deletePipelineRuns(cg, gitURL, deleteWebhook.Namespace, deleteWebhook.Pipeline); err != nil {
			utils.RespondError(response, err, http.StatusInternalServerError)
			return
		}
	}
}

// GetAllWebhooks returns all of the webhooks triggers on the EventListener
func GetAllWebhooks(request *restful.Request, response *restful.Response, cg *client.Group) {
	logging.Log.Debugf("GetAllWebhooks()")
	el, err := getWebhookEventListener(cg)
	if err != nil {
		utils.RespondError(response, err, http.StatusInternalServerError)
		return
	}
	webhooks := getWebhooksFromEventListener(*el)
	response.WriteEntity(webhooks)
}

// deletePipelineRuns deletes PipelineRuns witin the specified namespace that
// have a matching PipelineRef and GitURL
func deletePipelineRuns(cg *client.Group, repoURL *url.URL, namespace, pipeline string) error {
	logging.Log.Debugf("deletePipelineRuns() repo: %s, namespace: %s, pipeline: %b", repoURL.String(), namespace, pipeline)
	labelSelector := fields.SelectorFromSet(makePipelineRunSelectorSet(repoURL)).String()
	pipelineRunList, err := cg.TektonClient.TektonV1alpha1().PipelineRuns(namespace).List(metav1.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		return err
	}
	for _, pipelineRun := range pipelineRunList.Items {
		err := cg.TektonClient.TektonV1alpha1().PipelineRuns(namespace).Delete(pipelineRun.Name, &metav1.DeleteOptions{})
		if err != nil {
			return err
		}
	}
	return nil
}

// makePipelineRunSelectorSet creates a label selector set
func makePipelineRunSelectorSet(repoURL *url.URL) map[string]string {
	server, org, repo := getGitValues(*repoURL)
	return map[string]string{
		pipelineRunServerName: server,
		pipelineRunOrgName:    org,
		pipelineRunRepoName:   repo,
	}
}

// createOpenshiftRoute attempts to create an Openshift Route on the service.
// The Route has the same name as the service
func createOpenshiftRoute(cg *client.Group, serviceName string) error {
	route := &routesv1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name: serviceName,
		},
		Spec: routesv1.RouteSpec{
			To: routesv1.RouteTargetReference{
				Kind: "Service",
				Name: serviceName,
			},
		},
	}
	_, err := cg.RoutesClient.RouteV1().Routes(cg.Defaults.Namespace).Create(route)
	return err
}

// deleteOpenshiftRoute attempts to delete an Openshift Route
func deleteOpenshiftRoute(cg *client.Group, routeName string) error {
	return cg.RoutesClient.RouteV1().Routes(cg.Defaults.Namespace).Delete(routeName, &metav1.DeleteOptions{})
}

// createIngress attempts to creates an ingress for the service. The Ingress has
// the same name as the service
func createIngress(cg *client.Group, serviceName string) error {
	// Unlike webhook creation, the ingress does not need a protocol specified
	callback := strings.TrimPrefix(cg.Defaults.CallbackURL, "http://")
	callback = strings.TrimPrefix(callback, "https://")

	ingress := &v1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: cg.Defaults.Namespace,
		},
		Spec: v1beta1.IngressSpec{
			Rules: []v1beta1.IngressRule{
				{
					Host: callback,
					IngressRuleValue: v1beta1.IngressRuleValue{
						HTTP: &v1beta1.HTTPIngressRuleValue{
							Paths: []v1beta1.HTTPIngressPath{
								{
									Backend: v1beta1.IngressBackend{
										ServiceName: serviceName,
										ServicePort: intstr.IntOrString{
											Type:   intstr.Int,
											IntVal: 8080,
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	_, err := cg.K8sClient.ExtensionsV1beta1().Ingresses(cg.Defaults.Namespace).Create(ingress)
	return err
}

// deleteIngress attempts to deletes the ingress
func deleteIngress(cg *client.Group, ingressName string) error {
	return cg.K8sClient.ExtensionsV1beta1().Ingresses(cg.Defaults.Namespace).Delete(ingressName, &metav1.DeleteOptions{})
}

// addWebhookTriggers updates the EventListener with additional triggers
// generated by the webhook. The webhook git URL is assumed to be a valid
// url. The created EventListenerTriggers have names in the
// form: `<webhookName>-<postfix>`. This change is only made in memory and needs
// to be persisted
func addWebhookTriggers(cg *client.Group, eventListener *triggersv1alpha1.EventListener, webhook models.Webhook) {
	pipelineTriggerParams := getPipelineTriggerParams(webhook)
	monitorTriggerParams := getMonitorTriggerParams(cg, webhook)

	newPushTrigger := newTrigger(fmt.Sprintf("%s-%s", webhook.Name, pushTriggerBindingPostfix),
		fmt.Sprintf("%s-%s", webhook.Pipeline, pushTriggerBindingPostfix),
		fmt.Sprintf("%s-%s", webhook.Pipeline, triggerTemplatePostfix),
		cg.Defaults.Namespace,
		webhook.GitRepositoryURL,
		"push",
		webhook.AccessTokenRef,
		pipelineTriggerParams)

	newPullRequestTrigger := newTrigger(fmt.Sprintf("%s-%s", webhook.Name, pullTriggerBindingPostfix),
		fmt.Sprintf("%s-%s", webhook.Pipeline, pullTriggerBindingPostfix),
		fmt.Sprintf("%s-%s", webhook.Pipeline, triggerTemplatePostfix),
		cg.Defaults.Namespace,
		webhook.GitRepositoryURL,
		"pull_request",
		webhook.AccessTokenRef,
		pipelineTriggerParams)

	monitorTrigger := newTrigger(fmt.Sprintf("%s-%s", webhook.Name, monitorTaskName),
		fmt.Sprintf("%s-%s", webhook.Pipeline, monitorTriggerBindingPostfix),
		fmt.Sprintf("%s-%s", webhook.Pipeline, triggerTemplatePostfix),
		cg.Defaults.Namespace,
		webhook.GitRepositoryURL,
		"pull_request",
		webhook.AccessTokenRef,
		monitorTriggerParams)

	newTriggers := []triggersv1alpha1.EventListenerTrigger{
		newPushTrigger,
		newPullRequestTrigger,
		monitorTrigger,
	}
	eventListener.Spec.Triggers = append(eventListener.Spec.Triggers, newTriggers...)
}

// removeWebhookTriggers removes the Triggers from the EventListener that match
// the webhook name. This change is only made in memory and needs to be
// persisted
func removeWebhookTriggers(cg *client.Group, eventListener *triggersv1alpha1.EventListener, webhookName string) {
	newTriggers := []triggersv1alpha1.EventListenerTrigger{}
	for _, trigger := range eventListener.Spec.Triggers {
		if isWebhookTrigger(trigger) && getWebhookNameFromTrigger(trigger) != webhookName {
			newTriggers = append(newTriggers, trigger)
		}
	}
	eventListener.Spec.Triggers = newTriggers
}

// newTrigger creates a new Trigger
func newTrigger(triggerName, bindingName, templateName, interceptorNamespace, repoURL, eventType, secretName string, params []pipelinesv1alpha1.Param) triggersv1alpha1.EventListenerTrigger {
	return triggersv1alpha1.EventListenerTrigger{
		Name: triggerName,
		Binding: triggersv1alpha1.EventListenerBinding{
			Name:       bindingName,
			APIVersion: "v1alpha1",
		},
		Params: params,
		Template: triggersv1alpha1.EventListenerTemplate{
			Name:       templateName,
			APIVersion: "v1alpha1",
		},
		Interceptor: &triggersv1alpha1.EventInterceptor{
			Header: []pipelinesv1alpha1.Param{
				{Name: WextInterceptorTriggerName, Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: triggerName}},
				{Name: WextInterceptorRepoURL, Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: repoURL}},
				{Name: WextInterceptorEvent, Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: eventType}},
				{Name: WextInterceptorSecretName, Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: secretName}}},
			ObjectRef: &corev1.ObjectReference{
				APIVersion: "v1",
				Kind:       "Service",
				Name:       wextValidator,
				Namespace:  interceptorNamespace,
			},
		},
	}
}

// getMonitorTriggerParams returns parameters to be used by the monitor trigger
func getMonitorTriggerParams(cg *client.Group, w models.Webhook) []pipelinesv1alpha1.Param {
	return []pipelinesv1alpha1.Param{
		{Name: wextMonitorSecretName, Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: w.AccessTokenRef}},
		{Name: wextMonitorSecretKey, Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: AccessToken}},
		{Name: wextMonitorDashboardURL, Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: getDashboardURL(cg)}},
	}
}

// getPipelineTriggerParams returns parameters according to the specified
// webhook for the pipeline trigger
func getPipelineTriggerParams(w models.Webhook) []pipelinesv1alpha1.Param {
	url, _ := sanitizeGitURL(w.GitRepositoryURL)
	server, org, repo := getGitValues(*url)
	return []pipelinesv1alpha1.Param{
		{Name: wextTargetNamespace, Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: w.Namespace}},
		{Name: wextServiceAccount, Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: w.ServiceAccount}},
		{Name: wextDockerRegistry, Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: w.DockerRegistry}},
		{Name: wextGitServer, Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: server}},
		{Name: wextGitOrg, Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: org}},
		{Name: wextGitRepo, Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: repo}},
	}
}

// triggerToWebhook converts a webhook EventListenerTrigger into a Webhook
func triggerToWebhook(t triggersv1alpha1.EventListenerTrigger) (*models.Webhook, error) {
	expectedParams := map[string]string{
		wextTargetNamespace: "",
		wextServiceAccount:  "",
		wextDockerRegistry:  "",
	}
	expectedInterceptorParams := map[string]string{
		WextInterceptorSecretName: "",
		WextInterceptorRepoURL:    "",
	}
	// Find expected parameters
	for _, param := range t.Params {
		if _, ok := expectedParams[param.Name]; ok {
			expectedParams[param.Name] = param.Value.StringVal
		}
	}
	for _, param := range t.Interceptor.Header {
		if _, ok := expectedInterceptorParams[param.Name]; ok {
			expectedInterceptorParams[param.Name] = param.Value.StringVal
		}
	}
	// Check for any empty values
	for _, expectMap := range []map[string]string{
		expectedParams,
		expectedInterceptorParams,
	} {
		for key, val := range expectMap {
			if val == "" {
				return nil, xerrors.Errorf("%s was not found", key)
			}
		}
	}
	w := &models.Webhook{
		Name:             getWebhookNameFromTrigger(t),
		Namespace:        expectedParams[wextTargetNamespace],
		ServiceAccount:   expectedParams[wextServiceAccount],
		DockerRegistry:   expectedParams[wextDockerRegistry],
		AccessTokenRef:   expectedInterceptorParams[WextInterceptorSecretName],
		Pipeline:         getPipelineNameFromTrigger(t),
		GitRepositoryURL: expectedInterceptorParams[WextInterceptorRepoURL],
	}
	return w, nil
}

// filterWebhooksByRepo returns the filtered set of webhooks that match the repo
func filterWebhooksByRepo(webhooks []models.Webhook, repoURL string) []models.Webhook {
	filteredWebhooks := []models.Webhook{}
	for _, webhook := range webhooks {
		if webhook.GitRepositoryURL == repoURL {
			filteredWebhooks = append(filteredWebhooks, webhook)
		}
	}
	return filteredWebhooks
}

// findWebhookByName the named webhook from the list and errors if not found
func findWebhookByName(webhooks []models.Webhook, name string) (*models.Webhook, error) {
	for _, webhook := range webhooks {
		if webhook.Name == name {
			return &webhook, nil
		}
	}
	return nil, xerrors.New("Webhook not found")
}

// isWebhookTrigger returns whether or not the Trigger is a webhook Trigger by
// checking for the existance of the existance of the extension validator
// interceptor
func isWebhookTrigger(t triggersv1alpha1.EventListenerTrigger) bool {
	if t.Interceptor == nil {
		return false
	}
	if t.Interceptor.ObjectRef == nil {
		return false
	}
	return (t.Interceptor.ObjectRef.Name == wextValidator)
}

// getWebhookNameFromTrigger gets the name of a webhook given a
// Trigger. The trigger is assumed to be a valid webhook trigger
func getWebhookNameFromTrigger(t triggersv1alpha1.EventListenerTrigger) string {
	delimiterIndex := strings.Index(t.Name, "-")
	return t.Name[:delimiterIndex]
}

// getPipelineNameFromTrigger gets the name of a pipeline given a
// Trigger. The trigger is assumed to be a valid webhook trigger
func getPipelineNameFromTrigger(t triggersv1alpha1.EventListenerTrigger) string {
	delimiterIndex := strings.Index(t.Template.Name, "-")
	return t.Template.Name[:delimiterIndex]
}

// getBaseEventListener returns the base EventListener. Triggers must be added
// before creating to pass validation
func getBaseEventListener(installNamespace string) *triggersv1alpha1.EventListener {
	return &triggersv1alpha1.EventListener{
		ObjectMeta: metav1.ObjectMeta{
			Name:      eventListenerName,
			Namespace: installNamespace,
		},
		Spec: triggersv1alpha1.EventListenerSpec{
			ServiceAccountName: eventListenerSA,
		},
	}
}

// getWebhookEventListener returns the singleton EventListener used for webhooks
func getWebhookEventListener(cg *client.Group) (*triggersv1alpha1.EventListener, error) {
	return cg.TriggersClient.TektonV1alpha1().EventListeners(cg.Defaults.Namespace).Get(eventListenerName, metav1.GetOptions{})
}

// createEventListener attempts to the create the EventListener
func createEventListener(cg *client.Group, el *triggersv1alpha1.EventListener) error {
	_, err := cg.TriggersClient.TektonV1alpha1().EventListeners(cg.Defaults.Namespace).Create(el)
	return err
}

// updateEventListener attempts to update the EventListener
func updateEventListener(cg *client.Group, el *triggersv1alpha1.EventListener) error {
	_, err := cg.TriggersClient.TektonV1alpha1().EventListeners(cg.Defaults.Namespace).Update(el)
	return err
}

// deleteEventListener attempts to delete the EventListener
func deleteEventListener(cg *client.Group) error {
	return cg.TriggersClient.TektonV1alpha1().EventListeners(cg.Defaults.Namespace).Delete(eventListenerName, &metav1.DeleteOptions{})
}

// getWebhooksFromEventListener returns all the webhooks on the EventListener.
// When webhooks are created, multiple triggers are created with identical
// information so the pull trigger is arbitrary choosen to represent the webhook
func getWebhooksFromEventListener(el triggersv1alpha1.EventListener) []models.Webhook {
	logging.Log.Info("Getting webhooks from eventlistener")
	hooks := []models.Webhook{}
	for _, trigger := range el.Spec.Triggers {
		if isWebhookTrigger(trigger) && strings.HasSuffix(trigger.Name, pullTriggerBindingPostfix) {
			if hook, err := triggerToWebhook(trigger); err != nil {
				logging.Log.Debug(err)
				hooks = append(hooks, *hook)
			}
		}
	}
	return hooks
}

// waitForEventListenerStatus polls the created webhook EventListener until the
// EventListenerStatus is populated, which ensures the backing service is
// created.
func waitForEventListenerStatus(cg *client.Group) (*triggersv1alpha1.EventListener, error) {
	for {
		el, err := cg.TriggersClient.TektonV1alpha1().EventListeners(cg.Defaults.Namespace).Get(eventListenerName, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		if el.Status.Configuration.GeneratedResourceName != "" {
			return el, nil
		}
		time.Sleep(time.Millisecond * 100)
	}
}

// getGitValues extracts information from the url assuming it has already been
// validated by `sanitizeGitURL`
func getGitValues(u url.URL) (server, org, repo string) {
	// The Path should be in the form: /org/repo
	lastIndex := strings.LastIndex(u.Path, "/")
	return u.Host, u.Path[1:lastIndex], u.Path[lastIndex+1:]
}

// getWebhookSecretTokens attempts to return the accessToken and secretToken
// stored in the Secret
func getWebhookSecretTokens(cg *client.Group, secretName string) (aToken string, sToken string, err error) {
	secret, err := cg.K8sClient.CoreV1().Secrets(cg.Defaults.Namespace).Get(secretName, metav1.GetOptions{})
	if err != nil {
		return "", "", xerrors.Errorf("Error getting Webhook secret. Error was: %w", err)
	}
	accessToken, ok := secret.Data[AccessToken]
	if !ok {
		return "", "", xerrors.New("Did not find access token")
	}
	secretToken, ok := secret.Data[SecretToken]
	if !ok {
		return "", "", xerrors.New("Did not find secret token")
	}
	return string(accessToken), string(secretToken), nil
}

// sanitizeGitURL returns a URL for the specified rawurl string, where
// the .git suffix is removed. The rawurl must have the following format:
// `http(s)://<git-site>.com/<some-org>/<some-repo>(.git)`
func sanitizeGitURL(rawurl string) (*url.URL, error) {
	url, err := url.ParseRequestURI(strings.TrimSuffix(rawurl, ".git"))
	if err != nil {
		return nil, err
	}
	if !strings.HasSuffix(url.Hostname(), ".com") ||
		len(url.Hostname()) == 0 ||
		strings.HasPrefix(url.Hostname(), ".") {
		return nil, xerrors.Errorf("URL hostname '%s' is invalid", url.Hostname())
	}
	if !(url.Scheme == "http" || url.Scheme == "https") {
		return nil, xerrors.Errorf("URL scheme '%s' is invalid", url.Scheme)
	}
	// Does not allow trailing slashes
	// Expects a path in the format: /<some-org>/<some-repo>
	s := strings.Split(url.Path, "/")
	if len(s) != 3 || s[1] == "" || s[2] == "" {
		return nil, xerrors.Errorf("URL path '%s' is invalid", url.Path)
	}
	return url, nil
}

// getDashboardURL gets the URL of the Dashboard
func getDashboardURL(cg *client.Group) string {
	type element struct {
		Type string `json:"type"`
		URL  string `json:"url"`
	}

	dashboardURL := "http://localhost:9097/"

	labelLookup := "app=tekton-dashboard"
	if cg.Defaults.Platform == "openshift" {
		labelLookup = "app=tekton-dashboard-internal"
	}

	services, err := cg.K8sClient.CoreV1().Services(cg.Defaults.Namespace).List(metav1.ListOptions{LabelSelector: labelLookup})
	if err != nil || len(services.Items) == 0 {
		logging.Log.Errorf("Could not find the Dashboard's Service")
		return dashboardURL
	}

	name := services.Items[0].Name
	scheme := services.Items[0].Spec.Ports[0].Name
	port := services.Items[0].Spec.Ports[0].Port
	dashboardURL = fmt.Sprintf("%s://%s:%d/v1/namespaces/%s/endpoints", scheme, name, port, cg.Defaults.Namespace)
	logging.Log.Debugf("Using url: %s", dashboardURL)
	resp, err := http.DefaultClient.Get(dashboardURL)
	if err != nil {
		logging.Log.Errorf("Error getting endpoints from url: %s", err.Error())
		return dashboardURL
	}
	if resp.StatusCode != 200 {
		logging.Log.Errorf("Return code was not 200 when hitting the endpoints REST endpoint, code returned was: %d", resp.StatusCode)
		return dashboardURL
	}

	bodyJSON := []element{}
	json.NewDecoder(resp.Body).Decode(&bodyJSON)
	// Return the first URL received from the Dashboard
	return bodyJSON[0].URL
}
