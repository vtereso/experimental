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
	"net/http/httptest"

	"github.com/tektoncd/experimental/webhooks-extension/pkg/client"
	fakeclient "github.com/tektoncd/experimental/webhooks-extension/pkg/client/fake"
	"github.com/tektoncd/experimental/webhooks-extension/pkg/router"
)

// DummyHTTPRequest reurns a new http with the specified method, url and body.
// The content type is also set to JSON
func DummyHTTPRequest(method string, url string, body io.Reader) *http.Request {
	httpReq := httptest.NewRequest(method, url, body)
	httpReq.Header.Set("Content-Type", "application/json")
	return httpReq
}

// DummyServer return a new httptest server and the client group used within
func DummyServer() (*httptest.Server, *client.Group) {
	cg := fakeclient.DummyGroup()
	return httptest.NewServer(router.New(cg)), cg
}
