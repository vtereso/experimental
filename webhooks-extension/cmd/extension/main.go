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

package main

import (
	"net/http"
	"os"

	logging "github.com/tektoncd/experimental/webhooks-extension/pkg/logging"
	"github.com/tektoncd/experimental/webhooks-extension/pkg/restapi"
)

func main() {
	logging.Log.Info("Registering all endpoints")
	cg, err := restapi.NewGroup()
	if err != nil {
		logging.Log.Fatal(err)
	}

	h := restapi.NewRouter(cg)

	port := ":8080"
	portnum := os.Getenv("PORT")
	if portnum != "" {
		port = ":" + portnum
		logging.Log.Infof("Port number from config: %s", portnum)
	}

	logging.Log.Info("Creating server and entering wait loop.")
	server := &http.Server{Addr: port, Handler: h}
	logging.Log.Fatal(server.ListenAndServe())
}
