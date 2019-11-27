package endpoints

import (
	"net/http"
	"testing"

	restful "github.com/emicklei/go-restful"
	"github.com/google/go-cmp/cmp"
)

func TestCheckHealth(t *testing.T) {
	response := restful.NewResponse()
	CheckHealth(restful.NewRequest(), response)
	if diff := cmp.Diff(http.StatusNoContent, response.StatusCode); diff != "" {
		t.Errorf("Status code mismatch (-want +got):\n%s", diff)
	}
}
