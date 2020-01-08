package restapi

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestCheckHealth(t *testing.T) {
	tests := []struct {
		name       string
		url        string
		statusCode int
	}{
		{
			name:       "Regular Path",
			url:        "/liveness/",
			statusCode: 204,
		},
		{
			name:       "Regular Path",
			url:        "/readiness/",
			statusCode: 204,
		},
	}
	for i := range tests {
		t.Run(tests[i].name, func(t *testing.T) {
			server := DummyServer(DummyGroup())
			httpReq := DummyHTTPRequest("GET", fmt.Sprintf("%s%s", server.URL, tests[i].url), nil)
			// Make request
			response, err := http.DefaultClient.Do(httpReq)
			if err != nil {
				t.Fatalf("Error on request: %s", err)
			}
			// Compare
			if diff := cmp.Diff(tests[i].statusCode, response.StatusCode); diff != "" {
				t.Errorf("Status code mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
