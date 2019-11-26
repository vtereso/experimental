package router

import (
	"net/http"
	"os"

	restful "github.com/emicklei/go-restful"
	"github.com/tektoncd/experimental/webhooks-extension/pkg/endpoints"
	logging "github.com/tektoncd/experimental/webhooks-extension/pkg/logging"
)

const (
	webDirEnvKey = "WEB_RESOURCES_DIR"
)

// New registers endpoints using the specified resource and returns an
// http.Handler
func New(resource endpoints.Resource) http.Handler {
	wsContainer := restful.NewContainer()
	registerWeb(wsContainer)
	registerExtensionWebService(wsContainer, resource)
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
func registerExtensionWebService(container *restful.Container, r endpoints.Resource) {
	ws := new(restful.WebService)
	ws.
		Path("/webhooks").
		Consumes(restful.MIME_JSON, restful.MIME_JSON).
		Produces(restful.MIME_JSON, restful.MIME_JSON)

	ws.Route(ws.POST("/").To(r.CreateWebhook))
	ws.Route(ws.GET("/").To(r.GetAllWebhooks))
	ws.Route(ws.DELETE("/{name}").To(r.DeleteWebhook))

	ws.Route(ws.POST("/credentials").To(r.CreateCredential))
	ws.Route(ws.GET("/credentials").To(r.GetAllCredentials))
	ws.Route(ws.DELETE("/credentials/{name}").To(r.DeleteCredential))

	container.Add(ws)
}

// registerWeb registers the extension web bundle on the container
func registerWeb(container *restful.Container) {
	var handler http.Handler
	webResourcesDir := os.Getenv(webDirEnvKey)
	if _, err := os.Stat(webResourcesDir); err != nil {
		logging.Log.Fatal(err)
	}
	logging.Log.Info("Serving from web bundle from %s", webResourcesDir)
	handler = http.FileServer(http.Dir(webResourcesDir))
	container.Handle("/web/", http.StripPrefix("/web/", handler))
}
