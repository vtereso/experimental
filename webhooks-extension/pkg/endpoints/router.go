package endpoints

import (
	"net/http"
	"os"

	restful "github.com/emicklei/go-restful"
	logging "github.com/tektoncd/experimental/webhooks-extension/pkg/logging"
)

const (
	// webDirEnvKey is the environment key for the web directory environment
	// variable
	webDirEnvKey = "WEB_RESOURCES_DIR"
)

// NewRouter registers endpoints and returns an http.Handler
func NewRouter(cg *Group) http.Handler {
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
	ws.Route(ws.GET("/").To(CheckHealth))
	container.Add(ws)
}

// registerReadinessWebService registers the readiness web service
func registerReadinessWebService(container *restful.Container) {
	ws := new(restful.WebService)
	ws.Path("/readiness")
	ws.Route(ws.GET("/").To(CheckHealth))
	container.Add(ws)
}

// registerExtensionWebService registers the webhook webservice, which consumes
// and produces JSON
func registerExtensionWebService(container *restful.Container, cg *Group) {
	ws := new(restful.WebService)
	ws.
		Path("/webhooks").
		Consumes(restful.MIME_JSON, restful.MIME_JSON).
		Produces(restful.MIME_JSON, restful.MIME_JSON)

	// /webhooks/
	ws.Route(ws.POST("/").To(cg.CreateWebhook))
	ws.Route(ws.GET("/").To(cg.GetAllWebhooks))

	// /webhooks/{name}
	ws.Route(ws.DELETE("/{name}").To(cg.DeleteWebhook))

	// /webhooks/credentials
	ws.Route(ws.POST("/credentials").To(cg.CreateCredential))
	ws.Route(ws.GET("/credentials").To(cg.GetAllCredentials))

	// /webhooks/credentials/{name}
	ws.Route(ws.DELETE("/credentials/{name}").To(cg.DeleteCredential))

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
