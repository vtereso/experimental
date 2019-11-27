package router

import (
	"net/http"
	"os"

	restful "github.com/emicklei/go-restful"
	"github.com/tektoncd/experimental/webhooks-extension/pkg/client"
	"github.com/tektoncd/experimental/webhooks-extension/pkg/endpoints"
	logging "github.com/tektoncd/experimental/webhooks-extension/pkg/logging"
)

const (
	// webDirEnvKey is the environment key for the web directory environment
	// variable
	webDirEnvKey = "WEB_RESOURCES_DIR"
)

// New registers endpoints and returns an http.Handler
func New(cg client.Group) http.Handler {
	wsContainer := restful.NewContainer()
	registerWeb(wsContainer)
	registerExtensionWebService(wsContainer, cg)
	registerLivenessWebService(wsContainer)
	registerReadinessWebService(wsContainer)
	return wsContainer
}

// registerLivenessWebService registers the liveness web service
func registerLivenessWebService(container *restful.Container) {
	ws := new(restful.WebService)
	ws.Path("/liveness")
	ws.Route(ws.GET("/").To(endpoints.CheckHealth))
	container.Add(ws)
}

// registerReadinessWebService registers the readiness web service
func registerReadinessWebService(container *restful.Container) {
	ws := new(restful.WebService)
	ws.Path("/readiness")
	ws.Route(ws.GET("/").To(endpoints.CheckHealth))
	container.Add(ws)
}

// registerExtensionWebService registers the webhook webservice, which consumes
// and produces JSON
func registerExtensionWebService(container *restful.Container, cg *client.Group) {
	ws := new(restful.WebService)
	ws.
		Path("/webhooks").
		Consumes(restful.MIME_JSON, restful.MIME_JSON).
		Produces(restful.MIME_JSON, restful.MIME_JSON)

	// /webhooks/
	ws.Route(ws.POST("/").To(routeFunctionWithClientGroup(endpoints.CreateWebhook)))
	ws.Route(ws.GET("/").To(routeFunctionWithClientGroup(endpoints.GetAllWebhooks)))

	// /webhooks/{name}
	ws.Route(ws.DELETE("/{name}").To(routeFunctionWithClientGroup(endpoints.DeleteWebhook)))

	// /webhooks/credentials
	ws.Route(ws.POST("/credentials").To(routeFunctionWithClientGroup(endpoints.CreateCredential)))
	ws.Route(ws.GET("/credentials").To(routeFunctionWithClientGroup(endpoints.GetAllCredentials)))

	// /webhooks/credentials/{name}
	ws.Route(ws.DELETE("/credentials/{name}").To(routeFunctionWithClientGroup(endpoints.DeleteCredential)))

	container.Add(ws)
}

// registerWeb registers the extension web bundle on the container
func registerWeb(container *restful.Container) {
	var handler http.Handler
	webResourcesDir := os.Getenv(webDirEnvKey)
	if _, err := os.Stat(webResourcesDir); err != nil {
		logging.Log.Fatalf("registerWeb() %s", err)
	}
	logging.Log.Infof("Serving from web bundle from %s", webResourcesDir)
	handler = http.FileServer(http.Dir(webResourcesDir))
	container.Handle("/web/", http.StripPrefix("/web/", handler))
}

// routeFunctionClientGroup returns a RouteFunction that redirects to a
// RouteFunction with an additional client group parameter
func routeFunctionWithClientGroup(cg *client.Group, redirect func(*restful.Request, *restful.Response, *client.Group)) restful.RouteFunction {
	return func(req *restful.Request, resp *restful.Response) {
		redirect(req, resp, cg)
	}
}
