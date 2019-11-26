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
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/xerrors"

	restful "github.com/emicklei/go-restful"
	logging "github.com/tektoncd/experimental/webhooks-extension/pkg/logging"
	"github.com/tektoncd/experimental/webhooks-extension/pkg/utils"
	pipelinesv1alpha1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	v1alpha1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var (
	// modifyingConfigMapLock is the lock that must be acquired when making any
	// changes to the ConfigMap
	modifyingConfigMapLock sync.Mutex
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
	// configMapName is the name of the ConfigMap that stores information for
	// GitHub webhooks
	configMapName = "githubwebhook"
	// eventListenrName is the name of the EventListener that is the singleton
	// source of truth for Triggers/events
	eventListenerName = "tekton-webhooks-eventlistener"
	// eventListenerSA is the name of the serviceAccount that should the
	// EventListener (eventListenerName) should be configured with
	eventListenerSA = "tekton-webhooks-extension-eventlistener"
)

// webhook contains only the form payload structure used to create a webhook.
// This is defined within /src/components/WebhookCreate/WebhookCreate.js
type webhook struct {
	// Name is the name of the webhook in the UI
	Name string `json:"name"`
	// Namespace is the namespace passed to the TriggerTemplate
	Namespace string `json:"namespace"`
	// ServiceAccount is the serviceAccount passed to the TriggerTemplate
	ServiceAccount string `json:"serviceaccount,omitempty"`
	// AccessTokenRef is the name of the git secret used. This is used for
	// validation
	AccessTokenRef string `json:"accesstoken"`
	// Pipeline is the pipeline that a webhook is being created for. The
	// pipeline must have corresponding triggers resources.
	Pipeline string `json:"pipeline"`
	// DockerRegistry is the registry used to upload images within the pipeline
	DockerRegistry string `json:"dockerregistry,omitempty"`
	// GitRepositoryURL is broken down into fields (server, org, and repo) and
	// passed to the TriggerTemplate. This is also used for validation.
	GitRepositoryURL string `json:"gitrepositoryurl"`
}

// createEventListener creates the singleton eventListener for webhooks. This
// should only be called if the EventListener does not exist
func (r Resource) createEventListener() (*v1alpha1.EventListener, error) {
	eventListener := v1alpha1.EventListener{
		ObjectMeta: metav1.ObjectMeta{
			Name:      eventListenerName,
			Namespace: r.Defaults.Namespace,
		},
		Spec: v1alpha1.EventListenerSpec{
			ServiceAccountName: eventListenerSA,
		},
	}
	return r.TriggersClient.TektonV1alpha1().EventListeners(r.Defaults.Namespace).Create(&eventListener)
}

// updateEventListener updates the EventListener with additional triggers
// according to the specified webhook.
func (r Resource) updateEventListener(eventListener *v1alpha1.EventListener, webhook webhook) (*v1alpha1.EventListener, error) {
	hookParams, monitorParams := r.getParams(webhook)

	newPushTrigger := r.newTrigger(fmt.Sprintf("%s-%s-push-event", webhook.Name, webhook.Namespace),
		fmt.Sprintf("%s-push-binding", webhook.Pipeline),
		fmt.Sprintf("%s-template", webhook.Pipeline),
		webhook.GitRepositoryURL,
		"push",
		webhook.AccessTokenRef,
		hookParams)

	newPullRequestTrigger := r.newTrigger(fmt.Sprintf("%s-%s-pullrequest-event", webhook.Name, webhook.Namespace),
		fmt.Sprintf("%s-pullrequest-binding", webhook.Pipeline),
		fmt.Sprintf("%s-template", webhook.Pipeline),
		webhook.GitRepositoryURL,
		"pull_request",
		webhook.AccessTokenRef,
		hookParams)

	monitorTrigger := r.newTrigger("monitor-task",
		fmt.Sprintf("%s-binding", webhook.Pipeline),
		fmt.Sprintf("%s-template", webhook.Pipeline),
		webhook.GitRepositoryURL,
		"pull_request",
		webhook.AccessTokenRef,
		monitorParams)

	newTriggers := []v1alpha1.EventListenerTrigger{newPushTrigger, newPullRequestTrigger, monitorTrigger}

	eventListener.Spec.Triggers = append(eventListener.Spec.Triggers, newTriggers...)
	return r.TriggersClient.TektonV1alpha1().EventListeners(eventListener.GetNamespace()).Update(eventListener)
}

// newTrigger creates a new Trigger according to the specified parameters
func (r Resource) newTrigger(name, bindingName, templateName, repoURL, event, secretName string, params []pipelinesv1alpha1.Param) v1alpha1.EventListenerTrigger {
	return v1alpha1.EventListenerTrigger{
		Name: name,
		Binding: v1alpha1.EventListenerBinding{
			Name:       bindingName,
			APIVersion: "v1alpha1",
		},
		Params: params,
		Template: v1alpha1.EventListenerTemplate{
			Name:       templateName,
			APIVersion: "v1alpha1",
		},
		Interceptor: &v1alpha1.EventInterceptor{
			Header: []pipelinesv1alpha1.Param{
				{Name: "Wext-Trigger-Name", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: name}},
				{Name: "Wext-Repository-Url", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: repoURL}},
				{Name: "Wext-Incoming-Event", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: event}},
				{Name: "Wext-Secret-Name", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: secretName}}},
			ObjectRef: &corev1.ObjectReference{
				APIVersion: "v1",
				Kind:       "Service",
				Name:       "tekton-webhooks-extension-validator",
				Namespace:  r.Defaults.Namespace,
			},
		},
	}
}

// getMonitorTriggerParams returns parameters according to the specified webhook
// for the monitor trigger
func (r Resource) getMonitorTriggerParams(w webhook) []pipelinesv1alpha1.Param {
	return []pipelinesv1alpha1.Param{
		{Name: "gitsecretname", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: w.AccessTokenRef}},
		{Name: "gitsecretkeyname", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: "accessToken"}},
		{Name: "dashboardurl", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: r.getDashboardURL()}},
	}
}

// getPipelineTriggerParams returns parameters according to the specified
// webhook for the pipeline trigger
func (r Resource) getPipelineTriggerParams(w webhook) []pipelinesv1alpha1.Param {
	return []pipelinesv1alpha1.Param{
		{Name: "webhooks-tekton-release-name", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: w.GitRepositoryURL}},
		{Name: "webhooks-tekton-target-namespace", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: w.Namespace}},
		{Name: "webhooks-tekton-service-account", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: w.ServiceAccount}},
		{Name: "webhooks-tekton-git-server", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: "server"}},
		{Name: "webhooks-tekton-git-org", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: "org"}},
		{Name: "webhooks-tekton-git-repo", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: "repo"}},
	}
}

func (r Resource) getParams(webhook webhook) (webhookParams, monitorParams []pipelinesv1alpha1.Param) {
	saName := webhook.ServiceAccount
	if saName == "" {
		saName = "default"
	}
	server, org, repo, err := getGitValues(webhook.GitRepositoryURL)
	if err != nil {
		logging.Log.Errorf("error returned from getGitValues: %s", err)
	}
	server = strings.TrimPrefix(server, "https://")
	server = strings.TrimPrefix(server, "http://")

	hookParams := []pipelinesv1alpha1.Param{
		{Name: "webhooks-tekton-release-name", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: webhook.GitRepositoryURL}},
		{Name: "webhooks-tekton-target-namespace", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: webhook.Namespace}},
		{Name: "webhooks-tekton-service-account", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: webhook.ServiceAccount}},
		{Name: "webhooks-tekton-git-server", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: server}},
		{Name: "webhooks-tekton-git-org", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: org}},
		{Name: "webhooks-tekton-git-repo", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: repo}}}

	if webhook.DockerRegistry != "" {
		hookParams = append(hookParams, pipelinesv1alpha1.Param{Name: "webhooks-tekton-docker-registry", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: webhook.DockerRegistry}})
	}

	prMonitorParams := []pipelinesv1alpha1.Param{
		{Name: "gitsecretname", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: webhook.AccessTokenRef}},
		{Name: "gitsecretkeyname", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: "accessToken"}},
		{Name: "dashboardurl", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: r.getDashboardURL()}},
	}

	return hookParams, prMonitorParams
}

func (r Resource) getDashboardURL() string {
	type element struct {
		Type string `json:"type"`
		URL  string `json:"url"`
	}

	toReturn := "http://localhost:9097/"

	labelLookup := "app=tekton-dashboard"
	if "openshift" == os.Getenv("PLATFORM") {
		labelLookup = "app=tekton-dashboard-internal"
	}

	services, err := r.K8sClient.CoreV1().Services(r.Defaults.Namespace).List(metav1.ListOptions{LabelSelector: labelLookup})
	if err != nil {
		logging.Log.Errorf("could not find the dashboard's service - error: %s", err.Error())
		return toReturn
	}

	if len(services.Items) == 0 {
		logging.Log.Error("could not find the dashboard's service")
		return toReturn
	}

	name := services.Items[0].GetName()
	proto := services.Items[0].Spec.Ports[0].Name
	port := services.Items[0].Spec.Ports[0].Port
	url := fmt.Sprintf("%s://%s:%d/v1/namespaces/%s/endpoints", proto, name, port, r.Defaults.Namespace)
	logging.Log.Debugf("using url: %s", url)
	resp, err := http.DefaultClient.Get(url)
	if err != nil {
		logging.Log.Errorf("error occurred when hitting the endpoints REST endpoint: %s", err.Error())
		return url
	}
	if resp.StatusCode != 200 {
		logging.Log.Errorf("return code was not 200 when hitting the endpoints REST endpoint, code returned was: %d", resp.StatusCode)
		return url
	}

	bodyJSON := []element{}
	json.NewDecoder(resp.Body).Decode(&bodyJSON)
	return bodyJSON[0].URL
}

// getGitValues processes a git URL into component parts, all of which are
// lowercased to try and avoid problems matching strings.
func getGitValues(url string) (gitServer, gitOwner, gitRepo string, err error) {
	repoURL := ""
	prefix := ""
	if url != "" {
		url = strings.ToLower(url)
		if strings.Contains(url, "https://") {
			repoURL = strings.TrimPrefix(url, "https://")
			prefix = "https://"
		} else {
			repoURL = strings.TrimPrefix(url, "http://")
			prefix = "http://"
		}
	}
	// example at this point: github.com/tektoncd/pipeline
	numSlashes := strings.Count(repoURL, "/")
	if numSlashes < 2 {
		return "", "", "", errors.New("URL didn't contain an owner and repository")
	}
	repoURL = strings.TrimSuffix(repoURL, "/")
	gitServer = prefix + repoURL[0:strings.Index(repoURL, "/")]
	gitOwner = repoURL[strings.Index(repoURL, "/")+1 : strings.LastIndex(repoURL, "/")]
	//need to cut off the .git
	if strings.HasSuffix(url, ".git") {
		gitRepo = repoURL[strings.LastIndex(repoURL, "/")+1 : len(repoURL)-4]
	} else {
		gitRepo = repoURL[strings.LastIndex(repoURL, "/")+1:]
	}

	return gitServer, gitOwner, gitRepo, nil
}

// createWebhook creates a webhook for a given repository and populates
// (creating if doesn't yet exist) a ConfigMap storing this information
func (r Resource) CreateWebhook(request *restful.Request, response *restful.Response) {
	modifyingConfigMapLock.Lock()
	defer modifyingConfigMapLock.Unlock()

	logging.Log.Infof("Webhook creation request received with request: %+v.", request)

	webhook := webhook{}
	if err := request.ReadEntity(&webhook); err != nil {
		err = xerrors.Errorf("Error trying to read request entity as webhook %s", err)
		utils.RespondError(response, err, http.StatusBadRequest)
		return
	}
	if err := checkWebhook(webhook); err != nil {
		utils.RespondError(response, err, http.StatusBadRequest)
		return
	}
	_, err := sanitizeGitURL(webhook.GitRepositoryURL)
	if err != nil {
		err = xerrors.Errorf("Invalid value webhook URL: %s", err)
		utils.RespondError(response, err, http.StatusBadRequest)
		return
	}

	// hooks, err := r.getGitHubWebhooksFromConfigMap(gitURL)
	// if len(hooks) > 0 {
	// 	for _, hook := range hooks {
	// 		if hook.Name == webhook.Name && hook.Namespace == webhook.Namespace {
	// 			logging.Log.Errorf("Error creating webhook: A webhook already exists for GitRepositoryURL %+v with the Name %s and Namespace %s.", webhook.GitRepositoryURL, webhook.Name, webhook.Namespace)
	// 			utils.RespondError(response, errors.New("Webhook already exists for the specified Git repository with the same name, targeting the same namespace"), http.StatusBadRequest)
	// 			return
	// 		}
	// 		if hook.Pipeline == webhook.Pipeline && hook.Namespace == webhook.Namespace {
	// 			logging.Log.Errorf("Error creating webhook: A webhook already exists for GitRepositoryURL %+v, running pipeline %s in namespace %s.", webhook.GitRepositoryURL, webhook.Pipeline, webhook.Namespace)
	// 			utils.RespondError(response, errors.New("Webhook already exists for the specified Git repository, running the same pipeline in the same namespace"), http.StatusBadRequest)
	// 			return
	// 		}
	// 	}
	// }

	_, templateErr := r.TriggersClient.TektonV1alpha1().TriggerTemplates(r.Defaults.Namespace).Get(webhook.Pipeline+"-template", metav1.GetOptions{})
	_, pushErr := r.TriggersClient.TektonV1alpha1().TriggerBindings(r.Defaults.Namespace).Get(webhook.Pipeline+"-push-binding", metav1.GetOptions{})
	_, pullrequestErr := r.TriggersClient.TektonV1alpha1().TriggerBindings(r.Defaults.Namespace).Get(webhook.Pipeline+"-pullrequest-binding", metav1.GetOptions{})
	if templateErr != nil || pushErr != nil || pullrequestErr != nil {
		msg := fmt.Sprintf("Could not find the required trigger template or trigger bindings in namespace: %s. Expected to find: %s, %s and %s", r.Defaults.Namespace, webhook.Pipeline+"-template", webhook.Pipeline+"-push-binding", webhook.Pipeline+"-pullrequest-binding")
		logging.Log.Errorf("%s", msg)
		utils.RespondError(response, errors.New(msg), http.StatusBadRequest)
		return
	}

	// Single monitor trigger for all triggers on a repo - thus name to use for monitor is
	// monitorTriggerName := strings.TrimPrefix(gitServer+"/"+gitOwner+"/"+gitRepo, "http://")
	// monitorTriggerName = strings.TrimPrefix(monitorTriggerName, "https://")
	// monitorTriggerName := webhook.GitRepositoryURL

	// eventListener, err := r.TriggersClient.TektonV1alpha1().EventListeners(r.Defaults.Namespace).Get(eventListenerName, metav1.GetOptions{})
	// if err != nil {
	// 	if !k8serrors.IsNotFound(err) {
	// 		err = errors.Wrap(err, "Webhook creation failure")
	// 		logging.Log.Error(err)
	// 		utils.RespondError(response, errors.New(msg), http.StatusInternalServerError)
	// 		return
	// 	}
	// 	eventListener, err = r.createEventListener()
	// 	if err != nil {
	// 		err = errors.Wrap(err, "Webhook creation failure")
	// 		logging.Log.Error(err)
	// 		utils.RespondError(response, err, http.StatusInternalServerError)
	// 		return
	// 	}
	// }
	// _, err = r.updateEventListener(eventListener, webhook, monitorTriggerName)
	// if err != nil {
	// 	err = errors.Wrap(err, "Webhook creation failure")
	// 	logging.Log.Error(err)
	// 	utils.RespondError(response, err, http.StatusInternalServerError)
	// 	return
	// }

	// // Loop until the EventListener status is populated
	// for eventListener.Status.Configuration.GeneratedResourceName == "" {
	// 	time.Sleep(time.Millisecond * 100)
	// 	eventListener, err := r.TriggersClient.TektonV1alpha1().EventListeners(r.Defaults.Namespace).Get(eventListenerName, metav1.GetOptions{})
	// 	if err != nil {
	// 		err = errors.Wrap(err, "Webhook creation failure")
	// 		logging.Log.Error(err)
	// 		utils.RespondError(response, err, http.StatusInternalServerError)
	// 		return
	// 	}
	// }

	// if strings.Contains(strings.ToLower(r.Defaults.Platform), "openshift") {
	// 	r.RoutesClient.RouteV1().Routes()
	// } else {
	// 	err = r.createDeleteIngress("create")
	// 	if err != nil {
	// 		err = errors.Wrap(err, "Webhook creation failure")
	// 		logging.Log.Error(err)
	// 		logging.Log.Debugf("Deleting eventlistener as failed creating Ingress")
	// 		err2 := r.TriggersClient.TektonV1alpha1().EventListeners(r.Defaults.Namespace).Delete(eventListenerName, &metav1.DeleteOptions{})
	// 		if err2 != nil {
	// 			updatedMsg := fmt.Sprintf("error creating webhook due to error creating taskrun to create ingress. Also failed to cleanup and delete eventlistener. Errors were: %s and %s", err, err2)
	// 			utils.RespondError(response, errors.New(updatedMsg), http.StatusInternalServerError)
	// 			return
	// 		}
	// 		utils.RespondError(response, errors.New(msg), http.StatusInternalServerError)
	// 		return
	// 	} else {
	// 		logging.Log.Debug("ingress creation taskrun succeeded")
	// 	}
	// }

	// if len(hooks) == 0 {
	// 	webhookTaskRun, err := r.createGitHubWebhookTaskRun("create", sanitisedURL, gitServer, webhook)
	// 	if err != nil {
	// 		msg := fmt.Sprintf("error creating taskrun to create github webhook. Error was: %s", err)
	// 		logging.Log.Errorf("%s", msg)
	// 		err2 := r.deleteFromEventListener(webhook.Name+"-"+webhook.Namespace, monitorTriggerName, webhook.GitRepositoryURL)
	// 		if err2 != nil {
	// 			updatedMsg := fmt.Sprintf("error creating webhook creation taskrun. Also failed to cleanup and delete entry from eventlistener. Errors were: %s and %s", err, err2)
	// 			utils.RespondError(response, errors.New(updatedMsg), http.StatusInternalServerError)
	// 			return
	// 		}
	// 		utils.RespondError(response, errors.New(msg), http.StatusInternalServerError)
	// 		return
	// 	}
	// 	webhookTaskRunResult, err := r.checkTaskRunSucceeds(webhookTaskRun)
	// 	if !webhookTaskRunResult && err != nil {
	// 		msg := fmt.Sprintf("error in taskrun creating webhook. Error was: %s", err)
	// 		logging.Log.Errorf("%s", msg)
	// 		utils.RespondError(response, errors.New(msg), http.StatusInternalServerError)
	// 		return
	// 	}
	// 	logging.Log.Debug("webhook taskrun succeeded")
	// } else {
	// 	logging.Log.Debugf("webhook already exists for repository %s - not creating new hook in GitHub", sanitisedURL)
	// }

	// webhooks, err := r.readGitHubWebhooksFromConfigMap()
	// if err != nil {
	// 	logging.Log.Errorf("error getting GitHub webhooks: %s.", err.Error())
	// 	utils.RespondError(response, err, http.StatusInternalServerError)
	// 	return
	// }

	// webhooks[sanitisedURL] = append(webhooks[sanitisedURL], webhook)
	// logging.Log.Debugf("Writing the GitHubSource webhook ConfigMap in namespace %s", r.Defaults.Namespace)
	// r.writeGitHubWebhooks(webhooks)
	// response.WriteHeader(http.StatusCreated)
}

func (r Resource) createDeleteIngress(mode string) error {
	if mode == "create" {
		// Unlike webhook creation, the ingress does not need a protocol specified
		callback := strings.TrimPrefix(r.Defaults.CallbackURL, "http://")
		callback = strings.TrimPrefix(callback, "https://")

		ingress := &v1beta1.Ingress{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "el-" + eventListenerName,
				Namespace: r.Defaults.Namespace,
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
											ServiceName: "el-" + eventListenerName,
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
		ingress, err := r.K8sClient.ExtensionsV1beta1().Ingresses(r.Defaults.Namespace).Create(ingress)
		if err != nil {
			return err
		}
		logging.Log.Debug("Ingress has been created")
		return nil
	} else if mode == "delete" {
		err := r.K8sClient.ExtensionsV1beta1().Ingresses(r.Defaults.Namespace).Delete("el-"+eventListenerName, &metav1.DeleteOptions{})
		if err != nil {
			return err
		}
		logging.Log.Debug("Ingress has been deleted")
		return nil
	} else {
		logging.Log.Debug("Wrong mode")
		return errors.New("Wrong mode for createDeleteIngress")
	}
}

func (r Resource) createGitHubWebhookTaskRun(mode, gitRepoURL, gitServer string, webhook webhook) (*pipelinesv1alpha1.TaskRun, error) {
	params := []pipelinesv1alpha1.Param{
		{Name: "Mode", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: mode}},
		{Name: "CallbackURL", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: r.Defaults.CallbackURL}},
		{Name: "GitHubRepoURL", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: gitRepoURL}},
		{Name: "GitHubSecretName", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: webhook.AccessTokenRef}},
		{Name: "GitHubAccessTokenKey", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: "accessToken"}},
		{Name: "GitHubUserNameKey", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: ""}},
		{Name: "GitHubSecretStringKey", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: "secretToken"}},
		{Name: "GitHubServerUrl", Value: pipelinesv1alpha1.ArrayOrString{Type: pipelinesv1alpha1.ParamTypeString, StringVal: gitServer}}}

	webhookTaskRun := pipelinesv1alpha1.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: mode + "-webhook-",
			Namespace:    r.Defaults.Namespace,
		},
		Spec: pipelinesv1alpha1.TaskRunSpec{
			Inputs: pipelinesv1alpha1.TaskRunInputs{
				Params: params,
			},
			ServiceAccount: os.Getenv("SERVICE_ACCOUNT"),
			TaskRef: &pipelinesv1alpha1.TaskRef{
				Name: "webhook-task",
			},
		},
	}

	tr, err := r.TektonClient.TektonV1alpha1().TaskRuns(r.Defaults.Namespace).Create(&webhookTaskRun)
	if err != nil {
		return &pipelinesv1alpha1.TaskRun{}, err
	}
	logging.Log.Debugf("Webhook being created/deleted under taskrun %s", tr.GetName())

	return tr, nil
}

func (r Resource) checkTaskRunSucceeds(originalTaskRun *pipelinesv1alpha1.TaskRun) (bool, error) {
	var err error
	retries := 1
	for retries < 120 {
		taskRun, err := r.TektonClient.TektonV1alpha1().TaskRuns(r.Defaults.Namespace).Get(originalTaskRun.Name, metav1.GetOptions{})
		if err != nil {
			logging.Log.Debugf("Error occured retrieving taskrun %s.", originalTaskRun.Name)
			return false, err
		}
		if taskRun.IsDone() {
			if taskRun.IsSuccessful() {
				return true, nil
			}
			if taskRun.IsCancelled() {
				err = errors.New("taskrun " + taskRun.Name + " is in a cancelled state")
				return false, err
			}
			err = errors.New("taskrun " + taskRun.Name + " is in a failed or unknown state")
			return false, err
		}
		time.Sleep(1 * time.Second)
		retries = retries + 1
	}

	err = errors.New("taskrun " + originalTaskRun.Name + " is not reporting as successful or cancelled")
	return false, err
}

// // Removes from ConfigMap, removes the actual GitHubSource, removes the webhook
func (r Resource) DeleteWebhook(request *restful.Request, response *restful.Response) {}

// 	modifyingConfigMapLock.Lock()
// 	defer modifyingConfigMapLock.Unlock()
// 	logging.Log.Debug("In deleteWebhook")
// 	name := request.PathParameter("name")
// 	repo := request.QueryParameter("repository")
// 	namespace := request.QueryParameter("namespace")
// 	deletePipelineRuns := request.QueryParameter("deletepipelineruns")

// 	var toDeletePipelineRuns = false
// 	var err error

// 	if deletePipelineRuns != "" {
// 		toDeletePipelineRuns, err = strconv.ParseBool(deletePipelineRuns)
// 		if err != nil {
// 			theError := errors.New("bad request information provided, cannot handle deletepipelineruns query (should be set to true or not provided)")
// 			logging.Log.Error(theError)
// 			utils.RespondError(response, theError, http.StatusInternalServerError)
// 			return
// 		}
// 	}

// 	if namespace == "" || repo == "" {
// 		theError := errors.New("bad request information provided, a namespace and a repository must be specified as query parameters")
// 		logging.Log.Error(theError)
// 		utils.RespondError(response, theError, http.StatusBadRequest)
// 		return
// 	}

// 	logging.Log.Debugf("in deleteWebhook, name: %s, repo: %s, delete pipeline runs: %s", name, repo, deletePipelineRuns)

// 	webhooks, err := r.getGitHubWebhooksFromConfigMap(repo)
// 	if err != nil {
// 		utils.RespondError(response, err, http.StatusNotFound)
// 		return
// 	}

// 	logging.Log.Debugf("Found %d webhooks/pipelines registered against repo %s", len(webhooks), repo)
// 	if len(webhooks) < 1 {
// 		err := fmt.Errorf("no webhook found for repo %s", repo)
// 		logging.Log.Error(err)
// 		utils.RespondError(response, err, http.StatusBadRequest)
// 		return
// 	}

// 	gitServer, gitOwner, gitRepo, err := getGitValues(repo)
// 	// Single monitor trigger for all triggers on a repo - thus name to use for monitor is
// 	monitorTriggerName := strings.TrimPrefix(gitServer+"/"+gitOwner+"/"+gitRepo, "http://")
// 	monitorTriggerName = strings.TrimPrefix(monitorTriggerName, "https://")

// 	found := false
// 	var remaining int
// 	for _, hook := range webhooks {
// 		if hook.Name == name && hook.Namespace == namespace {
// 			found = true
// 			if len(webhooks) == 1 {
// 				logging.Log.Debug("No other pipelines triggered by this GitHub webhook, deleting webhook")
// 				remaining = 0
// 				sanitisedURL := gitServer + "/" + gitOwner + "/" + gitRepo
// 				deleteWebhookTaskRun, err := r.createGitHubWebhookTaskRun("delete", sanitisedURL, gitServer, hook)
// 				if err != nil {
// 					logging.Log.Error(err)
// 					theError := errors.New("error during creation of taskrun to delete webhook. ")
// 					utils.RespondError(response, theError, http.StatusInternalServerError)
// 					return
// 				}

// 				webhookDeleted, err := r.checkTaskRunSucceeds(deleteWebhookTaskRun)
// 				if !webhookDeleted && err != nil {
// 					logging.Log.Error(err)
// 					theError := errors.New("error during taskrun deleting webhook.")
// 					utils.RespondError(response, theError, http.StatusInternalServerError)
// 					return
// 				} else {
// 					logging.Log.Debug("Webhook deletion taskrun succeeded")
// 				}
// 			} else {
// 				remaining = len(webhooks) - 1
// 			}
// 			if toDeletePipelineRuns {
// 				r.deletePipelineRuns(repo, namespace, hook.Pipeline)
// 			}
// 			err := r.deleteWebhookFromConfigMap(repo, name, namespace, remaining)
// 			if err != nil {
// 				logging.Log.Error(err)
// 				theError := errors.New("error deleting webhook from configmap.")
// 				utils.RespondError(response, theError, http.StatusInternalServerError)
// 				return
// 			}

// 			eventListenerEntryPrefix := name + "-" + namespace
// 			err = r.deleteFromEventListener(eventListenerEntryPrefix, monitorTriggerName, repo)
// 			if err != nil {
// 				logging.Log.Error(err)
// 				theError := errors.New("error deleting webhook from eventlistener.")
// 				utils.RespondError(response, theError, http.StatusInternalServerError)
// 				return
// 			}

// 			response.WriteHeader(204)
// 		}
// 	}

// 	if !found {
// 		err := fmt.Errorf("no webhook found for repo %s with name %s associated with namespace %s", repo, name, namespace)
// 		logging.Log.Error(err)
// 		utils.RespondError(response, err, http.StatusNotFound)
// 		return
// 	}

// }

// func (r Resource) deleteFromEventListener(name, monitorTriggerName, repoOnParams string) error {
// 	logging.Log.Debugf("Deleting triggers for %s from the eventlistener", name)
// 	el, err := r.TriggersClient.TektonV1alpha1().EventListeners(r.Defaults.Namespace).Get(eventListenerName, metav1.GetOptions{})
// 	if err != nil {
// 		return err
// 	}

// 	toRemove := []string{name + "-push-event", name + "-pullrequest-event"}

// 	newTriggers := []v1alpha1.EventListenerTrigger{}
// 	currentTriggers := el.Spec.Triggers

// 	monitorTrigger := v1alpha1.EventListenerTrigger{}
// 	triggersOnRepo := 0
// 	triggersDeleted := 0

// 	for _, t := range currentTriggers {
// 		if t.Name == monitorTriggerName {
// 			monitorTrigger = t
// 		} else {
// 			interceptorParams := t.Interceptor.Header
// 			for _, p := range interceptorParams {
// 				if p.Name == "Wext-Repository-Url" && p.Value.StringVal == repoOnParams {
// 					triggersOnRepo++
// 				}
// 			}
// 			found := false
// 			for _, triggerName := range toRemove {
// 				if triggerName == t.Name {
// 					triggersDeleted++
// 					found = true
// 					break
// 				}
// 			}
// 			if !found {
// 				newTriggers = append(newTriggers, t)
// 			}
// 		}
// 	}

// 	if triggersOnRepo > triggersDeleted {
// 		newTriggers = append(newTriggers, monitorTrigger)
// 	}

// 	if len(newTriggers) == 0 {
// 		err = r.TriggersClient.TektonV1alpha1().EventListeners(r.Defaults.Namespace).Delete(el.GetName(), &metav1.DeleteOptions{})
// 		if err != nil {
// 			return err
// 		}

// 		_, varExists := os.LookupEnv("PLATFORM")
// 		if !varExists {
// 			err = r.createDeleteIngress("delete")
// 			if err != nil {
// 				logging.Log.Errorf("error deleting ingress: %s", err)
// 				return err
// 			} else {
// 				logging.Log.Debug("Ingress deleted")
// 				return nil
// 			}
// 		} else {
// 			routeTaskRun, err := r.createRouteTaskRun("delete")
// 			if err != nil {
// 				msg := fmt.Sprintf("error deleting webhook due to error creating taskrun to delete route. Error was: %s", err)
// 				logging.Log.Errorf("%s", msg)
// 				return err
// 			}
// 			routeTaskRunResult, err := r.checkTaskRunSucceeds(routeTaskRun)
// 			if !routeTaskRunResult && err != nil {
// 				msg := fmt.Sprintf("error deleting webhook due to error in taskrun to delete route. Error was: %s", err)
// 				logging.Log.Errorf("%s", msg)
// 				return err
// 			} else {
// 				logging.Log.Debug("route deletion taskrun succeeded")
// 			}
// 		}

// 	} else {
// 		el.Spec.Triggers = newTriggers
// 		_, err = r.TriggersClient.TektonV1alpha1().EventListeners(r.Defaults.Namespace).Update(el)
// 		if err != nil {
// 			logging.Log.Errorf("error updating eventlistener: %s", err)
// 			return err
// 		}
// 	}

// 	return err
// }

// Delete the webhook information from our ConfigMap
// func (r Resource) deleteWebhookFromConfigMap(repository, webhookName, namespace string, remainingCount int) error {
// 	logging.Log.Debugf("Deleting webhook info named %s on repository %s running in namespace %s from ConfigMap", webhookName, repository, namespace)
// 	repository = strings.ToLower(strings.TrimSuffix(repository, ".git"))
// 	allHooks, err := r.readGitHubWebhooksFromConfigMap()
// 	if err != nil {
// 		return err
// 	}

// 	if remainingCount > 0 {
// 		logging.Log.Debugf("Finding webhook for repository %s", repository)
// 		for i, hook := range allHooks[repository] {
// 			if hook.Name == webhookName && hook.Namespace == namespace {
// 				logging.Log.Debugf("Removing webhook from ConfigMap")
// 				allHooks[repository][i] = allHooks[repository][len(allHooks[repository])-1]
// 				allHooks[repository] = allHooks[repository][:len(allHooks[repository])-1]
// 			}
// 		}
// 	} else {
// 		logging.Log.Debugf("Deleting last webhook for repository %s", repository)
// 		delete(allHooks, repository)
// 	}

// 	err = r.writeGitHubWebhooks(allHooks)
// 	if err != nil {
// 		return err
// 	}
// 	return nil
// }

func (r Resource) GetAllWebhooks(request *restful.Request, response *restful.Response) {}

// 	logging.Log.Debugf("Get all webhooks")
// 	sources, err := r.readGitHubWebhooksFromConfigMap()
// 	if err != nil {
// 		logging.Log.Errorf("error trying to get webhooks: %s.", err.Error())
// 		utils.RespondError(response, err, http.StatusInternalServerError)
// 		return
// 	}
// 	sourcesList := []webhook{}
// 	for _, value := range sources {
// 		sourcesList = append(sourcesList, value...)
// 	}
// 	response.WriteEntity(sourcesList)

// }

// func (r Resource) deletePipelineRuns(gitRepoURL, namespace, pipeline string) error {
// 	logging.Log.Debugf("Looking for PipelineRuns in namespace %s with repository URL %s for pipeline %s", namespace, gitRepoURL, pipeline)

// 	allPipelineRuns, err := r.TektonClient.TektonV1alpha1().PipelineRuns(namespace).List(metav1.ListOptions{})

// 	if err != nil {
// 		logging.Log.Errorf("Unable to retrieve PipelineRuns in the namespace %s! Error: %s", namespace, err.Error())
// 		return err
// 	}

// 	found := false
// 	for _, pipelineRun := range allPipelineRuns.Items {
// 		if pipelineRun.Spec.PipelineRef.Name == pipeline {
// 			labels := pipelineRun.GetLabels()
// 			serverURL := labels["gitServer"]
// 			orgName := labels["gitOrg"]
// 			repoName := labels["gitRepo"]
// 			foundRepoURL := fmt.Sprintf("https://%s/%s/%s", serverURL, orgName, repoName)

// 			gitRepoURL = strings.ToLower(strings.TrimSuffix(gitRepoURL, ".git"))
// 			foundRepoURL = strings.ToLower(strings.TrimSuffix(foundRepoURL, ".git"))

// 			if foundRepoURL == gitRepoURL {
// 				found = true
// 				err := r.TektonClient.TektonV1alpha1().PipelineRuns(namespace).Delete(pipelineRun.Name, &metav1.DeleteOptions{})
// 				if err != nil {
// 					logging.Log.Errorf("failed to delete %s, error: %s", pipelineRun.Name, err.Error())
// 					return err
// 				}
// 				logging.Log.Infof("Deleted PipelineRun %s", pipelineRun.Name)
// 			}
// 		}
// 	}
// 	if !found {
// 		logging.Log.Infof("No matching PipelineRuns found")
// 	}
// 	return nil
// }

// // Retrieve webhook entry from configmap for the GitHub URL
// func (r Resource) getGitHubWebhooksFromConfigMap(gitURL *url.URL) ([]webhook, error) {
// 	if gitURL == nil {
// 		err := xerrors.New("Invalid URL")
// 		return nil, err
// 	}
// 	logging.Log.Debugf("Getting GitHub webhooks for repository URL %s", gitURL)

// 	sources, err := r.readGitHubWebhooksFromConfigMap()
// 	if err != nil {
// 		return []webhook{}, err
// 	}
// 	gitRepoURL = strings.ToLower(strings.TrimSuffix(gitRepoURL, ".git"))
// 	if sources[gitRepoURL] != nil {
// 		return sources[gitRepoURL], nil
// 	}

// 	return []webhook{}, fmt.Errorf("could not find webhook with GitRepositoryURL: %s", gitRepoURL)

// }

// // getGitHubWebhooksConfigMap returns the ConfigMap which stores information for
// // GitHub webhooks
// func (r Resource) getGitHubWebhooksConfigMap() (*corev1.ConfigMap, error) {
// 	return r.K8sClient.CoreV1().ConfigMaps(r.Defaults.Namespace).Get(ConfigMapName, metav1.GetOptions{})
// }

// func (r Resource) writeGitHubWebhooks(sources map[string][]webhook) error {
// 	logging.Log.Debugf("In writeGitHubWebhooks")
// 	configMapClient := r.K8sClient.CoreV1().ConfigMaps(r.Defaults.Namespace)
// 	configMap, err := configMapClient.Get(ConfigMapName, metav1.GetOptions{})
// 	var create = false
// 	if err != nil {
// 		configMap = &corev1.ConfigMap{
// 			ObjectMeta: metav1.ObjectMeta{
// 				Name:      ConfigMapName,
// 				Namespace: r.Defaults.Namespace,
// 			},
// 		}
// 		configMap.BinaryData = make(map[string][]byte)
// 		create = true
// 	}
// 	buf, err := json.Marshal(sources)
// 	if err != nil {
// 		logging.Log.Errorf("error marshalling GitHub webhooks: %s.", err.Error())
// 		return err
// 	}
// 	configMap.BinaryData["GitHubSource"] = buf
// 	if create {
// 		_, err = configMapClient.Create(configMap)
// 		if err != nil {
// 			logging.Log.Errorf("error creating configmap for GitHub webhooks: %s.", err.Error())
// 			return err
// 		}
// 	} else {
// 		_, err = configMapClient.Update(configMap)
// 		if err != nil {
// 			logging.Log.Errorf("error updating configmap for GitHub webhooks: %s.", err.Error())
// 		}
// 	}
// 	return nil
// }

// func (r Resource) getDefaults(request *restful.Request, response *restful.Response) {
// 	logging.Log.Debugf("getDefaults returning: %v", r.Defaults)
// 	response.WriteEntity(r.Defaults)
// }

// func (r Resource) createOpenshiftRoute() (*routev1.Route, error) {
// 	return r.RoutesClient.RouteV1().Routes(r.Defaults.Namespace).Create(
// 		&routev1.Route{},
// 	)
// }

// func (r Resource) deleteOpenshiftRoute() (*routev1.Route, error) {
// 	return r.RoutesClient.RouteV1().Routes(r.Defaults.Namespace).Delete(
// 		&routev1.Route{},
// 	)
// }
