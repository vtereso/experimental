package router

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	restful "github.com/emicklei/go-restful"
	"github.com/google/go-cmp/cmp"
	"github.com/tektoncd/experimental/webhooks-extension/pkg/client"
	fakeclient "github.com/tektoncd/experimental/webhooks-extension/pkg/client/fake"
)

var (
	// expectedExtensionRoutes are the routes registered by
	// registerExtensionWebService, which serve the RESTful backend
	expectedExtensionRoutes = map[string][]string{
		"/webhooks/": []string{
			http.MethodGet,
			http.MethodPost,
		},
		"/webhooks/{name}": []string{
			http.MethodDelete,
		},
		"/webhooks/credentials": []string{
			http.MethodPost,
			http.MethodGet,
		},
		"/webhooks/credentials/{name}": []string{
			http.MethodDelete,
		},
	}
	// expectedLivenessRoute is the route registered by
	// registerLivenessWebService, which serves the liveness endpoint
	expectedLivenessRoute = map[string][]string{
		"/liveness/": []string{
			http.MethodGet,
		},
	}
	// expectedReadinessRoute is the route registered by
	// registerReadinessWebService, which serves the readiness endpoint
	expectedReadinessRoute = map[string][]string{
		"/readiness/": []string{
			http.MethodGet,
		},
	}
	// expectedWebRoute is the route registered bY registerWeb, which serves
	// the web bundle
	expectedWebRoute = map[string][]string{
		"/web/": []string{
			http.MethodGet,
		},
	}
)

func init() {
	// Set the webDirEnvKey env so the stats checks pass
	// The value can be any file/dir within this directory
	os.Setenv(webDirEnvKey, "router.go")
}

func TestRegister(t *testing.T) {
	handler := New(fakeclient.DummyGroup())
	container, ok := handler.(*restful.Container)
	if !ok {
		t.Fatalf("Underlying handler type was not restful.Container")
	}
	mux := container.ServeMux
	for _, registeredRoutes := range []map[string][]string{
		expectedExtensionRoutes,
		expectedLivenessRoute,
		expectedReadinessRoute,
		expectedWebRoute,
	} {
		for path, methods := range registeredRoutes {
			for _, method := range methods {
				if _, pattern := mux.Handler(httptest.NewRequest(method, path, nil)); pattern == "" {
					t.Errorf("Route %s %s not found", method, path)
				}
			}
		}
	}
}

func Test_registerWeb(t *testing.T) {
	wsContainer := restful.NewContainer()
	registerWeb(wsContainer)
	mux := wsContainer.ServeMux
	if _, pattern := mux.Handler(httptest.NewRequest("", "/web/", nil)); pattern == "" {
		t.Errorf("File server was not located")
	}
}

func Test_registerExtensionWebService(t *testing.T) {
	wsContainer := restful.NewContainer()
	registerExtensionWebService(wsContainer, fakeclient.DummyGroup())
	webServices := wsContainer.RegisteredWebServices()

	if diff := cmp.Diff(1, len(webServices)); diff != "" {
		t.Fatalf("Webservice count mismatch (-want +got):\n%s", diff)
	}
	for _, route := range webServices[0].Routes() {
		checkExpectedRoute(t, expectedExtensionRoutes, route)
		checkJSONConsumer(t, route)
	}

}

func Test_registerLivenessWebService(t *testing.T) {
	wsContainer := restful.NewContainer()
	registerLivenessWebService(wsContainer)
	webServices := wsContainer.RegisteredWebServices()

	if diff := cmp.Diff(1, len(webServices)); diff != "" {
		t.Fatalf("Webservice count mismatch (-want +got):\n%s", diff)
	}
	for _, route := range webServices[0].Routes() {
		checkExpectedRoute(t, expectedLivenessRoute, route)
	}
}

func Test_registerReadinessWebService(t *testing.T) {
	wsContainer := restful.NewContainer()
	registerReadinessWebService(wsContainer)
	webServices := wsContainer.RegisteredWebServices()

	if diff := cmp.Diff(1, len(webServices)); diff != "" {
		t.Fatalf("Webservice count mismatch (-want +got):\n%s", diff)
	}
	for _, route := range webServices[0].Routes() {
		checkExpectedRoute(t, expectedReadinessRoute, route)
	}
}

func Test_routeFunctionWithClientGroup(t *testing.T) {
	redirect := func(_ *restful.Request, resp *restful.Response, _ *client.Group) {
		resp.WriteHeader(http.StatusNoContent)
	}
	routeFunction := routeFunctionWithClientGroup(fakeclient.DummyGroup(), redirect)
	response := restful.NewResponse()
	routeFunction(restful.NewRequest(), response)
	if diff := cmp.Diff(http.StatusNoContent, response.StatusCode); diff != "" {
		t.Fatalf("Status code mismatch (-want +got):\n%s", diff)
	}
}

// checkExpectedRoute checks for the existance the specified route method and
// path within the specified map, which maps paths to http methods
func checkExpectedRoute(t *testing.T, muxPath map[string][]string, route restful.Route) {
	t.Helper()
	methods, pathFound := muxPath[route.Path]
	if !pathFound {
		t.Errorf("Route path %s not found", route.Path)
	}
	for _, method := range methods {
		if method == route.Method {
			return
		}
	}
	t.Errorf("Route method %s not found with path %s", route.Method, route.Path)
}

// checkJSONConsumer checks that the specified route has a JSON consumption type
func checkJSONConsumer(t *testing.T, route restful.Route) {
	t.Helper()
	var hasJSONMIME bool
	for _, mime := range route.Consumes {
		if mime == restful.MIME_JSON {
			hasJSONMIME = true
			break
		}
	}
	if !hasJSONMIME {
		t.Errorf("Route '%s' does not have a JSON consumption MIME type specified", route)
	}
}
