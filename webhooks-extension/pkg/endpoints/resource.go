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
	"errors"
	"os"

	routeclientset "github.com/openshift/client-go/route/clientset/versioned"
	logging "github.com/tektoncd/experimental/webhooks-extension/pkg/logging"
	tektoncdclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	triggersclientset "github.com/tektoncd/triggers/pkg/client/clientset/versioned"
	k8sclientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// Resource stores all types here that are reused throughout files
type Resource struct {
	TektonClient   tektoncdclientset.Interface
	K8sClient      k8sclientset.Interface
	TriggersClient triggersclientset.Interface
	RoutesClient   routeclientset.Interface
	Defaults       EnvDefaults
}

// NewResource returns a new Resource instantiated with its clientsets
func NewResource() (*Resource, error) {
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

	// Currently Openshift does not have a top level client, but instead one for each apiGroup
	routesClient, err := routeclientset.NewForConfig(config)
	if err != nil {
		logging.Log.Errorf("Error building routes clientset: %s.", err.Error())
		return nil, err
	}

	defaults := EnvDefaults{
		Namespace:   os.Getenv("INSTALLED_NAMESPACE"),
		CallbackURL: os.Getenv("WEBHOOK_CALLBACK_URL"),
		Platform:    os.Getenv("PLATFORM"),
	}

	if defaults.Namespace == "" {
		return nil, errors.New("INSTALLED_NAMESPACE env value not found")
	}

	r := &Resource{
		K8sClient:      k8sClient,
		TektonClient:   tektonClient,
		TriggersClient: triggersClient,
		RoutesClient:   routesClient,
		Defaults:       defaults,
	}
	return r, nil
}

// EnvDefaults are the environment defaults
type EnvDefaults struct {
	Namespace   string `json:"namespace"`
	CallbackURL string `json:"endpointurl"`
	Platform    string `json:"platform"`
}
