package utils

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"math/rand"

	"github.com/google/go-cmp/cmp"

	"golang.org/x/xerrors"
)

func TestRespondError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		statusCode int
		response   *httptest.ResponseRecorder
	}{
		{
			name:       "Status 300",
			err:        xerrors.New("300 Status"),
			statusCode: 300,
			response:   newTestResponseRecorder([]byte("300 Status"), 300),
		},
		{
			name:       "Status 301",
			err:        xerrors.New("301 Status"),
			statusCode: 301,
			response:   newTestResponseRecorder([]byte("301 Status"), 301),
		},
		{
			name:       "Status 302",
			err:        xerrors.New("302 Status"),
			statusCode: 302,
			response:   newTestResponseRecorder([]byte("302 Status"), 302),
		},
	}
	for i := range tests {
		t.Run(tests[i].name, func(t *testing.T) {
			response := httptest.NewRecorder()
			RespondError(response, tests[i].err, tests[i].statusCode)
			if diff := cmp.Diff(tests[i].response.Code, response.Code); diff != "" {
				t.Errorf("Response code mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tests[i].response.Body.String(), response.Body.String()); diff != "" {
				t.Errorf("Response body mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestWriteResponseLocation(t *testing.T) {
	tests := []struct {
		name       string
		identifier string
		req        *http.Request
		response   *httptest.ResponseRecorder
	}{
		// Correct
		{
			name:       "POST Request",
			identifier: "1",
			req:        httptest.NewRequest(http.MethodPost, "/some/path", nil),
			response: withHeader(
				newTestResponseRecorder(nil, http.StatusCreated),
				"Content-Location",
				"/some/path/1"),
		},
		// Incorrect
		{
			name:       "GET Request",
			identifier: "1",
			req:        httptest.NewRequest(http.MethodGet, "/some/path", nil),
			response:   newTestResponseRecorder(nil, http.StatusOK),
		},
		{
			name:       "PUT Request",
			identifier: "1",
			req:        httptest.NewRequest(http.MethodPut, "/some/path", nil),
			response:   newTestResponseRecorder(nil, http.StatusOK),
		},
		{
			name:       "DELETE Request",
			identifier: "1",
			req:        httptest.NewRequest(http.MethodDelete, "/some/path", nil),
			response:   newTestResponseRecorder(nil, http.StatusOK),
		},
	}
	for i := range tests {
		t.Run(tests[i].name, func(t *testing.T) {
			response := httptest.NewRecorder()
			WriteResponseLocation(tests[i].req, response, tests[i].identifier)
			if diff := cmp.Diff(tests[i].response.Code, response.Code); diff != "" {
				t.Errorf("Response code mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tests[i].response.Body.String(), response.Body.String()); diff != "" {
				t.Errorf("Response body mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestGetRandomToken(t *testing.T) {
	src := rand.NewSource(0)
	tests := []struct {
		name  string
		bytes []byte
	}{
		{
			name:  "Random Token",
			bytes: []byte("sJyQs22cRR81AZcI3qh2"),
		},
		{
			name:  "Another Random Token",
			bytes: []byte("Ze7gKS3PSbsRMjIFYHmz"),
		},
	}
	for i := range tests {
		t.Run(tests[i].name, func(t *testing.T) {
			bytes := GetRandomToken(src)
			if diff := cmp.Diff(tests[i].bytes, bytes); diff != "" {
				t.Errorf("getRandomToken() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// withHeader adds a header to the response and returns it back
func withHeader(r *httptest.ResponseRecorder, key, value string) *httptest.ResponseRecorder {
	r.Header().Add(key, value)
	return r
}

// newTestResponseRecorder creates a new response recording with the body and
// status code set
func newTestResponseRecorder(buf []byte, statusCode int) *httptest.ResponseRecorder {
	r := httptest.NewRecorder()
	r.WriteHeader(statusCode)
	if buf != nil {
		r.Write(buf)
	}
	return r
}
