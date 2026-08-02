package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestProxyRejectsUnauthenticatedRequest(t *testing.T) {
	authServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"code":401}`))
	}))
	defer authServer.Close()

	server := &proxyServer{authURL: authServer.URL, authClient: &http.Client{Timeout: time.Second}}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/openchat/robot123/robot/ai-providers", nil)
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, response.Code)
	}
}

func TestProxyRejectsInvalidRobotCode(t *testing.T) {
	authServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Cookie") != "session=test" {
			t.Error("proxy did not forward the session cookie to the auth endpoint")
		}
		_, _ = w.Write([]byte(`{"code":200}`))
	}))
	defer authServer.Close()

	server := &proxyServer{authURL: authServer.URL, authClient: &http.Client{Timeout: time.Second}}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/openchat/../robot/ai-providers", nil)
	request.Header.Set("Cookie", "session=test")
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, response.Code)
	}
}
