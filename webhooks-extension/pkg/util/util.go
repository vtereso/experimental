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

package util

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"

	"golang.org/x/oauth2"

	logging "github.com/tektoncd/experimental/webhooks-extension/pkg/logging"
)

const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
	letterBytes   = "123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
)

// GetRandomToken generates a random 20-character secret using a random source
// Source: https://stackoverflow.com/questions/22892120/how-to-generate-a-random-string-of-a-fixed-length-in-go
func GetRandomToken(src rand.Source) []byte {
	n := 20
	b := make([]byte, n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}
	return b
}

// CreateOAuth2Client returns an HTTP client with oauth2 authentication using
// the provided accessToken
func CreateOAuth2Client(ctx context.Context, accessToken string) *http.Client {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: accessToken})
	return oauth2.NewClient(ctx, ts)
}

// RespondError logs the error, sets the status status code and writes the error
// string as plain text to the response. The caller should return
// immediately after
func RespondError(response http.ResponseWriter, err error, statusCode int) {
	logging.Log.Error(err)
	response.WriteHeader(statusCode)
	response.Header().Add("Content-Type", "text/plain")
	response.Write([]byte(err.Error()))
}

// WriteResponseLocation sets the http response "Content-Location" header and
// sets the status code to 201 for POST requests. The caller should return
// immediately after
func WriteResponseLocation(request *http.Request, response http.ResponseWriter, identifier string) {
	if request.Method != http.MethodPost {
		return
	}
	location := fmt.Sprintf("%s/%s", request.URL.Path, identifier)
	response.Header().Add("Content-Location", location)
	response.WriteHeader(http.StatusCreated)
}
