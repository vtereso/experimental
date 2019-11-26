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

package utils

import (
	"fmt"
	"net/http"

	restful "github.com/emicklei/go-restful"
	logging "github.com/tektoncd/dashboard/pkg/logging"
)

// RespondError logs the error, sets the status status code and writes the error
// string as plain text to the response
func RespondError(response http.ResponseWriter, err error, statusCode int) {
	logging.Log.Error(err)
	response.WriteHeader(statusCode)
	response.Header().Add("Content-Type", "text/plain")
	response.Write([]byte(err.Error()))
}

// WriteResponseLocation sets the http response "Content-Location" header and
// sets the status code to 201 for POST requests
func WriteResponseLocation(request *restful.Request, response *restful.Response, identifier string) {
	if request.Request.Method != http.MethodPost {
		return
	}
	location := fmt.Sprintf("%s/%s", request.Request.URL.Path, identifier)
	response.AddHeader("Content-Location", location)
	response.WriteHeader(http.StatusCreated)
}
