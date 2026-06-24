package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func executeRequest(req *http.Request, s *Server) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	s.Router.ServeHTTP(rr, req)
	return rr
}

func checkResponseCode(t *testing.T, expected, actual int) {
	if expected != actual {
		t.Errorf("Expected response code %d. Got %d\n", expected, actual)
	}
}

func TestHelloWorld(t *testing.T) {
	s := CreateNewServer()
	s.MountHandlers()

	req, _ := http.NewRequest("GET", "/", nil)
	response := executeRequest(req, s)

	checkResponseCode(t, http.StatusOK, response.Code)
	require.Equal(t, "Hello World!", response.Body.String())
}
