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

package client

import (
	"os"

	"golang.org/x/xerrors"

	routeclientset "github.com/openshift/client-go/route/clientset/versioned"
	logging "github.com/tektoncd/experimental/webhooks-extension/pkg/logging"
	tektoncdclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	triggersclientset "github.com/tektoncd/triggers/pkg/client/clientset/versioned"
	k8sclientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	// installNamespace is the ENV for the installed namespace
	installNamespace = "INSTALLED_NAMESPACE"
	// callbackURL is the ENV for the callback URL
	callbackURL = "WEBHOOK_CALLBACK_URL"
	// platform is the ENV for the platform
	platform = "PLATFORM"
)

// Group is a group of clients with environment defaults
type Group struct {
	TektonClient   tektoncdclientset.Interface
	K8sClient      k8sclientset.Interface
	TriggersClient triggersclientset.Interface
	RoutesClient   routeclientset.Interface
	Defaults       EnvDefaults
}

// EnvDefaults are the environment defaults
type EnvDefaults struct {
	Namespace   string `json:"namespace"`
	CallbackURL string `json:"endpointurl"`
	Platform    string `json:"platform"`
}

// NewGroup returns a new Group
func NewGroup() (*Group, error) {
	// Get cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		logging.Log.Errorf("Error getting in cluster config: %s.", err.Error())
		return nil, err
	}

	tektonClient, err := tektoncdclientset.NewForConfig(config)
	if err != nil {
		logging.Log.Errorf("Error building tekton clientset: %s.", err.Error())
		return nil, err
	}

	k8sClient, err := k8sclientset.NewForConfig(config)
	if err != nil {
		logging.Log.Errorf("Error building k8s clientset: %s.", err.Error())
		return nil, err
	}

	triggersClient, err := triggersclientset.NewForConfig(config)
	if err != nil {
		logging.Log.Errorf("Error building triggers clientset: %s.", err.Error())
		return nil, err
	}

	// Currently Openshift does not have a top level client, but instead one for
	// each apiGroup
	routesClient, err := routeclientset.NewForConfig(config)
	if err != nil {
		logging.Log.Errorf("Error building routes clientset: %s.", err.Error())
		return nil, err
	}

	defaults := EnvDefaults{
		Namespace:   os.Getenv(installNamespace),
		CallbackURL: os.Getenv(callbackURL),
		Platform:    os.Getenv(platform),
	}

	if defaults.Namespace == "" {
		return nil, xerrors.Errorf("%s env value not found", installNamespace)
	}

	g := &Group{
		K8sClient:      k8sClient,
		TektonClient:   tektonClient,
		TriggersClient: triggersClient,
		RoutesClient:   routesClient,
		Defaults:       defaults,
	}
	return g, nil
}
