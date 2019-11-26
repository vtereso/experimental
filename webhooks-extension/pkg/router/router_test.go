package router

import (
	"net/http"
	"testing"

	restful "github.com/emicklei/go-restful"
	"github.com/google/go-cmp/cmp"
	"github.com/tektoncd/experimental/webhooks-extension/pkg/testutils"
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
)

func TestRegister(t *testing.T) {
	handler := New(testutils.DummyResource())
	container, ok := handler.(*restful.Container)
	if !ok {
		t.Fatalf("Underlying handler type was not restful.Container")
	}
	expectedRoutes := map[string][]string{}
	for _, registeredRoutes := range []map[string][]string{
		expectedExtensionRoutes,
		expectedLivenessRoute,
		expectedReadinessRoute,
	} {
		for path, methods := range registeredRoutes {
			expectedRoutes[path] = methods
		}
	}
	for _, ws := range container.RegisteredWebServices() {
		for _, route := range ws.Routes() {
			checkExpectedRoute(t, expectedExtensionRoutes, route)
		}
	}

}

// func Test_registerWeb(t *testing.T) {
// 	wsContainer := restful.NewContainer()
// 	registerWeb(wsContainer)
// }

func Test_registerExtensionWebService(t *testing.T) {
	wsContainer := restful.NewContainer()
	registerExtensionWebService(wsContainer, testutils.DummyResource())
	webServices := wsContainer.RegisteredWebServices()

	if diff := cmp.Diff(1, len(webServices)); diff != "" {
		t.Fatalf("registerExtensionWebService() webservice count mismatch (-want +got):\n%s", diff)
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
