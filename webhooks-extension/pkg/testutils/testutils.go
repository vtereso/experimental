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

package testutils

import (
	"io"
	"net/http"

	fakerouteclientset "github.com/openshift/client-go/route/clientset/versioned/fake"
	"github.com/tektoncd/experimental/webhooks-extension/pkg/endpoints"
	faketektonclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned/fake"
	faketriggerclientset "github.com/tektoncd/triggers/pkg/client/clientset/versioned/fake"
	fakek8sclientset "k8s.io/client-go/kubernetes/fake"
)

// DummyResource returns a resource using fake clients
func DummyResource() endpoints.Resource {
	return endpoints.Resource{
		K8sClient:      dummyK8sClientset(),
		TektonClient:   dummyTektonClientset(),
		TriggersClient: dummyTriggersClientset(),
		RoutesClient:   dummyRoutesClientset(),
		Defaults:       dummyDefaults(),
	}
}

func dummyK8sClientset() *fakek8sclientset.Clientset {
	return fakek8sclientset.NewSimpleClientset()
}

func dummyTektonClientset() *faketektonclientset.Clientset {
	return faketektonclientset.NewSimpleClientset()
}

func dummyTriggersClientset() *faketriggerclientset.Clientset {
	return faketriggerclientset.NewSimpleClientset()
}

func dummyRoutesClientset() *fakerouteclientset.Clientset {
	return fakerouteclientset.NewSimpleClientset()
}

func dummyHTTPRequest(method string, url string, body io.Reader) *http.Request {
	httpReq, _ := http.NewRequest(method, url, body)
	httpReq.Header.Set("Content-Type", "application/json")
	return httpReq
}

func dummyDefaults() endpoints.EnvDefaults {
	return endpoints.EnvDefaults{
		Namespace: "default",
		Platform:  "openshift",
	}
}
