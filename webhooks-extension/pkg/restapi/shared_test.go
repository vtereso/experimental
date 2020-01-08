package restapi

import (
	"io"
	"net/http"
	"net/http/httptest"

	fakerouteclientset "github.com/openshift/client-go/route/clientset/versioned/fake"
	faketektonclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned/fake"
	faketriggerclientset "github.com/tektoncd/triggers/pkg/client/clientset/versioned/fake"
	fakek8sclientset "k8s.io/client-go/kubernetes/fake"
)

// DummyGroup returns a group using fake clients and defaults
func DummyGroup() *Group {
	return &Group{
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

func dummyDefaults() EnvDefaults {
	return EnvDefaults{
		Namespace:  "default",
		Platform:   "openshift",
		SSLEnabled: "false",
	}
}

// DummyHTTPRequest reurns a new http with the specified method, url and body.
// The content type is also set to JSON
func DummyHTTPRequest(method string, url string, body io.Reader) *http.Request {
	httpReq := httptest.NewRequest(method, url, body)
	httpReq.Header.Set("Content-Type", "application/json")
	return httpReq
}

// DummyServer return a new httptest server and the client group used within
func DummyServer(cg *Group) *httptest.Server {
	return httptest.NewServer(NewRouter(cg))
}
