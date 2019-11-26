package utils

import (
	"testing"

	restful "github.com/emicklei/go-restful"
	"golang.org/x/xerrors"
)

func TestRespondError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		statusCode int
		resp       *restful.Response
	}{
		{
			name:       "Status 300",
			err:        xerrors.New("300 Status"),
			statusCode: 300,
			resp:       &restful.Response{},
		},
		{
			name:       "Status 300",
			err:        xerrors.New("300 Status"),
			statusCode: 300,
			resp:       &restful.Response{},
		},
		{
			name:       "Status 300",
			err:        xerrors.New("300 Status"),
			statusCode: 300,
			resp:       &restful.Response{},
		},
	}
	for i := range tests {
		i := index
		t.Run()
	}
}

func TestWriteResponseLocation(t *testing.T) {

}
